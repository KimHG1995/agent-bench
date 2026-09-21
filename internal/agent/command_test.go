package agent

import (
	"context"
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

func TestCommandRunnerTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := (CommandRunner{}).Run(ctx, `sleep 1`, domain.RunRequest{})
	if err == nil {
		t.Fatal("expected timeout")
	}
}
