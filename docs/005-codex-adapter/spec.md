# 005 — Codex CLI adapter specification

상태: 구현됨. 계약·실행·비교 테스트 및 실측 결과는 [tasks](tasks.md), [실측](live-benchmark.md)에 기록한다.
작성일: 2026-09-21 KST. 기준 main: `9059358749c326fadd5e94172c99b04aaff71d76`.

## 목적

기존 ChatGPT 로그인으로 `codex exec`를 실행해 baseline/graph를 비교한다. 사용자 Codex 사용량을 사용하며 API key를 읽거나 ChatGPT 인증 정보를 추출하지 않는다. 인증·quota 실패에 대한 API/provider/model fallback은 없다.

현재 구현의 목적은 탐색 전략의 효과를 측정하는 것이다. 실험의 실행 성공, 비교 조건 충족, evidence 점수를 서로 구분한다. 특정 전략의 일반적인 우위를 보장하지 않는다.

## 비교 계약

| 항목 | baseline | graph |
| --- | --- | --- |
| 모델·effort·tier·CLI | 명시된 동일 설정 | 동일 |
| target | 같은 Git SHA의 새 archive | 동일 |
| 일반 도구 | Codex 읽기 전용 shell | 동일 |
| 추가 도구 | 없음 | `inspect_typescript_graph` 하나 |
| 세션 | 새 ephemeral process | 동일 |
| Graph 미사용 | 해당 없음 | 정상 제공됐으면 유효한 결과로 유지 |

기본 비교는 도구 제공 여부의 비교다. Graph를 사용한 실행만 선별하지 않는다. `AGENT_BENCH_CODEX_REQUIRE_GRAPH_USE=true`는 별도의 연결 smoke 정책이며 실제 Graph fact 호출을 최소 1회 요구한다. 이 설정도 두 전략의 공통 설정 hash에 포함한다.

Codex shell 한 호출은 여러 명령을 포함할 수 있으므로 다른 adapter의 파일 도구 호출 수와 직접 비교하지 않는다.

## 입력과 출력

`cmd/codex-adapter`는 stdin에서 기존 `RunRequest {strategy, run, task}`와 선택적인 `timeoutMs`를 받는다. task에는 `expected`가 없다. `timeoutMs`는 harness의 전체 deadline이며 adapter 예산보다 2초 넘게 길어야 한다.

stdout에는 `AgentOutput` JSON 하나를 출력한다. 실패한 경우에도 관측 가능한 answer/metrics/runtime/status를 보존하고 nonzero로 종료한다. CLI 원문은 artifact 파일에 둔다.

최종 모델 응답은 `answer`와 `evidence {symbols, paths, relationships}`만 허용한다. 세 근거 배열을 모두 요구하고 null·추가 속성·잘린 JSON을 거절한다. 마지막 agent message와 output-last-message의 구조화 값이 같아야 한다.

## 실행 절차

1. 명시적인 model, effort, tier, 양수 timeout과 supported strategy를 검사한다.
2. source HEAD가 task의 전체 40자리 SHA와 일치하고 작업 트리에 변경·untracked 파일이 없는지 확인한다.
3. `git archive`로 실행별 target을 만들고 content hash를 기록한다. source의 ignored 파일이나 node_modules를 복사하지 않는다.
4. target symlink, `.codex`, `.agents` 항목은 거절한다. archive 이전에 pinned tree의 Git submodule(mode 160000)도 거절하여 내용이 빠진 측정을 막는다. 이 제한은 외부 경로·프로젝트별 설정의 영향을 줄이기 위한 MVP 계약이다.
5. CLI binary hash/version과 `codex login status`를 확인한다. 기존 ChatGPT 로그인이 아니면 `auth` 실패다.
6. `--ignore-user-config --ephemeral --strict-config --json -s read-only`, output schema, model/effort/tier를 개별 argv로 전달한다. web·apps·plugins·hooks·추가 agent·browser 등의 기능을 비활성화한다.
7. 필요한 기본 OS 환경변수만 전달하며 OPENAI_API_KEY/CODEX_API_KEY/ORCAROUTER_API_KEY는 전달하지 않는다. CODEX_HOME을 변경하거나 credential 파일을 복제하지 않는다.
8. 실행 후 target hash와 최종 응답을 확인한다. timeout 시 Codex process group을 종료하고 기다린다. harness는 TERM 후 최대 2초의 종료 시간을 주어 adapter가 partial result를 남길 수 있게 한다.

검증한 CLI는 `codex-cli 0.155.0-alpha.9.2`, 플랫폼은 macOS arm64다. 다른 CLI 버전도 실제 binary/version을 기록하며 옵션·이벤트가 호환되지 않으면 실패로 남긴다. Windows 실행은 현재 지원하지 않는다.

## MCP

Codex가 stdio MCP process를 소유하며 cwd는 실행별 target이다. graph host의 Node server, 플랫폼 graph binary, tsgo binary를 절대 경로로 전달한다. `required=true`, startup 30초, tool timeout 90초, enabled_tools는 조회 도구 하나다. 해당 도구에만 `approval_mode="approve"`를 설정한다.

