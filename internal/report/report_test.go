package report

import (
	"strings"
	"testing"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func ptr(v int64) *int64 { return &v }

func TestMarkdownPreservesUnmeasuredMetrics(t *testing.T) {
	results := []domain.RunResult{{
		TaskID: "t1", Strategy: "baseline", Run: 1, Success: true, Accuracy: 1, DurationMS: 10,
	}}
	got := Markdown(results)
	want := "| baseline | 1 | 100.0% | 1.000 / 1.000 | n/a | n/a | 10.0 / 10.0 ms |"
	if !strings.Contains(got, want) {
		t.Fatalf("unexpected report:\n%s", got)
	}
}

func TestMarkdownIncludesPairedComparison(t *testing.T) {
	results := []domain.RunResult{
		{TaskID: "t1", Strategy: "baseline", Run: 1, Success: true, Accuracy: 0.8, DurationMS: 100, Metrics: domain.AgentMetrics{ToolCalls: ptr(10), InputTokens: ptr(900), OutputTokens: ptr(100)}},
		{TaskID: "t1", Strategy: "graph", Run: 1, Success: true, Accuracy: 1.0, DurationMS: 50, Metrics: domain.AgentMetrics{ToolCalls: ptr(5), InputTokens: ptr(400), OutputTokens: ptr(100)}},
	}
	got := Markdown(results)
	for _, want := range []string{
		"| Accuracy delta | 1 | +0.200 |",
		"| Tool calls change | 1 | -50.0% |",
		"| I/O tokens change | 1 | -50.0% |",
		"| Latency change | 1 | -50.0% |",
		"| t1 | 0.800 | 1.000 | +0.200 |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in report:\n%s", want, got)
		}
	}
}

func TestPercentileUsesNearestRank(t *testing.T) {
	values := []float64{1, 2, 3, 4, 100}
	if got := percentile(values, 0.50); got != 3 {
		t.Fatalf("median=%v", got)
	}
	if got := percentile(values, 0.95); got != 100 {
		t.Fatalf("p95=%v", got)
	}
}
