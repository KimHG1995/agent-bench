package runner

import (
	"context"
	"time"

	"github.com/KimHG1995/agent-bench/internal/domain"
	"github.com/KimHG1995/agent-bench/internal/grader"
)

type Agent interface {
	Run(context.Context, string, domain.RunRequest) (domain.AgentOutput, error)
}

type Runner struct {
	Agent   Agent
	Timeout time.Duration
}

func (r Runner) Run(tasks []domain.Task, strategy, command string, repeat int) []domain.RunResult {
	if repeat < 1 {
		repeat = 1
	}
	if r.Timeout <= 0 {
		r.Timeout = 120 * time.Second
	}

	results := make([]domain.RunResult, 0, len(tasks)*repeat)
	for _, task := range tasks {
		for run := 1; run <= repeat; run++ {
			results = append(results, r.runOne(task, strategy, command, run))
		}
	}
	return results
}

func (r Runner) runOne(task domain.Task, strategy, command string, run int) domain.RunResult {
	started := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	request := domain.RunRequest{Strategy: strategy, Run: run, Task: task}
	output, err := r.Agent.Run(ctx, command, request)
	duration := time.Since(started)

	result := domain.RunResult{
		TaskID: task.ID, Category: task.Category, Strategy: strategy, Run: run,
		Repository: task.Repository, StartedAt: started, DurationMS: duration.Milliseconds(),
		Success: err == nil,
	}
	if err != nil {
		result.Error = err.Error()
		return result
	}

	grade := grader.Grade(task.Expected, output.Evidence)
	result.Accuracy = grade.Accuracy
	result.Precision = grade.Precision
	result.Recall = grade.Recall
	result.Matched = grade.Matched
	result.Missing = grade.Missing
	result.Unexpected = grade.Unexpected
	result.Answer = output.Answer
	result.Metrics = output.Metrics
	return result
}
