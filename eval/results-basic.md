# FreshGuard eval — search_depth=basic, time_range=year

Run: 2026-11-06 12:03 · 30 cases · model gemini/gemini-3.5-flash-lite

| Metric | Result | Target |
|---|---|---|
| Stale-claim recall | **13/13 (100%)** | ≥ 70% |
| Wrong on current + timeless (flagged or errored) | **0/17 (0%)** | ≤ 15% |
| Checks that errored | 0/30 | 0 |
| Claims judged unverifiable | 1 | — |
| Latency p50 / p90 (all checks) | 6177 / 20532 ms | < 3000 ms p50 |
| Stage p50: extract / search / judge | 1101 / 320 / 1635 ms | — |

| Case | Expect | Pass | Status | Verdicts | Current value | ms |
|---|---|---|---|---|---|---|
| s01 The latest stable version of Python is 3.11. | stale | ✅ | checked | changed | 3.14.3 | 6227 |
| s02 The newest iPhone you can buy is the iPhone 15. | stale | ✅ | checked | changed | iPhone 17e and iPhone 17 lineup | 6177 |
| s03 The current Prime Minister of the United Kingdom i… | stale | ✅ | checked | changed | Andy Burnham | 18232 |
| s04 Twitter's CEO is Parag Agrawal. | stale | ✅ | checked | changed | Elon Musk (as owner/executive chair) or Linda Yaccarino (as … | 20731 |
| s05 The latest long-term support release of Ubuntu is … | stale | ✅ | checked | changed | Ubuntu 26.04 LTS | 4801 |
| s06 The latest major version of React is React 17. | stale | ✅ | checked | changed | React 19.2 | 12981 |
| s07 OpenAI's most advanced publicly available model is… | stale | ✅ | checked | changed | GPT-5.6 Sol | 2615 |
| s08 The reigning FIFA men's World Cup champion is Fran… | stale | ✅ | checked | changed | Spain | 23148 |
| s09 The newest Node.js LTS line is Node 18. | stale | ✅ | checked | changed | Node 24 | 20532 |
| s10 Windows 10 is Microsoft's newest desktop operating… | stale | ✅ | checked | changed | Windows 11 | 4776 |
| s11 The US Federal Reserve's target for the federal fu… | stale | ✅ | checked | changed | 3.75% to 4.00% | 4428 |
| s12 The Secretary-General of the United Nations is Ban… | stale | ✅ | checked | changed | António Guterres | 21015 |
| s13 The latest long-term support version of Java is Ja… | stale | ✅ | checked | changed | Java 25 | 19749 |
| c01 Satya Nadella is the CEO of Microsoft. | current | ✅ | checked | supported | Satya Nadella | 11995 |
| c02 Jensen Huang is the CEO of Nvidia. | current | ✅ | checked | supported |  | 8402 |
| c03 Sundar Pichai is the CEO of Alphabet. | current | ✅ | checked | supported |  | 2861 |
| c04 Sam Altman is the CEO of OpenAI. | current | ✅ | checked | supported |  | 4213 |
| c05 Python 3.14 was released in October 2025. | current | ✅ | checked | unverifiable |  | 7226 |
| c06 Keir Starmer became Prime Minister of the United K… | current | ✅ | checked | supported |  | 19539 |
| c07 Los Angeles will host the 2028 Summer Olympics. | current | ✅ | checked | supported |  | 5119 |
| c08 Ubuntu 24.04 is a long-term support release. | current | ✅ | checked | supported |  | 18749 |
| c09 Argentina won the 2022 FIFA World Cup in Qatar. | current | ✅ | no_claims |  |  | 697 |
| c10 Lisa Su is the CEO of AMD. | current | ✅ | checked | supported |  | 2684 |
| t01 Water boils at 100 degrees Celsius at sea level. | timeless | ✅ | no_claims |  |  | 9184 |
| t02 In Python, a list is mutable while a tuple is immu… | timeless | ✅ | no_claims |  |  | 1123 |
| t03 HTTP status code 404 means the requested resource … | timeless | ✅ | no_claims |  |  | 11290 |
| t04 The Pythagorean theorem says that in a right trian… | timeless | ✅ | no_claims |  |  | 802 |
| t05 To reverse a list in Python you can call list.reve… | timeless | ✅ | no_claims |  |  | 632 |
| t06 TCP is connection-oriented while UDP is connection… | timeless | ✅ | no_claims |  |  | 822 |
| t07 William Shakespeare wrote Hamlet. | timeless | ✅ | no_claims |  |  | 761 |
