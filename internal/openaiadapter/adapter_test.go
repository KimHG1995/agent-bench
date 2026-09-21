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
	out, err := Run(context.Background(), domain.RunRequest{Strategy:"baseline",Task:domain.AgentTask{Repository:domain.RepositoryRef{Path:dir},Question:"find UserService"}}, Config{BaseURL:server.URL,APIKey:"x",Model:"test-model",MaxTurns:3,Client:server.Client()})
	if err != nil { t.Fatal(err) }
	if out.Metrics.InputTokens == nil || *out.Metrics.InputTokens != 30 { t.Fatalf("input=%v", out.Metrics.InputTokens) }
	if out.Metrics.Partial { t.Fatal("expected complete metrics") }
}

func TestToolSchemaOmitsEmptyRequired(t *testing.T) {
	for _, tool := range toolSpecs() {
		if v, ok := tool.Function.Parameters["required"]; ok {
			if _, ok := v.([]string); !ok { t.Fatalf("%s required must be []string, got %T", tool.Function.Name, v) }
		}
	}
	if _, ok := toolSpecs()[0].Function.Parameters["required"]; ok { t.Fatal("list_files must omit empty required") }
}

func TestParseFinalRejectsNullAndMissingEvidence(t *testing.T) {
	for _, input := range []string{"null", `{"answer":"x"}`, `{"answer":"x","evidence":{}}`} {
		if _, err := parseFinal(input); err == nil { t.Fatalf("expected error for %s", input) }
	}
}

func TestMissingUsageStaysPartial(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type","application/json")
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"role":"assistant","content":"{\"answer\":\"ok\",\"evidence\":{\"symbols\":[],\"paths\":[],\"relationships\":[]}}"}}]}`))
	}))
	defer server.Close()
	out, err := Run(context.Background(), domain.RunRequest{Strategy:"baseline",Task:domain.AgentTask{Repository:domain.RepositoryRef{Path:dir},Question:"q"}}, Config{BaseURL:server.URL,APIKey:"x",Model:"m",MaxTurns:1,Client:server.Client()})
	if err != nil { t.Fatal(err) }
	if out.Metrics.InputTokens != nil || out.Metrics.OutputTokens != nil { t.Fatal("unobserved tokens must stay nil") }
	if !out.Metrics.Partial { t.Fatal("missing usage must mark metrics partial") }
}

func TestPartialUsageSurvivesLaterFailure(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Content-Type","application/json")
			_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"role":"assistant","tool_calls":[{"id":"1","type":"function","function":{"name":"list_files","arguments":"{}"}}]}}],"usage":{"prompt_tokens":123,"completion_tokens":45}}`))
			return
		}
		http.Error(w,"nope",http.StatusServiceUnavailable)
	}))
	defer server.Close()
	out, err := Run(context.Background(), domain.RunRequest{Strategy:"baseline",Task:domain.AgentTask{Repository:domain.RepositoryRef{Path:dir},Question:"q"}}, Config{BaseURL:server.URL,APIKey:"x",Model:"m",MaxTurns:2,Client:server.Client()})
	if err == nil { t.Fatal("expected error") }
	if out.Metrics.InputTokens == nil || *out.Metrics.InputTokens != 123 { t.Fatalf("partial input=%v", out.Metrics.InputTokens) }
	if !out.Metrics.Partial { t.Fatal("expected partial metrics") }
}

func TestSafePathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("marker"), 0o644); err != nil { t.Fatal(err) }
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(outside, link); err != nil { t.Skipf("symlink unavailable: %v", err) }
	if _, err := readFile(root, "link.txt", 1, 10); err == nil { t.Fatal("expected symlink escape rejection") }
}

func TestBaselineDoesNotAdvertiseGraphTool(t *testing.T) {
	dir := t.TempDir()
	var request struct { Tools []tool `json:"tools"` }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close(); if err := json.NewDecoder(r.Body).Decode(&request); err != nil { t.Fatal(err) }
		w.Header().Set("Content-Type","application/json")
		_, _ = w.Write([]byte(`{"model":"m","choices":[{"message":{"role":"assistant","content":"{\"answer\":\"ok\",\"evidence\":{\"symbols\":[],\"paths\":[],\"relationships\":[]}}"}}],"usage":{}}`))
	}))
	defer server.Close()
	_, err := Run(context.Background(), domain.RunRequest{Strategy:"baseline",Task:domain.AgentTask{Repository:domain.RepositoryRef{Path:dir},Question:"q"}}, Config{BaseURL:server.URL,APIKey:"x",Model:"m",MaxTurns:1,Client:server.Client()})
	if err != nil { t.Fatal(err) }
	for _, tool := range request.Tools { if tool.Function.Name == graphToolName { t.Fatal("graph tool leaked into baseline") } }
}
