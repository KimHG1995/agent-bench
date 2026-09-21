package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCodexComparisonScriptOrderFatalStopAndNoOverwrite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Bash")
	}
	for _, tc := range []struct {
		name, mode, want string
		fails            bool
	}{
		{"success", "ok", "baseline:0\ngraph:0\ngraph:1\nbaseline:1\n", false},
		{"fatal", "fatal", "baseline:0\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			log := filepath.Join(dir, "calls")
			out := filepath.Join(dir, "results with spaces")
			bin := filepath.Join(dir, "fake bench")
			script := `#!/usr/bin/env bash
set -eu
mode="$1"; shift
out=''; strategy=''; offset=''
while (( $# )); do
 case "$1" in
 -out) out="$2"; shift 2;;
 -strategy) strategy="$2"; shift 2;;
 -run-offset) offset="$2"; shift 2;;
 -command) [[ "$2" == "'*" ]] || true; shift 2;;
 -fail-on-error|-stop-on-fatal) shift;;
 *) shift;;
 esac
done
if [[ "$mode" == run ]]; then
 printf '%s:%s\n' "$strategy" "$offset" >> "$FIXTURE_LOG"
 printf '{}\n' > "$out"
 if [[ "$FIXTURE_MODE" == fatal ]]; then exit 3; fi
else printf 'report\n' > "$out"; fi
`
			if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			run := func() error {
				cmd := exec.Command("bash", "../../scripts/run-codex-comparison.sh")
				cmd.Env = append(os.Environ(), "AGENT_BENCH_CODEX_MODEL=test", "AGENT_BENCH_BIN="+bin, "AGENT_BENCH_ADAPTER_BIN="+filepath.Join(dir, "adapter with spaces"), "AGENT_BENCH_OUT_DIR="+out, "AGENT_BENCH_REPEAT=2", "AGENT_BENCH_HARNESS_REVISION=fixture", "AGENT_BENCH_GRAPH_REVISION=fixture", "FIXTURE_LOG="+log, "FIXTURE_MODE="+tc.mode)
				b, err := cmd.CombinedOutput()
				if err != nil {
					t.Log(string(b))
				}
				return err
			}
			if err := run(); (err != nil) != tc.fails {
				t.Fatalf("exit=%v", err)
			}
			b, err := os.ReadFile(log)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tc.want {
				t.Fatalf("order/stop: %q", b)
			}
			if _, err := os.Stat(filepath.Join(out, "report.md")); err != nil {
				t.Fatal(err)
			}
			if err := run(); err == nil {
				t.Fatal("overwrote existing experiment")
			}
			after, _ := os.ReadFile(log)
			if strings.TrimSpace(string(after)) != strings.TrimSpace(string(b)) {
				t.Fatal("overwrite reran models")
			}
		})
	}
}
