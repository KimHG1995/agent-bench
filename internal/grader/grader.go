package grader

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func Grade(expected, actual domain.Evidence) domain.Grade {
	expectedSet := facts(expected)
	actualSet := facts(actual)

	matched := intersection(expectedSet, actualSet)
	missing := difference(expectedSet, actualSet)
	unexpected := difference(actualSet, expectedSet)

	precision := ratio(len(matched), len(actualSet))
	recall := ratio(len(matched), len(expectedSet))
	if len(expectedSet) == 0 && len(actualSet) == 0 {
		precision, recall = 1, 1
	}
	f1 := 0.0
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}

	return domain.Grade{
		Accuracy:    f1,
		Precision:   precision,
		Recall:      recall,
		Matched:     keys(matched),
		Missing:     keys(missing),
		Unexpected:  keys(unexpected),
	}
}

func facts(e domain.Evidence) map[string]struct{} {
	out := map[string]struct{}{}
	for _, symbol := range e.Symbols {
		symbol = strings.TrimSpace(symbol)
		if symbol != "" {
			out["symbol:"+symbol] = struct{}{}
		}
	}
	for _, path := range e.Paths {
		path = normalizePath(path)
		if path != "" {
			out["path:"+path] = struct{}{}
		}
	}
	for _, rel := range e.Relationships {
		from := strings.TrimSpace(rel.From)
		to := strings.TrimSpace(rel.To)
		if from != "" && to != "" {
			out["relationship:"+from+"->"+to] = struct{}{}
		}
	}
	return out
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "./")
	return path
}

func intersection(a, b map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for k := range a {
		if _, ok := b[k]; ok {
			out[k] = struct{}{}
		}
	}
	return out
}

func difference(a, b map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for k := range a {
		if _, ok := b[k]; !ok {
			out[k] = struct{}{}
		}
	}
	return out
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ratio(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}
