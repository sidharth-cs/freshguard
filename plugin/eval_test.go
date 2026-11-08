//go:build eval

// Live accuracy/latency eval against a running gateway (./dev.sh run).
//
//	GOWORK=off go test -mod=mod -tags eval -run TestEval -v -timeout 30m .
//
// Env: FG_GATEWAY (default http://localhost:8080), FG_DEPTH (advanced|basic), FG_TIME_RANGE (default year).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type evalCase struct {
	ID     string `json:"id"`
	Expect string `json:"expect"` // stale | current | timeless
	Answer string `json:"answer"`
}

type evalRow struct {
	evalCase
	Report  Report
	Flagged bool // any claim judged "changed"
	Pass    bool
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func TestEval(t *testing.T) {
	raw, err := os.ReadFile("../eval/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []evalCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if only := os.Getenv("FG_ONLY"); only != "" { // subset: FG_ONLY=c06,s14
		var subset []evalCase
		for _, ec := range cases {
			if strings.Contains(","+only+",", ","+ec.ID+",") {
				subset = append(subset, ec)
			}
		}
		cases = subset
	}
	vkRaw, err := os.ReadFile("../run/demo_vk")
	if err != nil {
		t.Fatal("start the gateway with ./dev.sh run first:", err)
	}
	vk := strings.TrimSpace(string(vkRaw))

	cfg = defaultConfig()
	cfg.GatewayURL = envOr("FG_GATEWAY", "http://localhost:8080")
	cfg.TavilyURL = cfg.GatewayURL + "/relay/tavily"
	cfg.SearchDepth = envOr("FG_DEPTH", cfg.SearchDepth)
	cfg.TimeRange = envOr("FG_TIME_RANGE", cfg.TimeRange)
	cfg.Fallbacks = []string{"gemini/gemini-3.1-flash-lite", "gemini/gemini-3.8-flash"}
	client = &http.Client{Timeout: 60 * time.Second}
	c := Checker{MaxClaims: cfg.MaxClaims, Chat: gatewayChat(vk), Search: tavilySearch(vk)}

	rows := make([]evalRow, len(cases))
	sem := make(chan struct{}, 3) // ponytail: low concurrency to stay under Gemini free-tier rate limits
	var wg sync.WaitGroup
	for i, ec := range cases {
		wg.Add(1)
		go func(i int, ec evalCase) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			rows[i] = grade(ec, c.Check(ctx, ec.Answer))
		}(i, ec)
	}
	wg.Wait()

	if len(rows) == 1 {
		dump, _ := json.MarshalIndent(rows[0].Report, "", "  ")
		t.Log(string(dump))
		return
	}
	md := summarize(rows, cfg.SearchDepth, cfg.TimeRange)
	out := fmt.Sprintf("../eval/results-%s.md", cfg.SearchDepth)
	if err := os.WriteFile(out, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Log("\n" + md)
}

func grade(ec evalCase, rep Report) evalRow {
	r := evalRow{evalCase: ec, Report: rep}
	for _, cl := range rep.Claims {
		if cl.Verdict == VerdictChanged {
			r.Flagged = true
		}
	}
	if ec.Expect == "stale" {
		r.Pass = r.Flagged
	} else { // current, timeless: must not be flagged, and the check must not have errored
		r.Pass = !r.Flagged && rep.Status != "error"
	}
	return r
}

func summarize(rows []evalRow, depth, timeRange string) string {
	count := map[string]int{}
	pass := map[string]int{}
	var lat, ext, srch, jdg []int64
	errs, unverifiable := 0, 0
	for _, r := range rows {
		count[r.Expect]++
		if r.Pass {
			pass[r.Expect]++
		}
		if r.Report.Status == "error" {
			errs++
			continue
		}
		lat = append(lat, r.Report.LatencyMS)
		ext = append(ext, r.Report.Timings.Extract)
		if r.Report.Status == "checked" {
			srch = append(srch, r.Report.Timings.Search)
			jdg = append(jdg, r.Report.Timings.Judge)
		}
		for _, cl := range r.Report.Claims {
			if cl.Verdict == VerdictUnverifiable {
				unverifiable++
			}
		}
	}
	notStale := count["current"] + count["timeless"]
	falseChanged := notStale - pass["current"] - pass["timeless"]

	var b strings.Builder
	fmt.Fprintf(&b, "# FreshGuard eval — search_depth=%s, time_range=%s\n\n", depth, timeRange)
	fmt.Fprintf(&b, "Run: %s · %d cases · model %s\n\n", time.Now().Format("2006-01-02 15:04"), len(rows), cfg.Model)
	b.WriteString("| Metric | Result | Target |\n|---|---|---|\n")
	fmt.Fprintf(&b, "| Stale-claim recall | **%d/%d (%.0f%%)** | ≥ 70%% |\n", pass["stale"], count["stale"], pct(pass["stale"], count["stale"]))
	fmt.Fprintf(&b, "| Wrong on current + timeless (flagged or errored) | **%d/%d (%.0f%%)** | ≤ 15%% |\n", falseChanged, notStale, pct(falseChanged, notStale))
	fmt.Fprintf(&b, "| Checks that errored | %d/%d | 0 |\n", errs, len(rows))
	fmt.Fprintf(&b, "| Claims judged unverifiable | %d | — |\n", unverifiable)
	fmt.Fprintf(&b, "| Latency p50 / p90 (all checks) | %d / %d ms | < 3000 ms p50 |\n", pctl(lat, 50), pctl(lat, 90))
	fmt.Fprintf(&b, "| Stage p50: extract / search / judge | %d / %d / %d ms | — |\n\n", pctl(ext, 50), pctl(srch, 50), pctl(jdg, 50))

	b.WriteString("| Case | Expect | Pass | Status | Verdicts | Current value | ms |\n|---|---|---|---|---|---|---|\n")
	for _, r := range rows {
		var verdicts, current []string
		for _, cl := range r.Report.Claims {
			verdicts = append(verdicts, cl.Verdict)
			if cl.CurrentValue != "" {
				current = append(current, cl.CurrentValue)
			}
		}
		mark := "✅"
		if !r.Pass {
			mark = "❌"
		}
		status := r.Report.Status
		if r.Report.Error != "" {
			status += ": " + truncate(r.Report.Error, 60)
		}
		fmt.Fprintf(&b, "| %s %s | %s | %s | %s | %s | %s | %d |\n", r.ID, truncate(r.Answer, 50), r.Expect, mark, status,
			strings.Join(verdicts, ", "), truncate(strings.Join(current, "; "), 60), r.Report.LatencyMS)
	}
	return b.String()
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func pctl(v []int64, p int) int64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]int64(nil), v...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[(len(s)-1)*p/100]
}
