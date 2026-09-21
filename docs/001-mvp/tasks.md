# Agent Bench MVP Tasks

## Phase 1. 명세

- [x] 문제와 비교 조건 정의
- [x] task schema 정의
- [x] external agent protocol 정의
- [x] deterministic grading 정의
- [x] raw result와 report 형식 정의

## Phase 2. Go Core

- [x] Go module 초기화
- [x] domain type 구현
- [x] JSON task loader 구현
- [x] command agent 구현
- [x] deterministic grader 구현
- [x] benchmark runner 구현
- [x] JSONL result writer 구현
- [x] Markdown report 구현
- [x] CLI run/report 구현

## Phase 3. Fixture

- [x] synthetic TypeScript fixture 추가
- [x] lookup benchmark 추가
- [x] caller benchmark 추가
- [x] flow benchmark 추가
- [x] impact benchmark 추가
- [x] architecture benchmark 추가

## Phase 4. Verification

- [x] grader unit test
- [x] task loader unit test
- [x] report unit test
- [x] command agent integration test
- [x] go test ./... 통과
- [x] go vet ./... 통과
- [x] GitHub Actions 추가
- [x] mock agent 기반 end-to-end run/report 확인

## Phase 5. Agent Integration

MVP core 완료 이후 별도 진행한다.

- [ ] baseline coding agent adapter
- [ ] ts-graph-tools 기반 graph agent adapter
- [ ] 동일 모델과 설정을 사용한 반복 실험
- [ ] 실제 token과 tool call 수집
- [ ] 결과 문서화
