package report

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

type aggregate struct {
	strategy   string
	runs       int
	successes  int
	accuracies []float64
	latencies  []float64
	toolCalls  []float64
	ioTokens   []float64
}

func Markdown(results []domain.RunResult) string {
	groups := aggregateByStrategy(results)
	strategies := sortedStrategies(groups)

	var b strings.Builder
	b.WriteString("# Agent Bench Report\n\n")
	b.WriteString("## Strategy Summary\n\n")
	b.WriteString("| Strategy | Runs | Success | Accuracy mean / median | Tool calls median / p95 | I/O tokens median / p95 | Latency median / p95 |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, strategy := range strategies {
		a := groups[strategy]
		b.WriteString(fmt.Sprintf("| %s | %d | %.1f%% | %s | %s | %s | %s |\n",
			strategy,
			a.runs,
			successRate(a),
			meanMedian(a.accuracies, 3),
			medianP95(a.toolCalls, 1),
			medianP95(a.ioTokens, 1),
			latencyMedianP95(a.latencies),
		))
	}

	writePairedComparison(&b, results)
	writeTaskAccuracy(&b, results)
	writeFailures(&b, results)
	return b.String()
}

func aggregateByStrategy(results []domain.RunResult) map[string]*aggregate {
	groups := map[string]*aggregate{}
	for _, r := range results {
		a := groups[r.Strategy]
		if a == nil {
			a = &aggregate{strategy: r.Strategy}
			groups[r.Strategy] = a
		}
		a.runs++
		if !r.Success {
			continue
		}
		a.successes++
		a.accuracies = append(a.accuracies, r.Accuracy)
		a.latencies = append(a.latencies, float64(r.DurationMS))
		if r.Metrics.ToolCalls != nil {
			a.toolCalls = append(a.toolCalls, float64(*r.Metrics.ToolCalls))
		}
		if r.Metrics.InputTokens != nil && r.Metrics.OutputTokens != nil {
			a.ioTokens = append(a.ioTokens, float64(*r.Metrics.InputTokens+*r.Metrics.OutputTokens))
		}
	}
	return groups
}

func sortedStrategies(groups map[string]*aggregate) []string {
	strategies := make([]string, 0, len(groups))
	for strategy := range groups {
		strategies = append(strategies, strategy)
	}
	sort.Strings(strategies)
	return strategies
}

func successRate(a *aggregate) float64 {
	if a.runs == 0 {
		return 0
	}
	return 100 * float64(a.successes) / float64(a.runs)
}

func meanMedian(values []float64, precision int) string {
	if len(values) == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.*f / %.*f", precision, mean(values), precision, percentile(values, 0.50))
}

func medianP95(values []float64, precision int) string {
	if len(values) == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.*f / %.*f", precision, percentile(values, 0.50), precision, percentile(values, 0.95))
}

func latencyMedianP95(values []float64) string {
	if len(values) == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f / %.1f ms", percentile(values, 0.50), percentile(values, 0.95))
}

