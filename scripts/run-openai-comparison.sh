#!/usr/bin/env bash
set -euo pipefail

REPEAT="${AGENT_BENCH_REPEAT:-3}"
TIMEOUT="${AGENT_BENCH_TIMEOUT:-5m}"
TASKS="${AGENT_BENCH_TASKS:-benchmarks/loglens}"
OUT_DIR="${AGENT_BENCH_OUT_DIR:-results/live}"
AGENT_BENCH_BIN="${AGENT_BENCH_BIN:-./bin/agent-bench}"
ADAPTER_BIN="${AGENT_BENCH_ADAPTER_BIN:-./bin/openai-adapter}"

mkdir -p "$OUT_DIR"
baseline_inputs=()
graph_inputs=()

run_strategy() {
  local strategy="$1"
  local run="$2"
  local out="$OUT_DIR/$strategy-$run.jsonl"

  echo "==> $strategy run $run"
  "$AGENT_BENCH_BIN" run     -tasks "$TASKS"     -strategy "$strategy"     -command "$ADAPTER_BIN"     -repeat 1     -run-offset "$((run - 1))"     -timeout "$TIMEOUT"     -out "$out"

  if [[ "$strategy" == "baseline" ]]; then
    baseline_inputs+=("$out")
  else
    graph_inputs+=("$out")
  fi
}

for ((run=1; run<=REPEAT; run++)); do
  if (( run % 2 == 1 )); then
    run_strategy baseline "$run"
    run_strategy graph "$run"
  else
    run_strategy graph "$run"
    run_strategy baseline "$run"
  fi
done

all_inputs=("${baseline_inputs[@]}" "${graph_inputs[@]}")
inputs_csv="$(IFS=,; echo "${all_inputs[*]}")"

"$AGENT_BENCH_BIN" report   -inputs "$inputs_csv"   -out "$OUT_DIR/report.md"

echo "report: $OUT_DIR/report.md"
