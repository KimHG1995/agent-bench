package openaiadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

type Config struct {
	BaseURL  string
	APIKey   string
	Model    string
	MaxTurns int
	Client   *http.Client
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
		BaseURL:  strings.TrimRight(baseURL, "/"),
		APIKey:   key,
		Model:    strings.TrimSpace(os.Getenv("AGENT_BENCH_OPENAI_MODEL")),
		MaxTurns: maxTurns,
		Client:   &http.Client{Timeout: 120 * time.Second},
	}
}

func Run(ctx context.Context, req domain.RunRequest, cfg Config) (domain.AgentOutput, error) {
	if req.Strategy != "baseline" {
		return domain.AgentOutput{}, fmt.Errorf("openai-compatible adapter currently supports baseline only")
	}
	if cfg.APIKey == "" {
		return domain.AgentOutput{}, fmt.Errorf("missing API key: set AGENT_BENCH_OPENAI_API_KEY, OPENAI_API_KEY, or ORCAROUTER_API_KEY")
	}
	if cfg.Model == "" {
		return domain.AgentOutput{}, fmt.Errorf("AGENT_BENCH_OPENAI_MODEL is required")
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 120 * time.Second}
	}

	repo, err := filepath.Abs(req.Task.Repository.Path)
	if err != nil {
		return domain.AgentOutput{}, err
	}
	if info, err := os.Stat(repo); err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("not a directory")
		}
		return domain.AgentOutput{}, fmt.Errorf("repository %s: %w", repo, err)
	}

	messages := []message{
		{Role: "system", Content: systemPrompt()},
		{Role: "user", Content: req.Task.Question},
	}

	var promptTokens, completionTokens int64
	var toolCalls int64
	for turn := 0; turn < cfg.MaxTurns; turn++ {
		resp, err := chat(ctx, cfg, messages)
		if err != nil {
			return domain.AgentOutput{}, err
		}
		promptTokens += resp.Usage.PromptTokens
		completionTokens += resp.Usage.CompletionTokens
		if len(resp.Choices) == 0 {
			return domain.AgentOutput{}, fmt.Errorf("model returned no choices")
		}
		msg := resp.Choices[0].Message

		if len(msg.ToolCalls) == 0 {
			result, err := parseFinal(msg.Content)
			if err != nil {
				return domain.AgentOutput{}, err
			}
			return domain.AgentOutput{
				Answer:   result.Answer,
				Evidence: result.Evidence,
				Metrics: domain.AgentMetrics{
					ToolCalls:   &toolCalls,
					InputTokens: &promptTokens,
					OutputTokens:&completionTokens,
				},
				Runtime: domain.AgentRuntime{
					Provider: providerName(cfg.BaseURL),
					Model:    resp.Model,
					Adapter:  "openai-compatible",
				},
			}, nil
		}

		messages = append(messages, msg)
		for _, tc := range msg.ToolCalls {
			toolCalls++
			result, err := runTool(repo, tc)
			if err != nil {
				result = map[string]any{"error": err.Error()}
			}
			b, _ := json.Marshal(result)
			messages = append(messages, message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    string(b),
			})
		}
	}
	return domain.AgentOutput{}, fmt.Errorf("max turns exceeded")
}

func systemPrompt() string {
	return `You are a read-only coding agent in a benchmark.
Use tools to inspect the repository. Do not guess.
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

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
}

func chat(ctx context.Context, cfg Config, messages []message) (chatResponse, error) {
	body, err := json.Marshal(chatRequest{
		Model: cfg.Model,
		Messages: messages,
		Tools: toolSpecs(),
	})
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
		return map[string]any{"type":"object","properties":properties,"required":required,"additionalProperties":false}
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
	if strings.Contains(baseURL, "orcarouter.ai") {
		return "orcarouter"
	}
	if strings.Contains(baseURL, "openai.com") {
		return "openai"
	}
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
	var out finalResult
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return finalResult{}, fmt.Errorf("final answer is not valid JSON: %w", err)
	}
	return out, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
