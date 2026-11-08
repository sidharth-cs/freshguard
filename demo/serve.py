"""FreshGuard demo server: serves index.html and proxies chat calls to Bifrost.

The demo virtual key stays server-side (read from run/demo_vk); the browser never sees it.
Run after `./dev.sh run`:   python3 demo/serve.py   ->  http://localhost:8000
Env: FG_GATEWAY (default http://localhost:8080), DEMO_HOST (default 127.0.0.1; 0.0.0.0 in Docker),
DEMO_PORT or PORT (default 8000), DEMO_MAX_PER_MIN (default 20, all visitors combined),
FRESHGUARD_DEMO_VK (else read from run/demo_vk).
"""

import json
import os
import pathlib
import threading
import time
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

ROOT = pathlib.Path(__file__).resolve().parent
GATEWAY = os.environ.get("FG_GATEWAY", "http://localhost:8080").rstrip("/")
HOST = os.environ.get("DEMO_HOST", "127.0.0.1")
PORT = int(os.environ.get("DEMO_PORT") or os.environ.get("PORT") or "8000")
MAX_PER_MIN = int(os.environ.get("DEMO_MAX_PER_MIN", "20"))
MODEL = "gemini/gemini-3.5-flash-lite"
FALLBACKS = ["gemini/gemini-3.1-flash-lite", "gemini/gemini-3.8-flash"]
MAX_QUESTION_CHARS = 500


def demo_vk() -> str:
    return os.environ.get("FRESHGUARD_DEMO_VK") or (ROOT.parent / "run" / "demo_vk").read_text().strip()


# Public-demo guard: a global cap on questions per minute, on top of the virtual key's
# Bifrost rate limits. ponytail: one shared window; per-visitor limits if it gets abused.
_lock = threading.Lock()
_recent: list[float] = []


def allow() -> bool:
    now = time.monotonic()
    with _lock:
        _recent[:] = [t for t in _recent if now - t < 60]
        if len(_recent) >= MAX_PER_MIN:
            return False
        _recent.append(now)
        return True


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
        if not allow():
            self._json(429, {"error": "The demo is busy. Try again in a minute."})
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
            self._json(502, {"error": f"gateway unreachable: {e}"})

    def _json(self, code, obj):
        self._send(code, json.dumps(obj).encode(), "application/json")

    def _send(self, code, body, ctype):
        self.send_response(code)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    print(f"FreshGuard demo on http://{HOST}:{PORT}  (gateway {GATEWAY})", flush=True)
    ThreadingHTTPServer((HOST, PORT), Handler).serve_forever()
