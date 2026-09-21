package resultio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

type Writer struct {
	file *os.File
	buf  *bufio.Writer
	enc  *json.Encoder
}

func NewWriter(path string, overwrite bool) (*Writer, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return nil, err
	}
	buf := bufio.NewWriter(f)
	return &Writer{file: f, buf: buf, enc: json.NewEncoder(buf)}, nil
}

func (w *Writer) Write(result domain.RunResult) error {
	if err := w.enc.Encode(result); err != nil {
		return err
	}
	if err := w.buf.Flush(); err != nil {
		return err
	}
	return w.file.Sync()
}

func (w *Writer) Close() error {
	if w == nil || w.file == nil {
		return nil
	}
	if err := w.buf.Flush(); err != nil {
		_ = w.file.Close()
		return err
	}
	return w.file.Close()
}

func WriteJSONL(path string, results []domain.RunResult) error {
	w, err := NewWriter(path, true)
	if err != nil {
		return err
	}
	for _, result := range results {
		if err := w.Write(result); err != nil {
			_ = w.Close()
			return err
		}
	}
	return w.Close()
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