func mean(values []float64) float64 {
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	idx := int(math.Ceil(p*float64(len(copyValues)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(copyValues) {
		idx = len(copyValues) - 1
	}
	return copyValues[idx]
}

func writePairedComparison(b *strings.Builder, results []domain.RunResult) {
	type pair struct {
		baseline *domain.RunResult
		graph    *domain.RunResult
	}
	pairs := map[string]*pair{}
	for i := range results {
		r := &results[i]
		if !r.Success || (r.Strategy != "baseline" && r.Strategy != "graph") {
			continue
		}
		key := fmt.Sprintf("%s#%d", r.TaskID, r.Run)
		p := pairs[key]
		if p == nil {
			p = &pair{}
			pairs[key] = p
		}
		if r.Strategy == "baseline" {
			p.baseline = r
		} else {
			p.graph = r
		}
	}

	var accuracyDelta, toolChange, tokenChange, latencyChange []float64
	for _, p := range pairs {
		if p.baseline == nil || p.graph == nil {
			continue
		}
		accuracyDelta = append(accuracyDelta, p.graph.Accuracy-p.baseline.Accuracy)
		if p.baseline.Metrics.ToolCalls != nil && p.graph.Metrics.ToolCalls != nil && *p.baseline.Metrics.ToolCalls > 0 {
			toolChange = append(toolChange, percentChange(float64(*p.baseline.Metrics.ToolCalls), float64(*p.graph.Metrics.ToolCalls)))
		}
		bTokens, bOK := ioTokenCount(*p.baseline)
		gTokens, gOK := ioTokenCount(*p.graph)
		if bOK && gOK && bTokens > 0 {
			tokenChange = append(tokenChange, percentChange(bTokens, gTokens))
		}
		if p.baseline.DurationMS > 0 {
			latencyChange = append(latencyChange, percentChange(float64(p.baseline.DurationMS), float64(p.graph.DurationMS)))
		}
	}

	if len(accuracyDelta) == 0 {
		return
	}
	b.WriteString("\n## Paired Baseline vs Graph\n\n")
	b.WriteString("Only successful runs with the same task ID and repeat number are paired. Negative percentage means graph used less than baseline.\n\n")
	b.WriteString("| Metric | Paired samples | Median graph vs baseline |\n")
	b.WriteString("| --- | ---: | ---: |\n")
	b.WriteString(fmt.Sprintf("| Accuracy delta | %d | %+.3f |\n", len(accuracyDelta), percentile(accuracyDelta, 0.50)))
	writeChangeRow(b, "Tool calls change", toolChange)
	writeChangeRow(b, "I/O tokens change", tokenChange)
	writeChangeRow(b, "Latency change", latencyChange)
}

func writeChangeRow(b *strings.Builder, label string, values []float64) {
	if len(values) == 0 {
		b.WriteString(fmt.Sprintf("| %s | 0 | n/a |\n", label))
		return
	}
	b.WriteString(fmt.Sprintf("| %s | %d | %+.1f%% |\n", label, len(values), percentile(values, 0.50)))
}

func percentChange(baseline, graph float64) float64 {
	return 100 * (graph - baseline) / baseline
}

func ioTokenCount(r domain.RunResult) (float64, bool) {
	if r.Metrics.InputTokens == nil || r.Metrics.OutputTokens == nil {
		return 0, false
	}
	return float64(*r.Metrics.InputTokens + *r.Metrics.OutputTokens), true
}

func writeTaskAccuracy(b *strings.Builder, results []domain.RunResult) {
	type taskStats struct {
		baseline []float64
		graph    []float64
	}
	byTask := map[string]*taskStats{}
	for _, r := range results {
		if !r.Success || (r.Strategy != "baseline" && r.Strategy != "graph") {
			continue
		}
		s := byTask[r.TaskID]
		if s == nil {
			s = &taskStats{}
			byTask[r.TaskID] = s
		}
		if r.Strategy == "baseline" {
			s.baseline = append(s.baseline, r.Accuracy)
		} else {
			s.graph = append(s.graph, r.Accuracy)
		}
	}
	if len(byTask) == 0 {
		return
	}
	ids := make([]string, 0, len(byTask))
	for id := range byTask {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	b.WriteString("\n## Accuracy by Task\n\n")
	b.WriteString("| Task | Baseline mean | Graph mean | Delta |\n")
	b.WriteString("| --- | ---: | ---: | ---: |\n")
	for _, id := range ids {
		s := byTask[id]
		baseline := formatMean(s.baseline)
		graph := formatMean(s.graph)
		delta := "n/a"
		if len(s.baseline) > 0 && len(s.graph) > 0 {
			delta = fmt.Sprintf("%+.3f", mean(s.graph)-mean(s.baseline))
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", id, baseline, graph, delta))
	}
}

func formatMean(values []float64) string {
	if len(values) == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.3f", mean(values))
}

func writeFailures(b *strings.Builder, results []domain.RunResult) {
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
}

func escapeInline(s string) string {
	return strings.ReplaceAll(s, "`", "'")
}
