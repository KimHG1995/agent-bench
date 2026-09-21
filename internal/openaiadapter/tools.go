package openaiadapter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func runTool(root string, tc toolCall) (any, error) {
	switch tc.Function.Name {
	case "list_files":
		var args struct{ Path string `json:"path"` }
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil { return nil, err }
		return listFiles(root, args.Path)
	case "search_text":
		var args struct { Query string `json:"query"`; Path string `json:"path"` }
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil { return nil, err }
		return searchText(root, args.Path, args.Query)
	case "read_file":
		var args struct { Path string `json:"path"`; StartLine int `json:"startLine"`; EndLine int `json:"endLine"` }
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil { return nil, err }
		return readFile(root, args.Path, args.StartLine, args.EndLine)
	default:
		return nil, fmt.Errorf("unsupported tool %q", tc.Function.Name)
	}
}

func rootResolved(root string) (string, error) {
	return filepath.EvalSymlinks(root)
}

func ensureWithinRoot(root, path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil { return "", err }
	check, err := filepath.Rel(root, resolved)
	if err != nil || check == ".." || strings.HasPrefix(check, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repository")
	}
	return resolved, nil
}

func safePath(root, rel string) (string, error) {
	base, err := rootResolved(root)
	if err != nil { return "", err }
	rel = filepath.Clean(strings.TrimSpace(rel))
	if rel == "." { rel = "" }
	if filepath.IsAbs(rel) { return "", fmt.Errorf("absolute paths are not allowed") }
	return ensureWithinRoot(base, filepath.Join(base, rel))
}

func skipDir(name string) bool {
	switch name {
	case ".git","node_modules","dist","build","coverage",".next":
		return true
	}
	return false
}

func listFiles(root, rel string) ([]string, error) {
	start, err := safePath(root, rel)
	if err != nil { return nil, err }
	base, err := rootResolved(root)
	if err != nil { return nil, err }
	var out []string
	err = filepath.WalkDir(start, func(path string, d os.DirEntry, err error) error {
		if err != nil { return err }
		if d.IsDir() {
			if path != start && skipDir(d.Name()) { return filepath.SkipDir }
			return nil
		}
		safe, err := ensureWithinRoot(base, path)
		if err != nil { return nil }
		r, _ := filepath.Rel(base, safe)
		out = append(out, filepath.ToSlash(r))
		if len(out) >= 300 { return filepath.SkipAll }
		return nil
	})
	sort.Strings(out)
	return out, err
}

type match struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

func searchText(root, rel, query string) ([]match, error) {
	if strings.TrimSpace(query) == "" { return nil, fmt.Errorf("query is required") }
	start, err := safePath(root, rel)
	if err != nil { return nil, err }
	base, err := rootResolved(root)
	if err != nil { return nil, err }
	var out []match
	err = filepath.WalkDir(start, func(path string, d os.DirEntry, err error) error {
		if err != nil { return err }
		if d.IsDir() {
			if path != start && skipDir(d.Name()) { return filepath.SkipDir }
			return nil
		}
		safe, err := ensureWithinRoot(base, path)
		if err != nil { return nil }
		info, err := os.Stat(safe)
		if err != nil || info.Size() > 1<<20 { return nil }
		f, err := os.Open(safe)
		if err != nil { return nil }
		s := bufio.NewScanner(f)
		line := 0
		for s.Scan() {
			line++
			if strings.Contains(s.Text(), query) {
				r, _ := filepath.Rel(base, safe)
				out = append(out, match{Path:filepath.ToSlash(r),Line:line,Text:strings.TrimSpace(s.Text())})
				if len(out) >= 50 { _ = f.Close(); return filepath.SkipAll }
			}
		}
		scanErr := s.Err()
		_ = f.Close()
		return scanErr
	})
	return out, err
}

func readFile(root, rel string, start, end int) (map[string]any, error) {
	path, err := safePath(root, rel)
	if err != nil { return nil, err }
	base, _ := rootResolved(root)
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	if start <= 0 { start = 1 }
	if end <= 0 || end < start { end = start + 199 }
	if end-start > 299 { end = start + 299 }
	var lines []string
	s := bufio.NewScanner(f)
	line := 0
	for s.Scan() {
		line++
		if line < start { continue }
		if line > end { break }
		lines = append(lines, fmt.Sprintf("%d: %s", line, s.Text()))
	}
	if err := s.Err(); err != nil { return nil, err }
	relPath, _ := filepath.Rel(base, path)
	return map[string]any{"path":filepath.ToSlash(relPath),"startLine":start,"endLine":start+len(lines)-1,"content":strings.Join(lines,"\n")}, nil
}
