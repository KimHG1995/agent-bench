package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

type aggregate struct {
	strategy     string
	runs         int
	successes    int
	accuracySum  float64
	latencySum   int64
	toolCallsSum int64
	toolCallsN   int64
	inputSum     int64
	inputN       int64
	outputSum    int64
	outputN      int64
}

func Markdown(results []domain.RunResult) string {
	groups := map[string]*aggregate{}
	for _, r := range results {
		a := groups[r.Strategy]
		if a == nil {
			a = &aggregate{strategy: r.Strategy}
			groups[r.Strategy] = a
		}
		a.runs++
		if r.Success {
			a.successes++
			a.accuracySum += r.Accuracy
		}
		a.latencySum += r.DurationMS
		if r.Metrics.ToolCalls != nil {
			a.toolCallsSum += *r.Metrics.ToolCalls
			a.toolCallsN++
		}
		if r.Metrics.InputTokens != nil {
			a.inputSum += *r.Metrics.InputTokens
			a.inputN++
		}
		if r.Metrics.OutputTokens != nil {
			a.outputSum += *r.Metrics.OutputTokens
			a.outputN++
		}
	}

	strategies := make([]string, 0, len(groups))
	for strategy := range groups {
		strategies = append(strategies, strategy)
	}
	sort.Strings(strategies)

	var b strings.Builder
	b.WriteString("# Agent Bench Report\n\n")
	b.WriteString("| Strategy | Runs | Success | Mean accuracy | Mean tool calls | Mean input tokens | Mean output tokens | Mean latency |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, strategy := range strategies {
		a := groups[strategy]
		accuracy := "n/a"
		if a.successes > 0 {
			accuracy = fmt.Sprintf("%.3f", a.accuracySum/float64(a.successes))
		}
		b.WriteString(fmt.Sprintf("| %s | %d | %.1f%% | %s | %s | %s | %s | %.1f ms |\n",
			strategy,
			a.runs,
			100*float64(a.successes)/float64(a.runs),
			accuracy,
			meanInt(a.toolCallsSum, a.toolCallsN),
			meanInt(a.inputSum, a.inputN),
			meanInt(a.outputSum, a.outputN),
			float64(a.latencySum)/float64(a.runs),
		))
	}

	b.WriteString("\n## Failure and Evidence Gaps\n\n")
	found := false
	for _, r := range results {
		if r.Success && len(r.Missing) == 0 && len(r.Unexpected) == 0 {
			continue
		}
		found = true
		b.WriteString(fmt.Sprintf("### %s / %s / run %d\n\n", r.Strategy, r.TaskID, r.Run))
		if !r.Success {
			b.WriteString("- execution error: `" + escapeInline(r.Error) + "`\n")
		}
		if len(r.Missing) > 0 {
			b.WriteString("- missing: `" + strings.Join(r.Missing, "`, `") + "`\n")
		}
		if len(r.Unexpected) > 0 {
			b.WriteString("- unexpected: `" + strings.Join(r.Unexpected, "`, `") + "`\n")
		}
		b.WriteString("\n")
	}
	if !found {
		b.WriteString("No failures or evidence gaps.\n")
	}
	return b.String()
}

func meanInt(sum, n int64) string {
	if n == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f", float64(sum)/float64(n))
}

func escapeInline(s string) string {
	return strings.ReplaceAll(s, "`", "'")
}
