# Live Model Benchmark

The repository includes a manual GitHub Actions workflow for running the real OpenAI-compatible adapter against the fixed loglens benchmark target without exposing API keys in source control.

## Required repository secret

Choose one:

- `ORCAROUTER_API_KEY`
- `OPENAI_API_KEY`

## Run

Open **Actions → live benchmark → Run workflow** and choose:

- provider
- model
- repeat

The workflow:

1. builds `agent-bench` and `openai-adapter`
2. checks out loglens at commit `985d81ee1fb97570ae1f6da39775c7b0dec38db2`
3. runs the three real-world benchmark tasks
4. generates JSONL and Markdown results
5. uploads them as a GitHub Actions artifact

Secrets are only passed to the adapter process through the job environment and are not written into benchmark result files.
