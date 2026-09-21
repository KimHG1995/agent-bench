# Claude Code Adapter Specification

## 목적

동일한 Claude Code 모델을 사용해 코드베이스 이해 방식만 비교한다.

- baseline: Claude Code 내장 읽기 도구만 사용
- graph: 동일한 내장 읽기 도구에 TypeScript Code Graph MCP만 추가

## 공정 비교 조건

두 전략은 다음 조건을 동일하게 사용한다.

- Claude Code executable
- model
- effort
- max turns
- target repository와 revision
- prompt와 structured output schema
- built-in tools

차이는 graph 전략의 Code Graph MCP 접근 권한뿐이다.

## 격리

Claude Code 실행 시 다음을 사용한다.

- `--bare`: CLAUDE.md, skills, plugins, hooks, auto-discovered MCP 등 커스터마이징 제거
- `--tools Read,Grep,Glob`: 읽기/검색 도구만 노출
- `--strict-mcp-config`: 명시한 MCP 외의 MCP 제거
- baseline은 MCP config를 전달하지 않음
- graph는 `ts_graph` MCP만 전달
- `--no-session-persistence`: 이전 대화 상태를 재사용하지 않음
- `--json-schema`: 최종 evidence를 구조화 JSON으로 수집

## 정답 격리

benchmark task의 `expected`는 harness 내부 grader만 사용한다.

외부 adapter에 전달되는 RunRequest에는 다음만 포함한다.

- id
- category
- repository
- question

`expected`가 subprocess stdin에 포함되면 벤치마크 무효로 본다.

## 수집 metric

- tool call 수
- graph tool call 수
- input/output token
- cache creation/read token
- estimated cost
- 실제 model
- wall-clock latency

## Graph MCP

기본 서버 이름은 `ts_graph`다.

기본 tool 이름:

`mcp__ts_graph__inspect_typescript_graph`

설정 방식:

1. `AGENT_BENCH_TS_GRAPH_HOST`로 ts-graph-tools 설치 경로 전달
2. `AGENT_BENCH_GRAPH_MCP_CONFIG`로 custom MCP JSON 직접 전달

## 환경 변수

- `AGENT_BENCH_CLAUDE_BIN`: 기본 `claude`
- `AGENT_BENCH_CLAUDE_MODEL`: 기본 `claude-sonnet-5`
- `AGENT_BENCH_CLAUDE_EFFORT`: 기본 `medium`
- `AGENT_BENCH_CLAUDE_MAX_TURNS`: 기본 `8`
- `AGENT_BENCH_TS_GRAPH_HOST`
- `AGENT_BENCH_GRAPH_MCP_CONFIG`
- `AGENT_BENCH_GRAPH_SERVER`: 기본 `ts_graph`
- `AGENT_BENCH_NODE_BIN`: 기본 `node`
