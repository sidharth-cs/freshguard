# FreshGuard eval — search_depth=advanced, time_range=year

Run: 2026-11-06 11:55 · 30 cases · model gemini/gemini-3.5-flash-lite

| Metric | Result | Target |
|---|---|---|
| Stale-claim recall | **10/13 (77%)** | ≥ 70% |
| Wrong on current + timeless (flagged or errored) | **1/17 (6%)** | ≤ 15% |
| Checks that errored | 3/30 | 0 |
| Claims judged unverifiable | 0 | — |
| Latency p50 / p90 (all checks) | 6616 / 10147 ms | < 3000 ms p50 |
| Stage p50: extract / search / judge | 1160 / 4237 / 1178 ms | — |

| Case | Expect | Pass | Status | Verdicts | Current value | ms |
|---|---|---|---|---|---|---|
| s01 The latest stable version of Python is 3.11. | stale | ✅ | checked | changed | 3.14.3 | 6616 |
| s02 The newest iPhone you can buy is the iPhone 15. | stale | ✅ | checked | changed | iPhone 18 Pro, iPhone 18 Pro Max, and iPhone Duo (or iPhone … | 5585 |
| s03 The current Prime Minister of the United Kingdom i… | stale | ✅ | checked | changed | Andy Burnham | 6633 |
| s04 Twitter's CEO is Parag Agrawal. | stale | ❌ | error: judge: Post "http://localhost:8089/v1/chat/completions": con… |  |  | 60004 |
| s05 The latest long-term support release of Ubuntu is … | stale | ❌ | error: judge: http://localhost:8089/v1/chat/completions: HTTP 429: … |  |  | 11641 |
| s06 The latest major version of React is React 17. | stale | ✅ | checked | changed | React 19.2 | 5520 |
| s07 OpenAI's most advanced publicly available model is… | stale | ✅ | checked | changed | GPT-5.6 | 7363 |
| s08 The reigning FIFA men's World Cup champion is Fran… | stale | ✅ | checked | changed | Argentina | 7560 |
| s09 The newest Node.js LTS line is Node 18. | stale | ✅ | checked | changed | Node 24 (or Node 26) | 6925 |
| s10 Windows 10 is Microsoft's newest desktop operating… | stale | ❌ | error: extract: http://localhost:8089/v1/chat/completions: HTTP 429… |  |  | 3976 |
| s11 The US Federal Reserve's target for the federal fu… | stale | ✅ | checked | changed | 3.75% to 4.00% | 5378 |
| s12 The Secretary-General of the United Nations is Ban… | stale | ✅ | checked | changed | António Guterres | 5644 |
| s13 The latest long-term support version of Java is Ja… | stale | ✅ | checked | changed | Java 25 | 6537 |
| c01 Satya Nadella is the CEO of Microsoft. | current | ✅ | checked | supported |  | 7461 |
| c02 Jensen Huang is the CEO of Nvidia. | current | ✅ | checked | supported | Jensen Huang | 10147 |
| c03 Sundar Pichai is the CEO of Alphabet. | current | ✅ | checked | supported | Sundar Pichai | 11060 |
| c04 Sam Altman is the CEO of OpenAI. | current | ✅ | checked | supported | Sam Altman | 16801 |
| c05 Python 3.14 was released in October 2025. | current | ✅ | checked | supported |  | 10890 |
| c06 Keir Starmer became Prime Minister of the United K… | current | ❌ | checked | changed | Andy Burnham | 7583 |
| c07 Los Angeles will host the 2028 Summer Olympics. | current | ✅ | checked | supported |  | 7032 |
| c08 Ubuntu 24.04 is a long-term support release. | current | ✅ | checked | supported |  | 7859 |
| c09 Argentina won the 2022 FIFA World Cup in Qatar. | current | ✅ | no_claims |  |  | 7538 |
| c10 Lisa Su is the CEO of AMD. | current | ✅ | checked | supported |  | 5429 |
| t01 Water boils at 100 degrees Celsius at sea level. | timeless | ✅ | no_claims |  |  | 1488 |
| t02 In Python, a list is mutable while a tuple is immu… | timeless | ✅ | no_claims |  |  | 1194 |
| t03 HTTP status code 404 means the requested resource … | timeless | ✅ | no_claims |  |  | 837 |
| t04 The Pythagorean theorem says that in a right trian… | timeless | ✅ | no_claims |  |  | 1244 |
| t05 To reverse a list in Python you can call list.reve… | timeless | ✅ | no_claims |  |  | 1365 |
| t06 TCP is connection-oriented while UDP is connection… | timeless | ✅ | no_claims |  |  | 1186 |
| t07 William Shakespeare wrote Hamlet. | timeless | ✅ | no_claims |  |  | 1374 |
