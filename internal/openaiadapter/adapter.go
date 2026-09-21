package openaiadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/KimHG1995/agent-bench/internal/domain"
	"github.com/KimHG1995/agent-bench/internal/graphhost"
	"github.com/KimHG1995/agent-bench/internal/mcpstdio"
)

const graphToolName = "inspect_typescript_graph"

var gitSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

type Config struct {
	BaseURL   string
	APIKey    string
	Model     string
	MaxTurns  int
	Client    *http.Client
	GraphHost string
	NodeBin   string
}

func ConfigFromEnv() Config {
	baseURL := envOr("AGENT_BENCH_OPENAI_BASE_URL", "https://api.openai.com/v1")
	key := strings.TrimSpace(os.Getenv("AGENT_BENCH_OPENAI_API_KEY"))
	if key == "" {
		if strings.Contains(baseURL, "orcarouter.ai") {
			key = strings.TrimSpace(os.Getenv("ORCAROUTER_API_KEY"))
		} else {
			key = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
		}
	}
	maxTurns := 10
	if raw := strings.TrimSpace(os.Getenv("AGENT_BENCH_OPENAI_MAX_TURNS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			maxTurns = n
		}
	}
	return Config{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		APIKey:    key,
		Model:     strings.TrimSpace(os.Getenv("AGENT_BENCH_OPENAI_MODEL")),
		MaxTurns:  maxTurns,
		Client:    &http.Client{Timeout: 120 * time.Second},
		GraphHost: strings.TrimSpace(os.Getenv("AGENT_BENCH_TS_GRAPH_HOST")),
		NodeBin:   envOr("AGENT_BENCH_NODE_BIN", "node"),
	}
}

