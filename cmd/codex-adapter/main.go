package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/KimHG1995/agent-bench/internal/codexadapter"
	"github.com/KimHG1995/agent-bench/internal/domain"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var req domain.RunRequest
	var out domain.AgentOutput
	err := json.NewDecoder(os.Stdin).Decode(&req)
	if err == nil {
		cfg, configErr := codexadapter.ConfigFromEnv()
		err = configErr
		if err == nil {
			out, err = codexadapter.Run(ctx, req, cfg)
		}
	}
	if err != nil {
		out.Error = err.Error()
		if out.Status.FailureKind == "" {
			out.Status = domain.RunStatus{Execution: "failed", Measurement: "invalid", FailureKind: "config"}
		}
		out.Metrics.Partial = out.Metrics.Partial || out.Metrics.InputTokens == nil || out.Metrics.OutputTokens == nil
	}
	if encodeErr := json.NewEncoder(os.Stdout).Encode(out); encodeErr != nil {
		fmt.Fprintln(os.Stderr, encodeErr)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
