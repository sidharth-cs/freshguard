# Milestone 0 spike — PASSED (2026-11-06)

Custom Go plugin loads on Bifrost v2.4.0 (dynamic build) and its PostLLMHook sees the full chat response.

## Verdict
- **Go plugin (.so)** — the only viable path. Requires Bifrost's documented dynamic build (`make build DYNAMIC=1`); stock static binaries/Docker images can't load .so plugins.
- **WASM** — ruled out: deprecated, and no host callbacks / network, so it cannot call Tavily.
- Gemini free tier: `gemini-3.8-flash` returns 503 under load; use `gemini-3.5-flash-lite` with Bifrost `fallbacks`.

## Build recipe (gotchas found the hard way)
1. Bifrost at release tag `transports/v2.4.0` (main HEAD doesn't build: missing framework package).
2. `make setup-workspace && make build DYNAMIC=1` -> `tmp/bifrost-http`. Makefile builds with `GOWORK=off`, `-trimpath`, `-tags sqlite_static`.
3. Plugin must match EXACTLY: same Go toolchain (1.27.3), `GOARCH=arm64` (local `go` is an x86_64 build -> defaults to amd64), `-trimpath -tags sqlite_static`, and dependency versions pinned to `transports/go.mod` (core v1.14.1). Easiest: start plugin go.mod from transports/go.mod.
   `GOWORK=off GOARCH=arm64 CGO_ENABLED=1 go build -mod=mod -trimpath -tags sqlite_static -buildmode=plugin -o freshguard.so .`
4. config.json: `"plugins": [{"enabled": true, "name": "...", "path": "/abs/path.so", "config": {...}}]`; provider key `"value": "env.GEMINI_API_KEY"`.

## Demo gold
Asked "latest stable Python version?" -> model said "3.13.2 (released February 2025)". Stale as of Nov 2026. Perfect FreshGuard test case.
