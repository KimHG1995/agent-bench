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

	evidence := fixtureEvidence(req.Task.ID)
	toolCalls := int64(2)
	inputTokens := int64(400)
	outputTokens := int64(100)

	out := domain.AgentOutput{
		Answer:   "mock fixture answer",
		Evidence: evidence,
		Metrics: domain.AgentMetrics{
			ToolCalls:    &toolCalls,
			InputTokens:  &inputTokens,
			OutputTokens: &outputTokens,
		},
		Runtime: domain.AgentRuntime{
			Provider: "mock",
			Model:    "fixture",
			Adapter:  "mock-agent",
		},
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func fixtureEvidence(id string) domain.Evidence {
	switch id {
	case "lookup-001":
		return domain.Evidence{
			Symbols: []string{"UserService.deleteUser"},
			Paths:   []string{"src/user/user.service.ts"},
		}
	case "caller-001":
		return domain.Evidence{
			Symbols: []string{"UserService.deleteUser", "UserController.deleteUser"},
			Paths:   []string{"src/user/user.service.ts", "src/user/user.controller.ts"},
			Relationships: []domain.Relationship{{
				From: "UserController.deleteUser",
				To:   "UserService.deleteUser",
			}},
		}
	default:
		return domain.Evidence{}
	}
}
