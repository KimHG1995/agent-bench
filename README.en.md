# agent-bench

[한국어](README.md)

A reproducible benchmark harness written in Go for comparing how the same AI coding agent understands a codebase under controlled context conditions.

Current comparison:

- `baseline`: Claude Code with Read, Grep, and Glob
- `graph`: the same model and built-in tools plus one TypeScript Code Graph MCP tool

Agent Bench records deterministic evidence accuracy, tool calls, token and cache usage, estimated cost, latency, and failures.

Expected answers remain inside the harness and are never sent to the evaluated agent subprocess.

## Build

```bash
go test ./...
go vet ./...
go build -o bin/agent-bench ./cmd/agent-bench
go build -o bin/claude-adapter ./cmd/claude-adapter
```

## Baseline

```bash
export AGENT_BENCH_CLAUDE_MODEL=claude-sonnet-5
export AGENT_BENCH_CLAUDE_EFFORT=medium

./bin/agent-bench run \
  -tasks benchmarks \
  -strategy baseline \
  -command './bin/claude-adapter' \
  -repeat 3 \
  -timeout 5m \
  -out results/baseline.jsonl
```

## Graph

```bash
export AGENT_BENCH_TS_GRAPH_HOST=../ts-graph-tools

./bin/agent-bench run \
  -tasks benchmarks \
  -strategy graph \
  -command './bin/claude-adapter' \
  -repeat 3 \
  -timeout 5m \
  -out results/graph.jsonl
```

No token-saving or accuracy-improvement claim is made until real repeated runs are recorded.

See [docs/002-claude-adapter](docs/002-claude-adapter/spec.md) for the isolation and measurement contract.
