# Agent Bench MVP Design

## 1. 설계 원칙

Agent Bench는 특정 모델 공급자나 특정 Coding Agent에 종속되지 않는다.

핵심 역할은 다음 네 가지로 제한한다.

1. benchmark task를 읽는다.
2. 외부 Agent process를 실행한다.
3. deterministic evidence로 결과를 채점한다.
4. raw result와 summary를 남긴다.

Agent와 LLM의 세부 동작은 adapter 밖으로 분리한다.

## 2. 전체 구조

```text
benchmarks/*.json
        |
        v
   task loader
        |
        v
 benchmark runner
        |
        +----------------------+
        |                      |
        v                      v
 baseline adapter          graph adapter
        |                      |
        +----------+-----------+
                   |
                   v
              AgentOutput
                   |
                   v
          deterministic grader
                   |
                   v
               RunResult
                   |
          +--------+--------+
          |                 |
          v                 v
     JSONL writer     Markdown report
```

## 3. Go package 구조

```text
cmd/agent-bench
internal/
  domain
  taskloader
  agent
  grader
  runner
  resultio
  report
benchmarks/
fixtures/
docs/
```

### domain

외부 protocol과 내부 result를 공유하는 타입을 정의한다.

### taskloader

benchmark 디렉터리 아래 JSON 파일을 재귀적으로 읽고 Task로 디코딩한다.

### agent

외부 command를 실행한다.

- stdin: RunRequest JSON
- stdout: AgentOutput JSON
- stderr: 실패 시 진단 정보
- context timeout 적용

### grader

ExpectedEvidence와 Agent Evidence를 fact set으로 정규화하고 precision, recall, F1을 계산한다.

### runner

task x repeat 조합을 순차 실행한다.

MVP는 결과 재현과 디버깅을 우선해 병렬 실행하지 않는다.

### resultio

RunResult를 JSONL로 append한다.

### report

여러 JSONL 파일을 읽어 strategy별 aggregate를 만들고 Markdown을 생성한다.

## 4. 외부 Agent Protocol

### Request

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
    "question": "UserService.deleteUser를 호출하는 API를 찾아줘.",
    "expected": {
      "symbols": ["UserService.deleteUser", "UserController.deleteUser"],
      "paths": [
        "src/user/user.service.ts",
        "src/user/user.controller.ts"
      ],
      "relationships": [
        {
          "from": "UserController.deleteUser",
          "to": "UserService.deleteUser"
        }
      ]
    }
  }
}
```

### Response

```json
{
  "answer": "UserController.deleteUser가 UserService.deleteUser를 호출합니다.",
  "evidence": {
    "symbols": ["UserService.deleteUser", "UserController.deleteUser"],
    "paths": [
      "src/user/user.service.ts",
      "src/user/user.controller.ts"
    ],
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

## 5. Metric 소유권

Agent Bench가 직접 측정:

- startedAt
- durationMs
- process success/failure
- exit/timeout error
- accuracy

Agent adapter가 제공:

- toolCalls
- inputTokens
- outputTokens

token과 tool call을 측정할 수 없는 adapter는 해당 값을 생략한다.

## 6. Timeout

run마다 context deadline을 적용한다.

기본값은 120초다.

timeout은 run 실패로 기록하고 전체 benchmark는 다음 run을 계속한다.

## 7. Grading 상세

expected와 actual의 각 evidence를 문자열 fact로 변환한다.

중복은 제거한다.

예:

```text
symbol:UserService.deleteUser
path:src/user/user.service.ts
relationship:UserController.deleteUser->UserService.deleteUser
```

계산:

```text
TP = intersection(expected, actual)
FP = actual - expected
FN = expected - actual

precision = TP / (TP + FP)
recall = TP / (TP + FN)
F1 = 2PR / (P + R)
```

경로는 slash를 정규화하고 앞의 `./`는 제거한다.

MVP에서는 symbol 이름의 대소문자나 spelling을 임의 보정하지 않는다.

## 8. 결과 파일

JSONL은 append-only raw evidence다.

Markdown report는 raw result에서 다시 생성할 수 있어야 한다.

따라서 summary만 저장하고 raw run을 버리는 구조는 사용하지 않는다.

## 9. 실패 처리

다음은 실패 run으로 기록한다.

- process start 실패
- timeout
- non-zero exit
- stdout이 비어 있음
- response JSON parse 실패

실패 run도 duration과 error를 기록한다.

metric이 없는 실패는 미측정 상태로 둔다.

## 10. 확장 지점

MVP 이후 추가 가능:

- Codex adapter
- Claude Code adapter
- OpenAI API adapter
- MCP capability manifest
- LLM Judge
- cost metric
- percentile latency
- concurrent runner
- public OSS fixture registry
- HTML report
- statistical comparison
