# 005 — Codex CLI adapter specification

상태: **Proposed — 실측으로 실행 가능성을 확인한 구현 전 명세**  
작성일: 2026-09-21 KST  
기준 agent-bench: `9059358749c326fadd5e94172c99b04aaff71d76`

## 목적과 결정

기존 ChatGPT 로그인으로 `codex exec`를 실행하는 adapter를 추가한다. OrcaRouter 무료 quota와 별도로 코드 탐색 비교를 실행할 수 있게 하되, 사용자 계정의 Codex 사용량을 사용한다. ChatGPT 인증 정보를 추출하여 OpenAI-compatible API의 bearer key로 재사용하지 않는다.

2026-09-21에 실제 Codex 호출 4회를 실행했다. 승인 설정이 빠진 Graph 실행 1회를 제외하고, 같은 반복 번호의 baseline/graph 한 쌍에서 구조화 응답, 토큰, 도구 이벤트, Graph 조회를 확인했다. 상세 수치와 제한은 [실측 보고서](../codex-smoke/measurement.md)에 있다. 이번 결과는 adapter 연결 가능성의 증거이며, 일반적인 Graph 성능 우위를 입증하지 않는다.

이 명세는 구현 요구사항이다. 현재 main에는 Codex adapter가 없으며, 아래의 신규 필드·명령·검증은 아직 제공되지 않는다.

## 비교 범위

| 항목 | baseline | graph |
| --- | --- | --- |
| 모델·effort·service tier | 명시적으로 고정 | 동일 |
| 인증 | 기존 ChatGPT CLI 로그인 | 동일 |
| 대상과 질문 | 동일 commit와 task hash | 동일 |
| 일반 탐색 | Codex의 읽기 전용 로컬 shell | 동일 |
| Graph MCP | 제공하지 않음 | pinned `inspect_typescript_graph`만 추가 |
| 세션 | 새 ephemeral process | 새 ephemeral process |
| 실행 순서 | 반복마다 교차 | 반복마다 교차 |

Codex shell 한 호출은 여러 검색·읽기 명령을 포함할 수 있다. 기존 OpenAI adapter의 `list_files/search_text/read_file` 호출 수와 직접 비교하지 않는다. 우선 Codex adapter 안에서만 전략을 비교한다. 모델이 다른 OrcaRouter 결과와도 성능 차이를 합산하지 않는다.

기본 실험은 **도구 제공 여부 비교**다. 정상 제공된 Graph를 모델이 선택하지 않은 실행도 보존하고 `graphUsed=false`로 집계한다. 사용한 실행만 골라 성능을 계산하면 선택 편향이 생긴다. 별도의 **연결 smoke**에서만 실제 Graph 근거 조회 성공을 최소 1회 요구한다. 승인 거절·MCP 기동 실패 등으로 도구 자체가 사용할 수 없었던 실행은 설정 실패로 분리한다.

## 입력·출력 계약

stdin은 기존 `domain.RunRequest`를 사용한다. `Task.ForAgent()`에 포함된 id/category/repository/question만 모델에 전달하고 `expected`는 grader에만 둔다. repository 경로는 로컬 preflight에서 검증하며, 대상과 정답 파일은 별도 디렉터리에 둔다.

stdout에는 `domain.AgentOutput` JSON 하나만 출력한다. Codex JSONL, stderr, 로그를 stdout에 섞지 않는다. 최종 모델 응답은 `answer`와 `evidence {symbols, paths, relationships}`만 포함하는 엄격한 JSON Schema로 제한한다. metrics/runtime/error는 adapter가 관측하여 작성한다.

| 데이터 | 규칙 |
| --- | --- |
| answer/evidence | output-last-message를 JSON Schema로 검증; 마지막 final event와 일치 확인 |
| runtime.provider | `openai` |
| runtime.adapter | `codex-cli` |
| runtime.requestedModel / effort | 실제 실행 인자로 전달한 값 |
| runtime.model | 실제 관측 값만 기록; JSONL에 없으면 비워 둠 |
| metrics.costUsd | 관측되지 않으면 생략; ChatGPT 사용을 `$0`으로 변환하지 않음 |
| error | 실패 종류와 짧은 원인; 비밀정보 제외 |

현재 타입에 없는 다음 필드는 additive migration으로 추가한다. 정확한 위치와 JSON 이름은 구현 전에 고정하고 계약 테스트에 넣는다.

- 실행 출처: `authMode`, `cliVersion`, `cliBinaryHash`, `serviceTier`, `modelProvenance` (`observed` / `requested_only` / `unknown`).
- 재현 조건: `commonConfigHash`, `promptTemplateHash`, `schemaHash`, `toolProfile`, `sandboxProfile`, `timeoutMs`, `budgetPolicy`, graph package/binary hash.
- 상태: `executionStatus`, `measurementStatus`, `failureKind`, `warnings`.
- 도구 관측: `toolCallsAttempted`, `toolCallsSucceeded`, `graphToolCallsAttempted`, `graphToolCallsSucceeded`, `graphFactCallsSucceeded`, `graphUsed`.
- metric별 관측 상태. 초기 구현에서 기존 `metrics.partial`을 유지할 경우, 누락·절단된 사용량을 complete로 표시하지 않는 보수적 판정을 사용한다.