기동 성공과 실제 호출 성공은 별개다. MCP가 승인 거절·실행 오류를 반환하면, CLI가 파일 탐색으로 답변을 완성해 exit 0이더라도 `graph_unavailable`과 invalid measurement를 기록한다. 이 MVP는 Graph 도구 오류를 보수적으로 invalid로 분류하며 실패 수를 반드시 공개한다.

`result.structured_content`와 text content의 Graph contract를 모두 읽는다. 성공한 escape 요청은 도구 성공 수에는 포함하되 Graph fact 성공 수에는 포함하지 않는다.

## 관측 필드

| 위치 | 필드와 의미 |
| --- | --- |
| `status` | `execution` completed/failed, `measurement` valid/invalid, `failureKind`, `warnings` |
| `runtime` | provider=openai, adapter=codex-cli, authMode=chatgpt, requestedModel, effort, serviceTier |
| `runtime` | cliVersion, cliBinaryHash, commonConfigHash, promptTemplateHash, schemaHash |
| `runtime` | toolProfile, sandboxProfile, timeoutMs, budgetPolicy, graphFingerprint, contextFingerprint, targetSnapshotHash, artifactDir |
| `runtime.modelProvenance` | JSONL에서 actual model이 관측되지 않아 현재 `requested_only`; model 필드는 비워 둠 |
| `metrics.toolCalls`, `graphToolCalls` | 고유 item id 기준 시도 수. started/completed 중복 집계 금지 |
| `metrics` | toolCallsSucceeded, graphToolCallsSucceeded, graphFactCallsSucceeded, graphUsed |
| `metrics` | inputTokens, outputTokens, cacheReadInputTokens, cacheCreationInputTokens, reasoningOutputTokens |

usage는 single terminal turn에 보고된 값만 사용한다. 필드가 없으면 nil이며 실제 0만 0이다. 입력에 cached input을 다시 더하거나 출력에 reasoning을 임의로 더하지 않는다. 핵심 usage 누락 또는 불완전한 stream은 `partial=true`; report의 토큰 비교에서 보수적으로 제외한다. 비용은 관측되지 않으므로 `costUsd`를 생략한다.

`item.type=error` warning과 terminal 실패를 구분한다. 잘린 JSONL, 누락/중복 terminal turn, 음수 usage, 예상 밖 MCP 사용, final schema 불일치는 실패다. 여러 오류는 함께 보존하며, 앞선 Graph/provider 오류 이후 발생한 quota/auth도 중단 분류에서 우선한다. CLI가 MaxTurns를 강제한다는 근거가 없어 시간 예산만 사용한다.

## 비교 검증과 채점

기존 experiment/task/run/revision/provider/model/adapter 조건에 effort, tier, 인증 방식, CLI/version, 공통 설정·prompt·schema, tool/sandbox, 시간 예산, graph/context/target fingerprint를 추가한다. 전략별 도구 등록과 prompt의 의도된 차이는 공통 설정과 구분한다.

Codex 비교에서 필수 fingerprint가 없으면 거절한다. `requested_only` 비교를 허용하되 보고서에 실제 serving model이 검증되지 않았음을 표시한다. invalid measurement는 성공 집계·paired delta에 들어가지 않는다. 실패와 짝이 없는 결과도 원본 및 전체 실행 분모에 남긴다.

기존 `accuracy`는 grader v1의 exact evidence F1이다. symbols/paths/relationships를 합친 집합을 채점하며, 올바른 추가 근거도 unexpected로 감점될 수 있다. F1·precision·required recall을 함께 보고하고 unexpected를 hallucination으로 부르지 않는다. 이번 변경은 gold나 grader를 사후 조정하지 않는다.

## 남아 있는 제약

- 읽기 전용 sandbox는 target 밖 모든 읽기까지 금지하는 완전 격리가 아니다. prompt와 실행 로그를 함께 확인해야 한다.
- user skill/rules 파일과 환경을 fingerprint하지만 실제 모델 context 전체를 관측하지는 못한다. symlink는 링크 정보 기준이며 외부 대상의 모든 변경을 보장하지 않는다. host-context 제한을 warnings로 표시한다.
- dependency 없는 archive로 내부 코드 탐색을 측정한다. 외부 타입까지 포함한 전체 TypeScript graph 완전성을 주장하지 않는다.
- 요청 모델과 serving model, token 수와 구독 사용량/청구액은 동일한 개념이 아니다.
- 소표본 median/p95와 paired delta는 기술 통계다. 일반적 성능 우위의 통계적 증거가 아니다.

## 근거

[공식 인증](https://developers.openai.com/codex/auth/), [비대화형 실행](https://developers.openai.com/codex/noninteractive/), [MCP 설정](https://developers.openai.com/codex/mcp/). 구현의 기준은 실제 캡처한 이벤트 fixture와 테스트이며 변경된 provider 동작은 별도 실측으로 검증한다.
