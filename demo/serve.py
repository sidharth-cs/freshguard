"""FreshGuard demo server: serves index.html and proxies chat calls to Bifrost.

The demo virtual key stays server-side (read from run/demo_vk); the browser never sees it.
Run after `./dev.sh run`:   python3 demo/serve.py   ->  http://localhost:8000
Env: FG_GATEWAY (default http://localhost:8080), DEMO_PORT (default 8000).
"""

import json
import os
import pathlib
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

ROOT = pathlib.Path(__file__).resolve().parent
GATEWAY = os.environ.get("FG_GATEWAY", "http://localhost:8080").rstrip("/")
PORT = int(os.environ.get("DEMO_PORT", "8000"))
MODEL = "gemini/gemini-3.5-flash-lite"
FALLBACKS = ["gemini/gemini-3.1-flash-lite", "gemini/gemini-3.8-flash"]
MAX_QUESTION_CHARS = 500


def demo_vk() -> str:
    return (ROOT.parent / "run" / "demo_vk").read_text().strip()


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path not in ("/", "/index.html"):
            self.send_error(404)
            return
        self._send(200, (ROOT / "index.html").read_bytes(), "text/html; charset=utf-8")

    def do_POST(self):
        if self.path != "/api/ask":
            self.send_error(404)
            return
        try:
            length = min(int(self.headers.get("Content-Length", 0)), 10_000)
            question = str(json.loads(self.rfile.read(length) or b"{}").get("question", "")).strip()
        except (ValueError, json.JSONDecodeError):
            question = ""
        if not question or len(question) > MAX_QUESTION_CHARS:
            self._json(400, {"error": f"question must be 1-{MAX_QUESTION_CHARS} characters"})
            return

        payload = {
            "model": MODEL,
            "fallbacks": FALLBACKS,
            "messages": [{"role": "user", "content": question + "\nAnswer in two or three short sentences."}],
        }
        req = urllib.request.Request(
            GATEWAY + "/v1/chat/completions",
            data=json.dumps(payload).encode(),
            headers={"Content-Type": "application/json", "x-bf-vk": demo_vk()},
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=90) as res:
                data = json.loads(res.read())
                data["_headers"] = {k: v for k, v in res.headers.items() if k.lower().startswith("x-freshguard")}
                self._json(200, data)
        except urllib.error.HTTPError as e:
            self._json(e.code, {"error": e.read().decode(errors="replace")[:500]})
        except (urllib.error.URLError, TimeoutError) as e:
            self._json(502, {"error": f"gateway unreachable at {GATEWAY}: {e}. Is ./dev.sh run up?"})

    def _json(self, code, obj):
        self._send(code, json.dumps(obj).encode(), "application/json")

    def _send(self, code, body, ctype):
        self.send_response(code)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    print(f"FreshGuard demo on http://localhost:{PORT}  (gateway {GATEWAY})")
    ThreadingHTTPServer(("127.0.0.1", PORT), Handler).serve_forever()
