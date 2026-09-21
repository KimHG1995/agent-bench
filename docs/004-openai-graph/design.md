# OpenAI-Compatible Graph Comparison Design

```text
                    +----------------------+
                    | OpenAI / OrcaRouter  |
                    +----------+-----------+
                               |
                         tool calling
                               |
                    +----------v-----------+
                    | openai-adapter (Go)  |
                    +----------+-----------+
                               |
              +----------------+----------------+
              |                                 |
      baseline local tools                 graph strategy
 list/search/read filesystem        list/search/read filesystem
                                                +
                                       MCP stdio client
                                                |
                                       ts-graph-tools
                                                |
                                  inspect_typescript_graph
```

## MCP schema handoff

`tools/list`에서 받은:

- name
- description
- inputSchema

를 별도 복사본 없이 OpenAI function schema로 변환한다.

## Tool result handoff

`tools/call` 결과에 `structuredContent`가 있으면 그것을 모델에 전달한다.
없으면 MCP content 배열을 전달한다.

## 프로세스 수명

하나의 benchmark run에서 graph MCP 프로세스 하나를 시작한다.
해당 Agent run의 모든 graph call이 같은 resident graph session을 공유한다.
run 종료 시 stdin을 닫고 subprocess를 정리한다.

## 버전 고정

loglens와 ts-graph-tools 모두 commit SHA를 고정한다.
그래프 도구 버전 변화와 모델 변화가 한 실험에 섞이지 않도록 한다.
