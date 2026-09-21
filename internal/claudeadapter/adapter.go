package claudeadapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

const graphToolNameDefault = "mcp__ts_graph__inspect_typescript_graph"

var outputSchema = `{"type":"object","additionalProperties":false,"properties":{"answer":{"type":"string"},"evidence":{"type":"object","additionalProperties":false,"properties":{"symbols":{"type":"array","items":{"type":"string"}},"paths":{"type":"array","items":{"type":"string"}},"relationships":{"type":"array","items":{"type":"object","additionalProperties":false,"properties":{"from":{"type":"string"},"to":{"type":"string"}},"required":["from","to"]}}},"required":["symbols","paths","relationships"]}},"required":["answer","evidence"]}`

type Config struct {
	ClaudeBin       string
	Model           string
	Effort          string
	MaxTurns        int
	GraphHost       string
	GraphConfigPath string
	GraphServerName string
	NodeBin         string
}

func ConfigFromEnv() Config {
	maxTurns := 8
	if raw := strings.TrimSpace(os.Getenv("AGENT_BENCH_CLAUDE_MAX_TURNS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			maxTurns = n
		}
	}
	return Config{
		ClaudeBin:       envOr("AGENT_BENCH_CLAUDE_BIN", "claude"),
		Model:           envOr("AGENT_BENCH_CLAUDE_MODEL", "claude-sonnet-5"),
		Effort:          envOr("AGENT_BENCH_CLAUDE_EFFORT", "medium"),
		MaxTurns:        maxTurns,
		GraphHost:       strings.TrimSpace(os.Getenv("AGENT_BENCH_TS_GRAPH_HOST")),
		GraphConfigPath: strings.TrimSpace(os.Getenv("AGENT_BENCH_GRAPH_MCP_CONFIG")),
		GraphServerName: envOr("AGENT_BENCH_GRAPH_SERVER", "ts_graph"),
		NodeBin:         envOr("AGENT_BENCH_NODE_BIN", "node"),
	}
}

func Run(ctx context.Context, req domain.RunRequest, cfg Config) (domain.AgentOutput, error) {
	if req.Strategy != "baseline" && req.Strategy != "graph" {
		return domain.AgentOutput{}, fmt.Errorf("unsupported strategy %q", req.Strategy)
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

	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--no-session-persistence",
		"--bare",
		"--disable-slash-commands",
		"--no-chrome",
		"--tools", "Read,Grep,Glob",
		"--strict-mcp-config",
		"--json-schema", outputSchema,
		"--max-turns", strconv.Itoa(cfg.MaxTurns),
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	if cfg.Effort != "" {
		args = append(args, "--effort", cfg.Effort)
	}

	graphToolName := "mcp__" + cfg.GraphServerName + "__inspect_typescript_graph"
	var cleanup func()
	if req.Strategy == "graph" {
		configPath := cfg.GraphConfigPath
		if configPath == "" {
			configPath, cleanup, err = writeGraphConfig(cfg)
			if err != nil {
				return domain.AgentOutput{}, err
			}
			defer cleanup()
		}
		args = append(args, "--mcp-config", configPath, "--allowedTools", graphToolName)
	}
	args = append(args, buildPrompt(req))

	cmd := exec.CommandContext(ctx, cfg.ClaudeBin, args...)
	cmd.Dir = repo

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return domain.AgentOutput{}, ctx.Err()
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return domain.AgentOutput{}, fmt.Errorf("claude failed: %s", msg)
	}

	out, err := parseStream(stdout.Bytes(), req.Strategy, graphToolName)
	if err != nil {
		return domain.AgentOutput{}, err
	}
	out.Runtime.Adapter = "claude-code"
	out.Runtime.Provider = "anthropic"
	out.Runtime.Effort = cfg.Effort
	if out.Runtime.Model == "" {
		out.Runtime.Model = cfg.Model
	}
	return out, nil
}

