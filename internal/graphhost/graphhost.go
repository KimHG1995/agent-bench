package graphhost

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/KimHG1995/agent-bench/internal/mcpstdio"
)

func Process(host, repo, nodeBin string) (mcpstdio.ProcessConfig, error) {
	if host == "" {
		return mcpstdio.ProcessConfig{}, fmt.Errorf("AGENT_BENCH_TS_GRAPH_HOST is required for graph strategy")
	}
	if nodeBin == "" {
		nodeBin = "node"
	}
	host, err := filepath.Abs(host)
	if err != nil {
		return mcpstdio.ProcessConfig{}, err
	}
	repo, err = filepath.Abs(repo)
	if err != nil {
		return mcpstdio.ProcessConfig{}, err
	}
	platform, err := npmPlatform()
	if err != nil {
		return mcpstdio.ProcessConfig{}, err
	}
	server := filepath.Join(host, "node_modules", "@ttsc", "graph", "lib", "bin.js")
	graphBinary := filepath.Join(host, "node_modules", "@ttsc", platform, "bin", "ttscgraph")
	tsgoBinary := filepath.Join(host, "node_modules", "@typescript", "typescript-"+platform, "lib", "tsc")
	for _, path := range []string{server, graphBinary, tsgoBinary} {
		if _, err := os.Stat(path); err != nil {
			return mcpstdio.ProcessConfig{}, fmt.Errorf("graph dependency %s: %w", path, err)
		}
	}
	return mcpstdio.ProcessConfig{
		Command: nodeBin,
		Args:    []string{server},
		Dir:     repo,
		Env: mcpstdio.InheritEnv(
			"TTSC_GRAPH_BINARY="+graphBinary,
			"TTSC_TSGO_BINARY="+tsgoBinary,
		),
	}, nil
}

func npmPlatform() (string, error) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		return "darwin-arm64", nil
	case "darwin/amd64":
		return "darwin-x64", nil
	case "linux/amd64":
		return "linux-x64", nil
	case "linux/arm64":
		return "linux-arm64", nil
	default:
		return "", fmt.Errorf("unsupported graph platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}
