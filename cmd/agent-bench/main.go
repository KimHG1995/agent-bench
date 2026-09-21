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
	runOffset := fs.Int("run-offset", 0, "offset added to run numbers, useful for split experiment files")
	timeout := fs.Duration("timeout", 120*time.Second, "timeout per run")
	out := fs.String("out", "results/results.jsonl", "output JSONL path")
	overwrite := fs.Bool("overwrite", false, "allow replacing an existing result file")
	failOnError := fs.Bool("fail-on-error", false, "return a non-zero exit code after persisting all runs if any run failed")
	experimentID := fs.String("experiment-id", os.Getenv("AGENT_BENCH_EXPERIMENT_ID"), "experiment identifier used for paired comparison")
	harnessRevision := fs.String("harness-revision", os.Getenv("AGENT_BENCH_HARNESS_REVISION"), "agent-bench revision")
	graphRevision := fs.String("graph-revision", os.Getenv("AGENT_BENCH_GRAPH_REVISION"), "graph tool revision")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*strategy) == "" {
		return fmt.Errorf("-strategy is required")
	}
	if strings.TrimSpace(*command) == "" {
		return fmt.Errorf("-command is required")
	}
	if *runOffset < 0 {
		return fmt.Errorf("-run-offset must be >= 0")
	}

	tasks, err := taskloader.LoadDir(*tasksDir)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return fmt.Errorf("no benchmark tasks found in %s", *tasksDir)
	}

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	writer, err := resultio.NewWriter(*out, *overwrite)
	if err != nil {
		return fmt.Errorf("open result file: %w", err)
	}

	r := runner.Runner{
		Agent: agent.CommandRunner{}, Timeout: *timeout,
		ExperimentID: *experimentID, HarnessRevision: *harnessRevision, GraphRevision: *graphRevision,
	}
	results, runErr := r.RunWithOffsetEach(tasks, *strategy, *command, *repeat, *runOffset, writer.Write)
	closeErr := writer.Close()
	if runErr != nil {
		return runErr
	}
	if closeErr != nil {
		return closeErr
	}

	succeeded := 0
	for _, result := range results {
		if result.Success {
			succeeded++
		}
	}
	fmt.Printf("wrote %d runs to %s (%d succeeded)\n", len(results), *out, succeeded)
	if *failOnError && succeeded != len(results) {
		return fmt.Errorf("%d of %d runs failed", len(results)-succeeded, len(results))
	}
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
