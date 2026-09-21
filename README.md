# agent-bench

[English](README.en.md)

**기존 Code Graph 도구를 Coding Agent에 연결했을 때 실제로 도움이 되는지 확인하기 위한 Eval 도구입니다.**

이 저장소는 TypeScript Code Graph 엔진 자체를 만드는 프로젝트가 아닙니다.  
[ts-graph-tools](https://github.com/KimHG1995/ts-graph-tools)를 통해 기존 `@ttsc/graph`를 Codex에 연결하고, file-only 탐색과 비교해서 **근거 정확도, 도구 호출 수, 토큰, 실행 시간**이 실제로 어떻게 달라지는지 측정합니다.

## 왜 만들었나

`@ttsc/graph`는 TypeScript 코드의 symbol, caller, flow, impact 같은 관계를 compiler-resolved graph로 제공합니다.

`ts-graph-tools`는 이 도구를 target repository에 설치하지 않고 외부 host로 사용하기 위한 연동 환경입니다.

문제는 Graph 도구를 붙였다고 해서 반드시 더 효율적인 것은 아니라는 점입니다.

```text
파일만 탐색하는 Codex
        vs
같은 Codex + @ttsc/graph MCP
```

`agent-bench`는 이 두 조건에 같은 문제를 주고 결과를 비교합니다.

이 도구의 목적은 **Graph가 좋다는 걸 증명하는 것**이 아니라, 연동 방식이나 설정을 바꿨을 때 실제로 좋아졌는지 확인하는 것입니다.

## 현재 어디에 쓰나

주로 다음 변경 전후를 확인할 때 사용합니다.

- `@ttsc/graph` 응답을 Agent에 전달하는 방식을 바꿨을 때
- MCP description이나 prompt를 바꿨을 때
- Graph 응답 크기를 줄였을 때 토큰이 실제로 줄었는지
- Codex model/effort를 바꿨을 때 결과가 어떻게 달라지는지
- 연동 변경 후 정확도, 토큰, 실행 시간 regression이 생겼는지

## 비교 방식

현재 실제 비교는 Codex CLI 기준입니다.

```text
baseline
  Codex
  + read-only shell

graph
  같은 Codex
  + 같은 read-only shell
  + @ttsc/graph의 inspect_typescript_graph MCP tool
```

두 전략은 같은 target commit, 같은 질문, 같은 model/effort, 같은 시간 예산을 사용합니다.

평가 대상은 `agent-bench` 자체가 아니라 별도 프로젝트입니다.

- target: [loglens](https://github.com/KimHG1995/loglens)
- target commit: `985d81ee1fb97570ae1f6da39775c7b0dec38db2`
- graph integration host: [ts-graph-tools](https://github.com/KimHG1995/ts-graph-tools)
- upstream graph: `@ttsc/graph@0.19.3`

## 실제로 측정된 결과

2026-09-21 Codex 실측:

- 3개 과제
- baseline / graph
- 각 3회 반복
- 총 18회 실행
- 9개 paired comparison
- 18/18 정상 실행

| 지표 | Graph 연동 시 변화 |
| --- | ---: |
| Evidence F1 | +0.039 |
| 도구 호출 수 | -25.0% |
| 총 input + output tokens | +23.9% |
| 실행 시간 | +24.7% |

Graph 연동 조건에서는 **도구 호출은 줄었지만 토큰과 시간은 오히려 늘었습니다.**

이 결과 때문에 현재는 "Graph가 더 빠르다"라고 주장하지 않습니다. 어떤 연동 방식과 Agent 전략이 실제 효율을 만드는지 찾는 데 이 저장소를 사용합니다.

자세한 조건과 해석은 [실측 결과](docs/005-codex-adapter/live-benchmark.md)에 기록합니다.

## 측정하는 것

- 필요한 symbol/path/relationship을 찾았는지
- required evidence recall
- evidence precision / F1
- 전체 tool call 수
- Graph tool call 수
- input / output token
- 실행 시간
- 성공 / 실패
- 같은 task/run의 baseline ↔ graph 변화

F1은 자연어 답변의 품질 점수가 아니라 **사전에 정의한 evidence 집합과 얼마나 맞았는지**를 보는 점수입니다.

## Codex로 실행

```bash
codex login status

go build -o bin/agent-bench ./cmd/agent-bench
go build -o bin/codex-adapter ./cmd/codex-adapter

bash scripts/prepare-loglens.sh
bash scripts/prepare-ts-graph-tools.sh

export AGENT_BENCH_CODEX_MODEL=<model>
export AGENT_BENCH_CODEX_EFFORT=<effort>
export AGENT_BENCH_TS_GRAPH_HOST=targets/ts-graph-tools
export AGENT_BENCH_OUT_DIR=results/codex-run

AGENT_BENCH_REPEAT=3 bash scripts/run-codex-comparison.sh
```

[Codex 실행 방법](docs/005-codex-adapter/runbook.md)

## 이 저장소가 아닌 것

- TypeScript Code Graph 엔진 구현체가 아닙니다.
- 범용 LLM leaderboard가 아닙니다.
- 모델 성능 순위를 매기는 프로젝트가 아닙니다.
- Graph가 항상 더 좋다는 것을 보여주기 위한 프로젝트가 아닙니다.

현재 목적은 **기존 Coding Agent 도구를 연동했을 때 실제로 효과가 있는지 재현 가능한 방법으로 확인하는 것**입니다.