func Run(ctx context.Context, req domain.RunRequest, cfg Config) (domain.AgentOutput, error) {
	state := domain.AgentOutput{
		Runtime: domain.AgentRuntime{
			Provider:       providerName(cfg.BaseURL),
			RequestedModel: cfg.Model,
			Adapter:        "openai-compatible",
			Endpoint:       cfg.BaseURL,
			MaxTurns:       cfg.MaxTurns,
		},
	}
	fail := func(err error) (domain.AgentOutput, error) {
		state.Metrics.Partial = true
		state.Error = err.Error()
		return state, err
	}

	if req.Strategy != "baseline" && req.Strategy != "graph" {
		return fail(fmt.Errorf("unsupported strategy %q", req.Strategy))
	}
	if cfg.APIKey == "" {
		return fail(fmt.Errorf("missing API key: set AGENT_BENCH_OPENAI_API_KEY, OPENAI_API_KEY, or ORCAROUTER_API_KEY"))
	}
	if cfg.Model == "" {
		return fail(fmt.Errorf("AGENT_BENCH_OPENAI_MODEL is required"))
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 120 * time.Second}
	}

	repo, err := filepath.Abs(req.Task.Repository.Path)
	if err != nil {
		return fail(err)
	}
	if info, err := os.Stat(repo); err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("not a directory")
		}
		return fail(fmt.Errorf("repository %s: %w", repo, err))
	}
	if err := verifyRevision(repo, req.Task.Repository.Revision); err != nil {
		return fail(err)
	}

	availableTools := toolSpecs()
	var graphClient *mcpstdio.Client
	if req.Strategy == "graph" {
		process, err := graphhost.Process(cfg.GraphHost, repo, cfg.NodeBin)
		if err != nil {
			return fail(err)
		}
		graphClient, err = mcpstdio.Start(ctx, process)
		if err != nil {
			return fail(err)
		}
		defer graphClient.Close()

		mcpTools, err := graphClient.ListTools()
		if err != nil {
			return fail(fmt.Errorf("list graph tools: %w", err))
		}
		found := false
		for _, t := range mcpTools {
			if t.Name != graphToolName {
				continue
			}
			found = true
			availableTools = append(availableTools, tool{
				Type: "function",
				Function: functionSpec{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.InputSchema,
				},
			})
		}
		if !found {
			return fail(fmt.Errorf("%s not advertised by graph MCP", graphToolName))
		}
	}

	messages := []message{
		{Role: "system", Content: systemPrompt(req.Strategy)},
		{Role: "user", Content: req.Task.Question},
	}

	var promptTokens, completionTokens int64
	var usageObserved bool
	var usageComplete = true
	var toolCalls, graphToolCalls int64
	state.Metrics.ToolCalls = &toolCalls
	state.Metrics.GraphToolCalls = &graphToolCalls

	for turn := 0; turn < cfg.MaxTurns; turn++ {
		resp, err := chat(ctx, cfg, messages, availableTools)
		if err != nil {
			if usageObserved {
				state.Metrics.InputTokens = &promptTokens
				state.Metrics.OutputTokens = &completionTokens
			}
			return fail(err)
		}
		if resp.Model != "" {
			state.Runtime.Model = resp.Model
		}
		if resp.Usage != nil {
			usageObserved = true
			promptTokens += resp.Usage.PromptTokens
			completionTokens += resp.Usage.CompletionTokens
		} else {
			usageComplete = false
		}
		if len(resp.Choices) == 0 {
			if usageObserved {
				state.Metrics.InputTokens = &promptTokens
				state.Metrics.OutputTokens = &completionTokens
			}
			return fail(fmt.Errorf("model returned no choices"))
		}
		msg := resp.Choices[0].Message

		if len(msg.ToolCalls) == 0 {
			result, err := parseFinal(msg.Content)
			if err != nil {
				if usageObserved {
					state.Metrics.InputTokens = &promptTokens
					state.Metrics.OutputTokens = &completionTokens
				}
				return fail(err)
			}
			state.Answer = result.Answer
			state.Evidence = result.Evidence
			if usageObserved {
				state.Metrics.InputTokens = &promptTokens
				state.Metrics.OutputTokens = &completionTokens
			}
			state.Metrics.Partial = !usageComplete
			return state, nil
		}

		messages = append(messages, msg)
		for _, tc := range msg.ToolCalls {
			toolCalls++
			var result any
			if tc.Function.Name == graphToolName {
				if req.Strategy != "graph" || graphClient == nil {
					result = map[string]any{"error": "graph tool unavailable in baseline strategy"}
				} else {
					graphToolCalls++
					var args map[string]any
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
						result = map[string]any{"error": "invalid graph arguments: " + err.Error()}
					} else {
						call, err := graphClient.CallTool(graphToolName, args)
						if err != nil {
							result = map[string]any{"error": err.Error()}
						} else {
							result = call.Value()
						}
					}
				}
			} else {
				localResult, err := runTool(repo, tc)
				if err != nil {
					result = map[string]any{"error": err.Error()}
				} else {
					result = localResult
				}
			}
			b, _ := json.Marshal(result)
			messages = append(messages, message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    string(b),
			})
		}
	}
	if usageObserved {
		state.Metrics.InputTokens = &promptTokens
		state.Metrics.OutputTokens = &completionTokens
	}
	return fail(fmt.Errorf("max turns exceeded"))
}

func verifyRevision(repo, revision string) error {
	if !gitSHA.MatchString(strings.TrimSpace(revision)) {
		return nil
	}
	head, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("verify repository revision: %w", err)
	}
	if strings.TrimSpace(string(head)) != revision {
		return fmt.Errorf("repository revision mismatch: expected %s got %s", revision, strings.TrimSpace(string(head)))
	}
	status, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output()
	if err != nil {
		return fmt.Errorf("verify repository dirty state: %w", err)
	}
	if strings.TrimSpace(string(status)) != "" {
		return fmt.Errorf("repository has uncommitted changes")
	}
	return nil
}

