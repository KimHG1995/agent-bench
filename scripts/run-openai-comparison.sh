#!/usr/bin/env bash
set -u

REPEAT="${AGENT_BENCH_REPEAT:-3}"
TIMEOUT="${AGENT_BENCH_TIMEOUT:-5m}"
TASKS="${AGENT_BENCH_TASKS:-benchmarks/loglens}"
OUT_DIR="${AGENT_BENCH_OUT_DIR:-results/live}"
AGENT_BENCH_BIN="${AGENT_BENCH_BIN:-./bin/agent-bench}"
ADAPTER_BIN="${AGENT_BENCH_ADAPTER_BIN:-./bin/openai-adapter}"

mkdir -p "$OUT_DIR"
export AGENT_BENCH_EXPERIMENT_ID="${AGENT_BENCH_EXPERIMENT_ID:-$(date -u +%Y%m%dT%H%M%SZ)-$$}"
export AGENT_BENCH_HARNESS_REVISION="${AGENT_BENCH_HARNESS_REVISION:-$(git rev-parse HEAD 2>/dev/null || true)}"
export AGENT_BENCH_GRAPH_REVISION="${AGENT_BENCH_GRAPH_REVISION:-$(git -C "${AGENT_BENCH_TS_GRAPH_HOST:-targets/ts-graph-tools}" rev-parse HEAD 2>/dev/null || true)}"

baseline_inputs=()
graph_inputs=()
failed=0

run_strategy() {
  local strategy="$1" run="$2" out="$OUT_DIR/$strategy-$run.jsonl"
  echo "==> $strategy run $run"
  if ! "$AGENT_BENCH_BIN" run       -tasks "$TASKS"       -strategy "$strategy"       -command "$ADAPTER_BIN"       -repeat 1       -run-offset "$((run - 1))"       -timeout "$TIMEOUT"       -fail-on-error       -out "$out"; then
    failed=1
  fi
  if [[ "$strategy" == "baseline" ]]; then baseline_inputs+=("$out"); else graph_inputs+=("$out"); fi
}

for ((run=1; run<=REPEAT; run++)); do
  if (( run % 2 == 1 )); then run_strategy baseline "$run"; run_strategy graph "$run"; else run_strategy graph "$run"; run_strategy baseline "$run"; fi
done

all_inputs=("${baseline_inputs[@]}" "${graph_inputs[@]}")
inputs_csv="$(IFS=,; echo "${all_inputs[*]}")"
if ! "$AGENT_BENCH_BIN" report -inputs "$inputs_csv" -out "$OUT_DIR/report.md"; then failed=1; fi

echo "report: $OUT_DIR/report.md"
exit "$failed"
