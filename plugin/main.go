// FreshGuard is a Bifrost plugin that checks chat-completion answers for
// time-sensitive claims that may be out of date, re-verifies them with live
// Tavily search, and attaches a per-claim freshness report to the response.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/maximhq/bifrost/core/schemas"
)

const (
	pluginName     = "freshguard"
	headerOptIn    = "x-freshguard"          // request header: "on" enables the check in trigger=header mode
	headerInternal = "x-freshguard-internal" // marks FreshGuard's own model calls so they are not re-checked
	headerScore    = "X-FreshGuard-Score"
	headerStale    = "X-FreshGuard-Stale"
	headerVK       = "x-bf-vk" // Bifrost virtual key, forwarded so checks are billed to the caller's budget
)

// Config is the plugin's "config" block in Bifrost's config.json.
type Config struct {
	GatewayURL   string   `json:"gateway_url"`    // this Bifrost's own base URL; model calls go back through it
	Model        string   `json:"model"`          // model used for claim extraction and judging
	Fallbacks    []string `json:"fallbacks"`      // Bifrost fallbacks for that model
	TavilyURL    string   `json:"tavily_url"`     // Tavily API base, or a Bifrost relay URL for Tavily
	TavilyAPIKey string   `json:"tavily_api_key"` // literal or "env.NAME"; leave empty when a relay injects it
	SearchDepth  string   `json:"search_depth"`   // Tavily depth: "basic" (1 credit) or "advanced" (2 credits, better snippets)
	TimeRange    string   `json:"time_range"`     // Tavily recency filter: "day"|"week"|"month"|"year", "" = none
	MaxClaims    int      `json:"max_claims"`
	Trigger      string   `json:"trigger"`      // "header" (opt-in per request) or "always"
	VirtualKeys  []string `json:"virtual_keys"` // Bifrost virtual keys (literal or "env.NAME") whose requests are always checked
	TimeoutSec   int      `json:"timeout_seconds"`
}

func defaultConfig() Config {
	return Config{
		GatewayURL:  "http://localhost:8080",
		Model:       "gemini/gemini-3.5-flash-lite",
		TavilyURL:   "https://api.tavily.com",
		SearchDepth: "basic",
		TimeRange:   "year",
		MaxClaims:   5,
		Trigger:     "header",
		TimeoutSec:  30,
	}
}

var (
	cfg     = defaultConfig()
	checker *Checker
	client  = &http.Client{}
)

// Init is called once by Bifrost with the plugin's config block.
func Init(config any) error {
	c := defaultConfig()
	if config != nil {
		raw, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("freshguard: config: %w", err)
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			return fmt.Errorf("freshguard: config: %w", err)
		}
	}
	c.TavilyAPIKey = resolveEnv(c.TavilyAPIKey)
	for i, vk := range c.VirtualKeys {
		c.VirtualKeys[i] = resolveEnv(vk)
	}
	if c.Trigger != "header" && c.Trigger != "always" {
		return fmt.Errorf("freshguard: trigger must be \"header\" or \"always\", got %q", c.Trigger)
	}
	if c.MaxClaims < 1 {
		c.MaxClaims = 1
	}
	if c.TimeoutSec < 1 {
		c.TimeoutSec = 30
	}
	cfg = c
	client = &http.Client{Timeout: time.Duration(c.TimeoutSec) * time.Second}
	checker = &Checker{MaxClaims: c.MaxClaims}
	return nil
}

// GetName returns the plugin name shown in Bifrost.
func GetName() string { return pluginName }

// Cleanup is called on Bifrost shutdown.
func Cleanup() error { return nil }

// HTTPTransportPostHook runs after Bifrost has produced a response. For eligible
// chat completions it adds a "freshguard" report to the JSON body and score headers.
// It fails open: any problem leaves the original response untouched or adds an error report.
func HTTPTransportPostHook(ctx *schemas.BifrostContext, req *schemas.HTTPRequest, resp *schemas.HTTPResponse) error {
	answer, body, ok := eligible(req, resp)
	if !ok {
		return nil
	}

	runCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()
	// Forward the caller's virtual key so FreshGuard's own model and search calls are
	// governed and billed under the same key as the request that triggered them.
	vk := requestVK(req)
	c := *checker
	c.Chat = gatewayChat(vk)
	c.Search = tavilySearch(vk)
	report := c.Check(runCtx, answer)

	attachReport(resp, body, report)
	return nil
}

// attachReport writes the report into the response body and score headers.
func attachReport(resp *schemas.HTTPResponse, body map[string]any, report Report) {
	body["freshguard"] = report
	out, err := json.Marshal(body)
	if err != nil {
		return
	}
	resp.Body = out
	if resp.Headers == nil {
		resp.Headers = map[string]string{}
	}
	resp.Headers[headerScore] = strconv.FormatFloat(report.Score, 'f', 2, 64)
	resp.Headers[headerStale] = strconv.Itoa(report.Stale)
}

