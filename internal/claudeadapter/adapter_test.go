package claudeadapter

import (
	"strings"
	"testing"
)

func TestParseStreamStructuredOutputAndUsage(t *testing.T) {
	stream := strings.Join([]string{
		`{"type":"assistant","message":{"id":"m1","model":"claude-sonnet-5","usage":{"input_tokens":100,"output_tokens":20,"cache_creation_input_tokens":5,"cache_read_input_tokens":10},"content":[{"type":"tool_use","id":"t1","name":"Read"}]}}`,
		`{"type":"assistant","message":{"id":"m1","model":"claude-sonnet-5","usage":{"input_tokens":100,"output_tokens":20,"cache_creation_input_tokens":5,"cache_read_input_tokens":10},"content":[{"type":"tool_use","id":"t1","name":"Read"}]}}`,
		`{"type":"result","subtype":"success","total_cost_usd":0.01,"usage":{"input_tokens":100,"output_tokens":20,"cache_creation_input_tokens":5,"cache_read_input_tokens":10},"structured_output":{"answer":"ok","evidence":{"symbols":["A"],"paths":[],"relationships":[]}}}`,
	}, "\n")

	out, err := parseStream([]byte(stream), "baseline", graphToolNameDefault)
	if err != nil {
		t.Fatal(err)
	}
	if out.Answer != "ok" || out.Runtime.Model != "claude-sonnet-5" {
		t.Fatalf("unexpected output: %#v", out)
	}
	if out.Metrics.ToolCalls == nil || *out.Metrics.ToolCalls != 1 {
		t.Fatalf("tool calls: %#v", out.Metrics.ToolCalls)
	}
	if out.Metrics.InputTokens == nil || *out.Metrics.InputTokens != 100 {
		t.Fatalf("input tokens: %#v", out.Metrics.InputTokens)
	}
}

func TestParseStreamRejectsMCPInBaseline(t *testing.T) {
	stream := `{"type":"assistant","message":{"id":"m1","content":[{"type":"tool_use","id":"t1","name":"mcp__ts_graph__inspect_typescript_graph"}]}}` + "\n" +
		`{"type":"result","subtype":"success","structured_output":{"answer":"ok","evidence":{"symbols":[],"paths":[],"relationships":[]}}}`

	if _, err := parseStream([]byte(stream), "baseline", graphToolNameDefault); err == nil {
		t.Fatal("expected MCP violation")
	}
}

func TestParseStreamCountsGraphTool(t *testing.T) {
	stream := `{"type":"assistant","message":{"id":"m1","content":[{"type":"tool_use","id":"t1","name":"mcp__ts_graph__inspect_typescript_graph"}]}}` + "\n" +
		`{"type":"result","subtype":"success","structured_output":{"answer":"ok","evidence":{"symbols":[],"paths":[],"relationships":[]}}}`

	out, err := parseStream([]byte(stream), "graph", graphToolNameDefault)
	if err != nil {
		t.Fatal(err)
	}
	if out.Metrics.GraphToolCalls == nil || *out.Metrics.GraphToolCalls != 1 {
		t.Fatalf("graph calls: %#v", out.Metrics.GraphToolCalls)
	}
}
