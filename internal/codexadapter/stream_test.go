package codexadapter

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const finalAnswer = `{"answer":"A calls B","evidence":{"symbols":["A","B"],"paths":["src/a.ts"],"relationships":[{"from":"A","to":"B"}]}}`

func finalEvent() string {
	b, _ := json.Marshal(map[string]any{"type": "item.completed", "item": map[string]any{"id": "answer", "type": "agent_message", "text": finalAnswer}})
	return string(b) + "\n"
}

const usageEvent = `{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":60,"cache_write_input_tokens":0,"output_tokens":20,"reasoning_output_tokens":4}}` + "\n"

func TestParseStreamDeduplicatesToolsAndObservesStructuredGraphFacts(t *testing.T) {
	stream := `{"type":"thread.started","thread_id":"t"}
{"type":"turn.started"}
{"type":"item.started","item":{"id":"s","type":"command_execution","command":"rg A src","status":"in_progress"}}
{"type":"item.completed","item":{"id":"s","type":"command_execution","command":"rg A src","status":"completed","exit_code":0}}
{"type":"item.started","item":{"id":"g","type":"mcp_tool_call","server":"ts_graph","tool":"inspect_typescript_graph","status":"in_progress"}}
{"type":"item.completed","item":{"id":"g","type":"mcp_tool_call","server":"ts_graph","tool":"inspect_typescript_graph","status":"completed","error":null,"result":{"content":[],"structured_content":{"result":{"type":"tour"}}}}}
` + finalEvent() + usageEvent
	out, err := parseStream(strings.NewReader(stream), "graph")
	if err != nil {
		t.Fatal(err)
	}
	if out.Answer != "A calls B" || out.Metrics.ToolCalls == nil || *out.Metrics.ToolCalls != 2 || *out.Metrics.ToolCallsSucceeded != 2 {
		t.Fatalf("wrong tools/answer: %#v", out)
	}
	if *out.Metrics.GraphToolCalls != 1 || *out.Metrics.GraphFactCallsSucceeded != 1 || !*out.Metrics.GraphUsed {
		t.Fatalf("graph not observed: %#v", out.Metrics)
	}
	if *out.Metrics.InputTokens != 100 || *out.Metrics.CacheReadInputTokens != 60 || *out.Metrics.OutputTokens != 20 || *out.Metrics.ReasoningOutputTokens != 4 {
		t.Fatal(out.Metrics)
	}
	if out.Metrics.CostUSD != nil || out.Runtime.Model != "" || out.Metrics.Partial {
		t.Fatal("invented data", out)
	}
}

func TestParseStreamRejectsGraphDenialEvenAfterCompletedTurn(t *testing.T) {
	stream := `{"type":"item.completed","item":{"id":"g","type":"mcp_tool_call","server":"ts_graph","tool":"inspect_typescript_graph","status":"failed","error":{"message":"MCP tool call requires approval, but approval policy is never"}}}` + "\n" + finalEvent() + usageEvent
	out, err := parseStream(strings.NewReader(stream), "graph")
	if err == nil || out.Status.FailureKind != "graph_unavailable" || out.Status.Measurement != "invalid" {
		t.Fatalf("denial accepted: %#v %v", out, err)
	}
	if out.Status.Execution != "completed" || out.Answer == "" || out.Metrics.InputTokens == nil || *out.Metrics.GraphToolCalls != 1 || *out.Metrics.GraphToolCallsSucceeded != 0 {
		t.Fatal("lost observations", out)
	}
}

func TestParseStreamMissingUsageIsUnknown(t *testing.T) {
	for _, usage := range []string{``, `,"usage":{}`, `,"usage":{"input_tokens":10}`} {
		t.Run(usage, func(t *testing.T) {
			out, err := parseStream(strings.NewReader(finalEvent()+`{"type":"turn.completed"`+usage+"}\n"), "baseline")
			if err != nil {
				t.Fatal(err)
			}
			if !out.Metrics.Partial || out.Metrics.OutputTokens != nil || out.Metrics.CostUSD != nil {
				t.Fatal("missing became zero", out.Metrics)
			}
		})
	}
}

