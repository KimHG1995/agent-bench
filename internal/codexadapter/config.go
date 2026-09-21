package codexadapter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/KimHG1995/agent-bench/internal/graphhost"
)

const outputSchema = `{"type":"object","additionalProperties":false,"required":["answer","evidence"],"properties":{"answer":{"type":"string","minLength":1},"evidence":{"type":"object","additionalProperties":false,"required":["symbols","paths","relationships"],"properties":{"symbols":{"type":"array","items":{"type":"string"}},"paths":{"type":"array","items":{"type":"string"}},"relationships":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["from","to"],"properties":{"from":{"type":"string"},"to":{"type":"string"}}}}}}}}`

const commonPrompt = `You are a read-only coding agent in a benchmark.
Use tools to inspect the current repository. Do not guess.
Work only in this repository. Do not inspect files outside it, other benchmark runs, credentials, or benchmark answer keys. Do not change files or access the network.
Use local shell tools to list, search, and read source files.
Return the structured answer and evidence required by the output schema.
Paths must be relative to the repository root. Include only evidence relevant to the question.
`

const graphPrompt = `You also have inspect_typescript_graph. Prefer it for TypeScript symbol lookup, callers/callees, flow, impact, implementers, and architecture. Use file tools when the graph does not carry the needed source-body evidence.
`

type Config struct {
	CodexBin, Model, Effort, ServiceTier string
	GraphHost, NodeBin, ArtifactRoot     string
	Timeout                              time.Duration
	RequireGraphUse                      bool
}

func ConfigFromEnv() (Config, error) {
	c := Config{CodexBin: envOr("AGENT_BENCH_CODEX_BIN", "codex"), Model: os.Getenv("AGENT_BENCH_CODEX_MODEL"),
		Effort: envOr("AGENT_BENCH_CODEX_EFFORT", "medium"), ServiceTier: envOr("AGENT_BENCH_CODEX_SERVICE_TIER", "default"),
		GraphHost: os.Getenv("AGENT_BENCH_TS_GRAPH_HOST"), NodeBin: envOr("AGENT_BENCH_NODE_BIN", "node"),
		ArtifactRoot: envOr("AGENT_BENCH_CODEX_ARTIFACTS", "results/codex-artifacts")}
	var err error
	c.Timeout, err = time.ParseDuration(envOr("AGENT_BENCH_CODEX_TIMEOUT", "5m"))
	if err != nil {
		return c, fmt.Errorf("invalid AGENT_BENCH_CODEX_TIMEOUT: %w", err)
	}
	c.RequireGraphUse, err = strconv.ParseBool(envOr("AGENT_BENCH_CODEX_REQUIRE_GRAPH_USE", "false"))
	if err != nil {
		return c, fmt.Errorf("invalid AGENT_BENCH_CODEX_REQUIRE_GRAPH_USE: %w", err)
	}
	return c, c.validate()
}

func (c Config) validate() error {
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("AGENT_BENCH_CODEX_MODEL must explicitly select a model")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("Codex timeout must be positive")
	}
	if c.CodexBin == "" || c.ArtifactRoot == "" {
		return fmt.Errorf("Codex binary and artifact root are required")
	}
	switch c.Effort {
	case "none", "minimal", "low", "medium", "high", "xhigh", "max", "ultra":
	default:
		return fmt.Errorf("invalid Codex effort %q", c.Effort)
	}
	switch c.ServiceTier {
	case "default", "priority", "flex":
	default:
		return fmt.Errorf("invalid Codex service tier %q", c.ServiceTier)
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
func quoted(s string) string { b, _ := json.Marshal(s); return string(b) }

func safeEnv() []string {
	var env []string
	for _, key := range []string{"HOME", "PATH", "USER", "LOGNAME", "SHELL", "TMPDIR", "LANG", "LC_ALL", "TERM", "CODEX_HOME", "SSL_CERT_FILE", "SSL_CERT_DIR"} {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	return env
}

func commonArgs(c Config) []string {
	args := []string{"exec", "--ignore-user-config", "--ephemeral", "--strict-config", "--json", "--color", "never", "--skip-git-repo-check", "-s", "read-only", "-m", c.Model}
	for _, setting := range []string{"model_reasoning_effort=" + quoted(c.Effort), "service_tier=" + quoted(c.ServiceTier), `forced_login_method="chatgpt"`, `approval_policy="never"`, `web_search="disabled"`, `project_doc_max_bytes=0`, `mcp_servers={}`} {
		args = append(args, "-c", setting)
	}
	args = append(args, "--enable", "skip_host_skill_discovery")
	for _, feature := range []string{"apps", "plugins", "hooks", "multi_agent", "multi_agent_v2", "browser_use", "browser_use_external", "computer_use", "image_generation", "in_app_browser", "memories", "shell_snapshot", "sleep_tool", "workspace_dependencies"} {
		args = append(args, "--disable", feature)
	}
	return args
}

func graphArgs(c Config, target string) ([]string, string, error) {
	process, err := graphhost.Process(c.GraphHost, target, c.NodeBin)
	if err != nil {
		return nil, "", err
	}
	args := []string{"-c", "mcp_servers.ts_graph.command=" + quoted(process.Command), "-c", "mcp_servers.ts_graph.args=[" + quoted(process.Args[0]) + "]", "-c", "mcp_servers.ts_graph.cwd=" + quoted(target)}
	files := []string{process.Args[0]}
	for _, entry := range process.Env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || (key != "TTSC_GRAPH_BINARY" && key != "TTSC_TSGO_BINARY") {
			continue
		}
		args = append(args, "-c", "mcp_servers.ts_graph.env."+key+"="+quoted(value))
		files = append(files, value)
	}
	for _, setting := range []string{`enabled_tools=["inspect_typescript_graph"]`, `required=true`, `startup_timeout_sec=30`, `tool_timeout_sec=90`, `tools.inspect_typescript_graph.approval_mode="approve"`} {
		args = append(args, "-c", "mcp_servers.ts_graph."+setting)
	}
	host, err := filepath.Abs(c.GraphHost)
	if err != nil {
		return nil, "", err
	}
	// Hash the installed graph JS tree as well as platform executables, not just its launcher.
	tree, err := treeHash(filepath.Join(host, "node_modules", "@ttsc", "graph"), false)
	if err != nil {
		return nil, "", err
	}
	parts := []string{tree}
	for _, path := range files {
		sum, err := fileHash(path)
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, sum)
	}
	sort.Strings(parts)
	return args, hashJSON(parts), nil
}
