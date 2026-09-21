package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KimHG1995/agent-bench/internal/agent"
	"github.com/KimHG1995/agent-bench/internal/domain"
	"github.com/KimHG1995/agent-bench/internal/report"
	"github.com/KimHG1995/agent-bench/internal/resultio"
	"github.com/KimHG1995/agent-bench/internal/runner"
	"github.com/KimHG1995/agent-bench/internal/taskloader"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "run":
		err = runCommand(os.Args[2:])
	case "report":
		err = reportCommand(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		usage()
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runCommand(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	tasksDir := fs.String("tasks", "benchmarks", "directory containing benchmark task JSON files")
	strategy := fs.String("strategy", "", "strategy name, e.g. baseline or graph")
	command := fs.String("command", "", "external agent command")
	repeat := fs.Int("repeat", 1, "number of runs per task")
	timeout := fs.Duration("timeout", 120*time.Second, "timeout per run")
	out := fs.String("out", "results/results.jsonl", "output JSONL path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*strategy) == "" {
		return fmt.Errorf("-strategy is required")
	}
	if strings.TrimSpace(*command) == "" {
		return fmt.Errorf("-command is required")
	}

	tasks, err := taskloader.LoadDir(*tasksDir)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return fmt.Errorf("no benchmark tasks found in %s", *tasksDir)
	}

	r := runner.Runner{Agent: agent.CommandRunner{}, Timeout: *timeout}
	results := r.Run(tasks, *strategy, *command, *repeat)
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	if err := resultio.WriteJSONL(*out, results); err != nil {
		return err
	}

	succeeded := 0
	for _, result := range results {
		if result.Success {
			succeeded++
		}
	}
	fmt.Printf("wrote %d runs to %s (%d succeeded)\n", len(results), *out, succeeded)
	return nil
}

func reportCommand(args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	inputs := fs.String("inputs", "", "comma-separated JSONL result files")
	out := fs.String("out", "results/report.md", "output Markdown path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*inputs) == "" {
		return fmt.Errorf("-inputs is required")
	}

	var merged []domain.RunResult
	for _, path := range strings.Split(*inputs, ",") {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		rows, err := resultio.ReadJSONL(path)
		if err != nil {
			return err
		}
		merged = append(merged, rows...)
	}
	if len(merged) == 0 {
		return fmt.Errorf("no results loaded")
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(*out, []byte(report.Markdown(merged)), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote report to %s\n", *out)
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "agent-bench: reproducible benchmark harness for coding agents")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  run     execute benchmark tasks against an external agent")
	fmt.Fprintln(os.Stderr, "  report  aggregate JSONL result files into Markdown")
}
