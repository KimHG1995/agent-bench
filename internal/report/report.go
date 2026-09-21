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
	b.WriteString("Accuracy is deterministic expected-evidence F1, not natural-language answer quality.\n\n")
	b.WriteString("## Strategy Summary\n\n")
	b.WriteString("| Strategy | Runs | Success | Accuracy mean / median | Tool calls median / p95 | I/O tokens median / p95 | Latency median / p95 |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, strategy := range strategies {
		a := groups[strategy]
		b.WriteString(fmt.Sprintf("| %s | %d | %.1f%% | %s | %s | %s | %s |\n",
			strategy,a.runs,successRate(a),meanMedian(a.accuracies,3),medianP95(a.toolCalls,1),medianP95(a.ioTokens,1),latencyMedianP95(a.latencies)))
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
		if a == nil { a=&aggregate{strategy:r.Strategy}; groups[r.Strategy]=a }
		a.runs++
		if !r.Success { continue }
		a.successes++
		a.accuracies=append(a.accuracies,r.Accuracy)
		a.latencies=append(a.latencies,float64(r.DurationMS))
		if r.Metrics.ToolCalls!=nil { a.toolCalls=append(a.toolCalls,float64(*r.Metrics.ToolCalls)) }
		if !r.Metrics.Partial && r.Metrics.InputTokens!=nil && r.Metrics.OutputTokens!=nil {
			a.ioTokens=append(a.ioTokens,float64(*r.Metrics.InputTokens+*r.Metrics.OutputTokens))
		}
	}
	return groups
}

func sortedStrategies(groups map[string]*aggregate) []string { out:=make([]string,0,len(groups)); for k:=range groups{out=append(out,k)}; sort.Strings(out); return out }
func successRate(a *aggregate) float64 { if a.runs==0{return 0}; return 100*float64(a.successes)/float64(a.runs) }
func meanMedian(v []float64,p int) string { if len(v)==0{return "n/a"}; return fmt.Sprintf("%.*f / %.*f",p,mean(v),p,percentile(v,.50)) }
func medianP95(v []float64,p int) string { if len(v)==0{return "n/a"}; return fmt.Sprintf("%.*f / %.*f",p,percentile(v,.50),p,percentile(v,.95)) }
func latencyMedianP95(v []float64) string { if len(v)==0{return "n/a"}; return fmt.Sprintf("%.1f / %.1f ms",percentile(v,.50),percentile(v,.95)) }
func mean(v []float64) float64 { var s float64; for _,x:=range v{s+=x}; return s/float64(len(v)) }
func percentile(v []float64,p float64) float64 { if len(v)==0{return 0}; c:=append([]float64(nil),v...); sort.Float64s(c); i:=int(math.Ceil(p*float64(len(c))))-1; if i<0{i=0}; if i>=len(c){i=len(c)-1}; return c[i] }

