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
		if r.Strategy == "graph" {
			target = &p.graph
		}
		if *target != nil {
			return fmt.Errorf("duplicate %s result for %s", r.Strategy, key)
		}
		*target = r
	}
	for key, p := range pairs {
		if p.baseline == nil || p.graph == nil || !eligible(*p.baseline) || !eligible(*p.graph) {
			continue
		}
		if err := samePair(*p.baseline, *p.graph); err != nil {
			return fmt.Errorf("incomparable pair %s: %w", key, err)
		}
	}
	return nil
}

func samePair(a, b domain.RunResult) error {
	if a.Experiment.TaskHash != b.Experiment.TaskHash {
		return fmt.Errorf("task hash differs")
	}
	if a.Repository.Revision != b.Repository.Revision {
		return fmt.Errorf("repository revision differs")
	}
	if a.Experiment.HarnessRevision != b.Experiment.HarnessRevision {
		return fmt.Errorf("harness revision differs")
	}
	if a.Experiment.GraphRevision != b.Experiment.GraphRevision {
		return fmt.Errorf("graph revision differs")
	}
	if a.Runtime.Provider != b.Runtime.Provider {
		return fmt.Errorf("provider differs")
	}
	if a.Runtime.Endpoint != b.Runtime.Endpoint {
		return fmt.Errorf("endpoint differs")
	}
	if a.Runtime.RequestedModel != b.Runtime.RequestedModel {
		return fmt.Errorf("requested model differs")
	}
	if a.Runtime.Model != b.Runtime.Model {
		return fmt.Errorf("actual model differs")
	}
	if a.Runtime.Adapter != b.Runtime.Adapter {
		return fmt.Errorf("adapter differs")
	}
	if a.Runtime.MaxTurns != b.Runtime.MaxTurns {
		return fmt.Errorf("max turns differs")
	}
	for _, field := range []struct{ name, a, b string }{
		{"effort", a.Runtime.Effort, b.Runtime.Effort},
		{"service tier", a.Runtime.ServiceTier, b.Runtime.ServiceTier},
		{"auth mode", a.Runtime.AuthMode, b.Runtime.AuthMode},
		{"CLI version", a.Runtime.CLIVersion, b.Runtime.CLIVersion},
		{"CLI binary", a.Runtime.CLIBinaryHash, b.Runtime.CLIBinaryHash},
		{"model provenance", a.Runtime.ModelProvenance, b.Runtime.ModelProvenance},
		{"common config", a.Runtime.CommonConfigHash, b.Runtime.CommonConfigHash},
		{"prompt template", a.Runtime.PromptTemplateHash, b.Runtime.PromptTemplateHash},
		{"schema", a.Runtime.SchemaHash, b.Runtime.SchemaHash},
		{"tool profile", a.Runtime.ToolProfile, b.Runtime.ToolProfile},
		{"sandbox", a.Runtime.SandboxProfile, b.Runtime.SandboxProfile},
		{"budget policy", a.Runtime.BudgetPolicy, b.Runtime.BudgetPolicy},
		{"graph fingerprint", a.Runtime.GraphFingerprint, b.Runtime.GraphFingerprint},
		{"context fingerprint", a.Runtime.ContextFingerprint, b.Runtime.ContextFingerprint},
		{"target snapshot", a.Runtime.TargetSnapshotHash, b.Runtime.TargetSnapshotHash},
	} {
		if field.a != field.b {
			return fmt.Errorf("%s differs", field.name)
		}
		if a.Runtime.Adapter == "codex-cli" && field.a == "" {
			return fmt.Errorf("Codex %s is missing", field.name)
		}
	}
	if a.Runtime.TimeoutMS != b.Runtime.TimeoutMS {
		return fmt.Errorf("timeout differs")
	}
	if a.Runtime.Adapter == "codex-cli" {
		if a.Runtime.TimeoutMS <= 0 {
			return fmt.Errorf("Codex time budget is missing")
		}
		if a.Runtime.ModelProvenance != "requested_only" && a.Runtime.ModelProvenance != "observed" {
			return fmt.Errorf("unknown Codex model provenance")
		}
		if a.Runtime.ModelProvenance == "observed" && a.Runtime.Model == "" {
			return fmt.Errorf("observed model is missing")
		}
	}
	return nil
}

func eligible(r domain.RunResult) bool { return r.Success && r.Status.Measurement != "invalid" }
