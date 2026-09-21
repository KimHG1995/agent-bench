# Agent Bench MVP Design

## Protocol note

This document describes the original MVP architecture. The current external Agent request deliberately excludes expected evidence. Expected evidence is retained only inside the benchmark harness and grader.

## Core flow

```text
benchmark task
   |
   +--> expected evidence --------> deterministic grader
   |
   +--> id/category/repository/question
                  |
                  v
             AgentTask
```

### Agent request

```json
{
  "strategy": "baseline",
  "run": 1,
  "task": {
    "id": "caller-001",
    "category": "caller",
    "repository": {
      "path": "fixtures/nestjs-sample",
      "revision": "fixture-v1"
    },
    "question": "UserService.deleteUser를 호출하는 API를 찾아줘."
  }
}
```

### Result persistence

Each completed run is written to JSONL immediately and flushed before the next run. Existing result files are rejected by default. A deliberate overwrite requires the CLI overwrite option.

### Comparison identity

Paired reports require matching task hash, repository revision, harness/graph revision, provider endpoint, requested/actual model, adapter and max-turn settings. Duplicate pair rows are rejected.

### Accuracy

Accuracy means deterministic F1 over expected evidence facts. It does not measure natural-language answer quality or code-editing capability.

For later protocol details, see:

- ../002-claude-adapter/
- ../003-openai-compatible/
- ../004-openai-graph/
