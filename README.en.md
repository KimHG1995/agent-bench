# agent-bench

[한국어](README.md)

**An eval harness for checking whether Code Graph actually helps coding agents.**

I built this to evaluate `ts-graph-tools` with Codex using the same repository, task, model settings, and time budget.

Instead of assuming that an MCP tool makes an agent faster or more accurate, agent-bench measures the trade-off.

## Why it exists

`ts-graph-tools` exposes compiler-resolved TypeScript symbols, callers, flows, and impact relationships to coding agents.

The question is whether that context actually helps.

```text
Codex with file exploration
          vs
the same Codex + Code Graph MCP
```

agent-bench runs both conditions repeatedly and records evidence quality, tool calls, tokens, latency, and failures.

Its job is not to prove that Graph wins. Its job is to catch when a change helps, hurts, or simply moves cost somewhere else.

## What I use it for

- measure ts-graph-tools changes before and after
- check whether smaller Graph responses actually reduce tokens
- test prompt or MCP-description changes
- compare model/effort settings under the same task
- detect regressions in accuracy, tool usage, or latency

## Current experiment

The current real-world target is [loglens](https://github.com/KimHG1995/loglens), pinned to a fixed commit.

```text
baseline
  Codex
  + read-only shell

graph
  same Codex
  + same read-only shell
  + inspect_typescript_graph
```

Pinned inputs:

- loglens: `985d81ee1fb97570ae1f6da39775c7b0dec38db2`
- ts-graph-tools: `6cc701bde596a955cb95824f67e896e13c10fff2`

The current dataset contains three code-understanding tasks covering request flow, dependency injection, and spike-detection logic.

## Measured result

Codex experiment from 2026-09-21:

- 3 tasks
- baseline vs graph
- 3 repeats each
- 18 executions
- 9 paired comparisons
- 18/18 completed successfully

| Metric | Graph change |
| --- | ---: |
| Evidence F1 | +0.039 |
| Tool calls | -25.0% |
| Input + output tokens | +23.9% |
| Latency | +24.7% |

Graph reduced tool calls, but it did **not** reduce tokens or execution time in this sample.

That result is the reason this repository exists: without an eval harness, "fewer tool calls" could easily be mistaken for "more efficient."

See [measured results](docs/005-codex-adapter/live-benchmark.md) for the full conditions and limitations.

## What it measures

- required evidence recall
- evidence precision / F1
- tool calls
- Graph calls
- observed input/output tokens
- latency
- completion/failure
- paired baseline-vs-graph deltas

Evidence F1 is not a natural-language answer-quality score. It compares extracted symbols, paths, and relationships against a fixed expected set.

## Run with Codex

```bash
codex login status

go build -o bin/agent-bench ./cmd/agent-bench
go build -o bin/codex-adapter ./cmd/codex-adapter

bash scripts/prepare-loglens.sh
bash scripts/prepare-ts-graph-tools.sh

export AGENT_BENCH_CODEX_MODEL=<model>
export AGENT_BENCH_CODEX_EFFORT=<effort>
export AGENT_BENCH_TS_GRAPH_HOST=targets/ts-graph-tools
export AGENT_BENCH_OUT_DIR=results/codex-run

AGENT_BENCH_REPEAT=3 bash scripts/run-codex-comparison.sh
```

[Codex runbook](docs/005-codex-adapter/runbook.md)

## OpenAI-compatible APIs

OpenAI-compatible Chat Completions endpoints are also supported.

```bash
export AGENT_BENCH_OPENAI_BASE_URL=<base-url>
export AGENT_BENCH_OPENAI_API_KEY=<key>
export AGENT_BENCH_OPENAI_MODEL=<model>
export AGENT_BENCH_TS_GRAPH_HOST=targets/ts-graph-tools

AGENT_BENCH_REPEAT=3 bash scripts/run-openai-comparison.sh
```

Tool-call counts should not be compared directly across different adapters because their tool semantics differ.

## What this is not

- not a general LLM leaderboard
- not a model-ranking project
- not a project designed to prove that Graph always wins
- not evidence that three tasks generalize to all TypeScript repositories

The current goal is simple: **when I change a coding-agent context tool, I want a reproducible way to tell whether it actually got better.**

## Docs

- [Codex adapter spec](docs/005-codex-adapter/spec.md)
- [Runbook](docs/005-codex-adapter/runbook.md)
- [Measured results](docs/005-codex-adapter/live-benchmark.md)
- [Roadmap](docs/005-codex-adapter/roadmap.md)
