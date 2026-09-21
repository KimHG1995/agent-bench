# Claude Comparison Runbook

## 목적

baseline과 graph 실행 순서가 항상 한쪽에 유리하지 않도록 반복 번호마다 실행 순서를 번갈아 배치한다.

```text
run 1: baseline -> graph
run 2: graph -> baseline
run 3: baseline -> graph
...
```

같은 모델, effort, task set을 유지한다.

## 준비

```bash
go build -o bin/agent-bench ./cmd/agent-bench
go build -o bin/claude-adapter ./cmd/claude-adapter

export AGENT_BENCH_CLAUDE_MODEL=claude-sonnet-5
export AGENT_BENCH_CLAUDE_EFFORT=medium
export AGENT_BENCH_TS_GRAPH_HOST=../ts-graph-tools
```

Claude Code CLI 로그인도 완료되어 있어야 한다.

## 실행

```bash
AGENT_BENCH_REPEAT=5 bash scripts/run-claude-comparison.sh
```

선택 환경 변수:

- `AGENT_BENCH_REPEAT`: 기본 3
- `AGENT_BENCH_TIMEOUT`: 기본 5m
- `AGENT_BENCH_TASKS`: 기본 benchmarks
- `AGENT_BENCH_OUT_DIR`: 기본 results/claude
- `AGENT_BENCH_BIN`: 기본 ./bin/agent-bench
- `AGENT_BENCH_ADAPTER_BIN`: 기본 ./bin/claude-adapter

## 결과

각 반복의 raw JSONL을 따로 보존한다.

```text
results/claude/
  baseline-1.jsonl
  graph-1.jsonl
  graph-2.jsonl
  baseline-2.jsonl
  ...
  report.md
```

리포트는 모든 raw result를 다시 읽어 생성한다.

## 해석

- paired delta는 같은 taskId와 같은 run 번호끼리 비교한다.
- 실패한 쌍은 paired delta에서 제외한다.
- token/tool call이 미측정이면 0으로 간주하지 않는다.
- 작은 synthetic fixture의 결과를 전체 코드베이스 성능으로 일반화하지 않는다.
