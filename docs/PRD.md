# FreshGuard — freshness checks at the AI gateway

Hackathon: Geek Room Global AI Hackathon · Primary PS #4 (Bifrost gateway feature) + PS #5 (stale-answer detector, Tavily) · Deadline 2026-11-15 23:59 · Solo.

## Problem
Teams running LLM apps behind a Bifrost gateway ship answers that confidently state outdated facts — old pricing, deprecated API versions, changed laws, former executives. Nothing in the request path flags these before an end user reads them. Cost: wrong customer-facing answers, lost trust, support load.

## Evidence
- Assumption — needs validation via prototype on a labelled stale-claim set and feedback from Bifrost maintainers (upstream PR review).
- Hackathon PS #5 names this exact problem (sponsor-stated, not user-measured).

## Users
- **Primary**: Gateway operator / platform engineer running Bifrost for one or more LLM apps with customer-facing answers (support bot, docs assistant, sales copilot). Trigger: wants a single place to add answer-quality controls without touching each app.
- **Not for**: End users fact-checking answers manually; teams not using Bifrost; general hallucination detection on non-time-sensitive claims.

## Hypothesis
We believe **a gateway-level freshness check on LLM responses** will **reduce outdated claims reaching users without per-app changes** for **Bifrost operators**.
We'll know we're right when, **on a labelled set of ~30 answers containing known stale claims, FreshGuard flags ≥70% of stale claims with ≤15% false "changed" flags, adding <3s per checked answer.**

## Success Metrics
| Metric | Target | How measured |
|---|---|---|
| Stale-claim recall | ≥ 70% | Labelled eval set (~30 answers) |
| False "changed" rate | ≤ 15% | Same eval set |
| Added latency per checked answer | < 3 s (p50) | Timed eval runs |
| Setup on stock Bifrost | ≤ 10 min following README | Fresh-machine dry run |
| Tavily credits per checked answer | ≤ 10 | Credit counter over eval run |

## Scope
**MVP**
- Non-streaming chat-completion responses only.
- Extract up to N (default 5) time-sensitive claims per answer.
- Re-verify each claim with live Tavily search.
- Per-claim verdict: supported / changed / unverifiable, with source links; overall freshness score exposed to the caller.
- Opt-in per virtual key (or equivalent operator control).
- Tests, setup docs, PR-ready repo.
- Small before/after demo for judges.

**Out of scope**
- Streaming responses — chunked verification is a separate design problem.
- Blocking or auto-rewriting answers — annotate only; enforcement is a later mode.
- Verdict caching — optimization after correctness is proven.
- Non-time-sensitive hallucination detection — different problem.

## Delivery Milestones
| # | Milestone | Outcome | Status | Plan |
|---|---|---|---|---|
| 0 | Feasibility spike | Custom plugin loads on a stock Bifrost and sees a chat response | complete | SPIKE.md |
| 1 | Freshness verdicts | A chat call through Bifrost returns per-claim verdicts + score | complete | plugin/, dev.sh, demo-sample-response.json |
| 2 | Operator controls + tests | Opt-in per key, configurable limits, tests green, docs | complete | README.md |
| 3 | Eval + demo | Metrics table from labelled set; before/after demo page | complete | eval/, demo/ |
| 4 | Submission | Public repo, README, pitch deck, demo video, HackCulture submission (target Nov 13) | pending | — |

## Open Questions
- [x] Custom plugin needs Bifrost dynamic build (documented); WASM ruled out. See SPIKE.md.
- [ ] Can a plugin call a model back through Bifrost from inside a hook without recursion or deadlock?
- [ ] Does Bifrost's Tavily relay need setup a judge won't have; is direct Tavily acceptable as fallback?
- [ ] How do maintainers prefer results surfaced (header vs response extra fields)? — check open issues / ask in Discord.

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Go plugin needs non-stock build | High | High | Spike first; WASM or HTTP-transport fallback |
| Go unfamiliar to builder | High | Med | Keep plugin thin; Claude writes, builder reviews |
| Extraction misses claims / over-flags | Med | Med | Labelled eval set drives prompt tuning |
| Latency too high | Med | Low | Parallel searches, cap N claims, cheap model |
| Deadline (9 days, solo) | Med | High | Submit rough version by Nov 13, improve after |

---
*Status: DRAFT — requirements only. Implementation planning pending via /plan.*