// eligible decides whether a response should be checked and returns its answer text and parsed body.
func eligible(req *schemas.HTTPRequest, resp *schemas.HTTPResponse) (string, map[string]any, bool) {
	if req == nil || resp == nil || checker == nil {
		return "", nil, false
	}
	if req.CaseInsensitiveHeaderLookup(headerInternal) != "" {
		return "", nil, false
	}
	if !strings.HasSuffix(req.Path, "/chat/completions") || resp.StatusCode != http.StatusOK || len(resp.Body) == 0 {
		return "", nil, false
	}
	if !wantsCheck(req) {
		return "", nil, false
	}
	var reqBody struct {
		Stream bool `json:"stream"`
	}
	if json.Unmarshal(req.Body, &reqBody) == nil && reqBody.Stream {
		return "", nil, false // ponytail: streaming unsupported in MVP
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return "", nil, false
	}
	answer := answerText(body)
	if strings.TrimSpace(answer) == "" {
		return "", nil, false
	}
	return answer, body, true
}

// answerText returns the first choice's message text, whether content is a string or text blocks.
func answerText(body map[string]any) string {
	choices, _ := body["choices"].([]any)
	if len(choices) == 0 {
		return ""
	}
	choice, _ := choices[0].(map[string]any)
	msg, _ := choice["message"].(map[string]any)
	switch content := msg["content"].(type) {
	case string:
		return content
	case []any:
		var parts []string
		for _, blk := range content {
			if m, ok := blk.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

// gatewayChat returns a ChatFunc that calls a model back through this Bifrost, so
// FreshGuard's own model usage gets Bifrost's routing, fallbacks, budgets and logs.
func gatewayChat(virtualKey string) ChatFunc {
	return func(ctx context.Context, system, user string) (string, error) {
		payload := map[string]any{
			"model":           cfg.Model,
			"temperature":     0,
			"response_format": map[string]string{"type": "json_object"}, // both prompts expect JSON; provider-enforced
			"messages": []map[string]string{
				{"role": "system", "content": system},
				{"role": "user", "content": user},
			},
		}
		if len(cfg.Fallbacks) > 0 {
			payload["fallbacks"] = cfg.Fallbacks
		}
		headers := map[string]string{headerInternal: "1"}
		if virtualKey != "" {
			headers[headerVK] = virtualKey
		}
		var out map[string]any
		if err := postJSON(ctx, strings.TrimRight(cfg.GatewayURL, "/")+"/v1/chat/completions", headers, payload, &out); err != nil {
			return "", err
		}
		// Content may be a string or a list of text blocks depending on provider.
		if text := answerText(out); text != "" {
			return text, nil
		}
		return "", fmt.Errorf("model returned no text")
	}
}

// tavilySearch returns a SearchFunc for Tavily's /search endpoint, called directly or
// through a Bifrost relay (tavily_url = "<gateway_url>/relay/<name>").
func tavilySearch(virtualKey string) SearchFunc {
	return func(ctx context.Context, query string) ([]Evidence, error) {
		headers := map[string]string{}
		if cfg.TavilyAPIKey != "" {
			headers["Authorization"] = "Bearer " + cfg.TavilyAPIKey
		}
		// The virtual key is a Bifrost credential: only send it to our own gateway, never to a third party.
		if virtualKey != "" && viaGateway(cfg.TavilyURL) {
			headers[headerVK] = virtualKey
		}
		payload := map[string]any{"query": query, "max_results": maxResultsPerSearch, "search_depth": cfg.SearchDepth}
		if cfg.TimeRange != "" {
			payload["time_range"] = cfg.TimeRange
		}
		var out struct {
			Results []Evidence `json:"results"`
		}
		if err := postJSON(ctx, strings.TrimRight(cfg.TavilyURL, "/")+"/search", headers, payload, &out); err != nil {
			return nil, err
		}
		return out.Results, nil
	}
}

// viaGateway reports whether url points at this Bifrost instance.
func viaGateway(url string) bool {
	base := strings.TrimRight(cfg.GatewayURL, "/")
	return url == base || strings.HasPrefix(url, base+"/")
}

// requestVK returns the Bifrost virtual key on a request: x-bf-vk, or an sk-bf- bearer token.
func requestVK(req *schemas.HTTPRequest) string {
	if vk := req.CaseInsensitiveHeaderLookup(headerVK); vk != "" {
		return vk
	}
	if tok, ok := strings.CutPrefix(req.CaseInsensitiveHeaderLookup("authorization"), "Bearer "); ok && strings.HasPrefix(tok, "sk-bf-") {
		return tok
	}
	return ""
}

// wantsCheck applies the operator's policy: always, per virtual key, or per-request header opt-in.
func wantsCheck(req *schemas.HTTPRequest) bool {
	if cfg.Trigger == "always" {
		return true
	}
	if vk := requestVK(req); vk != "" {
		for _, enabled := range cfg.VirtualKeys {
			if enabled != "" && vk == enabled {
				return true
			}
		}
	}
	return isOn(req.CaseInsensitiveHeaderLookup(headerOptIn))
}

func postJSON(ctx context.Context, url string, headers map[string]string, payload, out any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	res, err := client.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d: %s", url, res.StatusCode, truncate(string(body), 300))
	}
	return json.Unmarshal(body, out)
}

func resolveEnv(v string) string {
	if name, ok := strings.CutPrefix(v, "env."); ok {
		return os.Getenv(name)
	}
	return v
}

func isOn(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "on", "1", "true", "yes":
		return true
	}
	return false
}
