package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// Verdict values for a checked claim.
const (
	VerdictSupported    = "supported"
	VerdictChanged      = "changed"
	VerdictUnverifiable = "unverifiable"
	unverifiableWeight  = 0.5 // ponytail: flat weight, calibrate against the eval set in milestone 3
	maxEvidenceChars    = 600
	maxResultsPerSearch = 3
)

// Claim is a time-sensitive statement pulled out of an LLM answer.
type Claim struct {
	Claim string `json:"claim"`
	Query string `json:"query"`
}

// Evidence is one web result used to judge a claim.
type Evidence struct {
	Title         string `json:"title"`
	URL           string `json:"url"`
	Content       string `json:"content"`
	PublishedDate string `json:"published_date,omitempty"`
}

// ClaimResult is the per-claim outcome returned to the caller.
type ClaimResult struct {
	Claim        string   `json:"claim"`
	Verdict      string   `json:"verdict"`
	CurrentValue string   `json:"current_value,omitempty"`
	Explanation  string   `json:"explanation,omitempty"`
	Sources      []string `json:"sources,omitempty"`
}

// Report is attached to the chat response under the "freshguard" key.
type Report struct {
	Status    string        `json:"status"` // "checked" | "no_claims" | "error"
	Score     float64       `json:"score"`  // 0 (all stale) .. 1 (all current); -1 on error
	Stale     int           `json:"stale"`
	CheckedAt string        `json:"checked_at"`
	LatencyMS int64         `json:"latency_ms"`
	Timings   Timings       `json:"timings_ms"`
	Claims    []ClaimResult `json:"claims"`
	Error     string        `json:"error,omitempty"`
}

// Timings breaks the check's latency down by stage, in milliseconds.
type Timings struct {
	Extract int64 `json:"extract"`
	Search  int64 `json:"search"`
	Judge   int64 `json:"judge"`
}

// ChatFunc sends a system+user prompt to a model and returns the text reply.
type ChatFunc func(ctx context.Context, system, user string) (string, error)

// SearchFunc runs a web search and returns evidence.
type SearchFunc func(ctx context.Context, query string) ([]Evidence, error)

// Checker runs the extract -> search -> judge pipeline.
type Checker struct {
	Chat      ChatFunc
	Search    SearchFunc
	MaxClaims int
	Now       func() time.Time
}

const extractPrompt = `You find claims that may have gone out of date. Today is %s.
From the ANSWER below, list at most %d factual claims whose truth depends on time:
versions and releases, prices and plans, who holds a role, laws and regulations, rankings, statistics, "latest"/"current" anything.
Skip opinions, definitions, timeless facts, how-to steps, and dated historical events ("X happened in <year>"), which stay true.
Quote each claim faithfully, keeping its tense and dates; never restate a past event ("became", "was released in") as a present-tense fact.
For each claim write a short web search query that would reveal the CURRENT truth: name the subject plainly, include the current year, and do not repeat the claimed value (e.g. "latest stable Python release %s", not "Python 3.12 latest").
Reply with JSON only: {"claims":[{"claim":"...","query":"..."}]}. Use {"claims":[]} if none.`

const judgePrompt = `You verify whether claims are still current. Today is %s.
For each numbered claim, use ONLY the web evidence given under it.
verdict:
- "supported": evidence confirms the claim is still true today
- "changed": evidence shows a newer or different value; put it in current_value
- "unverifiable": evidence is missing, off-topic or conflicting
Judge the claim exactly as worded. A claim about a dated past event ("became CEO in 2024", "was released in 2025") is "supported" if that event happened, even if things changed later.
Cite the evidence URLs you relied on in sources. Never invent facts beyond the evidence.
Reply with JSON only: {"verdicts":[{"id":1,"verdict":"...","current_value":"...","explanation":"one sentence","sources":["url"]}]}`

// Check runs the full pipeline on one answer. Bad model output or failed calls
// come back as a Report with Status "error"; it never fails the user's response.
func (c *Checker) Check(ctx context.Context, answer string) (rep Report) {
	start := c.now()
	rep = Report{CheckedAt: start.UTC().Format(time.RFC3339), Claims: []ClaimResult{}}
	defer func() { rep.LatencyMS = time.Since(start).Milliseconds() }()

	stage := time.Now()
	claims, err := c.extract(ctx, answer)
	rep.Timings.Extract = time.Since(stage).Milliseconds()
	if err != nil {
		return errReport(rep, fmt.Errorf("extract: %w", err))
	}
	if len(claims) == 0 {
		rep.Status, rep.Score = "no_claims", 1
		return rep
	}

	stage = time.Now()
	evidence := c.searchAll(ctx, claims)
	rep.Timings.Search = time.Since(stage).Milliseconds()

	stage = time.Now()
	results, err := c.judge(ctx, claims, evidence)
	rep.Timings.Judge = time.Since(stage).Milliseconds()
	if err != nil {
		return errReport(rep, fmt.Errorf("judge: %w", err))
	}
	rep.Status = "checked"
	rep.Claims = results
	rep.Score, rep.Stale = score(results)
	return rep
}

