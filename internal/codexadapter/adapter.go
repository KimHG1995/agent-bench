package codexadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

// Run makes one bounded CLI invocation. Raw artifacts are outside the fresh
// target snapshot and survive every failure; no provider/model fallback exists.
func Run(parent context.Context, req domain.RunRequest, cfg Config) (out domain.AgentOutput, runErr error) {
	out.Status = domain.RunStatus{Execution: "failed", Measurement: "invalid"}
	out.Runtime = domain.AgentRuntime{Provider: "openai", Adapter: "codex-cli", AuthMode: "chatgpt", RequestedModel: cfg.Model, Effort: cfg.Effort, ServiceTier: cfg.ServiceTier, ModelProvenance: "requested_only", ToolProfile: "codex-shell-v1", SandboxProfile: "read-only", BudgetPolicy: "wall-clock", TimeoutMS: cfg.Timeout.Milliseconds(), SchemaHash: hashJSON(outputSchema), PromptTemplateHash: hashJSON([]string{commonPrompt, graphPrompt})}
	if err := cfg.validate(); err != nil {
		return fail(out, "config", err)
	}
	if req.Strategy != "baseline" && req.Strategy != "graph" {
		return fail(out, "config", fmt.Errorf("unsupported strategy %q", req.Strategy))
	}
	if strings.TrimSpace(req.Task.Question) == "" {
		return fail(out, "config", fmt.Errorf("task question is required"))
	}
	ctx, cancel := context.WithTimeout(parent, cfg.Timeout)
	defer cancel()
	root, err := filepath.Abs(cfg.ArtifactRoot)
	if err != nil {
		return fail(out, "config", err)
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return fail(out, "artifact", err)
	}
	dir, err := os.MkdirTemp(root, req.Strategy+"-")
	if err != nil {
		return fail(out, "artifact", err)
	}
	out.Runtime.ArtifactDir = dir
	manifest := map[string]any{"strategy": req.Strategy, "run": req.Run, "taskId": req.Task.ID, "startedAt": time.Now().UTC(), "targetRevision": req.Task.Repository.Revision}
	defer func() {
		manifest["runtime"] = out.Runtime
		manifest["status"] = out.Status
		manifest["finishedAt"] = time.Now().UTC()
		if runErr != nil {
			manifest["error"] = runErr.Error()
		}
		b, e := json.MarshalIndent(manifest, "", "  ")
		if e == nil {
			e = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0600)
		}
		if e != nil && runErr == nil {
			out, runErr = fail(out, "artifact", e)
		}
	}()
	target := filepath.Join(dir, "target")
	if err = prepareTarget(ctx, req.Task.Repository.Path, req.Task.Repository.Revision, target); err != nil {
		return failContext(ctx, out, "target", err)
	}
	out.Runtime.TargetSnapshotHash, err = treeHash(target, true)
	if err != nil {
		return fail(out, "target", err)
	}
	before := out.Runtime.TargetSnapshotHash
	bin, err := exec.LookPath(cfg.CodexBin)
	if err != nil {
		return fail(out, "config", err)
	}
	bin, err = filepath.Abs(bin)
	if err != nil {
		return fail(out, "config", err)
	}
	out.Runtime.CLIBinaryHash, err = fileHash(bin)
	if err != nil {
		return fail(out, "config", err)
	}
	preflight := func(args ...string) (string, error) {
		cmd := exec.Command(bin, args...)
		cmd.Env = safeEnv()
		cmd.Dir = target
		var b bytes.Buffer
		cmd.Stdout = &b
		cmd.Stderr = &b
		err := runProcess(ctx, cmd)
		return strings.TrimSpace(b.String()), err
	}
	version, err := preflight("--version")
	if err != nil {
		return failContext(ctx, out, "config", fmt.Errorf("Codex version preflight failed: %w", err))
	}
	if !strings.HasPrefix(version, "codex-cli ") {
		return fail(out, "config", fmt.Errorf("unrecognized Codex CLI version"))
	}
	out.Runtime.CLIVersion = version
	login, err := preflight("login", "status")
	if err != nil || !strings.Contains(login, "Logged in using ChatGPT") {
		return failContext(ctx, out, "auth", fmt.Errorf("existing ChatGPT Codex login is required; run codex login separately"))
	}
	out.Runtime.ContextFingerprint, err = contextFingerprint()
	if err != nil {
		return fail(out, "config", err)
	}
	var graph []string
	if cfg.GraphHost != "" || req.Strategy == "graph" {
		graph, out.Runtime.GraphFingerprint, err = graphArgs(cfg, target)
		if err != nil {
			return fail(out, "graph_unavailable", err)
		}
	}
	args := commonArgs(cfg)
	out.Runtime.CommonConfigHash = hashJSON(struct {
		Args            []string
		RequireGraphUse bool
	}{args, cfg.RequireGraphUse})
	args = append(args, "-C", target, "--output-schema", filepath.Join(dir, "schema.json"), "-o", filepath.Join(dir, "answer.json"))
	if req.Strategy == "graph" {
		args = append(args, graph...)
	}
	args = append(args, "-")
	prompt := commonPrompt
	if req.Strategy == "graph" {
		prompt += graphPrompt
	}
	prompt += "\nQuestion: " + req.Task.Question + "\n"
	for name, value := range map[string]string{"prompt.txt": prompt, "schema.json": outputSchema} {
		if err = os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); err != nil {
			return fail(out, "artifact", err)
		}
	}
	manifest["command"] = append([]string{bin}, args...)
	manifest["promptHash"] = hashJSON(prompt)
	stdout, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fail(out, "artifact", err)
	}
	stderr, err := os.OpenFile(filepath.Join(dir, "stderr.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		_ = stdout.Close()
		return fail(out, "artifact", err)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = target
	cmd.Env = safeEnv()
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	processErr := runProcess(ctx, cmd)
	closeOut, closeErr := stdout.Close(), stderr.Close()
	manifest["exitCode"] = nil
	if cmd.ProcessState != nil {
		manifest["exitCode"] = cmd.ProcessState.ExitCode()
	}
	raw, err := os.Open(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		return fail(out, "artifact", err)
	}
	info, _ := raw.Stat()
	if info != nil && info.Size() > 64*1024*1024 {
		_ = raw.Close()
		return fail(out, "protocol", fmt.Errorf("Codex JSONL exceeds 64 MiB"))
	}
	parsed, parseErr := parseStream(raw, req.Strategy)
	_ = raw.Close()
	parsed.Runtime = out.Runtime
	out = parsed
	out.Status.Warnings = append(out.Status.Warnings, "requested model only: CLI did not expose the serving model", "host context is fingerprinted, not hermetically isolated; target dependencies are not installed")
	after, integrityErr := treeHash(target, true)
	manifest["targetUnchanged"] = integrityErr == nil && after == before
	if integrityErr != nil || after != before {
		return fail(out, "target_mutated", fmt.Errorf("target snapshot changed during execution"))
	}
	if ctx.Err() != nil {
		return failContext(ctx, out, "timeout", ctx.Err())
	}
	if closeOut != nil {
		return fail(out, "artifact", closeOut)
	}
	if closeErr != nil {
		return fail(out, "artifact", closeErr)
	}
	if processErr != nil {
		if parseErr != nil && out.Status.FailureKind != "protocol" && out.Status.FailureKind != "output_schema" {
			return out, parseErr
		}
		return fail(out, "process", fmt.Errorf("Codex process failed: %w (see stderr.log)", processErr))
	}
	if parseErr != nil {
		return out, parseErr
	}
	final, err := os.ReadFile(filepath.Join(dir, "answer.json"))
	if err != nil {
		return fail(out, "output_schema", err)
	}
	answer, err := decodeFinal(final)
	if err != nil {
		return fail(out, "output_schema", err)
	}
	if answer.Answer != out.Answer || !reflect.DeepEqual(answer.Evidence, out.Evidence) {
		return fail(out, "output_schema", fmt.Errorf("final output file differs from final event"))
	}
	if req.Strategy == "graph" && cfg.RequireGraphUse && (out.Metrics.GraphFactCallsSucceeded == nil || *out.Metrics.GraphFactCallsSucceeded == 0) {
		return fail(out, "graph_not_used", fmt.Errorf("connectivity smoke requires a successful graph fact call"))
	}
	return out, nil
}

func fail(out domain.AgentOutput, kind string, err error) (domain.AgentOutput, error) {
	out.Error = err.Error()
	out.Status.Measurement = "invalid"
	out.Status.FailureKind = kind
	if out.Status.Execution == "" {
		out.Status.Execution = "failed"
	}
	if out.Metrics.InputTokens == nil || out.Metrics.OutputTokens == nil {
		out.Metrics.Partial = true
	}
	return out, err
}

func failContext(ctx context.Context, out domain.AgentOutput, kind string, err error) (domain.AgentOutput, error) {
	if ctx.Err() != nil {
		kind = "timeout"
		err = ctx.Err()
		out.Metrics.Partial = true
		out.Status.Execution = "failed"
	}
	return fail(out, kind, err)
}
