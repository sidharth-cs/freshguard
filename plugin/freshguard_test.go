package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maximhq/bifrost/core/schemas"
)

// fakeChat answers the extract prompt and the judge prompt with canned replies.
func fakeChat(extract, judge string) ChatFunc {
	return func(_ context.Context, system, _ string) (string, error) {
		if strings.Contains(system, "find claims") {
			return extract, nil
		}
		return judge, nil
	}
}

func fakeSearch(_ context.Context, q string, _ []string) ([]Evidence, error) {
	return []Evidence{{Title: "Python releases", URL: "https://python.org/downloads", Content: "Python 3.15.0 is the latest stable release.", PublishedDate: "2026-10-01"}}, nil
}

func newChecker(chat ChatFunc) *Checker {
	return &Checker{Chat: chat, Search: fakeSearch, MaxClaims: 5, Now: func() time.Time { return time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC) }}
}

func TestCheckFlagsStaleClaimAndDropsHallucinatedSources(t *testing.T) {
	c := newChecker(fakeChat(
		"```json\n{\"claims\":[{\"claim\":\"Latest stable Python is 3.13.2\",\"query\":\"latest stable python version\"}]}\n```",
		`{"verdicts":[{"id":1,"verdict":"Changed","current_value":"3.15.0","explanation":"newer release","sources":["https://python.org/downloads","https://made-up.example"]}]}`,
	))
	rep := c.Check(context.Background(), "The latest stable version is Python 3.13.2.")

	if rep.Status != "checked" || rep.Score != 0 || rep.Stale != 1 {
		t.Fatalf("got status=%s score=%v stale=%d", rep.Status, rep.Score, rep.Stale)
	}
	got := rep.Claims[0]
	if got.Verdict != VerdictChanged || got.CurrentValue != "3.15.0" {
		t.Fatalf("claim = %+v", got)
	}
	if len(got.Sources) != 1 || got.Sources[0] != "https://python.org/downloads" {
		t.Fatalf("sources not filtered to evidence: %v", got.Sources)
	}
}

func TestCheckNoClaimsScoresFresh(t *testing.T) {
	rep := newChecker(fakeChat(`{"claims":[]}`, "")).Check(context.Background(), "Water boils at 100C at sea level.")
	if rep.Status != "no_claims" || rep.Score != 1 {
		t.Fatalf("got %+v", rep)
	}
}

func TestCheckBadModelOutputFailsOpen(t *testing.T) {
	rep := newChecker(fakeChat("sorry, I can't do that", "")).Check(context.Background(), "x")
	if rep.Status != "error" || rep.Score != -1 || rep.Error == "" {
		t.Fatalf("got %+v", rep)
	}
}

func TestCheckChatErrorFailsOpen(t *testing.T) {
	c := newChecker(func(context.Context, string, string) (string, error) { return "", errors.New("503") })
	if rep := c.Check(context.Background(), "x"); rep.Status != "error" {
		t.Fatalf("got %+v", rep)
	}
}

func TestJudgeDefaultsMissingAndUnknownVerdictsToUnverifiable(t *testing.T) {
	c := newChecker(fakeChat(
		`{"claims":[{"claim":"a","query":"a"},{"claim":"b","query":"b"},{"claim":"c"}]}`,
		`{"verdicts":[{"id":1,"verdict":"supported"},{"id":2,"verdict":"probably"},{"id":9,"verdict":"changed"}]}`,
	))
	rep := c.Check(context.Background(), "x")
	want := []string{VerdictSupported, VerdictUnverifiable, VerdictUnverifiable}
	for i, w := range want {
		if rep.Claims[i].Verdict != w {
			t.Fatalf("claim %d verdict = %s, want %s", i, rep.Claims[i].Verdict, w)
		}
	}
	if rep.Score != 0.67 { // (1 + 0.5 + 0.5) / 3
		t.Fatalf("score = %v", rep.Score)
	}
}

