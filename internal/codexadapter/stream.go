package codexadapter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

type usage struct {
	Input      *int64 `json:"input_tokens"`
	Output     *int64 `json:"output_tokens"`
	Cached     *int64 `json:"cached_input_tokens"`
	CacheWrite *int64 `json:"cache_write_input_tokens"`
	Reasoning  *int64 `json:"reasoning_output_tokens"`
}

type streamItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Text    string `json:"text"`
	Message string `json:"message"`
	Server  string `json:"server"`
	Tool    string `json:"tool"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
	Result *struct {
		IsError      bool            `json:"isError"`
		IsErrorSnake bool            `json:"is_error"`
		Structured   json.RawMessage `json:"structured_content"`
		Content      []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result"`
}

func parseStream(r io.Reader, strategy string) (domain.AgentOutput, error) {
	out := domain.AgentOutput{Status: domain.RunStatus{Execution: "failed", Measurement: "invalid"}}
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64*1024), 8*1024*1024)
	items := map[string]streamItem{}
	var firstErr error
	kind := "protocol"
	fail := func(k, msg string) {
		if firstErr == nil {
			firstErr = fmt.Errorf("%s", msg)
			kind = k
		}
	}
	var terminal int
	var completed bool
	var observed *usage
	var finalText string
	for s.Scan() {
		if strings.TrimSpace(s.Text()) == "" {
			continue
		}
		var ev struct {
			Type    string      `json:"type"`
			Item    *streamItem `json:"item"`
			Usage   *usage      `json:"usage"`
			Message string      `json:"message"`
			Error   *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(s.Bytes(), &ev); err != nil {
			fail("protocol", "malformed Codex JSONL: "+err.Error())
			continue
		}
		switch ev.Type {
		case "item.started", "item.updated", "item.completed":
			if ev.Item == nil || ev.Item.ID == "" {
				fail("protocol", "item event missing identity")
				continue
			}
			i := *ev.Item
			items[i.ID] = i
			if ev.Type == "item.completed" && i.Type == "agent_message" {
				finalText = i.Text
			}
			if ev.Type == "item.completed" && i.Type == "error" {
				out.Status.Warnings = append(out.Status.Warnings, i.Message)
			}
			if i.Type == "mcp_tool_call" && (strategy != "graph" || i.Server != "ts_graph" || i.Tool != "inspect_typescript_graph") {
				fail("unexpected_tool", "unexpected MCP tool: "+i.Server+"/"+i.Tool)
			}
			if i.Type == "file_change" {
				fail("target_mutated", "Codex attempted to change target files")
			}
			if i.Type == "mcp_tool_call" && (i.Status == "failed" || i.Error != nil || (i.Result != nil && (i.Result.IsError || i.Result.IsErrorSnake))) {
				msg := "Graph MCP call failed"
				if i.Error != nil {
					msg += ": " + i.Error.Message
				}
				fail("graph_unavailable", msg)
			}
		case "turn.completed":
			terminal++
			completed = true
			observed = ev.Usage
		case "turn.failed":
			terminal++
			msg := "Codex turn failed"
			if ev.Error != nil {
				msg += ": " + ev.Error.Message
			}
			fail(classifyError(msg), msg)
		case "error":
			fail(classifyError(ev.Message), "Codex error: "+ev.Message)
		}
	}
	if err := s.Err(); err != nil {
		fail("protocol", "read Codex stream: "+err.Error())
	}
	if terminal != 1 {
		fail("protocol", fmt.Sprintf("expected one terminal turn, got %d", terminal))
	}
	if completed {
		out.Status.Execution = "completed"
	}
	var total, succeeded, graph, graphOK, facts int64
	for _, i := range items {
		if i.Type != "command_execution" && i.Type != "mcp_tool_call" {
			continue
		}
		total++
		ok := i.Status == "completed" && i.Error == nil && (i.Result == nil || (!i.Result.IsError && !i.Result.IsErrorSnake))
		if ok {
			succeeded++
		}
		if i.Type == "mcp_tool_call" && i.Server == "ts_graph" && i.Tool == "inspect_typescript_graph" {
			graph++
			if ok {
				graphOK++
				if graphFact(i) {
					facts++
				}
			}
		}
		if i.Status != "completed" && i.Status != "failed" {
			fail("protocol", "tool has no terminal status: "+i.ID)
		}
	}
	used := facts > 0
	out.Metrics = domain.AgentMetrics{ToolCalls: &total, ToolCallsSucceeded: &succeeded, GraphToolCalls: &graph, GraphToolCallsSucceeded: &graphOK, GraphFactCallsSucceeded: &facts, GraphUsed: &used}
	if observed != nil {
		for _, v := range []*int64{observed.Input, observed.Output, observed.Cached, observed.CacheWrite, observed.Reasoning} {
			if v != nil && *v < 0 {
				fail("protocol", "negative token usage")
			}
		}
		if observed.Input != nil && observed.Cached != nil && *observed.Cached > *observed.Input {
			fail("protocol", "cached input exceeds total input")
		}
		out.Metrics.InputTokens, out.Metrics.OutputTokens = observed.Input, observed.Output
		out.Metrics.CacheReadInputTokens, out.Metrics.CacheCreationInputTokens = observed.Cached, observed.CacheWrite
		out.Metrics.ReasoningOutputTokens = observed.Reasoning
	}
	out.Metrics.Partial = !completed || terminal != 1 || firstErr != nil || out.Metrics.InputTokens == nil || out.Metrics.OutputTokens == nil
	if finalText != "" {
		answer, err := decodeFinal([]byte(finalText))
		if err != nil {
			fail("output_schema", err.Error())
		} else {
			out.Answer, out.Evidence = answer.Answer, answer.Evidence
		}
	} else {
		fail("output_schema", "missing final Codex answer")
	}
	if firstErr != nil {
		out.Status.FailureKind = kind
		out.Error = firstErr.Error()
		return out, firstErr
	}
	out.Status.Measurement = "valid"
	return out, nil
}

