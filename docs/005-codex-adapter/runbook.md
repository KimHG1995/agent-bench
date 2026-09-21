# Codex adapter 실행 안내

기존 ChatGPT 로그인을 사용하는 로컬 CLI benchmark다. API key는 필요하지 않으며 사용자 Codex 사용량을 사용한다.

## 준비

```bash
codex login status

go build -o bin/agent-bench ./cmd/agent-bench
go build -o bin/codex-adapter ./cmd/codex-adapter
bash scripts/prepare-loglens.sh
bash scripts/prepare-ts-graph-tools.sh
```

로그인 상태가 `Logged in using ChatGPT`여야 한다. 로그인 자체는 필요할 때 사용자가 별도로 `codex login`으로 수행한다. macOS 앱 번들 CLI를 사용하는 경우 `AGENT_BENCH_CODEX_BIN=/Applications/ChatGPT.app/Contents/Resources/codex`를 지정할 수 있다.

## 비교 실행

```bash
export AGENT_BENCH_CODEX_MODEL=gpt-6-astra
export AGENT_BENCH_CODEX_EFFORT=xhigh
export AGENT_BENCH_TS_GRAPH_HOST=targets/ts-graph-tools
export AGENT_BENCH_OUT_DIR=results/codex-example

AGENT_BENCH_REPEAT=3 bash scripts/run-codex-comparison.sh
```

예시 모델은 실측에 사용한 요청값이다. 계정에서 사용할 모델을 명시해야 하며 자동 모델 선택은 하지 않는다. 기본 과제는 loglens 3개, 기본 반복은 3회이므로 최대 18번의 agent 실행이다. 각 agent 실행 안에서 모델 요청이 여러 번 발생할 수 있다.

출력 디렉터리는 새 경로여야 한다. 기존 디렉터리를 재사용하면 호출 전에 실패한다. 홀수 반복은 baseline → graph, 짝수는 graph → baseline이다. auth/quota 오류에서는 해당 결과를 저장한 뒤 남은 실행을 중단한다. 재시도나 다른 provider fallback은 하지 않는다.

## 설정

| 환경변수 | 기본값 / 역할 |
| --- | --- |
| AGENT_BENCH_CODEX_BIN | codex 실행 파일 |
| AGENT_BENCH_CODEX_MODEL | 필수, 요청 모델 |
| AGENT_BENCH_CODEX_EFFORT | medium |
| AGENT_BENCH_CODEX_SERVICE_TIER | default |
| AGENT_BENCH_CODEX_TIMEOUT | 5m, adapter 전체 시간 |
| AGENT_BENCH_TIMEOUT | 비교 script에서 6m, harness 시간 |
| AGENT_BENCH_CODEX_REQUIRE_GRAPH_USE | false; true면 연결 smoke에서 fact 호출 필수 |
| AGENT_BENCH_TS_GRAPH_HOST | script에서 targets/ts-graph-tools |
| AGENT_BENCH_NODE_BIN | node |
| AGENT_BENCH_TASKS | script에서 benchmarks/loglens |
| AGENT_BENCH_REPEAT | script에서 3 |
| AGENT_BENCH_OUT_DIR | script에서 timestamp/PID가 포함된 새 결과 경로 |
| AGENT_BENCH_CODEX_ARTIFACTS | adapter 직접 실행 시 results/codex-artifacts; script는 OUT_DIR/raw로 지정 |

harness timeout은 adapter timeout보다 2초 넘게 길어야 한다. 실제 적용하지 않는 max-turns 값을 기록하지 않는다.

## 결과

`OUT_DIR`에 전략/반복별 JSONL과 `report.md`가 생긴다. `raw/`의 실행별 디렉터리에는 prompt, schema, events, stderr, final answer, manifest 및 target snapshot을 보존한다. 이 디렉터리는 소스 코드 내용을 포함할 수 있으므로 원문 공개 전 검토한다. 저장소에는 credential과 raw 실행 디렉터리를 커밋하지 않는다.

report는 evidence F1, required recall, precision, 토큰·캐시, Graph 사용 수, 비교 가능한 pair 수와 제외 이유를 표시한다. `requested_only`는 요청 모델만 확인했다는 뜻이다. `costUsd`가 없으면 비용 미관측이지 무료 사용량 무제한이 아니다.

## 실패 처리

| failureKind | 의미와 다음 행동 |
| --- | --- |
| auth / quota | 계정 문제; 상태를 확인한 뒤 새 실험 경로로 실행 |
| config | CLI/모델/시간 예산/호환 옵션 확인 |
| target | SHA 불일치, dirty target, 지원하지 않는 archive 항목 확인 |
| target_mutated | 실행 중 target 변경; 원문을 확인하고 해당 비교 폐기 |
| graph_unavailable | MCP 기동·승인·도구 실행 오류; stderr/events 확인 |
| graph_not_used | 연결 smoke에서 실제 Graph fact 조회가 없었음 |
| protocol / output_schema | stream 또는 final 응답 계약 불일치; fixture로 회귀 재현 |
| timeout | deadline 종료; partial 결과 보존 |
| process / provider / artifact | exit 상태·서비스 오류·로컬 파일 저장 실패 확인 |

`agent-bench run -stop-on-fatal`은 auth/quota 결과를 저장하고 exit 3을 반환한다. 비교 script는 보고서 생성까지 시도한 뒤 실패 시 exit 1을 반환한다. exit 0만으로 Graph 호출이 성공했다고 판단하지 않는다.

## 검증

```bash
go test -race ./...
go vet ./...
bash -n scripts/run-codex-comparison.sh
```

CI는 외부 모델 호출 없이 실제 subprocess fixture와 캡처된 JSONL을 사용한다. 개인 ChatGPT 인증 파일을 GitHub Actions secret으로 복사하는 방식은 제공하지 않는다.
