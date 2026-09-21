package taskloader

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func LoadDir(root string) ([]domain.Task, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(paths)
	tasks := make([]domain.Task, 0, len(paths))
	seen := map[string]struct{}{}
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var task domain.Task
		if err := json.Unmarshal(b, &task); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		if err := validate(task); err != nil {
			return nil, fmt.Errorf("validate %s: %w", path, err)
		}
		if _, ok := seen[task.ID]; ok {
			return nil, fmt.Errorf("duplicate task id %q", task.ID)
		}
		seen[task.ID] = struct{}{}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func validate(task domain.Task) error {
	if strings.TrimSpace(task.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(task.Category) == "" {
		return fmt.Errorf("category is required")
	}
	if strings.TrimSpace(task.Question) == "" {
		return fmt.Errorf("question is required")
	}
	if strings.TrimSpace(task.Repository.Path) == "" {
		return fmt.Errorf("repository.path is required")
	}
	return nil
}
