package runner

import (
	"context"
	"testing"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

type stubAgent struct{}

type invalidAgent struct {
	kind  string
	calls *int
}

func (a invalidAgent) Run(_ context.Context, _ string, _ domain.RunRequest) (domain.AgentOutput, error) {
	*a.calls++
	return domain.AgentOutput{Status: domain.RunStatus{Execution: "completed", Measurement: "invalid", FailureKind: a.kind}}, nil
}

func TestInvalidMeasurementFailsAndFatalStops(t *testing.T) {
	calls := 0
	r := Runner{Agent: invalidAgent{kind: "quota", calls: &calls}, StopOnFatal: true}
	rows := r.Run([]domain.Task{{ID: "a"}, {ID: "b"}}, "graph", "ignored", 3)
	if calls != 1 || len(rows) != 1 || rows[0].Success || rows[0].Status.FailureKind != "quota" {
		t.Fatalf("invalid/fatal not handled: calls=%d rows=%#v", calls, rows)
	}
}

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
