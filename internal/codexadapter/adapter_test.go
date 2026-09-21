package codexadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/KimHG1995/agent-bench/internal/agent"
	"github.com/KimHG1995/agent-bench/internal/domain"
)

// The external CLI is the only fake. Run executes the real process boundary,
// snapshot creation, environment filtering, parser and integrity checks.
func TestMain(m *testing.M) {
	if len(os.Args) > 2 && os.Args[1] == "adapter-fixture" {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
		defer cancel()
		var req domain.RunRequest
		_ = json.NewDecoder(os.Stdin).Decode(&req)
		bin, _ := os.Executable()
		out, err := Run(ctx, req, Config{CodexBin: bin, Model: "test-model", Effort: "high", ServiceTier: "default", Timeout: 15 * time.Second, ArtifactRoot: os.Args[2]})
		_ = json.NewEncoder(os.Stdout).Encode(out)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("codex-cli 0.155.0-alpha.9.2")
		os.Exit(0)
	}
	if len(os.Args) > 1 && os.Args[1] == "login" {
		fmt.Fprintln(os.Stderr, "Logged in using ChatGPT")
		os.Exit(0)
	}
	if len(os.Args) > 1 && os.Args[1] == "exec" {
		args := os.Args[2:]
		value := func(key string) string {
			for n, a := range args {
				if a == key && n+1 < len(args) {
					return args[n+1]
				}
			}
			return ""
		}
		joined := strings.Join(args, "\n")
		for _, token := range []string{"--ignore-user-config", "--ephemeral", "--json", "forced_login_method=\"chatgpt\"", "approval_policy=\"never\""} {
			if !strings.Contains(joined, token) {
				fmt.Fprintln(os.Stderr, "missing required argument", token)
				os.Exit(9)
			}
		}
		if value("-s") != "read-only" || os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("CODEX_API_KEY") != "" || os.Getenv("ORCAROUTER_API_KEY") != "" {
			fmt.Fprintln(os.Stderr, "unsafe execution environment")
			os.Exit(9)
		}
		prompt, _ := io.ReadAll(os.Stdin)
		if strings.Contains(string(prompt), "secret-expected") {
			os.Exit(9)
		}
		if strings.Contains(string(prompt), "scenario:timeout") {
			fmt.Fprintf(os.Stderr, "child-pid=%d\n", os.Getpid())
			time.Sleep(30 * time.Second)
			os.Exit(0)
		}
		if strings.Contains(string(prompt), "scenario:mutate") {
			_ = os.WriteFile("mutated.ts", []byte("changed"), 0600)
		}
		answer := finalAnswer
		if strings.Contains(string(prompt), "scenario:mismatch") {
			answer = strings.ReplaceAll(answer, "A calls B", "different")
		}
		_ = os.WriteFile(value("-o"), []byte(answer), 0600)
		fmt.Print(finalEvent(), usageEvent)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestOuterHarnessTimeoutReapsNestedCodexProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix process groups")
	}
	repo, rev := fixtureRepo(t)
	bin, _ := os.Executable()
	artifacts := t.TempDir()
	// Race-instrumented helper processes each wait at exit; allow preflight to
	// finish before exercising cancellation of the actual nested exec process.
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	command := fmt.Sprintf("exec '%s' adapter-fixture '%s'", bin, artifacts)
	out, err := (agent.CommandRunner{}).Run(ctx, command, domain.RunRequest{Strategy: "baseline", Task: domain.AgentTask{Question: "scenario:timeout", Repository: domain.RepositoryRef{Path: repo, Revision: rev}}})
	if err == nil || out.Status.FailureKind != "timeout" || out.Runtime.ArtifactDir == "" {
		t.Fatalf("nested partial result lost: %#v %v", out, err)
	}
	b, err := os.ReadFile(filepath.Join(out.Runtime.ArtifactDir, "stderr.log"))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(string(b), "child-pid=")))
	if err != nil {
		t.Fatalf("child never launched: %s", b)
	}
	process, _ := os.FindProcess(pid)
	if process.Signal(syscall.Signal(0)) == nil {
		_ = process.Kill()
		t.Fatal("nested Codex process survived outer timeout")
	}
}

func fixtureRepo(t *testing.T) (string, string) {
	t.Helper()
	p := t.TempDir()
	if err := os.WriteFile(filepath.Join(p, "a.ts"), []byte("export const A = 1;\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "a.ts"}, {"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "fixture"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = p
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %s %v", b, err)
		}
	}
	cmd := exec.Command("git", "-C", p, "rev-parse", "HEAD")
	b, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return p, strings.TrimSpace(string(b))
}

