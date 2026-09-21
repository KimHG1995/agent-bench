# agent-bench

[한국어](README.md)

A reproducible Go benchmark harness for comparing how AI coding agents understand codebases under controlled conditions.

The harness is model and vendor agnostic. It runs external agents through a JSON stdin/stdout protocol and records deterministic accuracy, tool calls, token usage, and wall-clock latency.

The first intended comparison is:

- `baseline`: file search and file reading based context
- `graph`: the same agent with TypeScript Code Graph MCP access

> The included `mock-agent` only verifies the harness end to end. Its numbers are not baseline-vs-graph benchmark results.

## SDD

- [MVP specification](docs/001-mvp/spec.md)
- [MVP design](docs/001-mvp/design.md)
- [Implementation tasks](docs/001-mvp/tasks.md)

## Verify

```bash
go test ./...
```

## End-to-end harness check

```bash
go run ./cmd/agent-bench run -tasks benchmarks -strategy baseline -command 'go run ./examples/mock-agent' -repeat 2 -out results/baseline.jsonl
go run ./cmd/agent-bench run -tasks benchmarks -strategy graph -command 'go run ./examples/mock-agent' -repeat 2 -out results/graph.jsonl
go run ./cmd/agent-bench report -inputs results/baseline.jsonl,results/graph.jsonl -out results/report.md
```

See the Korean README and SDD documents for the full protocol, grading model, and roadmap.
