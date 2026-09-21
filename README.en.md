# agent-bench

[한국어](README.md)

**An eval harness for checking whether an existing Code Graph tool actually helps a coding agent.**

This project does not implement a TypeScript Code Graph engine.

It connects the existing `@ttsc/graph` tool to Codex through [ts-graph-tools](https://github.com/KimHG1995/ts-graph-tools), then compares that setup with file-only exploration under the same repository, task, model settings, and time budget.

## Why it exists

`@ttsc/graph` provides compiler-resolved TypeScript symbols, callers, flows, and impact relationships.

`ts-graph-tools` is the external host/integration setup used to expose that existing tool to coding agents without installing it into the target repository.

The question is whether that integration actually helps.

```text
Codex with file exploration
          vs
the same Codex + @ttsc/graph MCP
```

agent-bench measures evidence quality, tool calls, tokens, latency, and failures.

Its job is not to prove that Graph wins. Its job is to catch when an integration or configuration change helps, hurts, or simply moves cost somewhere else.

## What I use it for

- evaluate changes in how `@ttsc/graph` is exposed to the agent
- test MCP-description or prompt changes
- check whether smaller Graph responses reduce tokens
- compare Codex model/effort settings under the same task
- detect regressions in accuracy, token usage, or latency

## Current experiment

```text
baseline
  Codex
  + read-only shell

graph
  same Codex
  + same read-only shell
  + inspect_typescript_graph from @ttsc/graph
```

Pinned inputs:

- target: [loglens](https://github.com/KimHG1995/loglens) at `985d81ee1fb97570ae1f6da39775c7b0dec38db2`
- graph integration host: [ts-graph-tools](https://github.com/KimHG1995/ts-graph-tools)
- upstream graph: `@ttsc/graph@0.19.3`

## Measured result

Codex experiment from 2026-09-21:

- 3 tasks
- baseline vs graph
- 3 repeats each
- 18 executions
- 9 paired comparisons
- 18/18 completed successfully

| Metric | Change with Graph integration |
| --- | ---: |
| Evidence F1 | +0.039 |
| Tool calls | -25.0% |
| Input + output tokens | +23.9% |
| Latency | +24.7% |

The Graph-enabled setup reduced tool calls, but did **not** reduce tokens or execution time in this sample.

That result is the reason this repository exists: to evaluate integration choices rather than assume that adding an MCP tool makes an agent more efficient.

See [measured results](docs/005-codex-adapter/live-benchmark.md) for the full conditions and limitations.

## What this is not

- not a TypeScript Code Graph implementation
- not a general LLM leaderboard
- not a model-ranking project
- not a project designed to prove that Graph always wins

The current goal is simple: **when I connect an existing developer tool to a coding agent, I want a reproducible way to tell whether the integration actually helped.**
