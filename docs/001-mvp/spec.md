# Agent Bench MVP Specification

## 1. 목적

Agent Bench는 AI Coding Agent가 코드베이스를 이해하는 서로 다른 컨텍스트 접근 방식을 같은 조건에서 비교하는 재현 가능한 벤치마크 도구다.

MVP의 첫 비교 대상은 다음 두 전략이다.

- baseline: 일반적인 파일 탐색, 검색, 파일 읽기 기반 Agent
- graph: TypeScript Code Graph MCP를 사용할 수 있는 Agent

Agent Bench 자체는 Go로 작성한다. TypeScript 코드는 평가 대상 fixture로만 사용한다.

## 2. 문제 정의

Coding Agent 성능 비교는 최종 답변만으로 판단하기 어렵다. 같은 정답을 내더라도 과도한 파일 읽기, 많은 도구 호출, 긴 실행 시간과 토큰 사용이 발생할 수 있다.

따라서 Agent Bench는 다음을 함께 기록한다.

- 정답 정확도
- 도구 호출 수
- 입력 토큰 수
- 출력 토큰 수
- 실행 시간
- 실행 실패 여부

정확도는 가능한 한 LLM Judge가 아니라 사전에 정의된 deterministic evidence로 평가한다.

## 3. 범위

### 포함

- JSON 기반 benchmark task 정의
- 외부 Agent 실행을 위한 표준 stdin/stdout protocol
- baseline, graph 등 임의 전략 이름 지원
- 동일 task 반복 실행
- harness가 직접 측정하는 wall-clock latency
- Agent가 보고한 tool call, token metric 수집
- symbol, file path, relationship 기반 deterministic grading
- JSONL raw result 저장
- 여러 result 파일의 Markdown summary 생성
- 실패 사례와 누락 evidence 출력
- Go 단위 테스트
- GitHub Actions에서 test 수행

### 제외

MVP에서는 다음을 직접 구현하지 않는다.

- 특정 LLM API 종속 adapter
- Claude Code, Codex 실행기 내장
- 비용 계산
- LLM Judge 기반 의미 평가
- 통계적 유의성 검정
- 웹 대시보드
- 실제 TypeScript Code Graph MCP 서버 구현

외부 Agent adapter는 Agent Bench protocol을 구현해 연결한다.

## 4. 공정 비교 조건

한 실험 안에서 비교 전략은 다음 조건을 동일하게 맞추는 것을 원칙으로 한다.

- 모델과 모델 설정
- 대상 repository와 commit
- 사용자 질문
- 사용 가능한 일반 도구
- 실행 환경과 권한
- 반복 횟수

전략별 차이는 평가하려는 컨텍스트 접근 방식에 한정한다.

예를 들어 baseline과 graph 비교에서는 graph 전략에만 Code Graph MCP 접근 권한을 추가하고 나머지 조건은 동일하게 유지한다.

## 5. Benchmark Task

Task는 다음 정보를 가진다.

- id
- category
- repository path와 고정 revision
- question
- expected evidence

MVP category:

- lookup
- caller
- flow
- impact
- architecture

expected evidence:

- symbols
- paths
- relationships

relationship은 from, to 두 symbol로 표현한다.

## 6. Agent Protocol

Agent Bench는 실행할 외부 프로그램의 stdin으로 단일 JSON request를 전달한다.

Request:

- task
- strategy
- run number

Agent는 stdout으로 단일 JSON response를 반환해야 한다.

Response:

- answer
- evidence.symbols
- evidence.paths
- evidence.relationships
- metrics.toolCalls
- metrics.inputTokens
- metrics.outputTokens

toolCalls와 token metric을 관측할 수 없으면 null 또는 필드 생략을 허용한다.

Agent가 비정상 종료하거나 유효하지 않은 JSON을 반환하면 해당 run은 실패로 기록한다.

## 7. Grading

expected evidence와 Agent evidence를 canonical fact set으로 변환한다.

- symbol:<name>
- path:<path>
- relationship:<from>-><to>

정확도는 두 set의 F1 score로 계산한다.

- precision = true positives / returned facts
- recall = true positives / expected facts
- F1 = 2 * precision * recall / (precision + recall)

expected와 actual이 모두 비어 있으면 1.0으로 처리한다.

결과에는 다음을 함께 남긴다.

- matched facts
- missing facts
- unexpected facts
- accuracy

## 8. 실행

예상 CLI:

```bash
agent-bench run \
  -tasks benchmarks \
  -strategy baseline \
  -command ./bin/baseline-agent \
  -repeat 3 \
  -out results/baseline.jsonl
```

graph 전략:

```bash
agent-bench run \
  -tasks benchmarks \
  -strategy graph \
  -command ./bin/graph-agent \
  -repeat 3 \
  -out results/graph.jsonl
```

리포트:

```bash
agent-bench report \
  -inputs results/baseline.jsonl,results/graph.jsonl \
  -out results/report.md
```

## 9. 결과

각 run은 JSONL 한 줄로 저장한다.

필수 항목:

- taskId
- category
- strategy
- run
- repository
- startedAt
- durationMs
- accuracy
- matched
- missing
- unexpected
- success
- error
- agent metrics

summary는 전략별로 다음을 보여준다.

- run count
- success rate
- mean accuracy
- mean tool calls
- mean input tokens
- mean output tokens
- mean total tokens
- mean latency

관측하지 못한 metric은 0으로 바꾸지 않고 미측정으로 유지한다.

## 10. 완료 조건

MVP 완료 조건:

1. `go test ./...` 통과
2. JSON task를 로드할 수 있다.
3. 외부 process를 protocol에 따라 실행할 수 있다.
4. timeout과 process failure를 result로 기록한다.
5. deterministic grader가 F1과 evidence 차이를 계산한다.
6. 반복 실행 결과를 JSONL로 저장한다.
7. JSONL 여러 개를 Markdown report로 집계할 수 있다.
8. 샘플 task와 synthetic TypeScript fixture가 포함된다.
9. README만 보고 로컬에서 build와 사용 방법을 이해할 수 있다.
