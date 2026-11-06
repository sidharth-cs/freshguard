# FreshGuard eval — search_depth=advanced, time_range=year

Run: 2026-11-06 12:05 · 30 cases · model gemini/gemini-3.5-flash-lite

| Metric | Result | Target |
|---|---|---|
| Stale-claim recall | **13/13 (100%)** | ≥ 70% |
| Wrong on current + timeless (flagged or errored) | **0/17 (0%)** | ≤ 15% |
| Checks that errored | 0/30 | 0 |
| Claims judged unverifiable | 1 | — |
| Latency p50 / p90 (all checks) | 6163 / 23271 ms | < 3000 ms p50 |
| Stage p50: extract / search / judge | 1145 / 309 / 1532 ms | — |

| Case | Expect | Pass | Status | Verdicts | Current value | ms |
|---|---|---|---|---|---|---|
| s01 The latest stable version of Python is 3.11. | stale | ✅ | checked | changed | 3.14.3 | 28531 |
| s02 The newest iPhone you can buy is the iPhone 15. | stale | ✅ | checked | changed | iPhone 18 Pro, iPhone 18 Pro Max, and iPhone Duo | 25380 |
| s03 The current Prime Minister of the United Kingdom i… | stale | ✅ | checked | changed | Andy Burnham | 2306 |
| s04 Twitter's CEO is Parag Agrawal. | stale | ✅ | checked | changed | Elon Musk (or X CEO) | 4949 |
| s05 The latest long-term support release of Ubuntu is … | stale | ✅ | checked | changed | 26.04 LTS | 29284 |
| s06 The latest major version of React is React 17. | stale | ✅ | checked | changed | React 19.2 | 2667 |
| s07 OpenAI's most advanced publicly available model is… | stale | ✅ | checked | changed | GPT-5.6 | 2779 |
| s08 The reigning FIFA men's World Cup champion is Fran… | stale | ✅ | checked | changed | Spain | 22516 |
| s09 The newest Node.js LTS line is Node 18. | stale | ✅ | checked | changed | Node 26 | 6163 |
| s10 Windows 10 is Microsoft's newest desktop operating… | stale | ✅ | checked | changed | Windows 11 | 20303 |
| s11 The US Federal Reserve's target for the federal fu… | stale | ✅ | checked | changed | 3.50% – 3.75% (or 3.75% – 4.00%) | 23271 |
| s12 The Secretary-General of the United Nations is Ban… | stale | ✅ | checked | changed | António Guterres | 2338 |
| s13 The latest long-term support version of Java is Ja… | stale | ✅ | checked | changed | Java 25 | 4196 |
| c01 Satya Nadella is the CEO of Microsoft. | current | ✅ | checked | supported |  | 21792 |
| c02 Jensen Huang is the CEO of Nvidia. | current | ✅ | checked | supported |  | 18203 |
| c03 Sundar Pichai is the CEO of Alphabet. | current | ✅ | checked | supported |  | 5378 |
| c04 Sam Altman is the CEO of OpenAI. | current | ✅ | checked | supported |  | 2585 |
| c05 Python 3.14 was released in October 2025. | current | ✅ | checked | unverifiable |  | 21267 |
| c06 Keir Starmer became Prime Minister of the United K… | current | ✅ | no_claims |  |  | 1145 |
| c07 Los Angeles will host the 2028 Summer Olympics. | current | ✅ | checked | supported |  | 17802 |
| c08 Ubuntu 24.04 is a long-term support release. | current | ✅ | checked | supported |  | 2764 |
| c09 Argentina won the 2022 FIFA World Cup in Qatar. | current | ✅ | no_claims |  |  | 1039 |
| c10 Lisa Su is the CEO of AMD. | current | ✅ | checked | supported |  | 10920 |
| t01 Water boils at 100 degrees Celsius at sea level. | timeless | ✅ | no_claims |  |  | 892 |
| t02 In Python, a list is mutable while a tuple is immu… | timeless | ✅ | no_claims |  |  | 13673 |
| t03 HTTP status code 404 means the requested resource … | timeless | ✅ | no_claims |  |  | 632 |
| t04 The Pythagorean theorem says that in a right trian… | timeless | ✅ | no_claims |  |  | 1022 |
| t05 To reverse a list in Python you can call list.reve… | timeless | ✅ | no_claims |  |  | 11979 |
| t06 TCP is connection-oriented while UDP is connection… | timeless | ✅ | no_claims |  |  | 16677 |
| t07 William Shakespeare wrote Hamlet. | timeless | ✅ | no_claims |  |  | 14858 |
