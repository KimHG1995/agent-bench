package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

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
	cmd := exec.Command(shell, "-c", command)
	configureProcessGroup(cmd)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return domain.AgentOutput{}, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var runErr error
	select {
	case runErr = <-done:
	case <-ctx.Done():
		terminateProcessGroup(cmd)
		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
		}
		runErr = ctx.Err()
	}

	out, decodeErr := decodeOutput(stdout.Bytes())
	if runErr != nil {
		if decodeErr == nil {
			return out, runErr
		}
		if ctx.Err() != nil {
			return domain.AgentOutput{}, ctx.Err()
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = runErr.Error()
		}
		return domain.AgentOutput{}, fmt.Errorf("agent command failed: %s", msg)
	}
	if decodeErr != nil {
		return domain.AgentOutput{}, decodeErr
	}
	return out, nil
}

func decodeOutput(data []byte) (domain.AgentOutput, error) {
	if strings.TrimSpace(string(data)) == "" {
		return domain.AgentOutput{}, fmt.Errorf("agent returned empty stdout")
	}
	var out domain.AgentOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return domain.AgentOutput{}, fmt.Errorf("decode agent response: %w", err)
	}
	return out, nil
}
