package report

import (
	"fmt"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func ValidateComparable(results []domain.RunResult) error {
	type slot struct {
		baseline *domain.RunResult
		graph    *domain.RunResult
	}
	pairs := map[string]*slot{}
	for i := range results {
		r := &results[i]
		if r.Strategy != "baseline" && r.Strategy != "graph" {
			continue
		}
		key := fmt.Sprintf("%s#%d#%s", r.TaskID, r.Run, r.Experiment.ID)
		p := pairs[key]
		if p == nil {
			p = &slot{}
			pairs[key] = p
		}
		target := &p.baseline
		if r.Strategy == "graph" { target = &p.graph }
		if *target != nil {
			return fmt.Errorf("duplicate %s result for %s", r.Strategy, key)
		}
		*target = r
	}
	for key, p := range pairs {
		if p.baseline == nil || p.graph == nil || !p.baseline.Success || !p.graph.Success {
			continue
		}
		if err := samePair(*p.baseline, *p.graph); err != nil {
			return fmt.Errorf("incomparable pair %s: %w", key, err)
		}
	}
	return nil
}

func samePair(a, b domain.RunResult) error {
	if a.Experiment.TaskHash != b.Experiment.TaskHash { return fmt.Errorf("task hash differs") }
	if a.Repository.Revision != b.Repository.Revision { return fmt.Errorf("repository revision differs") }
	if a.Experiment.HarnessRevision != b.Experiment.HarnessRevision { return fmt.Errorf("harness revision differs") }
	if a.Experiment.GraphRevision != b.Experiment.GraphRevision { return fmt.Errorf("graph revision differs") }
	if a.Runtime.Provider != b.Runtime.Provider { return fmt.Errorf("provider differs") }
	if a.Runtime.Endpoint != b.Runtime.Endpoint { return fmt.Errorf("endpoint differs") }
	if a.Runtime.RequestedModel != b.Runtime.RequestedModel { return fmt.Errorf("requested model differs") }
	if a.Runtime.Model != b.Runtime.Model { return fmt.Errorf("actual model differs") }
	if a.Runtime.Adapter != b.Runtime.Adapter { return fmt.Errorf("adapter differs") }
	if a.Runtime.MaxTurns != b.Runtime.MaxTurns { return fmt.Errorf("max turns differs") }
	return nil
}
