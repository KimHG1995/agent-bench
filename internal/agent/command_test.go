package agent

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func TestCommandRunner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	command := `cat >/dev/null; printf '%s' '{"answer":"ok","evidence":{"symbols":["A"]},"metrics":{"toolCalls":1}}'`
	out, err := (CommandRunner{}).Run(ctx, command, domain.RunRequest{Task: domain.AgentTask{ID: "t"}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Answer != "ok" || len(out.Evidence.Symbols) != 1 || out.Metrics.ToolCalls == nil || *out.Metrics.ToolCalls != 1 {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestCommandRunnerTimeoutKillsProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process group assertion is unix-specific")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := (CommandRunner{}).Run(ctx, `sleep 1; true`, domain.RunRequest{})
	if err == nil {
		t.Fatal("expected timeout")
	}
	if elapsed := time.Since(start); elapsed > 300*time.Millisecond {
		t.Fatalf("timeout took too long: %s", elapsed)
	}
}

func TestCommandRunnerTimeoutPreservesAdapterPartialOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix signals")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	command := `trap 'printf '\''%s'\'' '\''{"answer":"partial","evidence":{},"metrics":{"inputTokens":7,"partial":true},"error":"cancelled"}'\''; exit 0' TERM; while :; do sleep .01; done`
	out, err := (CommandRunner{}).Run(ctx, command, domain.RunRequest{})
	if err == nil || out.Metrics.InputTokens == nil || *out.Metrics.InputTokens != 7 {
		t.Fatalf("lost partial output on cancellation: %#v %v", out, err)
	}
}