기존 `toolCalls`와 `graphToolCalls`의 의미도 문서화한다. 제안된 정의는 terminal result의 성공 여부와 무관한 **고유 시도 수**다. 새 성공 수 필드로 실제 실행 여부를 구분한다.

## 실행과 인증

1. CLI 버전과 `codex login status`로 준비 상태를 확인한다. 인증 파일 내용은 읽거나 복사하지 않는다. 비대화형 benchmark가 로그인 flow를 자동 시작하지 않는다.
2. `forced_login_method="chatgpt"`를 사용하고, 실행 환경에서 `OPENAI_API_KEY`, `CODEX_API_KEY`, provider secret을 제외한다. API 인증·다른 모델로 자동 전환하지 않는다.
3. model/effort/tier/timeout을 명시한다. 사용자 설정의 기본값이 바뀌어도 기존 실험 결과와 조용히 섞이지 않아야 한다.
4. `--ephemeral`, `--json`, `--output-schema`, `--output-last-message`, 읽기 전용 sandbox를 사용한다. 각 실행은 이전 대화를 이어받지 않는다.
5. target revision, 작업 트리 상태, 의존성 버전과 변경 전후 파일 hash를 검증한다. archive를 쓰면 원본 commit와 archive/snapshot hash로 출처를 증명한다.
6. API 환경변수뿐 아니라 shell 환경, 전역 instruction/skill/MCP 영향도 통제·기록한다. `--ignore-user-config`만으로 완전 격리를 선언하지 않는다.

이번 CLI `0.155.0-alpha.9.2`는 `--ignore-user-config`와 `skip_host_skill_discovery`를 사용해도 사용자 skill YAML을 읽으려는 로그가 남았다. 따라서 이 옵션 조합은 격리 완료의 근거가 아니다. 전역 skill이 실제 모델 context에 얼마나 들어갔는지는 이번 capture로 확정할 수 없다. 정식 실험은 지원되는 격리 방법을 검증하거나 동일한 환경 manifest를 저장하고 제한을 명시해야 한다. 원래 `CODEX_HOME`을 다른 경로로 바꾸거나 인증 파일을 복제하는 방법은 사용하지 않는다.

## MCP 연결과 승인

adapter가 Codex process를 소유하고, Codex가 stdio MCP process를 소유한다. MCP의 cwd는 target root다. graph host의 pinned Node server와 플랫폼별 graph/tsgo binary를 절대경로로 전달한다.

- 허용 도구: `inspect_typescript_graph` 하나.
- `required=true`; 시작 실패를 조용히 무시하지 않는다.
- 검토한 로컬 도구에만 `mcp_servers.ts_graph.tools.inspect_typescript_graph.approval_mode="approve"`를 설정한다. 전역 sandbox/승인을 해제하지 않는다.
- 시작 timeout과 도구 timeout을 명시한다. 전체 run timeout과 구분한다.
- MCP 기동과 도구 조회가 성공해도 실제 호출의 승인·실행이 성공했다고 간주하지 않는다.
- `result.structured_content`를 지원한다. 이번 응답의 `content`는 빈 배열이었다. content 텍스트만 읽는 parser는 Graph 결과를 놓친다.
- `escape` 결과는 도구 실행 성공과 Graph 근거 조회 성공을 구분한다.

실측에서 일반 process는 exit 0으로 끝났지만 MCP는 `MCP tool call requires approval, but approval policy is never`로 실패했다. 이 경우 answer와 usage는 보존하되, Graph 실측이 성공했다고 보고하지 않는다.

## 이벤트·토큰·실패 처리

| 관측 | 처리 요구사항 |
| --- | --- |
| `thread.started` | thread id를 artifact 식별자로 보존 |
| `item.started/updated/completed` | thread/item id로 합쳐 같은 도구를 두 번 세지 않음 |
| `command_execution` | shell 도구 1회; 내부 shell 명령 수로 임의 분해하지 않음 |
| `mcp_tool_call` | server/tool/status/error와 구조화 결과를 기록 |
| `turn.completed.usage` | 실제 존재하는 정수 필드만 metric에 반영 |
| `item.type=error` | warning과 치명적 실패를 분류; 모든 item error를 run 실패로 처리하지 않음 |
| `turn.failed`, 최상위 error, nonzero exit | 실패, 부분 관측 보존 |
| malformed/truncated JSONL, final schema 불일치 | protocol/output 실패; 정상 답변으로 대체하지 않음 |
| 외부 timeout/cancel | process group 종료와 종료 확인, partial result 보존 |

