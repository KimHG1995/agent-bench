package report

import (
	"strings"
	"testing"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func TestMarkdownPreservesUnmeasuredMetrics(t *testing.T) {
	results := []domain.RunResult{{
		TaskID: "t1", Strategy: "baseline", Run: 1, Success: true, Accuracy: 1, DurationMS: 10,
	}}
	got := Markdown(results)
	if !strings.Contains(got, "| baseline | 1 | 100.0% | 1.000 | n/a | n/a | n/a | 10.0 ms |") {
		t.Fatalf("unexpected report:\n%s", got)
	}
}
