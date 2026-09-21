#!/usr/bin/env bash
set -euo pipefail

: "${AGENT_BENCH_CODEX_MODEL:?Set an explicit Codex model before benchmarking}"
REPEAT="${AGENT_BENCH_REPEAT:-3}"
[[ "$REPEAT" =~ ^[1-9][0-9]*$ ]] || { echo 'AGENT_BENCH_REPEAT must be a positive integer' >&2; exit 2; }
TASKS="${AGENT_BENCH_TASKS:-benchmarks/loglens}"
OUT_DIR="${AGENT_BENCH_OUT_DIR:-results/codex-$(date -u +%Y%m%dT%H%M%SZ)-$$}"
BENCH_BIN="${AGENT_BENCH_BIN:-./bin/agent-bench}"
ADAPTER_BIN="${AGENT_BENCH_ADAPTER_BIN:-./bin/codex-adapter}"
# The outer timeout leaves time for the adapter to persist partial observations.
OUTER_TIMEOUT="${AGENT_BENCH_TIMEOUT:-6m}"
export AGENT_BENCH_CODEX_TIMEOUT="${AGENT_BENCH_CODEX_TIMEOUT:-5m}"
export AGENT_BENCH_CODEX_EFFORT="${AGENT_BENCH_CODEX_EFFORT:-medium}"
export AGENT_BENCH_CODEX_SERVICE_TIER="${AGENT_BENCH_CODEX_SERVICE_TIER:-default}"
export AGENT_BENCH_TS_GRAPH_HOST="${AGENT_BENCH_TS_GRAPH_HOST:-targets/ts-graph-tools}"
export AGENT_BENCH_EXPERIMENT_ID="${AGENT_BENCH_EXPERIMENT_ID:-$(date -u +%Y%m%dT%H%M%SZ)-$$}"
export AGENT_BENCH_HARNESS_REVISION="${AGENT_BENCH_HARNESS_REVISION:-$(git rev-parse HEAD)}"
export AGENT_BENCH_GRAPH_REVISION="${AGENT_BENCH_GRAPH_REVISION:-$(git -C "$AGENT_BENCH_TS_GRAPH_HOST" rev-parse HEAD)}"

mkdir -p "$(dirname "$OUT_DIR")"
mkdir "$OUT_DIR" || { echo "Refusing to reuse experiment directory: $OUT_DIR" >&2; exit 2; }
export AGENT_BENCH_CODEX_ARTIFACTS="$OUT_DIR/raw"
# POSIX shell quote: CommandRunner executes this string using /bin/sh.
ADAPTER_COMMAND="'${ADAPTER_BIN//\'/\'\\\'\'}'"
inputs=()
failed=0
fatal=0

run_strategy() {
  local strategy="$1" run="$2" status=0
  local out="$OUT_DIR/$strategy-$run.jsonl"
  echo "==> $strategy run $run"
  "$BENCH_BIN" run -tasks "$TASKS" -strategy "$strategy" \
    -command "$ADAPTER_COMMAND" -repeat 1 -run-offset "$((run-1))" \
    -timeout "$OUTER_TIMEOUT" -fail-on-error -stop-on-fatal -out "$out" || status=$?
  if [[ -s "$out" ]]; then inputs+=("$out"); fi
  if ((status != 0)); then failed=1; fi
  if ((status == 3)); then fatal=1; fi
}

for ((run=1; run<=REPEAT; run++)); do
  if ((run % 2 == 1)); then order=(baseline graph); else order=(graph baseline); fi
  for strategy in "${order[@]}"; do
    run_strategy "$strategy" "$run"
    if ((fatal)); then break 2; fi
  done
done

if ((${#inputs[@]})); then
  inputs_csv="$(IFS=,; echo "${inputs[*]}")"
  "$BENCH_BIN" report -inputs "$inputs_csv" -out "$OUT_DIR/report.md" || failed=1
else
  echo 'No results were produced' >&2
  failed=1
fi
if ((fatal)); then echo 'Stopped on auth/quota failure; remaining runs were not attempted.' >&2; fi
echo "artifacts: $OUT_DIR"
exit "$failed"