func buildPrompt(req domain.RunRequest) string {
	strategy := "Use only the built-in read/search tools to gather evidence."
	if req.Strategy == "graph" {
		strategy = "Use the configured TypeScript Code Graph MCP as the primary evidence source when it can answer the question. Use built-in read/search tools only when additional source evidence is necessary."
	}

	return fmt.Sprintf(`This is a read-only codebase understanding benchmark. Do not edit files, run shell commands, use the web, or infer facts without repository evidence.

%s

Question:
%s

Return exact symbol names and paths relative to the repository root. Include only evidence relevant to the question.`, strategy, req.Task.Question)
}

func writeGraphConfig(cfg Config) (string, func(), error) {
	if strings.TrimSpace(cfg.GraphHost) == "" {
		return "", func() {}, fmt.Errorf("graph strategy requires AGENT_BENCH_TS_GRAPH_HOST or AGENT_BENCH_GRAPH_MCP_CONFIG")
	}

	host, err := filepath.Abs(cfg.GraphHost)
	if err != nil {
		return "", func() {}, err
	}
	platform, err := npmPlatform()
	if err != nil {
		return "", func() {}, err
	}

	server := filepath.Join(host, "node_modules", "@ttsc", "graph", "lib", "bin.js")
	graphBinary := filepath.Join(host, "node_modules", "@ttsc", platform, "bin", "ttscgraph")
	tsgoBinary := filepath.Join(host, "node_modules", "@typescript", "typescript-"+platform, "lib", "tsc")

	for _, path := range []string{server, graphBinary, tsgoBinary} {
		if _, err := os.Stat(path); err != nil {
			return "", func() {}, fmt.Errorf("graph dependency %s: %w", path, err)
		}
	}

	payload := map[string]any{
		"mcpServers": map[string]any{
			cfg.GraphServerName: map[string]any{
				"command": cfg.NodeBin,
				"args":    []string{server},
				"env": map[string]string{
					"TTSC_GRAPH_BINARY": graphBinary,
					"TTSC_TSGO_BINARY":  tsgoBinary,
				},
			},
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", func() {}, err
	}

	f, err := os.CreateTemp("", "agent-bench-mcp-*.json")
	if err != nil {
		return "", func() {}, err
	}
	name := f.Name()
	cleanup := func() { _ = os.Remove(name) }

	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}

	return name, cleanup, nil
}

func npmPlatform() (string, error) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		return "darwin-arm64", nil
	case "darwin/amd64":
		return "darwin-x64", nil
	case "linux/amd64":
		return "linux-x64", nil
	case "linux/arm64":
		return "linux-arm64", nil
	default:
		return "", fmt.Errorf("unsupported graph platform %s/%s; set AGENT_BENCH_GRAPH_MCP_CONFIG", runtime.GOOS, runtime.GOARCH)
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

type streamEvent struct {
	Type             string                `json:"type"`
	Subtype          string                `json:"subtype"`
	StructuredOutput json.RawMessage       `json:"structured_output"`
	TotalCostUSD     *float64              `json:"total_cost_usd"`
	Usage            *usage                `json:"usage"`
	ModelUsage       map[string]modelUsage `json:"modelUsage"`
	Message          *assistantMessage     `json:"message"`
	Errors           []string              `json:"errors"`
}

type usage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

type modelUsage struct {
	InputTokens              int64   `json:"inputTokens"`
	OutputTokens             int64   `json:"outputTokens"`
	CacheCreationInputTokens int64   `json:"cacheCreationInputTokens"`
	CacheReadInputTokens     int64   `json:"cacheReadInputTokens"`
	CostUSD                  float64 `json:"costUSD"`
}

type assistantMessage struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Usage   usage          `json:"usage"`
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type structured struct {
	Answer   string          `json:"answer"`
	Evidence domain.Evidence `json:"evidence"`
}

