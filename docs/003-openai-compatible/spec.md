# OpenAI-Compatible Adapter Specification

## 목적

Claude CLI와 독립적으로 OpenAI 호환 Chat Completions API를 사용해 실제 코드 탐색 Agent를 실행한다.

지원 대상:

- OpenAI API
- OrcaRouter
- 그 외 OpenAI-compatible `/v1/chat/completions` endpoint

OrcaRouter는 공식 문서상 OpenAI 호환 API이며 base URL을 `https://api.orcarouter.ai/v1`로 사용할 수 있다.

## MVP 범위

- baseline 전략만 지원
- 모델이 function tool calling으로 저장소를 탐색
- list_files
- search_text
- read_file
- 최종 evidence JSON 수집
- prompt/completion token 합산
- 외부 프로젝트인 loglens 고정 commit에서 benchmark

graph 전략은 baseline 측정 경로가 안정된 후 같은 agent loop에 추가한다.

## 환경 변수

- AGENT_BENCH_OPENAI_BASE_URL
- AGENT_BENCH_OPENAI_API_KEY
- AGENT_BENCH_OPENAI_MODEL
- AGENT_BENCH_OPENAI_MAX_TURNS
- OPENAI_API_KEY fallback
- ORCAROUTER_API_KEY fallback

## 안전성

tool은 repository root 아래의 읽기 동작만 허용한다.
상위 경로 탈출을 거부하고 node_modules, .git, dist 등은 탐색에서 제외한다.
