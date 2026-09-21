package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/KimHG1995/agent-bench/internal/domain"
	"github.com/KimHG1995/agent-bench/internal/grader"
)

type Agent interface {
	Run(context.Context, string, domain.RunRequest) (domain.AgentOutput, error)
}

type Runner struct {
	Agent           Agent
	Timeout         time.Duration
	ExperimentID    string
	HarnessRevision string
	GraphRevision   string
	StopOnFatal     bool
}

func (r Runner) Run(tasks []domain.Task, strategy, command string, repeat int) []domain.RunResult {
	results, _ := r.RunWithOffsetEach(tasks, strategy, command, repeat, 0, nil)
	return results
}

func (r Runner) RunWithOffset(tasks []domain.Task, strategy, command string, repeat, runOffset int) []domain.RunResult {
	results, _ := r.RunWithOffsetEach(tasks, strategy, command, repeat, runOffset, nil)
	return results
}

func (r Runner) RunWithOffsetEach(
	tasks []domain.Task,
	strategy, command string,
	repeat, runOffset int,
	onResult func(domain.RunResult) error,
) ([]domain.RunResult, error) {
	if repeat < 1 {
		repeat = 1
	}
	if runOffset < 0 {
		runOffset = 0
	}
	if r.Timeout <= 0 {
		r.Timeout = 120 * time.Second
	}

	results := make([]domain.RunResult, 0, len(tasks)*repeat)
	for _, task := range tasks {
		for i := 1; i <= repeat; i++ {
			run := runOffset + i
			result := r.runOne(task, strategy, command, run)
			results = append(results, result)
			if onResult != nil {
				if err := onResult(result); err != nil {
					return results, err
				}
			}
			if r.StopOnFatal && (result.Status.FailureKind == "auth" || result.Status.FailureKind == "quota") {
				return results, nil
			}
		}
	}
	return results, nil
}

func (r Runner) runOne(task domain.Task, strategy, command string, run int) domain.RunResult {
	started := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	request := domain.RunRequest{Strategy: strategy, Run: run, Task: task.ForAgent(), TimeoutMS: r.Timeout.Milliseconds()}
	output, err := r.Agent.Run(ctx, command, request)
	duration := time.Since(started)

	result := domain.RunResult{
		TaskID: task.ID, Category: task.Category, Strategy: strategy, Run: run,
		Repository: task.Repository,
		Experiment: domain.ExperimentMeta{
			ID:              r.ExperimentID,
			HarnessRevision: r.HarnessRevision,
			GraphRevision:   r.GraphRevision,
			TaskHash:        taskHash(task),
		},
		StartedAt: started, DurationMS: duration.Milliseconds(),
		Success: err == nil && output.Error == "" && output.Status.Measurement != "invalid",
		Answer:  output.Answer, Metrics: output.Metrics, Runtime: output.Runtime,
		Status: output.Status,
	}
	if err != nil {
		result.Error = err.Error()
		if output.Error != "" {
			result.Error = output.Error + ": " + result.Error
		}
		return result
	}
	if output.Error != "" {
		result.Error = output.Error
		return result
	}
	if output.Status.Measurement == "invalid" {
		result.Error = "invalid measurement: " + output.Status.FailureKind
		return result
	}

	grade := grader.Grade(task.Expected, output.Evidence)
	result.Accuracy = grade.Accuracy
	result.Precision = grade.Precision
	result.Recall = grade.Recall
	result.Matched = grade.Matched
	result.Missing = grade.Missing
	result.Unexpected = grade.Unexpected
	return result
}

func taskHash(task domain.Task) string {
	b, _ := json.Marshal(task)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
