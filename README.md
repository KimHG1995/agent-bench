# agent-bench

[English](README.en.md)

AI Coding Agent가 코드베이스를 이해하는 방식을 같은 조건에서 비교하는 **Go 기반 재현 가능한 벤치마크 도구**입니다.

현재는 같은 Claude Code 모델을 두 조건으로 비교합니다.

- `baseline`: Read, Grep, Glob 기반 코드 탐색
- `graph`: 동일한 도구에 TypeScript Code Graph MCP만 추가

```text
같은 model / effort / question / repository
              |
       +------+------+
       |             |
   baseline        graph
 Read/Grep/Glob   Read/Grep/Glob
                       +
                  Code Graph MCP
       |             |
       +------+------+
              |
 accuracy / tool calls / tokens / latency
```

## 측정 항목

- deterministic evidence accuracy
- 전체 tool call 수
- Code Graph tool call 수
- input/output token
- cache creation/read token
- estimated cost
- wall-clock latency
- 전략별 median / p95 분포
- 동일 task와 repeat를 묶은 baseline↔graph paired delta
- 실행 실패와 evidence gap

실측 데이터를 확보하기 전에는 특정 절감률이나 정확도 향상을 결과로 주장하지 않습니다.

## SDD

### MVP Core

- [요구사항](docs/001-mvp/spec.md)
- [설계](docs/001-mvp/design.md)
- [작업](docs/001-mvp/tasks.md)

### Claude Code Adapter

- [요구사항](docs/002-claude-adapter/spec.md)
- [설계](docs/002-claude-adapter/design.md)
- [작업](docs/002-claude-adapter/tasks.md)

## 정답 누출 방지

Task 파일의 `expected`는 harness 내부 grader만 사용합니다.

```text
Task
 |\
 | +--> expected -----------------> grader only
 |
 +----> id/category/repo/question -> Agent
```

외부 Agent subprocess에는 정답이 전달되지 않으며 unit test로 고정합니다.

## 빌드

```bash
go test ./...
go vet ./...

go build -o bin/agent-bench ./cmd/agent-bench
go build -o bin/claude-adapter ./cmd/claude-adapter
```

외부 Go package는 사용하지 않습니다.

## Claude Code 비교 실행

두 조건의 모델과 effort를 동일하게 고정합니다.

```bash
export AGENT_BENCH_CLAUDE_MODEL=claude-sonnet-5
export AGENT_BENCH_CLAUDE_EFFORT=medium
```

### Baseline

```bash
./bin/agent-bench run \
  -tasks benchmarks \
  -strategy baseline \
  -command './bin/claude-adapter' \
  -repeat 3 \
  -timeout 5m \
  -out results/baseline.jsonl
```

baseline은 Claude Code를 읽기 전용으로 격리합니다.

```text
--bare
--tools Read,Grep,Glob
--strict-mcp-config
```

프로젝트 CLAUDE.md, plugin, skill, hook, 사용자 MCP가 비교에 섞이지 않도록 합니다.

### Graph

먼저 `ts-graph-tools`를 설치합니다.

```bash
git clone https://github.com/KimHG1995/ts-graph-tools.git ../ts-graph-tools
cd ../ts-graph-tools
npm install
cd -

export AGENT_BENCH_TS_GRAPH_HOST=../ts-graph-tools
```

실행:

```bash
./bin/agent-bench run \
  -tasks benchmarks \
  -strategy graph \
  -command './bin/claude-adapter' \
  -repeat 3 \
  -timeout 5m \
  -out results/graph.jsonl
```

graph는 baseline과 동일한 built-in tool에 다음 MCP 하나만 추가합니다.

```text
mcp__ts_graph__inspect_typescript_graph
```

직접 관리하는 MCP config가 있다면:

```bash
export AGENT_BENCH_GRAPH_MCP_CONFIG=/path/to/mcp.json
```

## Report

```bash
./bin/agent-bench report \
  -inputs results/baseline.jsonl,results/graph.jsonl \
  -out results/report.md
```

Raw JSONL은 보존하고 Markdown report는 다시 생성할 수 있습니다.

리포트는 단순 평균만 보여주지 않습니다.

- 전략별 accuracy mean / median
- tool calls median / p95
- I/O tokens median / p95
- latency median / p95
- 동일 `taskId + run`의 baseline과 graph를 직접 묶은 paired delta
- task별 accuracy 차이
- 실패 및 missing/unexpected evidence

paired 비교는 양쪽 실행이 모두 성공한 경우만 포함하며, 관측되지 않은 metric은 0으로 대체하지 않습니다.

## deterministic grading

Agent evidence를 canonical fact로 바꿉니다.

```text
symbol:UserService.deleteUser
path:src/user/user.service.ts
relationship:UserController.deleteUser->UserService.deleteUser
```

expected와 actual의 precision, recall, F1을 계산하고 F1을 accuracy로 기록합니다.

현재 category:

- lookup
- caller
- flow
- impact
- architecture

## 다음 단계

1. 실제 Claude Code baseline / graph 반복 실행
2. 공개 TypeScript OSS fixture 추가
3. 실제 결과를 바탕으로 task별 failure case 기록
4. 공개 TypeScript OSS fixture 추가
5. Codex adapter 추가