func classifyError(message string) string {
	m := strings.ToLower(message)
	for _, marker := range []string{"usage limit", "quota", "rate limit", "insufficient_quota", "limit reached"} {
		if strings.Contains(m, marker) {
			return "quota"
		}
	}
	for _, marker := range []string{"unauthorized", "authentication", "not logged in", "token expired", "invalid api key"} {
		if strings.Contains(m, marker) {
			return "auth"
		}
	}
	return "provider"
}

func graphFact(i streamItem) bool {
	if i.Result == nil {
		return false
	}
	data := i.Result.Structured
	if len(data) == 0 {
		for _, c := range i.Result.Content {
			if c.Type == "text" {
				data = []byte(c.Text)
				break
			}
		}
	}
	var contract struct {
		Result struct {
			Type    string `json:"type"`
			Skipped bool   `json:"skipped"`
		} `json:"result"`
	}
	if json.Unmarshal(data, &contract) != nil || contract.Result.Skipped {
		return false
	}
	switch contract.Result.Type {
	case "tour", "lookup", "trace", "details", "entrypoints", "overview":
		return true
	}
	return false
}

func decodeFinal(data []byte) (domain.AgentOutput, error) {
	var root map[string]any
	bad := func() (domain.AgentOutput, error) {
		return domain.AgentOutput{}, fmt.Errorf("final answer violates answer/evidence schema")
	}
	if json.Unmarshal(data, &root) != nil || len(root) != 2 {
		return bad()
	}
	answer, ok := root["answer"].(string)
	if !ok || strings.TrimSpace(answer) == "" {
		return bad()
	}
	evidence, ok := root["evidence"].(map[string]any)
	if !ok || len(evidence) != 3 {
		return bad()
	}
	stringsArray := func(value any) ([]string, bool) {
		a, ok := value.([]any)
		if !ok {
			return nil, false
		}
		out := make([]string, 0, len(a))
		for _, v := range a {
			s, ok := v.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	}
	symbols, ok := stringsArray(evidence["symbols"])
	if !ok {
		return bad()
	}
	paths, ok := stringsArray(evidence["paths"])
	if !ok {
		return bad()
	}
	rels, ok := evidence["relationships"].([]any)
	if !ok {
		return bad()
	}
	relationships := make([]domain.Relationship, 0, len(rels))
	for _, raw := range rels {
		obj, ok := raw.(map[string]any)
		if !ok || len(obj) != 2 {
			return bad()
		}
		from, ok := obj["from"].(string)
		if !ok {
			return bad()
		}
		to, ok := obj["to"].(string)
		if !ok {
			return bad()
		}
		relationships = append(relationships, domain.Relationship{From: from, To: to})
	}
	return domain.AgentOutput{Answer: answer, Evidence: domain.Evidence{Symbols: symbols, Paths: paths, Relationships: relationships}}, nil
}
