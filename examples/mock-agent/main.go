package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func main() {
	var req domain.RunRequest
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	toolCalls := int64(7)
	inputTokens := int64(1200)
	outputTokens := int64(180)
	if req.Strategy == "graph" {
		toolCalls = 2
		inputTokens = 420
	}

	out := domain.AgentOutput{
		Answer:   "mock agent returns the expected evidence so the harness can be verified end-to-end",
		Evidence: req.Task.Expected,
		Metrics: domain.AgentMetrics{
			ToolCalls:    &toolCalls,
			InputTokens:  &inputTokens,
			OutputTokens: &outputTokens,
		},
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
