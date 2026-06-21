from __future__ import annotations

import importlib.util
import json
import os
import subprocess
import sys
import time
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
EXAMPLE_DIR = Path(__file__).resolve().parent
LOG_DIR = ROOT / "logs" / "personal_line_bridge"
REPLY_EXE = EXAMPLE_DIR / "personal-line-hakua-reply.exe"
REPLY_URL = "http://127.0.0.1:9102/reply"
HEALTH_URL = "http://127.0.0.1:9102/health"
PLUGIN_CORE = Path.home() / ".hermes" / "plugins" / "line-personal-bridge" / "core.py"
HERMES_ENV = Path.home() / ".hermes" / ".env"


def load_dotenv(path: Path) -> None:
    if not path.exists():
        return
    for raw in path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip().strip('"').strip("'")
        if key and key not in os.environ:
            os.environ[key] = value


def http_ok(url: str, timeout: float = 1.5) -> bool:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as resp:
            return 200 <= resp.status < 300
    except Exception:
        return False


def start_reply_server() -> None:
    LOG_DIR.mkdir(parents=True, exist_ok=True)
    if http_ok(HEALTH_URL):
        print("reply_server=already_running")
        return
    if not REPLY_EXE.exists():
        print("reply_server=build")
        subprocess.run(["go", "build", "-o", str(REPLY_EXE), "./examples/personal_line_bridge"], cwd=str(ROOT), check=True)
    log_path = LOG_DIR / "reply-server.log"
    log = log_path.open("ab")
    subprocess.Popen(
        [str(REPLY_EXE), "-addr", "127.0.0.1:9102"],
        cwd=str(ROOT),
        stdin=subprocess.DEVNULL,
        stdout=log,
        stderr=subprocess.STDOUT,
        creationflags=(subprocess.CREATE_NEW_PROCESS_GROUP if os.name == "nt" else 0),
    )
    for _ in range(20):
        if http_ok(HEALTH_URL):
            print(f"reply_server=started log={log_path}")
            return
        time.sleep(0.5)
    print(f"reply_server=unhealthy log={log_path}")


def load_core():
    spec = importlib.util.spec_from_file_location("line_personal_bridge_core", PLUGIN_CORE)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"cannot load plugin core: {PLUGIN_CORE}")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def main() -> int:
    load_dotenv(HERMES_ENV)

    # Non-secret runtime policy. Credentials are not embedded in this script.
    os.environ.setdefault("LINEJS_PERSONAL_USE_LINE_PERSONAL_FALLBACK", "1")
    os.environ["LINEJS_PERSONAL_AUTO_REPLY"] = "1"
    os.environ["LINEJS_PERSONAL_AUTO_REPLY_TRIGGERS"] = "#はくあ,#hermesagent"
    os.environ["LINEJS_PERSONAL_AUTO_REPLY_ONLY_GROUPS"] = "1"
    os.environ["LINEJS_PERSONAL_AUTO_REPLY_COOLDOWN_MS"] = "120000"
    os.environ["LINEJS_PERSONAL_AUTO_REPLY_WEBHOOK"] = REPLY_URL
    os.environ["LINEJS_PERSONAL_AUTO_REPLY_WEBHOOK_TIMEOUT_MS"] = "90000"
    # Restrict Hakua auto-replies/admin reactions to approved groups only.
    # 日本メンタル(雑談/通話グル), 生成AI、LLMなど
    os.environ["LINEJS_PERSONAL_ALLOWED_GROUP_MIDS"] = "c0431273df4f01cbc7afbb23bf4624b85,c3c47bde1c8b6bdb8ca0ddaab9f2089d7"

    start_reply_server()

    core = load_core()
    current = core.status_payload({})
    current_http = current.get("bridge", {}).get("http", {})
    auto_reply = current_http.get("autoReply") if isinstance(current_http, dict) else {}
    needs_restart = (
        current_http.get("loginState") == "error"
        or not (isinstance(auto_reply, dict) and auto_reply.get("webhookConfigured"))
    )
    result = core.start_payload({"force": needs_restart, "wait_seconds": 15})
    print(json.dumps({
        "line_personal_start_ok": result.get("ok"),
        "forced_restart": needs_restart,
        "already_running": result.get("already_running", False),
        "status": result.get("status") or result.get("bridge", {}).get("http"),
    }, ensure_ascii=False, default=str))

    # If a PIN is required, keep it visible in the local startup log only.
    status = core.status_payload({})
    http = status.get("bridge", {}).get("http", {})
    pin = http.get("pinCode")
    if pin:
        print(f"LINE_PIN_REQUIRED={pin}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
