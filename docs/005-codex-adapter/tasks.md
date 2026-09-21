# Codex adapter 진행 상태

## 구현 및 자동 검증

- [x] Go stdin/stdout adapter와 명시적 모델·effort·tier 설정.
- [x] ChatGPT 로그인 확인, 환경 allowlist, API fallback 없는 실행.
- [x] commit·dirty 상태 검증과 실행별 archive/hash.
- [x] per-tool Graph 승인, structured_content, escape 구분.
- [x] item id 중복 방지, usage nil/0 구분, strict final 일치 검증.
- [x] timeout/process group 종료, outer timeout partial 출력과 nested CLI 종료 테스트.
- [x] effort/config/CLI/context/target/budget 비교 검증.
- [x] F1과 required recall, cache, Graph 사용 비율, pair 제외 이유 보고.
- [x] 교차 실행 script, 기존 결과 덮어쓰기 거절, fatal 계정 오류 중단.
- [x] 캡처 fixture와 실제 subprocess 테스트, 전체 Go race suite, vet/build.
- [x] 실제 pinned Graph MCP lookup smoke.

## 통합 완료 확인

- [x] 새 adapter로 3개 task × 2전략 × 3반복 실측과 결과 감사: 18/18 유효, 9/9 comparable pair.
- [x] 구현에 맞춘 명세·실측 보고서·운영 문서의 최종 대조.
- [x] 독립 코드 리뷰와 Important 2건의 RED→GREEN 회귀 테스트; 최종 전체 race suite 통과.

리뷰 수정: 앞선 Graph/provider 오류 이후 auth/quota가 발생해도 fatal 중단 분류를 유지하며, archive가 소스를 누락할 수 있는 Git submodule은 실행 전에 거절한다. 실측 commit과 리뷰 수정 commit은 [실측 보고서](live-benchmark.md)에 구분했다. 실행별 검증 수치는 [live-results.json](live-results.json)에 보존한다.

일회성 CLI 실측과 정식 adapter 통합 실측은 별도 실험으로 기록한다. 기존 4회 exploratory 결과를 이번 18회 데이터에 합치지 않는다. 후속 개발 범위는 [roadmap.md](roadmap.md)를 따른다.
