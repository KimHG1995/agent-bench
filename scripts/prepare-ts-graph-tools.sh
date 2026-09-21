#!/usr/bin/env bash
set -euo pipefail

TARGET="${AGENT_BENCH_TS_GRAPH_HOST:-targets/ts-graph-tools}"
REPO="https://github.com/KimHG1995/ts-graph-tools.git"
COMMIT="6cc701bde596a955cb95824f67e896e13c10fff2"

if [[ -d "$TARGET/.git" ]]; then
  git -C "$TARGET" fetch --quiet origin "$COMMIT"
else
  mkdir -p "$(dirname "$TARGET")"
  git clone --quiet --no-checkout "$REPO" "$TARGET"
fi

git -C "$TARGET" checkout --quiet --detach "$COMMIT"
npm --prefix "$TARGET" ci --silent

actual="$(git -C "$TARGET" rev-parse HEAD)"
if [[ "$actual" != "$COMMIT" ]]; then
  echo "unexpected commit: $actual" >&2
  exit 1
fi

echo "prepared $TARGET @ $actual"
