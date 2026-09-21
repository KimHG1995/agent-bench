# Codex 실측 재현 안내

현재 사용 가능한 것은 [일회성 실측 스크립트](../codex-smoke/run_smoke.py)다. `agent-bench` 정식 Codex adapter나 비교 명령은 아직 구현되지 않았다. 이 스크립트는 이번 컴퓨터의 절대경로와 pinned commit에 맞춰져 있다.

## 준비

- Codex CLI: `/Applications/ChatGPT.app/Contents/Resources/codex`, 실측 버전 `0.155.0-alpha.9.2`.
- `codex login status`가 `Logged in using ChatGPT`인지 확인한다.
- `loglens` commit `985d81ee1fb97570ae1f6da39775c7b0dec38db2`와 `ts-graph-tools` commit `6cc701bde596a955cb95824f67e896e13c10fff2`가 깨끗한 작업 트리로 있어야 한다.
- Graph host는 `@ttsc/graph 0.19.3` 및 플랫폼별 binary가 설치되어 있어야 한다. 별도의 target dependency 설치는 이번 smoke에서 하지 않았다.
- 저장된 API key가 없어도 실행 가능하나, 사용자 Codex 계정의 사용량이 필요하다.

## 새로운 실측

다음 명령은 **실제 모델 호출을 추가로 발생시킨다**. 기존 run 번호를 재사용하면 디렉터리 생성 단계에서 실패하므로 raw 결과를 덮어쓰지 않는다. 새 번호를 사용한다.

```bash
python3 /Users/hgkim/Documents/PERSONAL/reviews/agent-bench-20260921/codex-smoke/run_smoke.py baseline --run 3
python3 /Users/hgkim/Documents/PERSONAL/reviews/agent-bench-20260921/codex-smoke/run_smoke.py graph --run 3
```

짝수 반복은 graph → baseline 순서로 실행한다. 이번 실행에서는 run 1의 Graph 승인 설정 실패를 남긴 뒤, run 2를 graph → baseline 순서로 완료했다. `run_smoke.py`의 현재 버전에는 조회 도구 승인 설정이 반영되어 있어 run 1의 실패 설정은 재현하지 않는다. 정확한 당시 argv는 각 `metadata.json`에 남아 있다.

스크립트는 300초 timeout을 적용한다. 모델 선택은 실측 당시 사용자 설정에서 가져온 `gpt-6-astra`, effort `xhigh`, tier `default`로 고정되어 있다. 모델이나 effort를 변경하면 새 실험으로 구분한다.

## 결과 확인

각 run 디렉터리에 `prompt.txt`, `schema.json`, `metadata.json`, `events.jsonl`, `stderr.log`, `answer.json`이 생성된다. credential 원문은 artifact에 저장하지 않는다. 실패한 실행도 삭제하지 않는다.

`metadata.exitCode=0`만 확인하지 않는다. 최종 schema가 유효한지, `turn.completed`가 있는지, usage가 온전한지, Graph 연결 smoke에서 실제 `mcp_tool_call.status=completed`와 Graph 근거 결과가 있는지 확인한다. Graph 승인 실패 후 shell로 작성된 답변은 정상 Graph smoke가 아니다.

채점은 모델 호출 없이 기존 Go grader를 사용했다. helper 원본은 [grade-main.go](../codex-smoke/grade-main.go)에 보존했다. Go `internal` import 규칙 때문에 agent-bench checkout 안의 임시 helper 경로에 놓고 실행해야 한다. 기존 원본 채점 결과는 [grades.json](../codex-smoke/grades.json)에 있다.

`summarize.py`는 이미 존재하는 grades와 events만 읽으며 모델을 호출하지 않는다. 새 실행을 추가한 경우 먼저 새 answer에 대해 Go grader와 schema validator를 다시 실행해 grades를 갱신해야 한다.

## 실패 대응

| 증상 | 처리 |
| --- | --- |
| ChatGPT 로그인 없음/만료 | 로그인 상태를 복구한 뒤 새 run; API key fallback 금지 |
| 계정 quota 소진 | 해당 실패 보존, 명시된 reset 정보 확인; 자동 연속 재시도 금지 |
| MCP 승인 필요 + approval never | 고정된 로컬 조회 도구의 per-tool 설정 확인 |
| MCP initialize 실패 | graph/node binary, cwd, dependency version 점검 |
| 전역 skill 경고 | 실행 성공과 분리해서 기록; 완전 격리라고 주장하지 않음 |
| timeout/final JSON 누락 | 실패·partial 기록; 기존 결과를 정상 값으로 보충하지 않음 |

공식 설명: [ChatGPT 인증](https://developers.openai.com/codex/auth/), [비대화형 실행](https://developers.openai.com/codex/noninteractive/), [MCP 설정](https://developers.openai.com/codex/mcp/).
