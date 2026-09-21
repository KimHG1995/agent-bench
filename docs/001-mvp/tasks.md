# Agent Bench MVP Tasks

## Phase 1. 명세

- [x] 문제와 비교 조건 정의
- [x] task schema 정의
- [x] external agent protocol 정의
- [x] deterministic grading 정의
- [x] raw result와 report 형식 정의

## Phase 2. Go Core

- [ ] Go module 초기화
- [ ] domain type 구현
- [ ] JSON task loader 구현
- [ ] command agent 구현
- [ ] deterministic grader 구현
- [ ] benchmark runner 구현
- [ ] JSONL result writer 구현
- [ ] Markdown report 구현
- [ ] CLI run/report 구현

## Phase 3. Fixture

- [ ] synthetic TypeScript fixture 추가
- [ ] lookup benchmark 추가
- [ ] caller benchmark 추가
- [ ] flow benchmark 추가
- [ ] impact benchmark 추가
- [ ] architecture benchmark 추가

## Phase 4. Verification

- [ ] grader unit test
- [ ] task loader unit test
- [ ] report unit test
- [ ] command agent integration test
- [ ] go test ./... 통과
- [ ] GitHub Actions 추가

## Phase 5. Agent Integration

MVP core 완료 이후 별도 진행한다.

- [ ] baseline coding agent adapter
- [ ] ts-graph-tools 기반 graph agent adapter
- [ ] 동일 모델과 설정을 사용한 반복 실험
- [ ] 실제 token과 tool call 수집
- [ ] 결과 문서화
