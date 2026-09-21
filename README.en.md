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

The Codex CLI adapter reuses an existing ChatGPT login. Its baseline uses Codex's read-only shell, and graph adds the pinned MCP inspection tool. Tool-call counts across adapters have different semantics and should not be compared directly.

## Fixed real-world target

- target: [loglens](https://github.com/KimHG1995/loglens)
- commit: `985d81ee1fb97570ae1f6da39775c7b0dec38db2`
- graph host: [ts-graph-tools](https://github.com/KimHG1995/ts-graph-tools)
- graph commit: `6cc701bde596a955cb95824f67e896e13c10fff2`

## Measurement

- deterministic expected-evidence F1
- required-evidence recall and precision; unexpected facts are not necessarily incorrect
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
go build -o bin/codex-adapter ./cmd/codex-adapter
```

For OrcaRouter:

```bash
export AGENT_BENCH_OPENAI_BASE_URL=https://api.orcarouter.ai/v1
export ORCAROUTER_API_KEY=...
export AGENT_BENCH_OPENAI_MODEL=<model>
export AGENT_BENCH_TS_GRAPH_HOST=targets/ts-graph-tools

AGENT_BENCH_REPEAT=3 bash scripts/run-openai-comparison.sh
```

## Codex with ChatGPT login

```bash
codex login status
export AGENT_BENCH_CODEX_MODEL=gpt-6-astra
export AGENT_BENCH_CODEX_EFFORT=xhigh
export AGENT_BENCH_TS_GRAPH_HOST=targets/ts-graph-tools
export AGENT_BENCH_OUT_DIR=results/codex-example

AGENT_BENCH_REPEAT=3 bash scripts/run-codex-comparison.sh
```

Select a model available to your account explicitly. This uses your Codex allowance, not an API key. No provider/model fallback occurs. The default experiment is 3 tasks × 2 strategies × 3 repeats. A new output directory is required; fatal authentication/quota failures stop remaining calls after persisting results.

Each run retains JSONL events, stderr, final JSON, a manifest and a fresh pinned target archive. The adapter checks schema, source integrity and Graph execution. Reports compare effort, tier, CLI/config fingerprints, time budgets and context/target hashes; a successful CLI exit cannot hide an invalid Graph measurement. Valid Graph nonuse is retained.

Actual serving model and dollar cost remain unknown when unobserved. Host context is fingerprinted, not fully isolated, and target dependencies are not installed in the archive. See the [specification](docs/005-codex-adapter/spec.md), [runbook](docs/005-codex-adapter/runbook.md), [measured results](docs/005-codex-adapter/live-benchmark.md), and [roadmap](docs/005-codex-adapter/roadmap.md).

The 2026-09-21 integration experiment completed 18/18 runs with 9/9 comparable pairs. Median paired Graph changes were −25.0% tool calls, +23.9% total input/output tokens, and +24.7% latency. This sample did not demonstrate overall token or time savings; task-level outcomes and exact-evidence formatting limitations are recorded alongside the measurements.

Small-sample measurements describe only the recorded experiment. Do not generalize them into universal savings or accuracy claims.
