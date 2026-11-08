# Upstream proposal: FreshGuard for Bifrost

Opened as [maximhq/bifrost#9748](https://github.com/maximhq/bifrost/issues/9748). It proposes FreshGuard as a community plugin and raises one small core change that would make it, and plugins like it, cleaner.

---

**Title:** Proposal: freshness-check plugin (re-verifies time-sensitive claims in answers) + plugin metadata on stream chunks

### Problem

LLMs answer from stale training data and state outdated facts confidently: versions, prices, who holds a role, laws. Teams running many apps behind one gateway have no single place to catch this; each app would need its own retrieval or fact-checking. The gateway already sees every answer.

### What FreshGuard does

A Go plugin using only public extension points:

- `HTTPTransportPostHook` (non-streaming) and `HTTPTransportStreamChunkHook` (streaming, checked on the final chunk).
- Extracts time-sensitive claims with a model call **back through Bifrost** (fallbacks, retries, governance and logs apply), searches each claim with Tavily through a **Bifrost relay** (the credential stays in the relay), judges claims against the evidence only, and attaches a per-claim report: `supported` / `changed` (+ current value) / `unverifiable`, with source URLs.
- All internal calls carry the caller's virtual key, so checks are governed and billed under the key that triggered them. The key is never sent to a non-gateway URL (tested).
- Operator control: per virtual key, for all traffic, or per request (`x-freshguard: on`). Fails open. In-memory claim cache. Official-site search via Tavily `include_domains`.
- Eval: 37 labelled answers (stale, current and timeless), run live through the gateway. Tests run with `-race` in CI.

Repo: https://github.com/sidharth-cs/freshguard

### Questions for maintainers

1. **Where should a plugin like this live?** In-tree under `plugins/`, or listed as a community plugin? Happy to restructure to match `plugins/*` conventions and add docs to the plugin guide.
2. **Plugin metadata on stream chunks.** A stream chunk has no free-form field for plugin output, so FreshGuard puts its report in `extra_fields.raw_response.freshguard` (moving any provider raw response to `raw_response.provider`). A small `plugin_metadata map[string]any` (or similar) on `BifrostResponseExtraFields` would give plugins a supported place for per-response output, for both buffered and streamed responses. Would you accept a PR for that?
3. **Shared cache.** The claim cache is per process. Is there a recommended way for a plugin to use Bifrost's cache store so replicas share it?

### Notes from building it

- Go `.so` plugins need the dynamic build (`make build DYNAMIC=1`) and an exact toolchain/flag/dependency match; the official images cannot load them. FreshGuard ships a Dockerfile that builds both together. A published plugin-capable base image would make third-party plugins much easier to try.
