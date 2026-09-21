package openaiadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func TestRunToolCallingLoop(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "service.ts"), []byte("export class UserService {}\n"), 0o644); err != nil { t.Fatal(err) }

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type","application/json")
		if calls == 1 {
			_, _ = w.Write([]byte(`{"model":"test-model","choices":[{"message":{"role":"assistant","tool_calls":[{"id":"1","type":"function","function":{"name":"search_text","arguments":"{\"query\":\"UserService\"}"}}]}}],"usage":{"prompt_tokens":10,"completion_tokens":3}}`))
			return
		}
		_, _ = w.Write([]byte(`{"model":"test-model","choices":[{"message":{"role":"assistant","content":"{\"answer\":\"found\",\"evidence\":{\"symbols\":[\"UserService\"],\"paths\":[\"service.ts\"],\"relationships\":[]}}"}}],"usage":{"prompt_tokens":20,"completion_tokens":5}}`))
	}))
	defer server.Close()

	out, err := Run(context.Background(), domain.RunRequest{
		Strategy:"baseline",
		Task:domain.AgentTask{Repository:domain.RepositoryRef{Path:dir},Question:"find UserService"},
	}, Config{BaseURL:server.URL,APIKey:"x",Model:"test-model",MaxTurns:3,Client:server.Client()})
	if err != nil { t.Fatal(err) }
	if out.Answer != "found" { t.Fatalf("answer=%q", out.Answer) }
	if out.Metrics.ToolCalls == nil || *out.Metrics.ToolCalls != 1 { t.Fatalf("tools=%v", out.Metrics.ToolCalls) }
	if out.Metrics.GraphToolCalls == nil || *out.Metrics.GraphToolCalls != 0 { t.Fatalf("graph tools=%v", out.Metrics.GraphToolCalls) }
	if out.Metrics.InputTokens == nil || *out.Metrics.InputTokens != 30 { t.Fatalf("input=%v", out.Metrics.InputTokens) }
}

func TestBaselineDoesNotAdvertiseGraphTool(t *testing.T) {
	dir := t.TempDir()
	var request struct {
		Tools []tool `json:"tools"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil { t.Fatal(err) }
		w.Header().Set("Content-Type","application/json")
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"role":"assistant","content":"{\"answer\":\"ok\",\"evidence\":{\"symbols\":[],\"paths\":[],\"relationships\":[]}}"}}],"usage":{}}`))
	}))
	defer server.Close()

	_, err := Run(context.Background(), domain.RunRequest{
		Strategy:"baseline",
		Task:domain.AgentTask{Repository:domain.RepositoryRef{Path:dir},Question:"q"},
	}, Config{BaseURL:server.URL,APIKey:"x",Model:"m",MaxTurns:1,Client:server.Client()})
	if err != nil { t.Fatal(err) }
	for _, tool := range request.Tools {
		if tool.Function.Name == graphToolName { t.Fatal("graph tool leaked into baseline") }
	}
}

func TestSafePathRejectsEscape(t *testing.T) {
	if _, err := safePath(t.TempDir(), "../secret"); err == nil { t.Fatal("expected error") }
}

func TestToolSpecsMarshal(t *testing.T) {
	if _, err := json.Marshal(toolSpecs()); err != nil { t.Fatal(err) }
}
