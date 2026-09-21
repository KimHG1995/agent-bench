# Codex CLI Benchmark Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans to implement this plan task-by-task. The user requested code work from the existing reviewed specification; execute inline and perform a fresh whole-branch review at the end.

**Goal:** Run reliable local ChatGPT-authenticated Codex baseline/graph experiments through agent-bench and document their measured limits.

**Architecture:** A Go adapter owns a bounded Codex CLI process, captures JSONL and strict final JSON, and emits the existing AgentOutput protocol. Additive domain metadata distinguishes execution, measurement validity and unknown metrics. Reports validate common experiment conditions and expose failures, recall and graph use.

**Tech Stack:** Go 1.25+, standard library, Bash, installed Codex CLI, pinned stdio @ttsc/graph.

**Spec:** [spec.md](spec.md), [design.md](design.md).

## Global Constraints

- ChatGPT login only; no credential extraction, CODEX_HOME relocation or API fallback.
- A new ephemeral CLI session per run; target reads only; expected evidence stays in the harness.
- Fixed model, effort, service tier, CLI/config fingerprints and time budget.
- Input/cache/output observations stay separate; absent usage/cost/model is unknown, not zero.
- Graph availability comparison retains valid runs with no Graph use; connectivity smoke requires an actual graph fact call.
- Keep evidence F1 v1; label recall and unexpected facts without claiming semantic answer accuracy.
- Start at main `9059358749c326fadd5e94172c99b04aaff71d76` on `codex/codex-cli-benchmark`.

## Review Focus

1. A CLI exits 0 after MCP denial: preserve answer/usage but invalidate that Graph experiment.
2. Partial/truncated streams and absent usage fields: never manufacture a complete zero measurement.
3. Context deadline kills a parent adapter before its nested CLI: no lingering CLI/MCP children and no buffer read race.
4. Dirty/mutated target, symlink escape, user context drift and mismatched effort/config: fail or disclose rather than compare silently.
5. Failed/auth-limited experiments and reused output directories: preserve failure evidence and do not overwrite or keep spending quota after a fatal account failure.

### Task 1: Observation contract and Codex stream parser

**Files:** `internal/domain/types.go`, `internal/codexadapter/stream.go`, `internal/codexadapter/stream_test.go`, `internal/codexadapter/testdata/`.

**Interfaces:** `parseStream(io.Reader, strategy string) (domain.AgentOutput, error)` produces observed usage, unique tool counts, warnings and measurement status. New `domain.RunStatus` appears in AgentOutput and RunResult; runtime fingerprints and successful tool counts are optional additive fields.

- [x] Add narrow captured fixtures and table tests for the five parser outcomes: normal graph structured_content, approval denial, absent/partial usage, warning and truncation. Derive counts from literal event ids.

```go
out, err := parseStream(strings.NewReader(deniedFixture), "graph")
if err == nil || out.Status.FailureKind != "graph_unavailable" { t.Fatal(out, err) }
if out.Metrics.GraphToolCalls == nil || *out.Metrics.GraphToolCalls != 1 { t.Fatal(out) }
if out.Metrics.GraphToolCallsSucceeded == nil || *out.Metrics.GraphToolCallsSucceeded != 0 { t.Fatal(out) }
```

- [x] Run `go test ./internal/codexadapter`; observe failure for missing parser/contract.
- [x] Implement strict final response decoding, pointer usage, item-id deduplication and graph fact success detection. Reject unexpected MCP tools in baseline and mismatched final schema. Preserve accumulated observations on error.

```go
type RunStatus struct {
    Execution string `json:"execution,omitempty"`
    Measurement string `json:"measurement,omitempty"`
    FailureKind string `json:"failureKind,omitempty"`
    Warnings []string `json:"warnings,omitempty"`
}
```

- [x] Run parser tests including actual smoke fixtures; expected all pass. Commit observation/parser change.

### Task 2: Codex process, configuration and target integrity

**Files:** `internal/codexadapter/{adapter,config,snapshot,process_unix,process_windows}.go`, matching tests; `cmd/codex-adapter/main.go`; targeted `internal/agent` cancellation update and tests.

**Interfaces:** `ConfigFromEnv() (Config,error)`; `Run(context.Context, domain.RunRequest, Config) (domain.AgentOutput,error)`; strict command entrypoint encodes one JSON even on failure. Parser from Task 1 supplies observations; process runner adds runtime and artifacts.

- [x] Add executable fixture process tests checking returned output for denied graph, timeout, wrong final JSON and request/argv/environment boundaries. Fixtures emulate only the external CLI; real process handling and parser run in tests.

```go
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()
out, err := Run(ctx, request, cfg)
if err == nil || out.Status.FailureKind != "timeout" { t.Fatal(out, err) }
```

- [x] Run tests and observe failure before implementing Run.
- [x] Implement model/effort/tier/timeout validation, installed CLI version/hash/login preflight, env allowlist, per-run artifact directory, pristine commit snapshot and before/after hash verification.
- [x] Generate argv with ignore-user-config, ephemeral JSONL, strict schema, read-only shell, disabled extra tools and only the selected pinned MCP server. Record common config, user-context fingerprint and the unresolved hermeticity limitation. Actual model remains requested_only when unobserved.
- [x] Bind adapter signals to context, terminate child process group and wait. Give CommandRunner a short TERM grace window so a nested adapter can persist partial results before final KILL; drain stdout only after Wait.
- [x] Run `go test ./internal/codexadapter ./internal/agent`; expected all pass including nested cancellation. Commit process/command change.