func TestExtractCapsClaims(t *testing.T) {
	c := newChecker(fakeChat(`{"claims":[{"claim":"a"},{"claim":"b"},{"claim":"c"}]}`, `{"verdicts":[]}`))
	c.MaxClaims = 2
	if rep := c.Check(context.Background(), "x"); len(rep.Claims) != 2 {
		t.Fatalf("got %d claims", len(rep.Claims))
	}
}

func TestEligible(t *testing.T) {
	checker = &Checker{}
	cfg = defaultConfig()
	okBody := []byte(`{"choices":[{"message":{"role":"assistant","content":"Python 3.13 is latest."}}]}`)
	req := func(headers map[string]string, body string) *schemas.HTTPRequest {
		return &schemas.HTTPRequest{Path: "/v1/chat/completions", Headers: headers, Body: []byte(body)}
	}
	resp := func() *schemas.HTTPResponse { return &schemas.HTTPResponse{StatusCode: 200, Body: okBody} }

	cases := []struct {
		name string
		req  *schemas.HTTPRequest
		resp *schemas.HTTPResponse
		want bool
	}{
		{"opted in", req(map[string]string{"X-FreshGuard": "on"}, `{}`), resp(), true},
		{"no opt-in header", req(nil, `{}`), resp(), false},
		{"internal call", req(map[string]string{"x-freshguard": "on", "x-freshguard-internal": "1"}, `{}`), resp(), false},
		{"streaming", req(map[string]string{"x-freshguard": "on"}, `{"stream":true}`), resp(), false},
		{"error status", req(map[string]string{"x-freshguard": "on"}, `{}`), &schemas.HTTPResponse{StatusCode: 500, Body: okBody}, false},
		{"other endpoint", &schemas.HTTPRequest{Path: "/v1/embeddings", Headers: map[string]string{"x-freshguard": "on"}}, resp(), false},
	}
	for _, tc := range cases {
		if _, _, got := eligible(tc.req, tc.resp); got != tc.want {
			t.Errorf("%s: eligible = %v, want %v", tc.name, got, tc.want)
		}
	}

	cfg.VirtualKeys = []string{"sk-bf-enabled"}
	if _, _, got := eligible(req(map[string]string{"x-bf-vk": "sk-bf-enabled"}, `{}`), resp()); !got {
		t.Error("enabled virtual key should be checked without opt-in header")
	}
	if _, _, got := eligible(req(map[string]string{"Authorization": "Bearer sk-bf-enabled"}, `{}`), resp()); !got {
		t.Error("enabled virtual key as bearer token should be checked")
	}
	if _, _, got := eligible(req(map[string]string{"x-bf-vk": "sk-bf-other"}, `{}`), resp()); got {
		t.Error("other virtual key must not be checked without opt-in header")
	}

	cfg.Trigger = "always"
	if _, _, got := eligible(req(nil, `{}`), resp()); !got {
		t.Error("trigger=always should check without opt-in header")
	}
}

