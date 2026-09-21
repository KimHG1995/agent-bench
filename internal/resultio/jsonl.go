package resultio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func WriteJSONL(path string, results []domain.RunResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	enc := json.NewEncoder(w)
	for _, result := range results {
		if err := enc.Encode(result); err != nil {
			return err
		}
	}
	return nil
}

func ReadJSONL(path string) ([]domain.RunResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []domain.RunResult
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		var result domain.RunResult
		if err := json.Unmarshal(scanner.Bytes(), &result); err != nil {
			return nil, fmt.Errorf("decode %s line %d: %w", path, line, err)
		}
		out = append(out, result)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
