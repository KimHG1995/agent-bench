package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/KimHG1995/agent-bench/internal/graphhost"
	"github.com/KimHG1995/agent-bench/internal/mcpstdio"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: graph-smoke <repository>")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	process, err := graphhost.Process(os.Getenv("AGENT_BENCH_TS_GRAPH_HOST"), os.Args[1], "node")
	if err != nil {
		fail(err)
	}
	client, err := mcpstdio.Start(ctx, process)
	if err != nil {
		fail(err)
	}
	defer client.Close()

	tools, err := client.ListTools()
	if err != nil {
		fail(err)
	}
	found := false
	for _, tool := range tools {
		if tool.Name == "inspect_typescript_graph" {
			found = true
			break
		}
	}
	if !found {
		fail(fmt.Errorf("inspect_typescript_graph not found"))
	}

	result, err := client.CallTool("inspect_typescript_graph", map[string]any{
		"question": "Where is ReportsService.latency declared?",
		"draft": map[string]any{
			"reason": "A named symbol lookup is sufficient.",
			"type":   "lookup",
		},
		"review": "Use one lookup request.",
		"request": map[string]any{
			"type":  "lookup",
			"query": "ReportsService.latency",
			"limit": 5,
		},
	})
	if err != nil {
		fail(err)
	}
	b, _ := json.Marshal(result.Value())
	fmt.Println(string(b))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