### Task 3: Valid comparisons and bounded comparison script

**Files:** `internal/runner/runner.go`, `internal/report/{validate,report,observations}.go`, tests; `scripts/run-codex-comparison.sh`; `cmd/agent-bench` optional failure-stop flag; `.github/workflows/verify.yml`.

**Interfaces:** Runner propagates RunStatus and excludes invalid measurements from Success. Report uses `samePair` with additive fingerprints and explicit requested-only disclosure. Script runs the adapter with inner timeout shorter than harness timeout, alternates order and persists raw artifacts and failures.

- [x] Add failing tests: effort/tier/config mismatch; zero Graph use is retained; invalid measurement cannot enter successful aggregates; missing usage remains n/a; report shows required recall, cache, failed counts and requested-only provenance.

```go
a, b := comparable("baseline"), comparable("graph")
b.Runtime.Effort = "high"
if ValidateComparable([]domain.RunResult{a,b}) == nil { t.Fatal("effort mismatch accepted") }
```

- [x] Add a script integration test with a fake external adapter/binary to exercise alternating order, paths with spaces, nonzero failures, fatal account failure stop and existing-directory rejection. Do not assert source text.
- [x] Run tests and observe intended failures, implement rules and summary sections without changing grader semantics.
- [x] Build all commands, `bash -n` scripts and run full Go suite. Expected all pass. Commit integration/report change.

### Task 4: Live validation, implementation-aligned docs and direction

**Files:** `README.md`, `README.en.md`, `docs/005-codex-adapter/{spec,design,tasks,runbook,live-benchmark,roadmap}.md`, optional sanitized fixture corrections and tests.

**Interfaces:** Only the built CLI and script from Tasks 1–3; raw outputs remain ignored local artifacts.

- [x] Run `go test -race ./...`, `go vet ./...`, all builds and real graph MCP smoke. Expected pass.
- [x] Run the real Codex adapter on the 3 pinned tasks, first one paired repeat then complete three paired repeats if auth/quota is available. Save all 18 runs or exact failure count and reason; no retries or API fallback.
- [x] Verify actual Graph success, schema validity, source integrity, comparable manifests, usage completeness and report denominators. If live behavior differs, reproduce the issue with a small fixture before fixing.
- [x] Rewrite spec/design/runbook/tasks around implemented behavior, evidence F1 limits, user-context disclosure and measured results. Roadmap separates measurement correctness, dataset breadth and strategy optimization.
- [x] Fresh whole-branch reviewer checks diff and Review Focus while local doc/artifact checks run. Address important findings with failing tests, then full suite.
- [x] Commit verified code/docs. Leave changes on the feature branch; publishing or merging is a separate action unless the user requests it.


## Completion evidence and decisions

Completed on 2026-09-21. Code commits: `3989e1b`, `5b098c4`, `af259e2`, and review fixes `37bb84f`. The measured revision is `af259e2`; [live-benchmark.md](live-benchmark.md) and [live-results.json](live-results.json) preserve the 18-run outcome separately from the later failure-path fixes.

Final verification: full `go test -race ./...`, `go vet ./...`, command builds and Bash syntax passed. The real Graph MCP lookup passed. Raw manifests, final answers, usage, item counts, target hashes and all 9 pair identities were audited. Rebuilding the report with the final code produced byte-identical Markdown.

One independent whole-branch review found two Important issues and no Critical issues. Both were reproduced with failing tests before fixes:

- `TestParseStreamFatalAccountErrorSurvivesEarlierFailures`: a prior Graph/provider error must not hide later fatal quota/auth; preserve both messages.
- `TestPrepareTargetRejectsCommittedSubmodule`: inspect pinned Git tree entries before archive extraction silently removes submodule contents.

No Minor findings were deferred by that review. Future CLI compatibility, complete host isolation, external dependency graph completeness and broad statistical superiority remain explicitly outside this release's claims.

| Decision | Reason / cost of the limit |
| --- | --- |
| Execute the already reviewed design inline without another design approval | The user requested implementation; integration remains on the local feature branch. |
| Per-run clean commit archive, excluding ignored files and node_modules | Prevent local source drift; does not cover complete external dependency types. |
| Reject target symlinks, submodules and top-level .codex/.agents | Avoid omitted source and ambiguous external context; these repository shapes need future support or a prepared target. |
| Ignore the Git global PAX commit metadata header | It is archive metadata, not a source file; unfamiliar source entry types still fail closed. |
| Propagate outer timeout and require a shutdown margin greater than 2 seconds | Allows nested process cleanup and partial output; too-small budgets are rejected. |
| Increase test-only fixture deadlines for race instrumentation | Instrumented helper processes wait at exit; production budgets are unchanged. |
| Preserve every live run at its measured revision | Avoid selection/retry bias; later failure-path fixes are verified by regression tests rather than consuming another 18 live calls. |
| Keep grader v1 and disclose exact-match formatting effects | Avoid post-hoc score changes; the next evidence contract must be versioned before remeasurement. |
| Record host context and requested model without claiming full isolation or observed serving identity | The installed CLI does not expose sufficient evidence; comparisons are conditional on these disclosed limitations. |

Local raw results and the execution scratch ledger are retained for inspection. Publishing or merging is not part of this implementation task.
