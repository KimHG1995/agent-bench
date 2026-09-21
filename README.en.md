# agent-bench

[한국어](README.md)

A reproducible Go benchmark for comparing how the same coding model understands a codebase with different context-access strategies.

## Strategies

```text
baseline
  list_files
  search_text
  read_file

graph
  the same local tools
  + inspect_typescript_graph (MCP)
```

The OpenAI-compatible adapter supports OpenAI, OrcaRouter, and compatible Chat Completions endpoints. A Claude Code adapter also remains available.

## Fixed real-world target

- target: [loglens](https://github.com/KimHG1995/loglens)
- commit: `985d81ee1fb97570ae1f6da39775c7b0dec38db2`
- graph host: [ts-graph-tools](https://github.com/KimHG1995/ts-graph-tools)
- graph commit: `6cc701bde596a955cb95824f67e896e13c10fff2`

## Measurement

- deterministic expected-evidence F1
- tool calls and graph tool calls
- observed input/output tokens
- latency and completion rate
- median/p95 summaries
- paired baseline-vs-graph deltas only for comparable experiment identities

Unobserved or partial token usage is not treated as zero.

## Integrity safeguards

- expected answers are never sent to the Agent
- Git targets with fixed SHA are checked for HEAD and dirty state before execution
- repository file tools reject symlink escapes
- each run is flushed to JSONL immediately
- existing result files are not overwritten by default
- timeout kills the subprocess process group on Unix
- reports reject mismatched model/revision pairs and duplicate rows
- live workflows preserve partial artifacts even when a run fails

## Run

```bash
bash scripts/prepare-loglens.sh
bash scripts/prepare-ts-graph-tools.sh

go build -o bin/agent-bench ./cmd/agent-bench
go build -o bin/openai-adapter ./cmd/openai-adapter
```

For OrcaRouter:

```bash
export AGENT_BENCH_OPENAI_BASE_URL=https://api.orcarouter.ai/v1
export ORCAROUTER_API_KEY=...
export AGENT_BENCH_OPENAI_MODEL=<model>
export AGENT_BENCH_TS_GRAPH_HOST=targets/ts-graph-tools

AGENT_BENCH_REPEAT=3 bash scripts/run-openai-comparison.sh
```

No token-saving or accuracy-improvement claim is made until real repeated model runs are recorded.
