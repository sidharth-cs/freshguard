# FreshGuard: Bifrost (plugin-capable build) + the FreshGuard plugin + the demo page.
#
#   docker compose up --build        then open http://localhost:8000
#
# Go plugins only load into a binary built by the same Go version with the same flags and
# dependency versions, so both are compiled here in one stage. Debian (glibc) because Go
# plugins need dlopen.

ARG BIFROST_VERSION=v2.4.0
ARG GO_IMAGE=golang:1.27.3-trixie

# 1. Gateway + plugin, same toolchain and flags. Bifrost's HTTP transport is fetched from
# the Go module proxy at the exact version the plugin's go.mod pins.
FROM ${GO_IMAGE} AS build
ARG BIFROST_VERSION
ENV CGO_ENABLED=1 GOWORK=off GOFLAGS=-mod=mod
RUN dir=$(GOFLAGS= go mod download -json github.com/maximhq/bifrost/transports/v2@${BIFROST_VERSION} | sed -n 's/.*"Dir": "\(.*\)",/\1/p') \
 && cp -r "$dir" /bifrost && chmod -R u+w /bifrost
WORKDIR /bifrost/bifrost-http
# The admin UI is never exposed from this image (the gateway listens on 127.0.0.1 only),
# so a placeholder page replaces Bifrost's Node.js UI build.
RUN mkdir -p ui && echo '<p>Bifrost UI is not bundled in the FreshGuard image.</p>' > ui/index.html \
 && go build -trimpath -tags sqlite_static -o /out/bifrost-http .
COPY plugin /src/plugin
WORKDIR /src/plugin
RUN go test ./... && go build -trimpath -tags sqlite_static -buildmode=plugin -o /out/freshguard.so .

# 2. Runtime.
FROM debian:trixie-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl openssl python3 \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -m -u 1000 app
WORKDIR /app
COPY --from=build /out/bifrost-http /app/bin/bifrost-http
COPY --from=build /out/freshguard.so /app/bin/freshguard.so
COPY dev.sh /app/dev.sh
COPY demo /app/demo
ENV BIFROST_BIN=/app/bin/bifrost-http \
    PLUGIN_SO=/app/bin/freshguard.so \
    RUN_DIR=/app/run \
    DEMO_HOST=0.0.0.0
RUN mkdir -p /app/run && chown -R app /app/run
USER app
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s CMD curl -sf http://127.0.0.1:8080/health || exit 1
CMD ["/app/dev.sh", "up"]
