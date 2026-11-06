# Contributing to FreshGuard

Thanks for helping. Issues and pull requests are welcome.

## Setup

```bash
cp .env.example .env     # GEMINI_API_KEY, TAVILY_API_KEY
./dev.sh run             # builds Bifrost (dynamic) + the plugin, starts the gateway on :8080
```

## Before you open a PR

```bash
cd plugin
gofmt -l .                                   # must print nothing
GOWORK=off go vet -mod=mod ./...
GOWORK=off go test -mod=mod -race ./...
```

If you change prompts or verdict logic, re-run the live eval and paste the summary table in the PR:

```bash
cd plugin && GOWORK=off go test -mod=mod -tags eval -run TestEval -v -timeout 20m .
```

## Ground rules

- **Fail open.** Nothing FreshGuard does may break or delay a user's answer beyond its timeout.
- **Evidence only.** Verdicts and sources must come from search results, never from the model's memory.
- **Never leak credentials.** Virtual keys go only to the configured gateway; keys never appear in logs or reports.
- **Keep the build matched.** The plugin's `go.mod` pins Bifrost's dependency versions; bump it together with `BIFROST_TAG` in `dev.sh`.
- Add or update a test with every behaviour change.
