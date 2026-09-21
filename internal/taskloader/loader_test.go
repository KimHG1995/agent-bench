package taskloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "id":"lookup-001",
  "category":"lookup",
  "repository":{"path":"fixture","revision":"v1"},
  "question":"where?",
  "expected":{"symbols":["A"]}
}`
	if err := os.WriteFile(filepath.Join(dir, "task.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tasks, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ID != "lookup-001" {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}
}

func TestLoadDirRejectsDuplicateIDs(t *testing.T) {
	dir := t.TempDir()
	content := `{"id":"same","category":"lookup","repository":{"path":"fixture"},"question":"q","expected":{}}`
	for _, name := range []string{"a.json", "b.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadDir(dir); err == nil {
		t.Fatal("expected duplicate id error")
	}
}