func (c *Checker) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Checker) today() string { return c.now().Format("2006-01-02") }

func (c *Checker) extract(ctx context.Context, answer string) ([]Claim, error) {
	raw, err := c.Chat(ctx, fmt.Sprintf(extractPrompt, c.today(), c.MaxClaims, c.now().Format("2006")), "ANSWER:\n"+answer)
	if err != nil {
		return nil, err
	}
	var out struct {
		Claims []Claim `json:"claims"`
	}
	if err := decodeModelJSON(raw, &out); err != nil {
		return nil, err
	}
	var claims []Claim
	for _, cl := range out.Claims {
		if strings.TrimSpace(cl.Claim) == "" {
			continue
		}
		if strings.TrimSpace(cl.Query) == "" {
			cl.Query = cl.Claim
		}
		claims = append(claims, cl)
	}
	if len(claims) > c.MaxClaims {
		claims = claims[:c.MaxClaims]
	}
	return claims, nil
}

// searchAll searches every claim in parallel. A failed search yields no evidence
// for that claim (the judge then marks it unverifiable) rather than failing the check.
func (c *Checker) searchAll(ctx context.Context, claims []Claim) [][]Evidence {
	out := make([][]Evidence, len(claims))
	var wg sync.WaitGroup
	for i, cl := range claims {
		wg.Add(1)
		go func(i int, q string) {
			defer wg.Done()
			if ev, err := c.Search(ctx, q); err == nil {
				out[i] = ev
			}
		}(i, cl.Query)
	}
	wg.Wait()
	return out
}

func (c *Checker) judge(ctx context.Context, claims []Claim, evidence [][]Evidence) ([]ClaimResult, error) {
	var b strings.Builder
	for i, cl := range claims {
		fmt.Fprintf(&b, "CLAIM %d: %s\n", i+1, cl.Claim)
		if len(evidence[i]) == 0 {
			b.WriteString("  (no evidence found)\n")
		}
		for _, ev := range evidence[i] {
			fmt.Fprintf(&b, "  - [%s] %s %s\n    %s\n", ev.PublishedDate, ev.Title, ev.URL, truncate(ev.Content, maxEvidenceChars))
		}
		b.WriteString("\n")
	}
	raw, err := c.Chat(ctx, fmt.Sprintf(judgePrompt, c.today()), b.String())
	if err != nil {
		return nil, err
	}
	var out struct {
		Verdicts []struct {
			ID           int      `json:"id"`
			Verdict      string   `json:"verdict"`
			CurrentValue string   `json:"current_value"`
			Explanation  string   `json:"explanation"`
			Sources      []string `json:"sources"`
		} `json:"verdicts"`
	}
	if err := decodeModelJSON(raw, &out); err != nil {
		return nil, err
	}

	// Default every claim to unverifiable, then fill in what the judge returned by id.
	results := make([]ClaimResult, len(claims))
	for i, cl := range claims {
		results[i] = ClaimResult{Claim: cl.Claim, Verdict: VerdictUnverifiable}
	}
	for _, v := range out.Verdicts {
		i := v.ID - 1
		if i < 0 || i >= len(results) {
			continue
		}
		verdict := strings.ToLower(strings.TrimSpace(v.Verdict))
		if verdict != VerdictSupported && verdict != VerdictChanged {
			verdict = VerdictUnverifiable
		}
		results[i].Verdict = verdict
		results[i].CurrentValue = v.CurrentValue
		results[i].Explanation = v.Explanation
		results[i].Sources = onlyKnownURLs(v.Sources, evidence[i])
	}
	return results, nil
}

// score returns the freshness score (1 = everything current) and the count of changed claims.
func score(results []ClaimResult) (float64, int) {
	if len(results) == 0 {
		return 1, 0
	}
	var total float64
	stale := 0
	for _, r := range results {
		switch r.Verdict {
		case VerdictSupported:
			total++
		case VerdictUnverifiable:
			total += unverifiableWeight
		case VerdictChanged:
			stale++
		}
	}
	return math.Round(total/float64(len(results))*100) / 100, stale
}

// onlyKnownURLs drops any source the model cited that was not in the evidence, so
// the report never links to a hallucinated URL.
func onlyKnownURLs(cited []string, evidence []Evidence) []string {
	known := map[string]bool{}
	for _, ev := range evidence {
		known[ev.URL] = true
	}
	var out []string
	for _, u := range cited {
		if known[u] {
			out = append(out, u)
		}
	}
	return out
}

// decodeModelJSON parses JSON from a model reply, tolerating ```json fences and surrounding prose.
func decodeModelJSON(raw string, v any) error {
	s := strings.TrimSpace(raw)
	start, end := strings.Index(s, "{"), strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return fmt.Errorf("no JSON object in model reply: %q", truncate(s, 200))
	}
	return json.Unmarshal([]byte(s[start:end+1]), v)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func errReport(rep Report, err error) Report {
	rep.Status, rep.Score, rep.Error = "error", -1, err.Error()
	return rep
}
