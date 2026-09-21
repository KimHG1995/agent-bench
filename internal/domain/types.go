package domain

import "time"

type Relationship struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Evidence struct {
	Symbols       []string       `json:"symbols,omitempty"`
	Paths         []string       `json:"paths,omitempty"`
	Relationships []Relationship `json:"relationships,omitempty"`
}

type RepositoryRef struct {
	Path     string `json:"path"`
	Revision string `json:"revision"`
}

type Task struct {
	ID         string        `json:"id"`
	Category   string        `json:"category"`
	Repository RepositoryRef `json:"repository"`
	Question   string        `json:"question"`
	Expected   Evidence      `json:"expected"`
}

type RunRequest struct {
	Strategy string `json:"strategy"`
	Run      int    `json:"run"`
	Task     Task   `json:"task"`
}

type AgentMetrics struct {
	ToolCalls    *int64 `json:"toolCalls,omitempty"`
	InputTokens  *int64 `json:"inputTokens,omitempty"`
	OutputTokens *int64 `json:"outputTokens,omitempty"`
}

type AgentOutput struct {
	Answer   string       `json:"answer"`
	Evidence Evidence     `json:"evidence"`
	Metrics  AgentMetrics `json:"metrics"`
}

type Grade struct {
	Accuracy   float64  `json:"accuracy"`
	Precision  float64  `json:"precision"`
	Recall     float64  `json:"recall"`
	Matched    []string `json:"matched"`
	Missing    []string `json:"missing"`
	Unexpected []string `json:"unexpected"`
}

type RunResult struct {
	TaskID     string        `json:"taskId"`
	Category   string        `json:"category"`
	Strategy   string        `json:"strategy"`
	Run        int           `json:"run"`
	Repository RepositoryRef `json:"repository"`
	StartedAt  time.Time     `json:"startedAt"`
	DurationMS int64         `json:"durationMs"`
	Accuracy   float64       `json:"accuracy"`
	Precision  float64       `json:"precision"`
	Recall     float64       `json:"recall"`
	Matched    []string      `json:"matched"`
	Missing    []string      `json:"missing"`
	Unexpected []string      `json:"unexpected"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
	Answer     string        `json:"answer,omitempty"`
	Metrics    AgentMetrics  `json:"metrics"`
}
