# OpenAI-Compatible Runbook

## 1. Build

```bash
go build -o bin/agent-bench ./cmd/agent-bench
go build -o bin/openai-adapter ./cmd/openai-adapter
bash scripts/prepare-loglens.sh
```

## 2. OrcaRouter

```bash
export AGENT_BENCH_OPENAI_BASE_URL=https://api.orcarouter.ai/v1
export ORCAROUTER_API_KEY=...
export AGENT_BENCH_OPENAI_MODEL=<model slug>

./bin/agent-bench run \
  -tasks benchmarks/loglens \
  -strategy baseline \
  -command './bin/openai-adapter' \
  -repeat 3 \
  -timeout 5m \
  -out results/orcarouter-loglens.jsonl
```

## 3. OpenAI

```bash
export AGENT_BENCH_OPENAI_BASE_URL=https://api.openai.com/v1
export OPENAI_API_KEY=...
export AGENT_BENCH_OPENAI_MODEL=<model>

./bin/agent-bench run \
  -tasks benchmarks/loglens \
  -strategy baseline \
  -command './bin/openai-adapter' \
  -repeat 3 \
  -timeout 5m \
  -out results/openai-loglens.jsonl
```

## 4. Report

```bash
./bin/agent-bench report \
  -inputs results/orcarouter-loglens.jsonl \
  -out results/orcarouter-loglens.md
```

실행 결과에는 실제 model, tool call, prompt/completion token, accuracy와 evidence gap이 기록된다.