func TestTavilySearchSendsVirtualKeyOnlyToGateway(t *testing.T) {
	var gotVK, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotVK, gotAuth = r.Header.Get("x-bf-vk"), r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"results":[{"title":"t","url":"https://u","content":"c"}]}`))
	}))
	defer srv.Close()
	client = srv.Client()

	// Relay through our own gateway: virtual key forwarded, no Tavily key needed.
	cfg = defaultConfig()
	cfg.GatewayURL, cfg.TavilyURL = srv.URL, srv.URL+"/relay/tavily"
	ev, err := tavilySearch("sk-bf-caller")(context.Background(), "q", nil)
	if err != nil || len(ev) != 1 || gotVK != "sk-bf-caller" || gotAuth != "" {
		t.Fatalf("relay: err=%v ev=%v vk=%q auth=%q", err, ev, gotVK, gotAuth)
	}

	// Direct to a third party: virtual key must not leak, Tavily key is used instead.
	cfg.GatewayURL, cfg.TavilyURL, cfg.TavilyAPIKey = "http://localhost:1", srv.URL, "tvly-test"
	if _, err := tavilySearch("sk-bf-caller")(context.Background(), "q", nil); err != nil {
		t.Fatal(err)
	}
	if gotVK != "" || gotAuth != "Bearer tvly-test" {
		t.Fatalf("direct: vk leaked=%q auth=%q", gotVK, gotAuth)
	}
}

func TestAnswerTextHandlesContentBlocks(t *testing.T) {
	var body map[string]any
	_ = json.Unmarshal([]byte(`{"choices":[{"message":{"content":[{"type":"text","text":"a"},{"type":"text","text":"b"}]}}]}`), &body)
	if got := answerText(body); got != "a\nb" {
		t.Fatalf("got %q", got)
	}
}

func TestAttachReportSetsBodyAndHeaders(t *testing.T) {
	resp := &schemas.HTTPResponse{StatusCode: 200}
	attachReport(resp, map[string]any{"id": "x"}, Report{Status: "checked", Score: 0.5, Stale: 1, Claims: []ClaimResult{}})
	if resp.Headers[headerScore] != "0.50" || resp.Headers[headerStale] != "1" {
		t.Fatalf("headers = %v", resp.Headers)
	}
	var out map[string]any
	if err := json.Unmarshal(resp.Body, &out); err != nil || out["freshguard"] == nil || out["id"] != "x" {
		t.Fatalf("body = %s", resp.Body)
	}
}

func TestInitRejectsBadTriggerAndResolvesEnv(t *testing.T) {
	if err := Init(map[string]any{"trigger": "sometimes"}); err == nil {
		t.Fatal("expected error for bad trigger")
	}
	t.Setenv("FG_TEST_KEY", "secret")
	if err := Init(map[string]any{"tavily_api_key": "env.FG_TEST_KEY", "max_claims": 0}); err != nil {
		t.Fatal(err)
	}
	if cfg.TavilyAPIKey != "secret" || cfg.MaxClaims != 1 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestCacheSkipsSearchAndJudgeForRecentlyVerifiedClaims(t *testing.T) {
	var judged, searched atomic.Int32
	c := newChecker(func(_ context.Context, system, user string) (string, error) {
		if strings.Contains(system, "find claims") {
			return `{"claims":[{"claim":"Latest Python is 3.13","query":"q"},{"claim":"Mars has two moons","query":"q2"}]}`, nil
		}
		judged.Add(1)
		if strings.Contains(user, "Latest Python") {
			return `{"verdicts":[{"id":1,"verdict":"changed","current_value":"3.15"},{"id":2,"verdict":"unverifiable"}]}`, nil
		}
		return `{"verdicts":[{"id":1,"verdict":"unverifiable"}]}`, nil // only the uncached claim is re-judged
	})
	c.Search = func(ctx context.Context, q string, d []string) ([]Evidence, error) {
		searched.Add(1)
		return fakeSearch(ctx, q, d)
	}
	c.Cache = NewClaimCache(time.Hour)

	first := c.Check(context.Background(), "x")
	second := c.Check(context.Background(), "x")

	if first.Cached != 0 || second.Cached != 1 {
		t.Fatalf("cached: first=%d second=%d", first.Cached, second.Cached)
	}
	if searched.Load() != 3 || judged.Load() != 2 { // 2 searches, then only the unverifiable claim again
		t.Fatalf("searched=%d judged=%d", searched.Load(), judged.Load())
	}
	if second.Claims[0].Verdict != VerdictChanged || second.Claims[0].CurrentValue != "3.15" || second.Stale != 1 {
		t.Fatalf("cached result not reused in order: %+v", second.Claims)
	}
}

func TestCacheExpires(t *testing.T) {
	now := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	cc := NewClaimCache(time.Hour)
	cc.now = func() time.Time { return now }
	cc.Put("  The Claim ", ClaimResult{Verdict: VerdictSupported})
	if _, ok := cc.Get("the claim"); !ok {
		t.Fatal("expected hit (normalized key)")
	}
	now = now.Add(61 * time.Minute)
	if _, ok := cc.Get("the claim"); ok {
		t.Fatal("expected expiry")
	}
	cc.Put("x", ClaimResult{Verdict: VerdictUnverifiable})
	if _, ok := cc.Get("x"); ok {
		t.Fatal("unverifiable results must not be cached")
	}
}

func TestOfficialSiteIsSearchedAndRankedFirst(t *testing.T) {
	c := newChecker(fakeChat(`{"claims":[{"claim":"c","query":"q","official_site":"python.org"}]}`, `{"verdicts":[]}`))
	var calls [][]string
	var mu sync.Mutex
	c.Search = func(_ context.Context, q string, domains []string) ([]Evidence, error) {
		mu.Lock()
		calls = append(calls, domains)
		mu.Unlock()
		if len(domains) > 0 {
			return []Evidence{{URL: "https://python.org/dl"}, {URL: "https://blog.example/a"}}, nil
		}
		return []Evidence{{URL: "https://blog.example/a"}, {URL: "https://news.example/b"}}, nil
	}
	ev := c.searchAll(context.Background(), []Claim{{Claim: "c", Query: "q", OfficialSite: "python.org"}})
	if len(calls) != 2 {
		t.Fatalf("want general + official search, got %v", calls)
	}
	got := []string{}
	for _, e := range ev[0] {
		got = append(got, e.URL)
	}
	if strings.Join(got, ",") != "https://python.org/dl,https://blog.example/a,https://news.example/b" {
		t.Fatalf("evidence order/dedupe = %v", got)
	}
	if !ev[0][0].Official {
		t.Fatal("official result not marked")
	}
}

func streamChunk(content string, finish *string) *schemas.BifrostStreamChunk {
	return &schemas.BifrostStreamChunk{BifrostChatResponse: &schemas.BifrostChatResponse{
		Choices: []schemas.BifrostResponseChoice{{
			FinishReason:             finish,
			ChatStreamResponseChoice: &schemas.ChatStreamResponseChoice{Delta: &schemas.ChatStreamResponseChoiceDelta{Content: &content}},
		}},
	}}
}

func TestStreamChunkHookChecksFullAnswerOnLastChunk(t *testing.T) {
	cfg = defaultConfig()
	checker = &Checker{MaxClaims: 5}
	var seen string
	newChat = func(string) ChatFunc {
		return func(_ context.Context, system, user string) (string, error) {
			if strings.Contains(system, "find claims") {
				seen = user
				return `{"claims":[{"claim":"Latest Python is 3.13","query":"q"}]}`, nil
			}
			return `{"verdicts":[{"id":1,"verdict":"changed","current_value":"3.15"}]}`, nil
		}
	}
	newSearch = func(string) SearchFunc { return fakeSearch }
	defer func() { newChat, newSearch = gatewayChat, tavilySearch }()

	ctx := schemas.NewBifrostContext(context.Background(), time.Time{})
	req := &schemas.HTTPRequest{Path: "/v1/chat/completions", Headers: map[string]string{"x-freshguard": "on"}, Body: []byte(`{"stream":true}`)}
	stop := "stop"
	for _, part := range []string{"Latest Python ", "is 3.13."} {
		out, err := HTTPTransportStreamChunkHook(ctx, req, streamChunk(part, nil))
		if err != nil || out.BifrostChatResponse.ExtraFields.RawResponse != nil {
			t.Fatalf("mid-stream chunk modified: %v", err)
		}
	}
	last, err := HTTPTransportStreamChunkHook(ctx, req, streamChunk("", &stop))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(seen, "Latest Python is 3.13.") {
		t.Fatalf("extractor saw %q", seen)
	}
	raw, _ := last.BifrostChatResponse.ExtraFields.RawResponse.(map[string]any)
	rep, ok := raw["freshguard"].(Report)
	if !ok || rep.Stale != 1 {
		t.Fatalf("last chunk report = %#v", last.BifrostChatResponse.ExtraFields.RawResponse)
	}

	// Not opted in: chunks pass through untouched.
	plain := &schemas.HTTPRequest{Path: "/v1/chat/completions", Body: []byte(`{"stream":true}`)}
	ctx2 := schemas.NewBifrostContext(context.Background(), time.Time{})
	out, _ := HTTPTransportStreamChunkHook(ctx2, plain, streamChunk("hi", &stop))
	if out.BifrostChatResponse.ExtraFields.RawResponse != nil {
		t.Fatal("unchecked stream was modified")
	}
}
