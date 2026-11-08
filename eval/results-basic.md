# FreshGuard eval — search_depth=basic, time_range=year

Run: 2026-11-08 22:04 · 37 cases · model gemini/gemini-3.5-flash-lite

| Metric | Result | Target |
|---|---|---|
| Stale-claim recall | **18/18 (100%)** | ≥ 70% |
| Wrong on current + timeless (flagged or errored) | **1/19 (5%)** | ≤ 15% |
| Checks that errored | 1/37 | 0 |
| Claims judged unverifiable | 0 | — |
| Latency p50 / p90 (all checks) | 16464 / 37721 ms | < 3000 ms p50 |
| Stage p50: extract / search / judge | 7566 / 2723 / 8216 ms | — |

| Case | Expect | Pass | Status | Verdicts | Current value | ms |
|---|---|---|---|---|---|---|
| s01 The latest stable version of Python is 3.11. | stale | ✅ | checked | changed | 3.15.2 | 49847 |
| s02 The newest iPhone you can buy is the iPhone 15. | stale | ✅ | checked | changed | iPhone 18 / iPhone Duo / iPhone 17 | 26238 |
| s03 The current Prime Minister of the United Kingdom i… | stale | ✅ | checked | changed | Andy Burnham | 19164 |
| s04 Twitter's CEO is Parag Agrawal. | stale | ✅ | checked | changed | Elon Musk (or no CEO / Linda Yaccarino previously) | 34000 |
| s05 The latest long-term support release of Ubuntu is … | stale | ✅ | checked | changed | Ubuntu 26.04 LTS | 11358 |
| s06 The latest major version of React is React 17. | stale | ✅ | checked | changed | React 19 | 16464 |
| s07 OpenAI's most advanced publicly available model is… | stale | ✅ | checked | changed | GPT-6-Astra or GPT-5.6 | 15245 |
| s08 The reigning FIFA men's World Cup champion is Fran… | stale | ✅ | checked | changed | Spain | 22661 |
| s09 The newest Node.js LTS line is Node 18. | stale | ✅ | checked | changed | Node.js 26 (and Node.js 24) | 17690 |
| s10 Windows 10 is Microsoft's newest desktop operating… | stale | ✅ | checked | changed | Windows 11 | 11181 |
| s11 The US Federal Reserve's target for the federal fu… | stale | ✅ | checked | changed | 3-3/4 to 4 percent | 4557 |
| s12 The Secretary-General of the United Nations is Ban… | stale | ✅ | checked | changed | António Guterres | 48681 |
| s13 The latest long-term support version of Java is Ja… | stale | ✅ | checked | changed | Java 25 | 31949 |
| s14 A US first-class Forever stamp costs 66 cents. | stale | ✅ | checked | changed | 82 cents | 48465 |
| s15 Netflix's Standard plan in the US costs $15.49 per… | stale | ✅ | checked | changed | $19.99 per month | 38446 |
| s16 Spotify Premium Individual costs $10.99 per month … | stale | ✅ | checked | changed | $12.99 per month | 4770 |
| s17 The UK National Living Wage for workers aged 21 an… | stale | ✅ | checked | changed | £12.71 per hour | 22293 |
| s18 The EU AI Act is still a proposal and has not been… | stale | ✅ | checked | changed | The EU AI Act has been passed into law and is actively being… | 23509 |
| c01 Satya Nadella is the CEO of Microsoft. | current | ✅ | checked | supported |  | 19071 |
| c02 Jensen Huang is the CEO of Nvidia. | current | ✅ | checked | supported |  | 20674 |
| c03 Sundar Pichai is the CEO of Alphabet. | current | ✅ | checked | supported |  | 12608 |
| c04 Sam Altman is the CEO of OpenAI. | current | ✅ | checked | supported |  | 4812 |
| c05 Python 3.14 was released in October 2025. | current | ✅ | checked | supported |  | 6909 |
| c06 Keir Starmer became Prime Minister of the United K… | current | ✅ | checked | supported |  | 35166 |
| c07 Los Angeles will host the 2028 Summer Olympics. | current | ✅ | checked | supported |  | 5063 |
| c08 Ubuntu 24.04 is a long-term support release. | current | ✅ | checked | supported |  | 15336 |
| c09 Argentina won the 2022 FIFA World Cup in Qatar. | current | ✅ | no_claims |  |  | 8967 |
| c10 Lisa Su is the CEO of AMD. | current | ✅ | checked | supported |  | 21556 |
| c11 The US federal minimum wage is $7.25 per hour. | current | ✅ | checked | supported | $7.25 | 37721 |
| c12 The UK income tax personal allowance is £12,570. | current | ❌ | error: judge: Post "http://localhost:8080/v1/chat/completions": con… |  |  | 60003 |
| t01 Water boils at 100 degrees Celsius at sea level. | timeless | ✅ | no_claims |  |  | 9005 |
| t02 In Python, a list is mutable while a tuple is immu… | timeless | ✅ | no_claims |  |  | 31135 |
| t03 HTTP status code 404 means the requested resource … | timeless | ✅ | no_claims |  |  | 4076 |
| t04 The Pythagorean theorem says that in a right trian… | timeless | ✅ | no_claims |  |  | 800 |
| t05 To reverse a list in Python you can call list.reve… | timeless | ✅ | no_claims |  |  | 719 |
| t06 TCP is connection-oriented while UDP is connection… | timeless | ✅ | no_claims |  |  | 15651 |
| t07 William Shakespeare wrote Hamlet. | timeless | ✅ | no_claims |  |  | 4862 |
