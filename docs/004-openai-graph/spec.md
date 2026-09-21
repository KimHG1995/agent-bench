# OpenAI-Compatible Graph Comparison Specification

## 목적

동일한 OpenAI-compatible 모델에서 코드 탐색 전략만 바꿔 비교한다.

- baseline: list_files, search_text, read_file
- graph: 동일한 세 도구 + inspect_typescript_graph

## Graph 연결

agent-bench가 Go stdio MCP client로 ts-graph-tools 프로세스를 직접 소유한다.

대상 ts-graph-tools commit:

`6cc701bde596a955cb95824f67e896e13c10fff2`

해당 저장소는 `@ttsc/graph 0.19.3`을 사용하므로 2025-era MCP lifecycle로 연결한다.

1. initialize (2025-11-25)
2. notifications/initialized
3. tools/list
4. tools/call

MCP가 반환한 tool description과 inputSchema를 OpenAI function tool에 그대로 전달한다.

## 비교 공정성

동일하게 유지:

- provider
- model
- target repository commit
- question
- local read/search tools
- max turns
- 반복 번호

graph 전략에만 compiler graph tool을 한 개 추가한다.

반복 실행 순서는 baseline/graph와 graph/baseline을 교차한다.

## 지표

기존 지표에 `graphToolCalls`를 포함한다.

paired 비교는 같은 taskId와 run 번호의 성공한 양쪽 결과만 계산한다.

## 테스트 대상

agent-bench 본체가 아닌 loglens commit
`985d81ee1fb97570ae1f6da39775c7b0dec38db2`.

## 완료 조건

- Go MCP stdio client 구현
- 실제 ts-graph-tools tools/list 성공
- 실제 loglens graph lookup 성공
- OpenAI adapter graph strategy 지원
- live workflow baseline/graph paired 실행
- CI test/vet/build/graph smoke 통과
