# Codex adapter 통합 실측 — 2026-09-21

3개 과제 × 2전략 × 3반복, 총 **18회 실행과 9 paired 비교**를 완료했다. 모두 정상 실행됐으며 Graph 9회 모두 실제 Graph fact 조회가 관측됐다. 이번 조건에서 Graph는 도구 호출 수를 줄였지만 토큰과 실행 시간의 전반적 절감은 보여주지 못했다.

이 결과는 agent-bench가 전략의 손익과 회귀를 측정할 수 있다는 근거다. Graph의 일반적 성능 우위나 자연어 답변 정확도 개선을 입증하는 자료는 아니다.

## 고정 조건과 출처

| 항목 | 값 |
| --- | --- |
| 실행 시간 | 2026-09-21 07:35:41–07:46:57 UTC / 16:35:41–16:46:57 KST |
| experiment id | `20260921T073541Z-76682` |
| 측정한 harness commit | `af259e211c2ff383c7ac08893d4c61ca2fd16546` |
| target | loglens `985d81ee1fb97570ae1f6da39775c7b0dec38db2` |
| graph host | ts-graph-tools `6cc701bde596a955cb95824f67e896e13c10fff2`, @ttsc/graph 0.19.3 |
| 요청 모델 / effort / tier | `gpt-6-astra` / `xhigh` / `default` |
| 실제 serving model | 미관측, `requested_only` |
| 인증 | 사용자의 기존 ChatGPT 로그인, API key/fallback 없음 |
| CLI / 환경 | `codex-cli 0.155.0-alpha.9.2`, macOS arm64, Node 22.22.3 |
| 시간 예산 | adapter 5분 / harness 6분, 실행마다 새 ephemeral 세션 |
| 실행 순서 | 반복 1 baseline→graph, 반복 2 graph→baseline, 반복 3 baseline→graph; 각 전략에서 3개 과제 실행 |
| Graph 정책 | 제공 여부 비교, `REQUIRE_GRAPH_USE=false` |

실측 후 코드 리뷰로 발견한 계정 오류 우선순위와 submodule 거절 문제는 `37bb84f`에서 회귀 테스트와 함께 수정했다. 위 18회는 **수정 전 측정 commit의 결과**이며 재실행으로 교체하지 않았다. 해당 실험에는 계정 오류와 submodule이 없었으므로 이 두 실패 처리 경로는 실행되지 않았다.

개인 절대 경로와 모델 답변 본문을 제외한 실행별 수치·채점 근거·공통 fingerprint·raw 파일 hash는 [live-results.json](live-results.json)에 보존한다. 원문 JSONL, answer, manifest, prompt, stderr와 target archive는 로컬 `results/codex-live-20260921/`에 보존하며 Git에는 포함하지 않는다. 앞선 4회 exploratory CLI 호출은 이번 표본에 합치지 않았다.

## 결과

| 지표 | baseline, n=9 | graph, n=9 |
| --- | ---: | ---: |
| 정상 실행 / 유효 측정 | 9 / 9 | 9 / 9 |
| Evidence F1 평균 | 0.795 | 0.841 |
| 필수 근거 recall 평균 | 0.956 | 0.935 |
| Precision 평균 | 0.714 | 0.775 |
| 도구 호출 수 중앙값 | 5 | 4 |
| Input tokens 중앙값 | 45,591 | 69,445 |
| Cached input 중앙값 | 32,768 | 43,008 |
| Output tokens 중앙값 | 563 | 672 |
| 총 input+output 중앙값 | 46,154 | 70,124 |
| 실행 시간 중앙값 | 32.520초 | 34.266초 |
| 실제 Graph fact 사용 | 0 / 9 | 9 / 9 |

**같은 task/run의 9쌍에서 계산한 변화량의 중앙값**은 다음과 같다. 위 전략별 중앙값끼리 나눈 값과는 다른 통계량이다.

| Paired 지표 | Graph − baseline 또는 변화율 |
| --- | ---: |
| Evidence F1 차이 | +0.039 |
| 도구 호출 수 | −25.0% |
| 총 input+output tokens | +23.9% |
| 실행 시간 | +24.7% |

토큰 총합은 baseline input 463,927 / output 5,890, graph input 601,947 / output 7,012다. Cached input은 각각 311,040과 428,800으로 input에 이미 포함된다. USD 비용과 Codex 구독 한도 차감량은 이 수치에서 추산하지 않는다.

## 과제별 해석