func systemPrompt(strategy string) string {
	graph := ""
	if strategy == "graph" {
		graph = `
You also have inspect_typescript_graph. Prefer it for TypeScript symbol lookup, callers/callees, flow, impact, implementers, and architecture. Treat compiler-resolved graph facts as trusted evidence. Use file tools only when the graph does not carry the needed source-body evidence.`
	}
	return `You are a read-only coding agent in a benchmark.
Use tools to inspect the repository. Do not guess.` + graph + `
When you have enough evidence, respond with JSON only and no markdown:
{"answer":"...","evidence":{"symbols":[],"paths":[],"relationships":[{"from":"...","to":"..."}]}}
Paths must be relative to the repository root. Include only evidence relevant to the question.`
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Tools    []tool    `json:"tools"`
}

type message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type tool struct {
	Type     string       `json:"type"`
	Function functionSpec `json:"function"`
}

type functionSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type usage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Usage *usage `json:"usage,omitempty"`
}

func chat(ctx context.Context, cfg Config, messages []message, tools []tool) (chatResponse, error) {
	body, err := json.Marshal(chatRequest{Model: cfg.Model, Messages: messages, Tools: tools})
	if err != nil {
		return chatResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return chatResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := cfg.Client.Do(req)
	if err != nil {
		return chatResponse{}, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return chatResponse{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return chatResponse{}, fmt.Errorf("chat completions %s: %s", res.Status, strings.TrimSpace(string(data)))
	}
	var out chatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return chatResponse{}, fmt.Errorf("decode chat response: %w", err)
	}
	return out, nil
}

func toolSpecs() []tool {
	object := func(properties map[string]any, required ...string) map[string]any {
		schema := map[string]any{
			"type":                 "object",
			"properties":           properties,
			"additionalProperties": false,
		}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	}
	return []tool{
		{Type:"function", Function:functionSpec{Name:"list_files",Description:"List repository files under an optional directory.",Parameters:object(map[string]any{
			"path":map[string]any{"type":"string"},
		})}},
		{Type:"function", Function:functionSpec{Name:"search_text",Description:"Search text in repository files.",Parameters:object(map[string]any{
			"query":map[string]any{"type":"string"},
			"path":map[string]any{"type":"string"},
		},"query")}},
		{Type:"function", Function:functionSpec{Name:"read_file",Description:"Read a UTF-8 repository file with line numbers.",Parameters:object(map[string]any{
			"path":map[string]any{"type":"string"},
			"startLine":map[string]any{"type":"integer","minimum":1},
			"endLine":map[string]any{"type":"integer","minimum":1},
		},"path")}},
	}
}

func providerName(baseURL string) string {
	if strings.Contains(baseURL, "orcarouter.ai") { return "orcarouter" }
	if strings.Contains(baseURL, "openai.com") { return "openai" }
	return "openai-compatible"
}

type finalResult struct {
	Answer   string          `json:"answer"`
	Evidence domain.Evidence `json:"evidence"`
}

func parseFinal(content string) (finalResult, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	if content == "" || content == "null" {
		return finalResult{}, fmt.Errorf("final answer must be a JSON object")
	}
	var raw struct {
		Answer   *string         `json:"answer"`
		Evidence json.RawMessage `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return finalResult{}, fmt.Errorf("final answer is not valid JSON: %w", err)
	}
	if raw.Answer == nil || len(raw.Evidence) == 0 || string(raw.Evidence) == "null" {
		return finalResult{}, fmt.Errorf("final answer must contain answer and evidence")
	}
	var evidence struct {
		Symbols       *[]string             `json:"symbols"`
		Paths         *[]string             `json:"paths"`
		Relationships *[]domain.Relationship `json:"relationships"`
	}
	if err := json.Unmarshal(raw.Evidence, &evidence); err != nil {
		return finalResult{}, fmt.Errorf("invalid evidence: %w", err)
	}
	if evidence.Symbols == nil || evidence.Paths == nil || evidence.Relationships == nil {
		return finalResult{}, fmt.Errorf("evidence must contain symbols, paths, and relationships arrays")
	}
	return finalResult{
		Answer: *raw.Answer,
		Evidence: domain.Evidence{
			Symbols: *evidence.Symbols,
			Paths: *evidence.Paths,
			Relationships: *evidence.Relationships,
		},
	}, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" { return v }
	return fallback
}