func writePairedComparison(b *strings.Builder, results []domain.RunResult) {
	type pair struct{ baseline,graph *domain.RunResult }
	pairs:=map[string]*pair{}
	for i:=range results {
		r:=&results[i]
		if !r.Success || (r.Strategy!="baseline"&&r.Strategy!="graph"){continue}
		key:=fmt.Sprintf("%s#%d#%s",r.TaskID,r.Run,r.Experiment.ID)
		p:=pairs[key]; if p==nil{p=&pair{};pairs[key]=p}
		if r.Strategy=="baseline"{p.baseline=r}else{p.graph=r}
	}
	var accuracyDelta,toolChange,tokenChange,latencyChange []float64
	for _,p:=range pairs{
		if p.baseline==nil||p.graph==nil{continue}
		if samePair(*p.baseline,*p.graph)!=nil{continue}
		accuracyDelta=append(accuracyDelta,p.graph.Accuracy-p.baseline.Accuracy)
		if p.baseline.Metrics.ToolCalls!=nil&&p.graph.Metrics.ToolCalls!=nil&&*p.baseline.Metrics.ToolCalls>0{toolChange=append(toolChange,percentChange(float64(*p.baseline.Metrics.ToolCalls),float64(*p.graph.Metrics.ToolCalls)))}
		if bt,bok:=ioTokenCount(*p.baseline);bok{if gt,gok:=ioTokenCount(*p.graph);gok&&bt>0{tokenChange=append(tokenChange,percentChange(bt,gt))}}
		if p.baseline.DurationMS>0{latencyChange=append(latencyChange,percentChange(float64(p.baseline.DurationMS),float64(p.graph.DurationMS)))}
	}
	if len(accuracyDelta)==0{return}
	b.WriteString("\n## Paired Baseline vs Graph\n\nOnly successful, comparable runs with the same experiment/task/run are paired.\n\n")
	b.WriteString("| Metric | Paired samples | Median graph vs baseline |\n| --- | ---: | ---: |\n")
	b.WriteString(fmt.Sprintf("| Accuracy delta | %d | %+.3f |\n",len(accuracyDelta),percentile(accuracyDelta,.50)))
	writeChangeRow(b,"Tool calls change",toolChange);writeChangeRow(b,"I/O tokens change",tokenChange);writeChangeRow(b,"Latency change",latencyChange)
}
func writeChangeRow(b *strings.Builder,l string,v []float64){if len(v)==0{b.WriteString(fmt.Sprintf("| %s | 0 | n/a |\n",l));return};b.WriteString(fmt.Sprintf("| %s | %d | %+.1f%% |\n",l,len(v),percentile(v,.50)))}
func percentChange(base,graph float64)float64{return 100*(graph-base)/base}
func ioTokenCount(r domain.RunResult)(float64,bool){if r.Metrics.Partial||r.Metrics.InputTokens==nil||r.Metrics.OutputTokens==nil{return 0,false};return float64(*r.Metrics.InputTokens+*r.Metrics.OutputTokens),true}

func writeTaskAccuracy(b *strings.Builder,results []domain.RunResult){type stats struct{baseline,graph []float64};m:=map[string]*stats{};for _,r:=range results{if !r.Success||(r.Strategy!="baseline"&&r.Strategy!="graph"){continue};s:=m[r.TaskID];if s==nil{s=&stats{};m[r.TaskID]=s};if r.Strategy=="baseline"{s.baseline=append(s.baseline,r.Accuracy)}else{s.graph=append(s.graph,r.Accuracy)}};if len(m)==0{return};ids:=make([]string,0,len(m));for id:=range m{ids=append(ids,id)};sort.Strings(ids);b.WriteString("\n## Accuracy by Task\n\n| Task | Baseline mean | Graph mean | Delta |\n| --- | ---: | ---: | ---: |\n");for _,id:=range ids{s:=m[id];delta:="n/a";if len(s.baseline)>0&&len(s.graph)>0{delta=fmt.Sprintf("%+.3f",mean(s.graph)-mean(s.baseline))};b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",id,formatMean(s.baseline),formatMean(s.graph),delta))}}
func formatMean(v []float64)string{if len(v)==0{return "n/a"};return fmt.Sprintf("%.3f",mean(v))}
func writeFailures(b *strings.Builder,results []domain.RunResult){b.WriteString("\n## Failure and Evidence Gaps\n\n");found:=false;for _,r:=range results{if r.Success&&len(r.Missing)==0&&len(r.Unexpected)==0{continue};found=true;b.WriteString(fmt.Sprintf("### %s / %s / run %d\n\n",r.Strategy,r.TaskID,r.Run));if !r.Success{b.WriteString("- execution error: `"+escapeInline(r.Error)+"`\n")};if len(r.Missing)>0{b.WriteString("- missing: `"+strings.Join(r.Missing,"`, `")+"`\n")};if len(r.Unexpected)>0{b.WriteString("- unexpected: `"+strings.Join(r.Unexpected,"`, `")+"`\n")};if r.Metrics.Partial{b.WriteString("- metrics: partial\n")};b.WriteString("\n")};if !found{b.WriteString("No failures or evidence gaps.\n")}}
func escapeInline(s string)string{return strings.ReplaceAll(s,"`","'")}
