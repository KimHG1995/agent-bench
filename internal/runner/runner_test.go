package runner

import (
	"context"
	"testing"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

type stubAgent struct{}

func (stubAgent) Run(_ context.Context, _ string, req domain.RunRequest) (domain.AgentOutput, error) {
	return domain.AgentOutput{Evidence: domain.Evidence{}}, nil
}

func TestRunWithOffsetPreservesRunNumbers(t *testing.T) {
	r := Runner{Agent: stubAgent{}}
	results := r.RunWithOffset([]domain.Task{
		{ID: "t1", Category: "lookup", Repository: domain.RepositoryRef{Path: "fixture"}},
	}, "baseline", "ignored", 2, 3)

	if len(results) != 2 {
		t.Fatalf("len=%d", len(results))
	}
	if results[0].Run != 4 || results[1].Run != 5 {
		t.Fatalf("run numbers=%d,%d", results[0].Run, results[1].Run)
	}
}
