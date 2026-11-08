#!/usr/bin/env bash
# Build Bifrost (dynamic, plugin-capable) + the FreshGuard plugin, then run the gateway.
#   ./dev.sh build   build everything
#   ./dev.sh run     build if needed, then start Bifrost on $GATEWAY_PORT (default 8080) with FreshGuard loaded
#   ./dev.sh up      run + the demo page (http://localhost:8000); what the Docker image runs
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
BIFROST_TAG="transports/v2.4.0"   # plugin go.mod is pinned to this release's dependency versions
BIFROST_DIR="$ROOT/.bifrost"
RUN_DIR="${RUN_DIR:-$ROOT/run}"
GATEWAY_PORT="${GATEWAY_PORT:-8080}"   # not $PORT: hosting platforms set PORT for the public demo page
BIFROST_BIN="${BIFROST_BIN:-$BIFROST_DIR/tmp/bifrost-http}"
PLUGIN_SO="${PLUGIN_SO:-$RUN_DIR/freshguard.so}"
# Caps on the demo virtual key, enforced by Bifrost governance (a public demo cannot drain your keys).
DEMO_REQ_PER_DAY="${DEMO_REQ_PER_DAY:-1000}"
DEMO_TOKENS_PER_DAY="${DEMO_TOKENS_PER_DAY:-2000000}"
ARCH="$(uname -m | sed 's/x86_64/amd64/')"   # force native arch: an Intel `go` on Apple Silicon defaults to amd64

build() {
  if [ ! -x "$BIFROST_BIN" ]; then
    [ -d "$BIFROST_DIR" ] || git clone --depth 1 --branch "$BIFROST_TAG" https://github.com/maximhq/bifrost "$BIFROST_DIR"
    (cd "$BIFROST_DIR" && make setup-workspace >/dev/null && make build DYNAMIC=1)
  fi
  # Must match Bifrost's own build flags exactly or plugin.Open rejects the .so.
  (cd "$ROOT/plugin" && GOWORK=off GOARCH="$ARCH" CGO_ENABLED=1 \
    go build -mod=mod -trimpath -tags sqlite_static -buildmode=plugin -o "$PLUGIN_SO" .)
  echo "built: $BIFROST_BIN + $PLUGIN_SO"
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
    "rate_limits": [{
      "id": "demo-cap",
      "request_max_limit": $DEMO_REQ_PER_DAY, "request_reset_duration": "1d",
      "token_max_limit": $DEMO_TOKENS_PER_DAY, "token_reset_duration": "1d"
    }],
    "virtual_keys": [{
      "id": "vk-freshguard-demo",
      "name": "freshguard-demo",
      "value": "env.FRESHGUARD_DEMO_VK",
      "is_active": true,
      "rate_limit_id": "demo-cap",
      "provider_configs": [{"provider": "gemini", "allowed_models": ["*"], "key_ids": ["*"], "weight": 1.0}],
      "relay_grants": [{"relay_name": "tavily"}]
    }]
  },
  "plugins": [{
    "enabled": true,
    "name": "freshguard",
    "path": "$PLUGIN_SO",
    "config": {
      "gateway_url": "http://localhost:$GATEWAY_PORT",
      "model": "gemini/gemini-3.5-flash-lite",
      "fallbacks": ["gemini/gemini-3.1-flash-lite", "gemini/gemini-3.8-flash"],
      "tavily_url": "http://localhost:$GATEWAY_PORT/relay/tavily",
      "virtual_keys": ["env.FRESHGUARD_DEMO_VK"],
      "max_claims": 5,
      "trigger": "header",
      "timeout_seconds": 30,
      "cache_ttl_minutes": 60
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

prepare() {
  mkdir -p "$RUN_DIR"
  [ -f "$PLUGIN_SO" ] || build
  config
  demo_vk
  set -a; [ -f "$ROOT/.env" ] && . "$ROOT/.env"; set +a
  : "${GEMINI_API_KEY:?set GEMINI_API_KEY (in .env or the environment)}"
  : "${TAVILY_API_KEY:?set TAVILY_API_KEY (in .env or the environment)}"
}

run() {
  prepare
  exec "$BIFROST_BIN" -app-dir "$RUN_DIR" -host 127.0.0.1 -port "$GATEWAY_PORT" -log-style pretty
}

# Gateway in the background (localhost only: its admin UI is never exposed), demo page in front.
up() {
  prepare
  "$BIFROST_BIN" -app-dir "$RUN_DIR" -host 127.0.0.1 -port "$GATEWAY_PORT" -log-style json &
  for _ in $(seq 1 60); do
    curl -sf "http://127.0.0.1:$GATEWAY_PORT/health" >/dev/null && break
    sleep 1
  done
  FG_GATEWAY="http://127.0.0.1:$GATEWAY_PORT" exec python3 "$ROOT/demo/serve.py"
}

case "${1:-run}" in
  build) mkdir -p "$RUN_DIR"; build ;;
  run) run ;;
  up) up ;;
  *) echo "usage: $0 [build|run|up]" >&2; exit 1 ;;
esac