각 과제는 전략별 3회다. 토큰과 시간 변화율은 해당 과제의 paired 변화율 중앙값이다.

| 과제 | F1 평균 baseline→graph | Recall 평균 baseline→graph | 총 토큰 변화 | 시간 변화 |
| --- | ---: | ---: | ---: | ---: |
| ingest-sink | 0.777→0.857 | 0.867→0.900 | +20.6% | +15.1% |
| reports-latency-flow | 0.608→0.790 | 1.000→1.000 | −4.5% | +24.7% |
| spike-evaluator-flow | 1.000→0.875 | 1.000→0.905 | +108.2% | +49.8% |

`reports-latency-flow`는 두 전략 모두 필수 근거를 찾았다. F1 차이는 주로 정답 집합 밖의 추가 근거 수 차이에서 발생했다. 따라서 F1 증가를 필요한 사실을 더 많이 발견한 결과로 해석하면 안 된다.

`spike-evaluator-flow`의 Graph 반복 2는 파일 경로에 `:24`, `:56` 같은 줄 번호를 붙였다. grader v1은 경로를 exact match하므로 해당 경로를 missing/unexpected로 나눠 감점했다. ingest에서는 관계 endpoint에 `(useExisting binding)` 같은 설명을 붙인 경우도 있었다. 이런 점수 차이는 의미적 오류와 출력 표기의 차이가 섞여 있음을 보여준다. 이번 결과에 맞춰 정답이나 정규화 규칙을 사후 변경하지 않았다.

도구 호출 수가 줄어도 한 번의 Graph 결과 크기, 도구 설명, 캐시, 추론 시간이 달라질 수 있다. 이번 관측만으로 토큰 증가의 원인을 특정할 수 없으며 후속 실험에서 분해해야 한다.

## 결과 감사

- 실행 조합 18개가 유일하며, report에서 성공한 비교 pair 9/9를 확인했다.
- 18개 manifest의 exit 0, valid measurement, targetUnchanged를 확인했다. 보존된 target 파일의 hash를 다시 계산해 runtime 값과 대조했다.
- 모든 실행에서 마지막 agent message와 answer.json이 일치했다. terminal turn은 하나였으며 input/output/cache usage가 JSONL 요약과 일치했다.
- 고유 tool item id를 다시 세고 Graph 성공과 fact type을 확인했다. Graph 9/9 사용, baseline의 MCP 호출은 0회다.
- CLI/config/prompt/schema/graph/context/target fingerprint는 18회에서 일치했다. taskHash는 같은 과제의 pair에서 고정됐다.
- 원본 loglens와 benchmark target의 HEAD/clean 상태를 다시 확인했다.
- 인증/quota 실패, 잘린 stream, partial core usage, 무효 측정은 이번 18회에서 관측되지 않았다.

## 검증과 재현 범위

최종 실패 처리 수정 후 `go test -race ./...`, `go vet ./...`, 전체 command build, 비교 script syntax 검사를 통과했다. 실제 Graph MCP lookup smoke도 통과했다. 독립 코드 리뷰의 Important 발견 2개는 각각 실패하는 테스트로 재현한 뒤 수정했으며 Critical 발견은 없었다.

새 실험은 [runbook](runbook.md)의 명령으로 실행한다. 기존 raw 결과를 다시 집계할 때는 다음과 같이 사용한다.

```bash
bin/agent-bench report \
  -inputs results/codex-live-20260921/baseline-1.jsonl,results/codex-live-20260921/graph-1.jsonl,results/codex-live-20260921/graph-2.jsonl,results/codex-live-20260921/baseline-2.jsonl,results/codex-live-20260921/baseline-3.jsonl,results/codex-live-20260921/graph-3.jsonl \
  -out results/codex-live-20260921/report-reproduced.md
```

이 실험은 고정 입력·관측값·검증 절차를 재현 가능하게 남긴다. 모델의 확률적 출력, 서버 캐시/부하, 알파 CLI 동작까지 동일하게 재생한다는 뜻은 아니다. user context는 fingerprint 수준으로 통제했고 완전 격리를 보장하지 않는다. 외부 의존성은 target archive에 설치하지 않았다. 한 저장소·3문제의 소표본이므로 통계적 유의성이나 다른 저장소로의 일반화는 주장하지 않는다.

후속 작업은 [개발 방향](roadmap.md)의 순서로 진행한다. 특히 채점 입력의 canonical 표기와 필수/허용 근거를 먼저 버전으로 고정하고, 데이터셋을 넓힌 뒤 전략 최적화를 평가해야 한다.
