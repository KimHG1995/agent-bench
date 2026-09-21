# agent-bench

[English](README.en.md)

AI Coding Agent가 코드베이스를 이해하는 방식을 같은 조건에서 비교하기 위한 재현 가능한 벤치마크 도구입니다.

Agent Bench 자체는 **Go**로 작성하며, 특정 LLM이나 Agent 제품에 종속되지 않습니다. 외부 Agent를 표준 JSON stdin/stdout protocol로 실행하고 정확도, 도구 호출 수, 토큰, 실행 시간을 수집합니다.

현재 MVP의 첫 비교 대상은 다음 두 전략입니다.

- `baseline`: 일반 파일 탐색, 검색, 파일 읽기 기반 Agent
- `graph`: TypeScript Code Graph MCP를 사용할 수 있는 Agent

> 현재 저장소에 포함된 `mock-agent`는 harness 검증용입니다. 실제 baseline/graph 성능 결과가 아니며 README에서 절감률을 주장하지 않습니다.

## 핵심 원칙

```text
같은 모델
같은 저장소와 revision
같은 질문
같은 일반 도구와 권한
        |
        +--> baseline: file/search context
        |
        +--> graph: Code Graph MCP context
                    |
                    v
        accuracy / tool calls / tokens / latency
```

최종 답변만 평가하지 않습니다. 가능한 경우 symbol, file path, relationship을 사전에 정의하고 deterministic F1 score로 채점합니다.

## SDD

MVP는 명세를 먼저 정의하고 구현했습니다.

- [요구사항 명세](docs/001-mvp/spec.md)
- [설계](docs/001-mvp/design.md)
- [구현 작업](docs/001-mvp/tasks.md)

## 구조

```text
cmd/agent-bench/       CLI
internal/domain/       task, protocol, result model
internal/taskloader/   benchmark JSON loader
internal/agent/        external process adapter
internal/grader/       deterministic evidence grader
internal/runner/       task x repeat benchmark runner
internal/resultio/     JSONL raw result I/O
internal/report/       Markdown aggregate report
benchmarks/            benchmark task definitions
fixtures/              synthetic target repositories
examples/mock-agent/   harness verification adapter
```

## 요구사항

- Go 1.25 이상
- 실제 Agent benchmark를 실행할 경우 해당 Agent와 모델 환경

외부 Go 패키지는 사용하지 않습니다.

## 테스트

```bash
go test ./...
```

## 동작 확인

mock agent는 task의 expected evidence를 그대로 반환합니다. 벤치마크 엔진과 리포트 파이프라인이 정상적으로 동작하는지 확인하기 위한 용도입니다.

```bash
go run ./cmd/agent-bench run \
  -tasks benchmarks \
  -strategy baseline \
  -command 'go run ./examples/mock-agent' \
  -repeat 2 \
  -out results/baseline.jsonl

go run ./cmd/agent-bench run \
  -tasks benchmarks \
  -strategy graph \
  -command 'go run ./examples/mock-agent' \
  -repeat 2 \
  -out results/graph.jsonl

go run ./cmd/agent-bench report \
  -inputs results/baseline.jsonl,results/graph.jsonl \
  -out results/report.md
```

## 외부 Agent Protocol

Agent Bench는 각 run마다 Agent command의 stdin으로 JSON을 전달합니다.

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
    "question": "UserService.deleteUser를 호출하는 API 진입점을 찾아라."
  }
}
```

Agent는 stdout에 단 하나의 JSON response를 출력해야 합니다.

```json
{
  "answer": "UserController.deleteUser가 호출합니다.",
  "evidence": {
    "symbols": ["UserService.deleteUser", "UserController.deleteUser"],
    "paths": ["src/user/user.service.ts", "src/user/user.controller.ts"],
    "relationships": [
      {
        "from": "UserController.deleteUser",
        "to": "UserService.deleteUser"
      }
    ]
  },
  "metrics": {
    "toolCalls": 4,
    "inputTokens": 3200,
    "outputTokens": 420
  }
}
```

관측할 수 없는 metric은 생략할 수 있습니다. Agent Bench는 누락된 값을 `0`으로 바꾸지 않고 미측정으로 유지합니다.

## 채점

expected evidence와 Agent evidence를 다음 fact로 변환합니다.

```text
symbol:UserService.deleteUser
path:src/user/user.service.ts
relationship:UserController.deleteUser->UserService.deleteUser
```

그 뒤 precision, recall, F1을 계산하며 F1을 `accuracy`로 기록합니다.

LLM Judge는 MVP 범위에 포함하지 않습니다. 정확한 symbol이나 호출 관계처럼 자동 검증 가능한 항목부터 평가합니다.

## 현재 benchmark fixture

synthetic TypeScript fixture에 다음 관계가 들어 있습니다.

```text
UserController.deleteUser
        |
        v
UserService.deleteUser
    |               |
    v               v
UserRepository   AuditService
.deleteById      .record

userServiceDeleteTest
        |
        +------> UserService.deleteUser
```

MVP task category:

- lookup
- caller
- flow
- impact
- architecture

## 결과 형식

각 실행은 JSONL raw result로 남습니다. 이후 여러 결과 파일을 다시 읽어 Markdown report를 생성합니다.

```text
raw results
  baseline.jsonl
  graph.jsonl
       |
       v
agent-bench report
       |
       v
report.md
```

리포트에는 다음을 집계합니다.

- run count
- success rate
- mean accuracy
- mean tool calls
- mean input tokens
- mean output tokens
- mean latency
- 실패 및 evidence gap

## 다음 단계

실제 성능 비교는 Go benchmark core와 분리된 adapter로 연결합니다.

1. baseline Coding Agent adapter
2. `ts-graph-tools` MCP를 활성화한 graph Agent adapter
3. 동일 모델, 동일 설정으로 반복 실행
4. 실제 tool call과 token usage 기록
5. synthetic fixture 이후 공개 TypeScript OSS로 확장

실측 데이터를 확보하기 전까지 특정 토큰 절감률이나 정확도 향상을 프로젝트의 결과로 주장하지 않습니다.