이번 single-turn 실행에서 `input_tokens`, `cached_input_tokens`, `cache_write_input_tokens`, `output_tokens`, `reasoning_output_tokens`를 관측했다. 입력 총량과 cache 사용량을 각각 저장한다. cached input을 input에 다시 더하지 않는다. reasoning과 output도 포함 관계가 검증되기 전 임의 합산하지 않는다. usage 객체가 없거나 필수 항목이 빠지면 unknown/partial이며, 0으로 채우지 않는다. 실제 0으로 반환된 값은 0이다.

현재 `codex exec --help`에서 강제 max-turns 옵션을 확인하지 못했다. 내부 model request 수와 user turn 수는 다르다. 지원하지 않는 maxTurns를 적용했다고 표시하지 않고, MVP는 전체 wall-clock timeout으로 제한한다. 도구 수 제한을 추가할 경우 stream 관측 후 중단하는 한계까지 별도 기록한다.

ChatGPT quota/auth 실패 시 요청을 멈추고 실패를 보존한다. 무한 재시도, 유료 API fallback, 다른 계정으로 전환은 하지 않는다. retry는 실제 reset/retry 정보가 있고 범위가 정해진 경우에만 별도 정책으로 구현한다.

## 채점과 비교 유효성

기존 grader v1은 symbols/paths/relationships를 모두 집합으로 합친 exact evidence F1이다. 보고서의 `accuracy`는 이 값이며, 답변 문장 자체의 의미 정확도와 같지 않다. 이번 정상 쌍은 필수 근거 8개를 모두 찾았지만 추가 근거 때문에 F1이 0.6154 / 0.7273으로 내려갔다.

MVP에서는 기존 점수를 유지하고 **evidence F1·precision·required recall·unexpected**를 함께 노출한다. `unexpected`를 곧바로 hallucination으로 부르지 않는다. 질문 범위를 좁히거나 허용 가능한 추가 근거를 정의하는 개선은 dataset/grader version을 올리는 별도 변경이다. 이번 출력에 맞춰 사후 정답을 바꾸지 않는다.

paired 검증은 기존 task/run/experiment/revision/provider/model/adapter 조건에 다음을 추가한다.

- effort, service tier, CLI/config/prompt/schema hash, 인증 방식, 공통 tool profile, sandbox, timeout 및 budget policy가 동일해야 한다.
- 전략별 의도된 차이는 Graph 도구/설명/승인 설정뿐이며 별도의 strategy config로 기록한다. 전체 prompt hash가 다른 사실만으로 잘못 거절하지 않는다.
- actual model이 관측되지 않은 `requested_only` 결과는 그 상태를 명시한 비교에서만 사용한다. actual-model 일치를 검증했다는 주장을 금지한다.
- 인프라·설정 실패와 모델의 과제 실패, Graph 미사용을 분리한다. 실패 수와 제외 이유를 항상 표시한다.
- unknown/partial metric은 해당 delta에서 제외하고, 다른 유효한 지표까지 0으로 바꾸지 않는다.
- cache와 실행 순서가 같았다고 가정하지 않는다. 입력 총량·cached 입력·출력·wall time을 따로 보고한다.

## 수용 기준

1. 기존 RunRequest/AgentOutput 계약을 지키는 Go `cmd/codex-adapter`가 빌드된다.
2. 실측 fixture로 정상 응답, structured_content, 이벤트 중복, warning, MCP 승인 실패, 잘린 usage, timeout을 검증한다.
3. mock CLI fixture를 통한 회귀 테스트는 로그인이나 모델 비용 없이 CI에서 실행된다.
4. 실제 ChatGPT 로그인으로 baseline/graph 연결 smoke가 성공하고 Graph 근거 호출 성공 이벤트를 남긴다.
5. target 파일이 바뀌지 않고, expected가 prompt/schema/tool output에 노출되지 않는다.
6. 비교 validator가 effort/config/CLI/budget mismatch를 거절하고 requested-only 모델 출처를 표시한다.
7. report가 Graph 승인 실패의 exit-0 실행을 정상 Graph 성공으로 세지 않는다.
8. 정식 통합 smoke는 현재 3개 task × 2전략 × 3반복으로 18회 실행하고 전체 성공/실패·사용량을 보고한다. 이것도 일반적인 성능 우위를 입증할 충분한 표본으로 간주하지 않는다.

## 근거

- [Codex 인증](https://developers.openai.com/codex/auth/): ChatGPT 로그인과 API key 방식 구분.
- [Codex 비대화형 실행](https://developers.openai.com/codex/noninteractive/): JSONL과 구조화 최종 응답.
- [Codex MCP 설정](https://developers.openai.com/codex/mcp/): required server, 도구 allowlist, 도구별 승인 설정.
- [현재 grader](https://github.com/KimHG1995/agent-bench/blob/9059358749c326fadd5e94172c99b04aaff71d76/internal/grader/grader.go#L11), [현재 비교 검증](https://github.com/KimHG1995/agent-bench/blob/9059358749c326fadd5e94172c99b04aaff71d76/internal/report/validate.go#L44).
