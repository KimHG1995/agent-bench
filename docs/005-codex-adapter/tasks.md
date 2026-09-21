# Codex adapter 구현 순서

상태: 스펙 정리 완료, production 구현 미착수. [명세](spec.md)의 수용 기준을 기준으로 진행한다.

## 완료한 실측

- [x] 기존 ChatGPT CLI 인증 확인, API key 없는 실제 추론.
- [x] 고정 target commit에서 baseline 2회, Graph 조건 2회 실행.
- [x] 초기 Graph 승인 실패를 보존하고 정상 비교에서 제외.
- [x] 도구별 승인 설정 후 실제 MCP `tour` 응답 확인.
- [x] 정상 반복 쌍의 JSON Schema, 사용량, 도구 이벤트, 기존 Go grader 채점 확인.
- [x] 전역 skill 격리·actual model 관측·채점 범위의 제한 기록.

## 1. 계약과 fixture

- [ ] `internal/codexadapter`와 `cmd/codex-adapter`의 최소 stdin/stdout 계약 구현.
- [ ] 신규 관측 상태/출처 필드와 기존 JSON 호환 정책 확정.
- [ ] 실측 JSONL에서 작은 비밀정보 없는 fixture 작성: 정상, warning, Graph 승인 거절, structured_content, partial usage, 비정상 종료.
- [ ] grader v1 의미를 evidence F1로 명시. score 변경은 이번 adapter 변경에 섞지 않음.

## 2. process와 MCP

- [ ] 지원 CLI 버전 preflight와 ChatGPT 인증 검사.
- [ ] 개별 argv 조립, 환경 allowlist, timeout/process group 종료, final schema 검증.
- [ ] target snapshot과 의존성 출처 검증.
- [ ] 전역 skill/context 영향을 재현 가능하게 통제하는 방법 검증. 불가능한 경우 환경 fingerprint와 제한 노출.
- [ ] pinned Graph MCP와 해당 조회 도구의 승인 설정만 추가.
- [ ] 기동 실패·승인 실패·model quota/auth 오류를 분류하고 fallback 없이 기록.

## 3. metric과 report

- [ ] 고유 item id 기준 attempted/succeeded 도구 수.
- [ ] cache 포함 input, output, reasoning 관측값과 unknown 상태 보존.
- [ ] requested-only model 표시, cost 미관측 시 생략.
- [ ] effort/CLI/config/tier/budget 등 pairing 조건 확장.
- [ ] Graph availability 비교와 연결 smoke를 구분하고 Graph 미사용 비율 보고.
- [ ] 실패 포함 분모, 제외 이유, valid-pair 수 표시. 소표본 p95의 해석 제한 명시.

## 4. 통합과 문서

- [ ] `scripts/run-codex-comparison.sh`를 추가하고 실행 순서를 교차.
- [ ] README와 SDD index에 로그인/사용량/측정 범위/제한 안내.
- [ ] fixture test, Go test/vet/build, 실제 두 전략 smoke.
- [ ] 3 task × 3 paired 반복 통합 검증. 계정 사용량 부족 시 실패로 남기고 무한 재시도하지 않음.

GitHub Actions에서 개인 ChatGPT 인증 파일을 secret으로 복사하는 자동화는 이번 범위에 포함하지 않는다. 먼저 사용자 로컬 로그인 환경에서 재현 가능한 adapter를 만든다.
