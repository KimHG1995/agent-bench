package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

type CommandRunner struct {
	Shell string
}

func (r CommandRunner) Run(ctx context.Context, command string, req domain.RunRequest) (domain.AgentOutput, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return domain.AgentOutput{}, fmt.Errorf("encode request: %w", err)
	}

	shell := r.Shell
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.CommandContext(ctx, shell, "-c", command)
	cmd.Stdin = bytes.NewReader(payload)
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
		return domain.AgentOutput{}, fmt.Errorf("agent command failed: %s", msg)
	}
	if strings.TrimSpace(stdout.String()) == "" {
		return domain.AgentOutput{}, fmt.Errorf("agent returned empty stdout")
	}

	var out domain.AgentOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return domain.AgentOutput{}, fmt.Errorf("decode agent response: %w", err)
	}
	return out, nil
}
