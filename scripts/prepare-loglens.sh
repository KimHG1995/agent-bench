#!/usr/bin/env bash
set -euo pipefail

TARGET="${AGENT_BENCH_LOGLENS_TARGET:-targets/loglens}"
REPO="https://github.com/KimHG1995/loglens.git"
COMMIT="985d81ee1fb97570ae1f6da39775c7b0dec38db2"

if [[ -d "$TARGET/.git" ]]; then
  git -C "$TARGET" fetch --quiet origin "$COMMIT"
else
  mkdir -p "$(dirname "$TARGET")"
  git clone --quiet --no-checkout "$REPO" "$TARGET"
fi

git -C "$TARGET" checkout --quiet --detach "$COMMIT"
actual="$(git -C "$TARGET" rev-parse HEAD)"
if [[ "$actual" != "$COMMIT" ]]; then
  echo "unexpected commit: $actual" >&2
  exit 1
fi

echo "prepared $TARGET @ $actual"