func fixtureConfig(t *testing.T) Config {
	t.Helper()
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return Config{CodexBin: bin, Model: "test-model", Effort: "high", ServiceTier: "default", Timeout: 10 * time.Second, ArtifactRoot: filepath.Join(t.TempDir(), "artifacts with spaces")}
}

func TestRunUsesSafeCLIAndPreservesArtifacts(t *testing.T) {
	for _, key := range []string{"OPENAI_API_KEY", "CODEX_API_KEY", "ORCAROUTER_API_KEY"} {
		t.Setenv(key, "must-not-reach-child")
	}
	repo, rev := fixtureRepo(t)
	cfg := fixtureConfig(t)
	out, err := Run(context.Background(), domain.RunRequest{Strategy: "baseline", Run: 1, Task: domain.AgentTask{Question: "Trace A", Repository: domain.RepositoryRef{Path: repo, Revision: rev}}}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if out.Answer != "A calls B" || out.Runtime.Model != "" || out.Runtime.ModelProvenance != "requested_only" || out.Runtime.CommonConfigHash == "" || out.Runtime.TargetSnapshotHash == "" {
		t.Fatalf("bad runtime: %#v", out)
	}
	for _, name := range []string{"events.jsonl", "stderr.log", "answer.json", "manifest.json", "prompt.txt", "schema.json"} {
		if _, err := os.Stat(filepath.Join(out.Runtime.ArtifactDir, name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunRejectsChangedTargetAndFinalMismatch(t *testing.T) {
	for _, tc := range []struct{ scenario, kind string }{{"mutate", "target_mutated"}, {"mismatch", "output_schema"}} {
		t.Run(tc.scenario, func(t *testing.T) {
			repo, rev := fixtureRepo(t)
			out, err := Run(context.Background(), domain.RunRequest{Strategy: "baseline", Task: domain.AgentTask{Question: "scenario:" + tc.scenario, Repository: domain.RepositoryRef{Path: repo, Revision: rev}}}, fixtureConfig(t))
			if err == nil || out.Status.FailureKind != tc.kind || out.Metrics.InputTokens == nil {
				t.Fatalf("not rejected/preserved: %#v %v", out, err)
			}
		})
	}
}

func TestRunTimeoutIsBounded(t *testing.T) {
	repo, rev := fixtureRepo(t)
	cfg := fixtureConfig(t)
	cfg.Timeout = 300 * time.Millisecond
	start := time.Now()
	out, err := Run(context.Background(), domain.RunRequest{Strategy: "baseline", Task: domain.AgentTask{Question: "scenario:timeout", Repository: domain.RepositoryRef{Path: repo, Revision: rev}}}, cfg)
	if err == nil || out.Status.FailureKind != "timeout" || !out.Metrics.Partial {
		t.Fatalf("bad timeout: %#v %v", out, err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("timeout did not reap child")
	}
}

func TestRunRejectsUnpinnedAndDirtySource(t *testing.T) {
	for _, scenario := range []string{"revision", "dirty", "project-config", "escaping-symlink"} {
		t.Run(scenario, func(t *testing.T) {
			repo, rev := fixtureRepo(t)
			switch scenario {
			case "revision":
				rev = "main"
			case "dirty":
				_ = os.WriteFile(filepath.Join(repo, "a.ts"), []byte("changed"), 0600)
			case "project-config":
				_ = os.Mkdir(filepath.Join(repo, ".codex"), 0700)
				_ = os.WriteFile(filepath.Join(repo, ".codex", "config.toml"), []byte("model='other'"), 0600)
			case "escaping-symlink":
				_ = os.Symlink("/etc/passwd", filepath.Join(repo, "external"))
			}
			if scenario == "project-config" || scenario == "escaping-symlink" {
				for _, args := range [][]string{{"add", "."}, {"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "unsafe fixture"}} {
					cmd := exec.Command("git", args...)
					cmd.Dir = repo
					if b, err := cmd.CombinedOutput(); err != nil {
						t.Fatalf("%s %v", b, err)
					}
				}
				b, _ := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
				rev = strings.TrimSpace(string(b))
			}
			_, err := Run(context.Background(), domain.RunRequest{Strategy: "baseline", Task: domain.AgentTask{Question: "Trace A", Repository: domain.RepositoryRef{Path: repo, Revision: rev}}}, fixtureConfig(t))
			if err == nil {
				t.Fatal("unsafe target accepted")
			}
		})
	}
}

func TestConfigRejectsBadBudgetAndRequiresModel(t *testing.T) {
	t.Setenv("AGENT_BENCH_CODEX_MODEL", "")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("missing model accepted")
	}
	t.Setenv("AGENT_BENCH_CODEX_MODEL", "test-model")
	t.Setenv("AGENT_BENCH_CODEX_TIMEOUT", "oops")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("invalid timeout accepted")
	}
}
