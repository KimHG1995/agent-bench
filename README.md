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

## 실제 대상

agent-bench 본체를 평가하지 않습니다.

- target: [loglens](https://github.com/KimHG1995/loglens)
- commit: `985d81ee1fb97570ae1f6da39775c7b0dec38db2`
- graph host: [ts-graph-tools](https://github.com/KimHG1995/ts-graph-tools)
- graph commit: `6cc701bde596a955cb95824f67e896e13c10fff2`

## 측정

- deterministic evidence F1
- tool calls
- graph tool calls
- input/output tokens
- latency
- success/failure
- median/p95
- 동일 task/run의 paired baseline↔graph delta

## 빌드와 검증

```bash
go test ./...
go vet ./...

go build -o bin/agent-bench ./cmd/agent-bench
go build -o bin/openai-adapter ./cmd/openai-adapter

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

실측 데이터를 확보하기 전에는 특정 절감률이나 정확도 향상을 결과로 주장하지 않습니다.
