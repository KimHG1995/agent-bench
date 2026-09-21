package codexadapter

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func hashJSON(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func treeHash(root string, rejectLinks bool) (string, error) {
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	var records []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			if rejectLinks {
				return fmt.Errorf("target symlink is not supported: %s", rel)
			}
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			records = append(records, rel+"=symlink:"+link)
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("unsupported file: %s", rel)
		}
		sum, err := fileHash(path)
		if err != nil {
			return err
		}
		records = append(records, rel+"="+sum)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(records)
	return hashJSON(records), nil
}

var fullRevision = regexp.MustCompile(`^[0-9a-f]{40}$`)

func prepareTarget(ctx context.Context, source, revision, dest string) error {
	if !fullRevision.MatchString(revision) {
		return fmt.Errorf("target revision must be a full Git SHA")
	}
	git := func(args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", source}, args...)...)
		return cmd.Output()
	}
	b, err := git("rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("verify target: %w", err)
	}
	if strings.TrimSpace(string(b)) != revision {
		return fmt.Errorf("target revision differs from requested commit")
	}
	b, err = git("status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return err
	}
	if len(b) > 0 {
		return fmt.Errorf("target has uncommitted/untracked changes")
	}
	if err = os.MkdirAll(dest, 0700); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "git", "-C", source, "archive", revision)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	extractErr := extractArchive(r, dest)
	if extractErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if extractErr != nil {
		return extractErr
	}
	return waitErr
}

func extractArchive(r io.Reader, dest string) error {
	tarReader := tar.NewReader(r)
	for {
		h, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(filepath.FromSlash(h.Name))
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return fmt.Errorf("archive path escapes target")
		}
		if name == ".codex" || strings.HasPrefix(name, ".codex"+string(filepath.Separator)) || name == ".agents" || strings.HasPrefix(name, ".agents"+string(filepath.Separator)) {
			return fmt.Errorf("target-local Codex config/skills are not supported")
		}
		path := filepath.Join(dest, name)
		switch h.Typeflag {
		case tar.TypeXGlobalHeader:
			// git archive stores its commit id in a global PAX header.
			continue
		case tar.TypeDir:
			err = os.MkdirAll(path, 0700)
		case tar.TypeReg:
			if h.Size > 64*1024*1024 {
				return fmt.Errorf("target file exceeds 64 MiB")
			}
			if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return err
			}
			f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(h.Mode)&0700)
			if e != nil {
				return e
			}
			_, err = io.Copy(f, tarReader)
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
		default:
			return fmt.Errorf("unsupported target archive entry %s (symlinks and submodules are not supported)", h.Name)
		}
		if err != nil {
			return err
		}
	}
}

// Hash readable instructions/skills, never credential files. This is a drift
// detector, not proof that the CLI exposes only this context to the model.
func contextFingerprint() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		codexHome = filepath.Join(home, ".codex")
	}
	paths := []string{filepath.Join(home, ".agents", "skills"), filepath.Join(codexHome, "skills"), filepath.Join(codexHome, "AGENTS.md"), filepath.Join(codexHome, "AGENTS.override.md"), filepath.Join(codexHome, "rules")}
	var records []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			records = append(records, path+"=absent")
			continue
		}
		if err != nil {
			return "", err
		}
		var sum string
		if info.IsDir() {
			sum, err = treeHash(path, false)
		} else {
			sum, err = fileHash(path)
		}
		if err != nil {
			return "", err
		}
		records = append(records, path+"="+sum)
	}
	records = append(records, "environment="+hashJSON(safeEnv()))
	return hashJSON(records), nil
}
