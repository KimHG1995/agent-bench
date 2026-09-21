package codexadapter

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareTargetRejectsCommittedSubmodule(t *testing.T) {
	repo, revision := fixtureRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, "vendor/dep"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"update-index", "--add", "--cacheinfo", "160000," + revision + ",vendor/dep"},
		{"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "submodule fixture"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %s %v", b, err)
		}
	}
	b, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	err = prepareTarget(context.Background(), repo, strings.TrimSpace(string(b)), filepath.Join(t.TempDir(), "target"))
	if err == nil || !strings.Contains(err.Error(), "submodule") || !strings.Contains(err.Error(), "vendor/dep") {
		t.Fatalf("submodule source silently omitted: %v", err)
	}
}
