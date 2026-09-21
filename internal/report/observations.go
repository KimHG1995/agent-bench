package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func writeObservations(b *strings.Builder, results []domain.RunResult) {
	b.WriteString("\n## Observation Quality\n\nRequired recall measures expected facts found; unexpected is not necessarily incorrect. F1 can penalize correct extra evidence. Cache counts are part of total input, not an additional token charge. USD cost is unknown unless explicitly observed.\n\n")
	b.WriteString("| Strategy | Required recall mean | Precision mean | Input median (n) | Cached input median (n) | Output median (n) | Graph fact use / observed runs |\n| --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, strategy := range sortedStrategies(aggregateByStrategy(results)) {
		var recall, precision, input, cached, output []float64
		used, observed := 0, 0
		for _, r := range results {
			if r.Strategy != strategy || !eligible(r) {
				continue
			}
			recall = append(recall, r.Recall)
			precision = append(precision, r.Precision)
			if r.Metrics.GraphUsed != nil {
				observed++
				if *r.Metrics.GraphUsed {
					used++
				}
			}
			if r.Metrics.Partial {
				continue
			}
			if r.Metrics.InputTokens != nil {
				input = append(input, float64(*r.Metrics.InputTokens))
			}
			if r.Metrics.CacheReadInputTokens != nil {
				cached = append(cached, float64(*r.Metrics.CacheReadInputTokens))
			}
			if r.Metrics.OutputTokens != nil {
				output = append(output, float64(*r.Metrics.OutputTokens))
			}
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %d / %d |\n", strategy, formatMean(recall), formatMean(precision), sampleMedian(input), sampleMedian(cached), sampleMedian(output), used, observed))
	}
	type pair struct{ a, b *domain.RunResult }
	pairs := map[string]*pair{}
	provenance := map[string]bool{}
	for i := range results {
		r := &results[i]
		if r.Runtime.ModelProvenance != "" {
			provenance[r.Runtime.ModelProvenance] = true
		}
		if r.Strategy != "baseline" && r.Strategy != "graph" {
			continue
		}
		key := fmt.Sprintf("%s / %s / run %d", r.Experiment.ID, r.TaskID, r.Run)
		p := pairs[key]
		if p == nil {
			p = &pair{}
			pairs[key] = p
		}
		if r.Strategy == "baseline" {
			p.a = r
		} else {
			p.b = r
		}
	}
	if len(provenance) > 0 {
		keys := make([]string, 0, len(provenance))
		for p := range provenance {
			keys = append(keys, p)
		}
		sort.Strings(keys)
		b.WriteString("\nModel provenance: `" + strings.Join(keys, "`, `") + "`. requested_only verifies the requested configuration, not the serving model.\n")
	}
	var excluded []string
	valid := 0
	for key, p := range pairs {
		reason := ""
		switch {
		case p.a == nil || p.b == nil:
			reason = "missing counterpart"
		case !eligible(*p.a) || !eligible(*p.b):
			reason = "failed or invalid measurement"
		default:
			if err := samePair(*p.a, *p.b); err != nil {
				reason = err.Error()
			}
		}
		if reason == "" {
			valid++
		} else {
			excluded = append(excluded, key+": "+reason)
		}
	}
	b.WriteString(fmt.Sprintf("\nComparable successful pairs: **%d / %d** candidate pairs. Incomplete/failed pairs are excluded from deltas but retained in the run denominator. Percentiles over small samples are descriptive only; this is not a significance test.\n", valid, len(pairs)))
	sort.Strings(excluded)
	for _, line := range excluded {
		b.WriteString("- " + escapeInline(line) + "\n")
	}
	warnings := map[string]bool{}
	for _, r := range results {
		if r.Status.FailureKind != "" {
			b.WriteString(fmt.Sprintf("- %s / %s / run %d: %s\n", r.Strategy, r.TaskID, r.Run, escapeInline(r.Status.FailureKind)))
		}
		for _, w := range r.Status.Warnings {
			warnings[w] = true
		}
	}
	if len(warnings) > 0 {
		b.WriteString("\nObserved warnings:\n\n")
		keys := make([]string, 0, len(warnings))
		for w := range warnings {
			keys = append(keys, w)
		}
		sort.Strings(keys)
		for _, w := range keys {
			b.WriteString("- " + escapeInline(w) + "\n")
		}
	}
}

func sampleMedian(v []float64) string {
	if len(v) == 0 {
		return "n/a (0)"
	}
	return fmt.Sprintf("%.0f (%d)", percentile(v, .5), len(v))
}