func parseStream(data []byte, strategy, graphToolName string) (domain.AgentOutput, error) {
	s := bufio.NewScanner(bytes.NewReader(data))
	s.Buffer(make([]byte, 64*1024), 8*1024*1024)

	seenMessages := map[string]usage{}
	seenTools := map[string]struct{}{}
	var totalTools, graphTools int64
	var final *streamEvent
	model := ""

	for s.Scan() {
		var ev streamEvent
		if err := json.Unmarshal(s.Bytes(), &ev); err != nil {
			return domain.AgentOutput{}, fmt.Errorf("decode claude stream: %w", err)
		}

		if ev.Type == "assistant" && ev.Message != nil {
			if model == "" {
				model = ev.Message.Model
			}
			if ev.Message.ID != "" {
				seenMessages[ev.Message.ID] = ev.Message.Usage
			}

			for i, block := range ev.Message.Content {
				if !strings.HasSuffix(block.Type, "tool_use") {
					continue
				}
				key := block.ID
				if key == "" {
					key = ev.Message.ID + ":" + strconv.Itoa(i) + ":" + block.Name
				}
				if _, ok := seenTools[key]; ok {
					continue
				}
				seenTools[key] = struct{}{}
				totalTools++

				if block.Name == graphToolName {
					graphTools++
				}
				if strings.HasPrefix(block.Name, "mcp__") {
					if strategy == "baseline" {
						return domain.AgentOutput{}, fmt.Errorf("baseline used MCP tool %s", block.Name)
					}
					if block.Name != graphToolName {
						return domain.AgentOutput{}, fmt.Errorf("graph strategy used unexpected MCP tool %s", block.Name)
					}
				}
			}
		}

		if ev.Type == "result" {
			copy := ev
			final = &copy
		}
	}
	if err := s.Err(); err != nil {
		return domain.AgentOutput{}, err
	}
	if final == nil {
		return domain.AgentOutput{}, fmt.Errorf("claude stream missing result event")
	}
	if final.Subtype != "success" {
		return domain.AgentOutput{}, fmt.Errorf("claude result subtype %q: %s", final.Subtype, strings.Join(final.Errors, ","))
	}
	if len(final.StructuredOutput) == 0 || string(final.StructuredOutput) == "null" {
		return domain.AgentOutput{}, fmt.Errorf("claude result missing structured_output")
	}

	var parsed structured
	if err := json.Unmarshal(final.StructuredOutput, &parsed); err != nil {
		return domain.AgentOutput{}, fmt.Errorf("decode structured output: %w", err)
	}

	metrics := domain.AgentMetrics{
		ToolCalls:      &totalTools,
		GraphToolCalls: &graphTools,
		CostUSD:        final.TotalCostUSD,
	}

	if len(final.ModelUsage) > 0 {
		var in, out, cacheCreate, cacheRead int64
		models := make([]string, 0, len(final.ModelUsage))
		for name, u := range final.ModelUsage {
			models = append(models, name)
			in += u.InputTokens
			out += u.OutputTokens
			cacheCreate += u.CacheCreationInputTokens
			cacheRead += u.CacheReadInputTokens
		}
		metrics.InputTokens = &in
		metrics.OutputTokens = &out
		metrics.CacheCreationInputTokens = &cacheCreate
		metrics.CacheReadInputTokens = &cacheRead
		if model == "" && len(models) == 1 {
			model = models[0]
		}
	} else if final.Usage != nil {
		in := final.Usage.InputTokens
		out := final.Usage.OutputTokens
		cacheCreate := final.Usage.CacheCreationInputTokens
		cacheRead := final.Usage.CacheReadInputTokens
		metrics.InputTokens = &in
		metrics.OutputTokens = &out
		metrics.CacheCreationInputTokens = &cacheCreate
		metrics.CacheReadInputTokens = &cacheRead
	} else {
		var in, out, cacheCreate, cacheRead int64
		for _, u := range seenMessages {
			in += u.InputTokens
			out += u.OutputTokens
			cacheCreate += u.CacheCreationInputTokens
			cacheRead += u.CacheReadInputTokens
		}
		metrics.InputTokens = &in
		metrics.OutputTokens = &out
		metrics.CacheCreationInputTokens = &cacheCreate
		metrics.CacheReadInputTokens = &cacheRead
	}

	return domain.AgentOutput{
		Answer:   parsed.Answer,
		Evidence: parsed.Evidence,
		Metrics:  metrics,
		Runtime: domain.AgentRuntime{
			Provider: "anthropic",
			Model:    model,
			Adapter:  "claude-code",
		},
	}, nil
}
