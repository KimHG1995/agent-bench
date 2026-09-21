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

type AgentTask struct {
	ID         string        `json:"id"`
	Category   string        `json:"category"`
	Repository RepositoryRef `json:"repository"`
	Question   string        `json:"question"`
}

func (t Task) ForAgent() AgentTask {
	return AgentTask{
		ID:         t.ID,
		Category:   t.Category,
		Repository: t.Repository,
		Question:   t.Question,
	}
}

type RunRequest struct {
	Strategy string    `json:"strategy"`
	Run      int       `json:"run"`
	Task     AgentTask `json:"task"`
}

type AgentRuntime struct {
	Provider           string `json:"provider,omitempty"`
	RequestedModel     string `json:"requestedModel,omitempty"`
	Model              string `json:"model,omitempty"`
	Adapter            string `json:"adapter,omitempty"`
	Effort             string `json:"effort,omitempty"`
	Endpoint           string `json:"endpoint,omitempty"`
	MaxTurns           int    `json:"maxTurns,omitempty"`
	AuthMode           string `json:"authMode,omitempty"`
	CLIVersion         string `json:"cliVersion,omitempty"`
	CLIBinaryHash      string `json:"cliBinaryHash,omitempty"`
	ServiceTier        string `json:"serviceTier,omitempty"`
	ModelProvenance    string `json:"modelProvenance,omitempty"`
	CommonConfigHash   string `json:"commonConfigHash,omitempty"`
	PromptTemplateHash string `json:"promptTemplateHash,omitempty"`
	SchemaHash         string `json:"schemaHash,omitempty"`
	ToolProfile        string `json:"toolProfile,omitempty"`
	SandboxProfile     string `json:"sandboxProfile,omitempty"`
	TimeoutMS          int64  `json:"timeoutMs,omitempty"`
	BudgetPolicy       string `json:"budgetPolicy,omitempty"`
	GraphFingerprint   string `json:"graphFingerprint,omitempty"`
	ContextFingerprint string `json:"contextFingerprint,omitempty"`
	TargetSnapshotHash string `json:"targetSnapshotHash,omitempty"`
	ArtifactDir        string `json:"artifactDir,omitempty"`
}

type AgentMetrics struct {
	ToolCalls                *int64   `json:"toolCalls,omitempty"`
	GraphToolCalls           *int64   `json:"graphToolCalls,omitempty"`
	InputTokens              *int64   `json:"inputTokens,omitempty"`
	OutputTokens             *int64   `json:"outputTokens,omitempty"`
	CacheCreationInputTokens *int64   `json:"cacheCreationInputTokens,omitempty"`
	CacheReadInputTokens     *int64   `json:"cacheReadInputTokens,omitempty"`
	CostUSD                  *float64 `json:"costUsd,omitempty"`
	Partial                  bool     `json:"partial,omitempty"`
	ToolCallsSucceeded       *int64   `json:"toolCallsSucceeded,omitempty"`
	GraphToolCallsSucceeded  *int64   `json:"graphToolCallsSucceeded,omitempty"`
	GraphFactCallsSucceeded  *int64   `json:"graphFactCallsSucceeded,omitempty"`
	GraphUsed                *bool    `json:"graphUsed,omitempty"`
	ReasoningOutputTokens    *int64   `json:"reasoningOutputTokens,omitempty"`
}

// RunStatus is additive: legacy adapters leave it empty. Execution is the
// CLI's terminal state; Measurement records whether experiment conditions held.
type RunStatus struct {
	Execution   string   `json:"execution,omitempty"`
	Measurement string   `json:"measurement,omitempty"`
	FailureKind string   `json:"failureKind,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

type AgentOutput struct {
	Answer   string       `json:"answer,omitempty"`
	Evidence Evidence     `json:"evidence"`
	Metrics  AgentMetrics `json:"metrics"`
	Runtime  AgentRuntime `json:"runtime"`
	Error    string       `json:"error,omitempty"`
	Status   RunStatus    `json:"status,omitempty"`
}

type Grade struct {
	Accuracy   float64  `json:"accuracy"`
	Precision  float64  `json:"precision"`
	Recall     float64  `json:"recall"`
	Matched    []string `json:"matched"`
	Missing    []string `json:"missing"`
	Unexpected []string `json:"unexpected"`
}

type ExperimentMeta struct {
	ID              string `json:"id,omitempty"`
	HarnessRevision string `json:"harnessRevision,omitempty"`
	GraphRevision   string `json:"graphRevision,omitempty"`
	TaskHash        string `json:"taskHash"`
}

type RunResult struct {
	TaskID     string         `json:"taskId"`
	Category   string         `json:"category"`
	Strategy   string         `json:"strategy"`
	Run        int            `json:"run"`
	Repository RepositoryRef  `json:"repository"`
	Experiment ExperimentMeta `json:"experiment"`
	StartedAt  time.Time      `json:"startedAt"`
	DurationMS int64          `json:"durationMs"`
	Accuracy   float64        `json:"accuracy"`
	Precision  float64        `json:"precision"`
	Recall     float64        `json:"recall"`
	Matched    []string       `json:"matched"`
	Missing    []string       `json:"missing"`
	Unexpected []string       `json:"unexpected"`
	Success    bool           `json:"success"`
	Error      string         `json:"error,omitempty"`
	Answer     string         `json:"answer,omitempty"`
	Metrics    AgentMetrics   `json:"metrics"`
	Runtime    AgentRuntime   `json:"runtime"`
	Status     RunStatus      `json:"status,omitempty"`
}
