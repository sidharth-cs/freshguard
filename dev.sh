#!/usr/bin/env bash
# Build Bifrost (dynamic, plugin-capable) + the FreshGuard plugin, then run the gateway.
#   ./dev.sh build   build everything
#   ./dev.sh run     build if needed, then start Bifrost on $PORT (default 8080) with FreshGuard loaded
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
BIFROST_TAG="transports/v2.4.0"   # plugin go.mod is pinned to this release's dependency versions
BIFROST_DIR="$ROOT/.bifrost"
RUN_DIR="$ROOT/run"
PORT="${PORT:-8080}"
ARCH="$(uname -m | sed 's/x86_64/amd64/')"   # force native arch: an Intel `go` on Apple Silicon defaults to amd64

build() {
  if [ ! -x "$BIFROST_DIR/tmp/bifrost-http" ]; then
    [ -d "$BIFROST_DIR" ] || git clone --depth 1 --branch "$BIFROST_TAG" https://github.com/maximhq/bifrost "$BIFROST_DIR"
    (cd "$BIFROST_DIR" && make setup-workspace >/dev/null && make build DYNAMIC=1)
  fi
  # Must match Bifrost's own build flags exactly or plugin.Open rejects the .so.
  (cd "$ROOT/plugin" && GOWORK=off GOARCH="$ARCH" CGO_ENABLED=1 \
    go build -mod=mod -trimpath -tags sqlite_static -buildmode=plugin -o "$RUN_DIR/freshguard.so" .)
  echo "built: $BIFROST_DIR/tmp/bifrost-http + $RUN_DIR/freshguard.so"
}

# Demo config: Tavily goes through a Bifrost relay (credential injected by Bifrost, spend
# logged per virtual key), and one demo virtual key has FreshGuard switched on for it.
config() {
  mkdir -p "$RUN_DIR"
  rm -f "$RUN_DIR"/config.db*   # ponytail: fresh store each run so config.json is always the source of truth
  cat > "$RUN_DIR/config.json" <<EOF
{
  "\$schema": "https://www.getbifrost.ai/schema",
  "config_store": {"enabled": true, "type": "sqlite", "config": {"path": "$RUN_DIR/config.db"}},
  "logs_store": {"enabled": false},
  "providers": {
    "gemini": {
      "keys": [{"name": "gemini", "value": "env.GEMINI_API_KEY", "weight": 1, "models": ["*"]}],
      "network_config": {"max_retries": 3, "retry_backoff_initial": 1000, "retry_backoff_max": 8000}
    }
  },
  "relays": [{
    "name": "tavily",
    "description": "Live web search for FreshGuard",
    "base_url": "https://api.tavily.com",
    "auth": {"type": "bearer"},
    "credential": {"value": "env.TAVILY_API_KEY"},
    "routes": [{"method": "POST", "path": "/search"}]
  }],
  "governance": {
    "virtual_keys": [{
      "id": "vk-freshguard-demo",
      "name": "freshguard-demo",
      "value": "env.FRESHGUARD_DEMO_VK",
      "is_active": true,
      "provider_configs": [{"provider": "gemini", "allowed_models": ["*"], "key_ids": ["*"], "weight": 1.0}],
      "relay_grants": [{"relay_name": "tavily"}]
    }]
  },
  "plugins": [{
    "enabled": true,
    "name": "freshguard",
    "path": "$RUN_DIR/freshguard.so",
    "config": {
      "gateway_url": "http://localhost:$PORT",
      "model": "gemini/gemini-3.5-flash-lite",
      "fallbacks": ["gemini/gemini-3.1-flash-lite", "gemini/gemini-3.8-flash"],
      "tavily_url": "http://localhost:$PORT/relay/tavily",
      "virtual_keys": ["env.FRESHGUARD_DEMO_VK"],
      "max_claims": 5,
      "trigger": "header",
      "timeout_seconds": 30
    }
  }]
}
EOF
}

# A strong random demo virtual key (Bifrost requires the sk-bf- prefix), kept across runs.
demo_vk() {
  [ -f "$RUN_DIR/demo_vk" ] || { echo "sk-bf-$(openssl rand -hex 24)" > "$RUN_DIR/demo_vk"; chmod 600 "$RUN_DIR/demo_vk"; }
  FRESHGUARD_DEMO_VK="$(cat "$RUN_DIR/demo_vk")"
  export FRESHGUARD_DEMO_VK
}

run() {
  mkdir -p "$RUN_DIR"
  [ -f "$RUN_DIR/freshguard.so" ] || build
  config
  demo_vk
  set -a; [ -f "$ROOT/.env" ] && . "$ROOT/.env"; set +a
  exec "$BIFROST_DIR/tmp/bifrost-http" -app-dir "$RUN_DIR" -port "$PORT" -log-style pretty
}

case "${1:-run}" in
  build) mkdir -p "$RUN_DIR"; build ;;
  run) run ;;
  *) echo "usage: $0 [build|run]" >&2; exit 1 ;;
esac