func TestParseStreamWarningAndUnusedGraphRemainValid(t *testing.T) {
	stream := `{"type":"item.completed","item":{"id":"w","type":"error","message":"Under-development features enabled"}}` + "\n" + finalEvent() + usageEvent
	out, err := parseStream(strings.NewReader(stream), "graph")
	if err != nil || out.Status.Measurement != "valid" || len(out.Status.Warnings) != 1 || *out.Metrics.GraphUsed {
		t.Fatalf("warning or nonuse discarded: %#v %v", out, err)
	}
}

func TestParseStreamRejectsBrokenOrForeignExecution(t *testing.T) {
	tests := map[string]string{
		"truncated":        finalEvent() + usageEvent + `{"type":`,
		"missing terminal": finalEvent(),
		"double terminal":  finalEvent() + usageEvent + usageEvent,
		"negative usage":   finalEvent() + `{"type":"turn.completed","usage":{"input_tokens":-1,"output_tokens":0}}`,
		"foreign MCP":      `{"type":"item.completed","item":{"id":"g","type":"mcp_tool_call","server":"external","tool":"send","status":"completed"}}` + "\n" + finalEvent() + usageEvent,
		"baseline MCP":     `{"type":"item.completed","item":{"id":"g","type":"mcp_tool_call","server":"ts_graph","tool":"inspect_typescript_graph","status":"completed"}}` + "\n" + finalEvent() + usageEvent,
	}
	for name, stream := range tests {
		t.Run(name, func(t *testing.T) {
			out, err := parseStream(strings.NewReader(stream), "baseline")
			if err == nil || out.Status.Measurement != "invalid" {
				t.Fatalf("accepted %s: %#v", name, out)
			}
		})
	}
}

func TestDecodeFinalEnforcesSchema(t *testing.T) {
	for _, raw := range []string{
		`{"answer":"ok","evidence":{"symbols":null,"paths":[],"relationships":[]}}`,
		`{"answer":"ok","evidence":{"symbols":[],"paths":[],"relationships":[{"from":"A"}]}}`,
		`{"answer":"ok","evidence":{"symbols":[],"paths":[],"relationships":[]},"metrics":{}}`,
		`{"answer":"ok","evidence":{"symbols":[],"paths":[]}}`,
		finalAnswer + `{}`, `null`,
	} {
		if _, err := decodeFinal([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestGraphEscapeDoesNotClaimFactUse(t *testing.T) {
	stream := `{"type":"item.completed","item":{"id":"g","type":"mcp_tool_call","server":"ts_graph","tool":"inspect_typescript_graph","status":"completed","result":{"structured_content":{"result":{"type":"escape","skipped":true}}}}}` + "\n" + finalEvent() + usageEvent
	out, err := parseStream(strings.NewReader(stream), "graph")
	if err != nil || *out.Metrics.GraphToolCallsSucceeded != 1 || *out.Metrics.GraphFactCallsSucceeded != 0 || *out.Metrics.GraphUsed {
		t.Fatalf("escape claimed facts: %#v %v", out, err)
	}
}

func TestCapturedSmokeEvents(t *testing.T) {
	for _, tc := range []struct {
		name, strategy   string
		valid            bool
		in, tools, facts int64
	}{
		{"baseline", "baseline", true, 57867, 5, 0},
		{"graph", "graph", true, 42115, 2, 1},
		{"graph-denied", "graph", false, 86791, 8, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open("testdata/" + tc.name + ".jsonl")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			out, err := parseStream(f, tc.strategy)
			if (err == nil) != tc.valid || out.Metrics.InputTokens == nil || *out.Metrics.InputTokens != tc.in || *out.Metrics.ToolCalls != tc.tools || *out.Metrics.GraphFactCallsSucceeded != tc.facts {
				t.Fatalf("wrong captured parse: %#v %v", out, err)
			}
		})
	}
}
