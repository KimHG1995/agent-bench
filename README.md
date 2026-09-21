# agent-bench

[English](README.en.md)

AI Coding Agent가 코드베이스를 이해하는 방식을 같은 조건에서 비교하는 **Go 기반 재현 가능한 벤치마크 도구**입니다.

## 비교 전략

같은 모델, 같은 질문, 같은 저장소에서 context 접근 방식만 바꿉니다.

```text
baseline
  list_files
  search_text
  read_file

graph
  list_files
  search_text
  read_file
  inspect_typescript_graph (MCP)
```

현재 OpenAI-compatible adapter는 OpenAI API, OrcaRouter 및 호환 Chat Completions endpoint를 지원합니다.

Codex CLI adapter는 기존 ChatGPT 로그인으로 로컬 실측을 실행합니다. 이 adapter의 baseline은 Codex의 읽기 전용 shell이며, graph는 여기에 같은 Graph MCP 조회 도구를 추가합니다. 서로 다른 adapter의 도구 호출 수는 직접 비교하지 않습니다.

## 실제 대상

agent-bench 본체를 평가하지 않습니다.

- target: [loglens](https://github.com/KimHG1995/loglens)
- commit: `985d81ee1fb97570ae1f6da39775c7b0dec38db2`
- graph host: [ts-graph-tools](https://github.com/KimHG1995/ts-graph-tools)
- graph commit: `6cc701bde596a955cb95824f67e896e13c10fff2`

## 측정

- deterministic evidence F1
- 필수 근거 recall과 precision (F1은 자연어 답변의 의미 정확도와 다릅니다)
- tool calls
- graph tool calls
- input/output tokens
- latency
- success/failure
- median/p95
- 동일 task/run의 paired baseline↔graph delta

Codex 결과는 요청 모델·effort·tier·CLI·실행 설정·시간 예산·target/graph/context hash를 함께 검증합니다. MCP가 실패했지만 모델이 답변을 완성한 실행은 정상 Graph 비교로 집계하지 않습니다. 정상 제공된 Graph를 모델이 선택하지 않은 결과는 유지합니다.

## 빌드와 검증

```bash
go test ./...
go vet ./...

go build -o bin/agent-bench ./cmd/agent-bench
go build -o bin/openai-adapter ./cmd/openai-adapter
go build -o bin/codex-adapter ./cmd/codex-adapter

bash scripts/prepare-loglens.sh
bash scripts/prepare-ts-graph-tools.sh
```

CI에서는 실제 `@ttsc/graph` MCP를 띄우고 loglens에서 symbol lookup smoke까지 수행합니다.

## OrcaRouter

```bash
export AGENT_BENCH_OPENAI_BASE_URL=https://api.orcarouter.ai/v1
export ORCAROUTER_API_KEY=...
export AGENT_BENCH_OPENAI_MODEL=<model>
export AGENT_BENCH_TS_GRAPH_HOST=targets/ts-graph-tools

AGENT_BENCH_REPEAT=3 bash scripts/run-openai-comparison.sh
```

## ChatGPT 로그인으로 Codex 실측

```bash
codex login status
export AGENT_BENCH_CODEX_MODEL=gpt-6-astra
export AGENT_BENCH_CODEX_EFFORT=xhigh
export AGENT_BENCH_TS_GRAPH_HOST=targets/ts-graph-tools
export AGENT_BENCH_OUT_DIR=results/codex-example

AGENT_BENCH_REPEAT=3 bash scripts/run-codex-comparison.sh
```

예시 모델은 실측에 사용한 값이며 모델을 명시적으로 선택해야 합니다. 기존 ChatGPT 로그인을 사용하고 사용자 Codex 사용량을 소비합니다. API key는 전달하지 않으며 다른 provider로 fallback하지 않습니다. 기본 구성은 3개 과제 × 2전략 × 3반복입니다. 새 출력 디렉터리가 필요하며 인증·quota 오류에서는 남은 호출을 중단합니다.

출력에는 report, JSONL, 실행별 raw events와 manifest가 포함됩니다. 실제 serving model과 USD 비용은 관측되지 않으면 unknown으로 남깁니다. host context의 완전 격리와 전체 외부 의존성 graph는 현재 보장하지 않습니다. [운영 안내](docs/005-codex-adapter/runbook.md), [실측 결과](docs/005-codex-adapter/live-benchmark.md), [개발 방향](docs/005-codex-adapter/roadmap.md)을 참고하세요.

2026-09-21 통합 실측은 18/18회 성공, 9/9쌍 비교 가능했습니다. Graph는 pair별 변화율 중앙값에서 도구 호출 수 −25.0%, 총 입출력 토큰 +23.9%, 실행 시간 +24.7%였습니다. 이 소표본에서는 전반적인 토큰·시간 절감을 확인하지 못했으며, 과제별 점수와 채점 표기 문제를 실측 문서에 함께 기록했습니다.

## OpenAI

```bash
export AGENT_BENCH_OPENAI_BASE_URL=https://api.openai.com/v1
export OPENAI_API_KEY=...
export AGENT_BENCH_OPENAI_MODEL=<model>
export AGENT_BENCH_TS_GRAPH_HOST=targets/ts-graph-tools

AGENT_BENCH_REPEAT=3 bash scripts/run-openai-comparison.sh
```

## GitHub Actions 실측

Repository Secret:

- `ORCAROUTER_API_KEY` 또는
- `OPENAI_API_KEY`

이후 **Actions → live benchmark → Run workflow**에서 provider, model, repeat를 선택합니다.

워크플로는 baseline과 graph 실행 순서를 반복마다 교차하고 raw JSONL 및 paired Markdown report를 artifact로 남깁니다.

## SDD

- [MVP](docs/001-mvp/spec.md)
- [Claude adapter](docs/002-claude-adapter/spec.md)
- [OpenAI-compatible adapter](docs/003-openai-compatible/spec.md)
- [OpenAI-compatible graph comparison](docs/004-openai-graph/spec.md)
- [Codex CLI adapter](docs/005-codex-adapter/spec.md)

성능 차이는 기록한 실험의 범위에서만 해석합니다. 소규모 실측의 절감률을 일반적인 정확도·비용 개선으로 확대하지 않습니다.
