package report

import (
	"strings"
	"testing"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func ptr(v int64) *int64 { return &v }

func comparable(strategy string) domain.RunResult {
	return domain.RunResult{
		TaskID:"t1", Strategy:strategy, Run:1, Success:true, Accuracy:1, DurationMS:10,
		Repository:domain.RepositoryRef{Revision:"abc"},
		Experiment:domain.ExperimentMeta{ID:"e1",HarnessRevision:"h1",GraphRevision:"g1",TaskHash:"t"},
		Runtime:domain.AgentRuntime{Provider:"p",Endpoint:"u",RequestedModel:"m",Model:"actual",Adapter:"a",MaxTurns:10},
	}
}

func TestValidateComparableRejectsDifferentModels(t *testing.T) {
	a:=comparable("baseline"); b:=comparable("graph"); b.Runtime.Model="other"
	if err:=ValidateComparable([]domain.RunResult{a,b}); err==nil { t.Fatal("expected mismatch") }
}

func TestValidateComparableRejectsDuplicate(t *testing.T) {
	a:=comparable("baseline")
	if err:=ValidateComparable([]domain.RunResult{a,a}); err==nil { t.Fatal("expected duplicate") }
}

func TestMarkdownIncludesPairedComparison(t *testing.T) {
	a:=comparable("baseline"); b:=comparable("graph")
	a.Accuracy=.8;a.DurationMS=100;a.Metrics=domain.AgentMetrics{ToolCalls:ptr(10),InputTokens:ptr(900),OutputTokens:ptr(100)}
	b.Accuracy=1;b.DurationMS=50;b.Metrics=domain.AgentMetrics{ToolCalls:ptr(5),InputTokens:ptr(400),OutputTokens:ptr(100)}
	if err:=ValidateComparable([]domain.RunResult{a,b});err!=nil{t.Fatal(err)}
	got:=Markdown([]domain.RunResult{a,b})
	for _,want:=range []string{"| Accuracy delta | 1 | +0.200 |","| Tool calls change | 1 | -50.0% |","| I/O tokens change | 1 | -50.0% |","| Latency change | 1 | -50.0% |"}{if !strings.Contains(got,want){t.Fatalf("missing %q\n%s",want,got)}}
}

func TestPartialTokensExcluded(t *testing.T){
	r:=comparable("baseline");r.Metrics=domain.AgentMetrics{InputTokens:ptr(10),OutputTokens:ptr(10),Partial:true}
	got:=Markdown([]domain.RunResult{r})
	if !strings.Contains(got,"| baseline | 1 | 100.0% | 1.000 / 1.000 | n/a | n/a |"){t.Fatalf("partial tokens leaked:\n%s",got)}
}
