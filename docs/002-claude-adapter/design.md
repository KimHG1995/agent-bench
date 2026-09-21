# Claude Code Adapter Design

## 흐름

```text
agent-bench
   |
   | RunRequest (expected 제외)
   v
claude-adapter
   |
   +-- baseline
   |     Read / Grep / Glob
   |
   +-- graph
         Read / Grep / Glob
               +
         ts_graph MCP
               |
               v
      inspect_typescript_graph

Claude stream-json
   |
   +-- assistant events -> tool calls
   +-- result event    -> structured output / usage / cost
   v
AgentOutput
   v
deterministic grader
```

## adapter 경계

`cmd/claude-adapter`는 Agent Bench protocol adapter다.

stdin은 정답이 제거된 RunRequest다. stdout은 단일 AgentOutput JSON이다.

Claude raw stream-json은 adapter 내부에서 파싱한다.

## structured output

```json
{
  "answer": "...",
  "evidence": {
    "symbols": [],
    "paths": [],
    "relationships": [
      {"from": "A", "to": "B"}
    ]
  }
}
```

schema는 `additionalProperties: false`를 사용한다.

## metric 집계

Tool call은 tool use id로 중복 제거한다.

Token은 다음 우선순위로 수집한다.

1. result event model usage
2. result event cumulative usage
3. assistant message usage를 message id 기준으로 중복 제거

## 안전성

Agent에는 Bash, Edit, Write, Web 도구를 제공하지 않는다.

이 benchmark는 코드 수정 능력이 아니라 코드 이해와 context 접근 효율을 비교한다.
