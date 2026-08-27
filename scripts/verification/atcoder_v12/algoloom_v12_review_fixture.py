#!/usr/bin/env python3
"""AlgoLoom V-12 review fixture: a single-file stand-in for the local helper.

Purpose
    Let a Chrome Web Store reviewer exercise "AlgoLoom Authentication Verification
    BETA" without running the compiled macOS helper. It speaks the same
    authenticated loopback protocol, applies the same checks, and requires no
    compiler, no developer mode, and no Gatekeeper override.

What this fixture deliberately does NOT do
    * It never writes the AtCoder session anywhere. The value is validated for
      shape and scope, then discarded. Nothing is stored in a keychain, a file,
      an environment variable, or a log.
    * It never contacts atcoder.jp or any other host. It only listens on
      127.0.0.1.
    * It never automates sign-in, Turnstile, or submission.

    The production helper additionally verifies the account against AtCoder and
    stores the verified session in the OS secret store. That behaviour is out of
    scope for reviewing the extension, so this fixture omits it.

Usage
    python3 algoloom_v12_review_fixture.py --extension-id <32-character-id>
    python3 algoloom_v12_review_fixture.py --self-test

Requires Python 3.9 or later. No third-party packages.
"""

import argparse
import ast
import hmac
import json
import re
import secrets
import sys
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

FIXTURE_VERSION = "0.1.0"
PROTOCOL_VERSION = 1
MAX_REQUEST_BYTES = 32 * 1024
MAX_COOKIE_VALUE_BYTES = 16 * 1024
DEFAULT_EXTENSION_VERSION = "0.1.0"
DEFAULT_CONSENT_VERSION = "1.0"

TOKEN_PATTERN = re.compile(r"^[0-9a-f]{64}$")
VERSION_PATTERN = re.compile(r"^\d+(?:\.\d+){1,3}$")
ACCOUNT_PATTERN = re.compile(r"^[A-Za-z0-9_]{1,64}$")
EXTENSION_ID_PATTERN = re.compile(r"^[a-p]{32}$")
ALLOWED_COOKIE_DOMAINS = frozenset(("atcoder.jp", ".atcoder.jp"))

STAGE_BOOTSTRAP = "await_bootstrap"
STAGE_CONSENT = "await_consent"
STAGE_ACCOUNT = "await_account"
STAGE_CAPTURE = "await_capture"
STAGE_COMPLETE = "complete"


class ProtocolError(Exception):
    """Carries a value-free reason identifier that is safe to return."""

    def __init__(self, reason):
        super().__init__(reason)
        self.reason = reason


def _exact_keys(raw, *expected):
    return len(raw) == len(expected) and all(key in raw for key in expected)


def _required(raw, key, kind):
    value = raw.get(key)
    if kind is bool:
        if not isinstance(value, bool):
            raise ProtocolError("%s_invalid" % key)
        return value
    if kind is int:
        if not isinstance(value, int) or isinstance(value, bool):
            raise ProtocolError("%s_invalid" % key)
        return value
    if not isinstance(value, str):
        raise ProtocolError("%s_invalid" % key)
    return value


def _valid_cookie_value(value):
    if not isinstance(value, str) or not value:
        return False
    if len(value.encode("utf-8")) > MAX_COOKIE_VALUE_BYTES:
        return False
    if value.strip() != value:
        return False
    if any(character in value for character in (";", "\r", "\n", "\x00")):
        return False
    return "REVEL_SESSION=" not in value


class ProtocolMachine:
    """Mirrors the helper's state order and validation, without persistence."""

    def __init__(self, extension_version, consent_version, expected_identity):
        if not VERSION_PATTERN.match(extension_version) or not VERSION_PATTERN.match(consent_version):
            raise ProtocolError("version_invalid")
        if not ACCOUNT_PATTERN.match(expected_identity):
            raise ProtocolError("expected_identity_invalid")
        self.extension_version = extension_version
        self.consent_version = consent_version
        self.expected_identity = expected_identity
        self.stage = STAGE_BOOTSTRAP
        self.completed = threading.Event()
        self._lock = threading.Lock()

    def apply_event(self, raw):
        with self._lock:
            event_type = raw.get("type")
            if not isinstance(event_type, str):
                raise ProtocolError("type_invalid")

            if event_type == "bootstrap_ready":
                if self.stage != STAGE_BOOTSTRAP or not _exact_keys(
                    raw, "type", "protocol_version", "extension_version",
                    "consent_version", "navigator_webdriver",
                ):
                    raise ProtocolError("bootstrap_state_invalid")
                if _required(raw, "protocol_version", int) != PROTOCOL_VERSION:
                    raise ProtocolError("protocol_version_mismatch")
                if _required(raw, "extension_version", str) != self.extension_version:
                    raise ProtocolError("extension_version_mismatch")
                if _required(raw, "consent_version", str) != self.consent_version:
                    raise ProtocolError("consent_version_mismatch")
                if _required(raw, "navigator_webdriver", bool):
                    raise ProtocolError("automation_signal_detected")
                self.stage = STAGE_CONSENT
                return

            if event_type == "consent_confirmed":
                if self.stage != STAGE_CONSENT or not _exact_keys(raw, "type", "consent_version"):
                    raise ProtocolError("consent_state_invalid")
                if _required(raw, "consent_version", str) != self.consent_version:
                    raise ProtocolError("consent_version_mismatch")
                self.stage = STAGE_ACCOUNT
                return

            if event_type == "account_checked":
                if self.stage != STAGE_ACCOUNT or not _exact_keys(
                    raw, "type", "identity", "identity_count", "navigator_webdriver",
                ):
                    raise ProtocolError("account_state_invalid")
                identity = _required(raw, "identity", str)
                if not ACCOUNT_PATTERN.match(identity):
                    raise ProtocolError("identity_invalid")
                if _required(raw, "identity_count", int) != 1:
                    raise ProtocolError("identity_count_invalid")
                if _required(raw, "navigator_webdriver", bool):
                    raise ProtocolError("automation_signal_detected")
                if not hmac.compare_digest(identity, self.expected_identity):
                    raise ProtocolError("identity_mismatch")
                self.stage = STAGE_CAPTURE
                return

            raise ProtocolError("event_type_invalid")

    def capture(self, raw):
        with self._lock:
            if self.stage != STAGE_CAPTURE:
                raise ProtocolError("capture_state_invalid")
            if not _exact_keys(
                raw, "candidate_count", "cookie_name", "cookie_domain", "cookie_path",
                "cookie_secure", "cookie_http_only", "cookie_host_only", "cookie_session",
                "cookie_partitioned", "cookie_value", "observed_identity",
            ):
                raise ProtocolError("capture_schema_invalid")

            if _required(raw, "candidate_count", int) != 1:
                raise ProtocolError("cookie_scope_not_unique")
            if _required(raw, "cookie_name", str) != "REVEL_SESSION":
                raise ProtocolError("cookie_scope_not_unique")
            if _required(raw, "cookie_domain", str) not in ALLOWED_COOKIE_DOMAINS:
                raise ProtocolError("cookie_domain_invalid")
            if (
                _required(raw, "cookie_path", str) != "/"
                or not _required(raw, "cookie_secure", bool)
                or not _required(raw, "cookie_http_only", bool)
                or _required(raw, "cookie_partitioned", bool)
            ):
                raise ProtocolError("cookie_attributes_invalid")
            _required(raw, "cookie_host_only", bool)
            _required(raw, "cookie_session", bool)
            if not _valid_cookie_value(raw.get("cookie_value")):
                raise ProtocolError("cookie_value_invalid")
            observed = _required(raw, "observed_identity", str)
            if not hmac.compare_digest(observed, self.expected_identity):
                raise ProtocolError("identity_mismatch")

            # The session value is intentionally never bound to any name that
            # outlives this call. Validation is the only use.
            self.stage = STAGE_COMPLETE
            self.completed.set()
            return {
                "ok": True,
                "capture_validated": True,
                "identity_matched": True,
                "secret_store_written": False,
            }


CONSENT_PAGE = """<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="algoloom-loopback-token" content="{token}">
<meta name="algoloom-consent-version" content="{consent_version}">
<title>AlgoLoom review fixture</title>
<style>
:root {{ color-scheme: light; font-family: ui-sans-serif, system-ui, sans-serif; }}
body {{ margin: 0; padding: 32px; background: #eef4ff; color: #172033; }}
main {{ max-width: 720px; margin: 0 auto; background: #fff; border: 1px solid #c7d7f5;
        border-radius: 12px; padding: 28px; line-height: 1.7; }}
h1 {{ font-size: 20px; margin: 0 0 4px; }}
p.tag {{ margin: 0 0 20px; font-size: 13px; color: #4a5b7a; }}
ul {{ padding-left: 20px; }}
li {{ margin-bottom: 6px; }}
.note {{ background: #f1f6ff; border-left: 4px solid #2563eb; padding: 12px 16px; margin: 20px 0; }}
button {{ font-size: 16px; padding: 12px 22px; border: 0; border-radius: 8px;
          background: #1d4ed8; color: #fff; cursor: pointer; }}
button[disabled] {{ background: #94a7c9; cursor: default; }}
</style>
</head>
<body>
<main>
<h1>AlgoLoom Authentication Verification BETA</h1>
<p class="tag">Chrome Web Store review fixture {fixture_version} &mdash; consent version {consent_version}</p>
<p>After you choose to continue, this extension will:</p>
<ul>
<li>send you to <code>https://atcoder.jp/settings</code>, where you sign in yourself;</li>
<li>read exactly one <code>REVEL_SESSION</code> cookie for <code>https://atcoder.jp/</code>, once;</li>
<li>hand that single cookie to this local fixture on <code>127.0.0.1</code>.</li>
</ul>
<p>The extension does not automate sign-in, Turnstile, or submission, and reads no other cookie.</p>
<div class="note">
<strong>This fixture stores nothing.</strong> It checks that the cookie has the expected
name, host, path, and security attributes, reports the result, and discards the value.
It writes no keychain item, no file, and no log entry, and it never contacts atcoder.jp.
The production helper instead verifies the account and stores the session in the OS
secret store; that step is out of scope here.
</div>
<p>この画面は Chrome ウェブストア審査用のフィクスチャです。値は検査後に破棄され、保存も外部送信も行いません。</p>
<button id="algoloom-consent" type="button">Consent and continue to AtCoder</button>
</main>
</body>
</html>
"""


def render_consent_page(token, consent_version):
    return CONSENT_PAGE.format(
        token=token, consent_version=consent_version, fixture_version=FIXTURE_VERSION,
    )


class ReviewFixture:
    """Transport checks and routing. Pure: it owns no socket."""

    def __init__(self, port, token, extension_id, consent_version, machine):
        if not (1024 <= port <= 65535):
            raise ProtocolError("handler_configuration_invalid")
        if not TOKEN_PATTERN.match(token) or not EXTENSION_ID_PATTERN.match(extension_id):
            raise ProtocolError("handler_configuration_invalid")
        self.port = port
        self.token = token
        self.extension_id = extension_id
        self.consent_version = consent_version
        self.machine = machine
        self.bootstrap_claimed = False
        self._lock = threading.Lock()

    def handle(self, method, path, host, client_host, headers, body):
        base = [
            ("Cache-Control", "no-store"),
            ("X-Content-Type-Options", "nosniff"),
        ]

        if host != "127.0.0.1:%d" % self.port or client_host != "127.0.0.1":
            return self._json(403, base, {"error": "transport_rejected"})

        if method == "GET" and path == "/bootstrap":
            return self._bootstrap(base)

        if method != "POST" or path not in ("/event", "/capture"):
            return self._json(404, base, {"error": "route_rejected"})

        origin = headers.get("origin", "")
        authorization = headers.get("authorization", "")
        expected_authorization = "Bearer " + self.token
        if origin != "chrome-extension://" + self.extension_id or not (
            len(authorization) == len(expected_authorization)
            and hmac.compare_digest(authorization, expected_authorization)
        ):
            return self._json(403, base, {"error": "authentication_rejected"})

        if headers.get("content-type", "") != "application/json":
            return self._json(415, base, {"error": "content_type_rejected"})

        if len(body) > MAX_REQUEST_BYTES:
            return self._json(400, base, {"error": "body_rejected"})
        try:
            raw = json.loads(body.decode("utf-8"))
        except (ValueError, UnicodeDecodeError):
            return self._json(400, base, {"error": "body_rejected"})
        if not isinstance(raw, dict):
            return self._json(400, base, {"error": "body_rejected"})

        try:
            if path == "/event":
                self.machine.apply_event(raw)
                return self._json(200, base, {"ok": True})
            return self._json(200, base, self.machine.capture(raw))
        except ProtocolError as error:
            return self._json(409, base, {"error": error.reason})

    def _bootstrap(self, base):
        with self._lock:
            if self.bootstrap_claimed:
                return self._json(410, base, {"error": "bootstrap_already_claimed"})
            self.bootstrap_claimed = True
        page = render_consent_page(self.token, self.consent_version).encode("utf-8")
        headers = base + [
            ("Content-Type", "text/html; charset=utf-8"),
            ("Content-Security-Policy",
             "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'"),
            ("Referrer-Policy", "no-referrer"),
            ("X-Frame-Options", "DENY"),
        ]
        return 200, headers, page

    @staticmethod
    def _json(status, base, payload):
        body = (json.dumps(payload) + "\n").encode("utf-8")
        return status, base + [("Content-Type", "application/json")], body


class _RequestHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    fixture = None

    def _respond(self, method):
        length_header = self.headers.get("Content-Length", "0")
        try:
            length = int(length_header)
        except ValueError:
            length = -1
        if length < 0 or length > MAX_REQUEST_BYTES:
            self._write(400, [("Content-Type", "application/json")],
                        b'{"error": "body_rejected"}\n')
            return
        body = self.rfile.read(length) if length else b""
        headers = {key.lower(): value for key, value in self.headers.items()}
        status, response_headers, payload = self.fixture.handle(
            method, self.path.split("?", 1)[0], self.headers.get("Host", ""),
            self.client_address[0], headers, body,
        )
        self._write(status, response_headers, payload)

    def _write(self, status, headers, payload):
        self.send_response(status)
        for key, value in headers:
            self.send_header(key, value)
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def do_GET(self):
        self._respond("GET")

    def do_POST(self):
        self._respond("POST")

    def log_message(self, fmt, *args):
        # Never write request bodies or query strings anywhere.
        sys.stderr.write("fixture: %s %s\n" % (self.command, self.path.split("?", 1)[0]))


def run_server(extension_id, extension_version, consent_version, expected_identity):
    machine = ProtocolMachine(extension_version, consent_version, expected_identity)
    server = ThreadingHTTPServer(("127.0.0.1", 0), _RequestHandler)
    port = server.server_address[1]
    handler_state = ReviewFixture(port, secrets.token_hex(32), extension_id, consent_version, machine)
    _RequestHandler.fixture = handler_state

    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    sys.stdout.write(
        "\nOpen this URL in the browser where the extension is installed:\n"
        "  http://127.0.0.1:%d/bootstrap\n\n"
        "Then sign in to AtCoder as '%s' when the page sends you there.\n"
        "Press Ctrl-C to stop. Nothing is stored on this machine.\n\n" % (port, expected_identity)
    )
    sys.stdout.flush()
    try:
        completed = machine.completed.wait(timeout=15 * 60)
    except KeyboardInterrupt:
        completed = False
    server.shutdown()
    server.server_close()
    result = {
        "ok": bool(completed),
        "fixture_version": FIXTURE_VERSION,
        "protocol_version": PROTOCOL_VERSION,
        "extension_version": extension_version,
        "stage": machine.stage,
        "secret_store_written": False,
        "session_value_retained": False,
    }
    sys.stdout.write(json.dumps(result, indent=2) + "\n")
    return 0 if completed else 1


# --- fixed-input self-test: no sockets, no external connections -------------

_ID = "a" * 32
_TOKEN = "0" * 64
_PORT = 50000
_IDENTITY = "reviewer_account"
_HOST = "127.0.0.1:%d" % _PORT
_AUTH = {"origin": "chrome-extension://" + _ID,
         "authorization": "Bearer " + _TOKEN,
         "content-type": "application/json"}


def _new_fixture():
    machine = ProtocolMachine(DEFAULT_EXTENSION_VERSION, DEFAULT_CONSENT_VERSION, _IDENTITY)
    return ReviewFixture(_PORT, _TOKEN, _ID, DEFAULT_CONSENT_VERSION, machine), machine


def _post(fixture, path, payload, headers=None, host=_HOST, client="127.0.0.1"):
    body = json.dumps(payload).encode("utf-8") if isinstance(payload, (dict, list)) else payload
    status, _, response = fixture.handle(
        "POST", path, host, client, dict(_AUTH if headers is None else headers), body)
    return status, json.loads(response.decode("utf-8"))


def _bootstrap_ready(**overrides):
    payload = {"type": "bootstrap_ready", "protocol_version": PROTOCOL_VERSION,
               "extension_version": DEFAULT_EXTENSION_VERSION,
               "consent_version": DEFAULT_CONSENT_VERSION, "navigator_webdriver": False}
    payload.update(overrides)
    return payload


def _capture_payload(**overrides):
    payload = {"candidate_count": 1, "cookie_name": "REVEL_SESSION",
               "cookie_domain": "atcoder.jp", "cookie_path": "/", "cookie_secure": True,
               "cookie_http_only": True, "cookie_host_only": True, "cookie_session": True,
               "cookie_partitioned": False, "cookie_value": "fixture-session-value",
               "observed_identity": _IDENTITY}
    payload.update(overrides)
    return payload


def _advance_to_capture(fixture):
    assert _post(fixture, "/event", _bootstrap_ready())[0] == 200
    assert _post(fixture, "/event",
                 {"type": "consent_confirmed", "consent_version": DEFAULT_CONSENT_VERSION})[0] == 200
    assert _post(fixture, "/event", {"type": "account_checked", "identity": _IDENTITY,
                                     "identity_count": 1, "navigator_webdriver": False})[0] == 200


def _cases():
    yield "happy_path_completes_and_reports_no_storage", _case_happy_path
    yield "bootstrap_page_is_served_once", _case_bootstrap_once
    yield "host_header_must_match_the_bound_port", _case_host
    yield "client_must_be_ipv4_loopback", _case_client
    yield "origin_must_be_the_fixed_extension", _case_origin
    yield "bearer_token_must_match", _case_token
    yield "content_type_must_be_json", _case_content_type
    yield "body_over_32_kib_is_rejected", _case_body_limit
    yield "unknown_routes_are_rejected", _case_route
    yield "state_order_is_enforced", _case_state_order
    yield "extra_json_keys_are_rejected", _case_extra_keys
    yield "protocol_and_version_mismatch_stop", _case_version_mismatch
    yield "automation_signal_stops_the_flow", _case_webdriver
    yield "cookie_scope_and_attributes_are_enforced", _case_cookie_scope
    yield "identity_mismatch_stops_before_capture", _case_identity
    yield "session_value_is_not_retained_anywhere", _case_no_retention


def _case_happy_path():
    fixture, machine = _new_fixture()
    status, _, page = fixture.handle("GET", "/bootstrap", _HOST, "127.0.0.1", {}, b"")
    assert status == 200 and b'name="algoloom-loopback-token"' in page
    assert b'id="algoloom-consent"' in page and _TOKEN.encode() in page
    _advance_to_capture(fixture)
    status, body = _post(fixture, "/capture", _capture_payload())
    assert status == 200 and body == {"ok": True, "capture_validated": True,
                                      "identity_matched": True, "secret_store_written": False}
    assert machine.stage == STAGE_COMPLETE and machine.completed.is_set()


def _case_bootstrap_once():
    fixture, _ = _new_fixture()
    assert fixture.handle("GET", "/bootstrap", _HOST, "127.0.0.1", {}, b"")[0] == 200
    status, _, body = fixture.handle("GET", "/bootstrap", _HOST, "127.0.0.1", {}, b"")
    assert status == 410 and b"bootstrap_already_claimed" in body


def _case_host():
    fixture, _ = _new_fixture()
    for host in ("127.0.0.1:1", "localhost:%d" % _PORT, "evil.example:%d" % _PORT, ""):
        status, body = _post(fixture, "/event", _bootstrap_ready(), host=host)
        assert status == 403 and body["error"] == "transport_rejected", host


def _case_client():
    fixture, _ = _new_fixture()
    for client in ("127.0.0.2", "192.168.0.5", "::1"):
        status, body = _post(fixture, "/event", _bootstrap_ready(), client=client)
        assert status == 403 and body["error"] == "transport_rejected", client


def _case_origin():
    fixture, _ = _new_fixture()
    for origin in ("", "https://atcoder.jp", "chrome-extension://" + "b" * 32, "null"):
        headers = dict(_AUTH, origin=origin)
        status, body = _post(fixture, "/event", _bootstrap_ready(), headers=headers)
        assert status == 403 and body["error"] == "authentication_rejected", origin


def _case_token():
    fixture, _ = _new_fixture()
    for value in ("", "Bearer " + "1" * 64, _TOKEN, "Bearer " + "0" * 63):
        headers = dict(_AUTH, authorization=value)
        status, body = _post(fixture, "/event", _bootstrap_ready(), headers=headers)
        assert status == 403 and body["error"] == "authentication_rejected", value


def _case_content_type():
    fixture, _ = _new_fixture()
    for value in ("", "text/plain", "application/json; charset=utf-8"):
        headers = dict(_AUTH)
        headers["content-type"] = value
        status, body = _post(fixture, "/event", _bootstrap_ready(), headers=headers)
        assert status == 415 and body["error"] == "content_type_rejected", value


def _case_body_limit():
    fixture, _ = _new_fixture()
    oversize = json.dumps(_bootstrap_ready(extension_version="0.1.0")).encode("utf-8")
    oversize = oversize[:-1] + b',"pad":"' + b"x" * MAX_REQUEST_BYTES + b'"}'
    status, body = _post(fixture, "/event", oversize)
    assert status == 400 and body["error"] == "body_rejected"
    status, body = _post(fixture, "/event", b"not json")
    assert status == 400 and body["error"] == "body_rejected"
    status, body = _post(fixture, "/event", b'{"type":"bootstrap_ready"} {"type":"x"}')
    assert status == 400 and body["error"] == "body_rejected"
    status, body = _post(fixture, "/event", b'["bootstrap_ready"]')
    assert status == 400 and body["error"] == "body_rejected"


def _case_route():
    fixture, _ = _new_fixture()
    for method, path in (("POST", "/"), ("POST", "/bootstrap"), ("GET", "/event"),
                         ("POST", "/captures"), ("PUT", "/event")):
        status, _, body = fixture.handle(method, path, _HOST, "127.0.0.1", dict(_AUTH), b"{}")
        assert status == 404 and b"route_rejected" in body, path


def _case_state_order():
    fixture, _ = _new_fixture()
    assert _post(fixture, "/capture", _capture_payload())[1]["error"] == "capture_state_invalid"
    assert _post(fixture, "/event", {"type": "consent_confirmed",
                                     "consent_version": DEFAULT_CONSENT_VERSION})[1]["error"] \
        == "consent_state_invalid"
    assert _post(fixture, "/event", _bootstrap_ready())[0] == 200
    assert _post(fixture, "/event", _bootstrap_ready())[1]["error"] == "bootstrap_state_invalid"
    assert _post(fixture, "/event", {"type": "account_checked", "identity": _IDENTITY,
                                     "identity_count": 1,
                                     "navigator_webdriver": False})[1]["error"] \
        == "account_state_invalid"
    assert _post(fixture, "/event", {"type": "unknown"})[1]["error"] == "event_type_invalid"


def _case_extra_keys():
    fixture, _ = _new_fixture()
    payload = _bootstrap_ready()
    payload["extra"] = 1
    assert _post(fixture, "/event", payload)[1]["error"] == "bootstrap_state_invalid"
    fixture, _ = _new_fixture()
    _advance_to_capture(fixture)
    payload = _capture_payload()
    payload["extra"] = 1
    assert _post(fixture, "/capture", payload)[1]["error"] == "capture_schema_invalid"
    fixture, _ = _new_fixture()
    _advance_to_capture(fixture)
    payload = _capture_payload()
    del payload["cookie_session"]
    assert _post(fixture, "/capture", payload)[1]["error"] == "capture_schema_invalid"


def _case_version_mismatch():
    for override, reason in (
        ({"protocol_version": 2}, "protocol_version_mismatch"),
        ({"extension_version": "0.1.1"}, "extension_version_mismatch"),
        ({"consent_version": "1.1"}, "consent_version_mismatch"),
        ({"protocol_version": "1"}, "protocol_version_invalid"),
    ):
        fixture, _ = _new_fixture()
        assert _post(fixture, "/event", _bootstrap_ready(**override))[1]["error"] == reason, override


def _case_webdriver():
    fixture, _ = _new_fixture()
    assert _post(fixture, "/event",
                 _bootstrap_ready(navigator_webdriver=True))[1]["error"] == "automation_signal_detected"
    fixture, _ = _new_fixture()
    assert _post(fixture, "/event", _bootstrap_ready())[0] == 200
    assert _post(fixture, "/event", {"type": "consent_confirmed",
                                     "consent_version": DEFAULT_CONSENT_VERSION})[0] == 200
    assert _post(fixture, "/event", {"type": "account_checked", "identity": _IDENTITY,
                                     "identity_count": 1,
                                     "navigator_webdriver": True})[1]["error"] \
        == "automation_signal_detected"


def _case_cookie_scope():
    for override, reason in (
        ({"candidate_count": 2}, "cookie_scope_not_unique"),
        ({"cookie_name": "SESSION"}, "cookie_scope_not_unique"),
        ({"cookie_domain": "example.com"}, "cookie_domain_invalid"),
        ({"cookie_domain": "sub.atcoder.jp"}, "cookie_domain_invalid"),
        ({"cookie_path": "/contests"}, "cookie_attributes_invalid"),
        ({"cookie_secure": False}, "cookie_attributes_invalid"),
        ({"cookie_http_only": False}, "cookie_attributes_invalid"),
        ({"cookie_partitioned": True}, "cookie_attributes_invalid"),
        ({"cookie_value": ""}, "cookie_value_invalid"),
        ({"cookie_value": " padded "}, "cookie_value_invalid"),
        ({"cookie_value": "a;b"}, "cookie_value_invalid"),
        ({"cookie_value": "a\nb"}, "cookie_value_invalid"),
        ({"cookie_value": "REVEL_SESSION=abc"}, "cookie_value_invalid"),
        ({"cookie_value": "x" * (MAX_COOKIE_VALUE_BYTES + 1)}, "cookie_value_invalid"),
    ):
        fixture, _ = _new_fixture()
        _advance_to_capture(fixture)
        assert _post(fixture, "/capture", _capture_payload(**override))[1]["error"] == reason, override
    fixture, _ = _new_fixture()
    _advance_to_capture(fixture)
    assert _post(fixture, "/capture", _capture_payload(cookie_domain=".atcoder.jp"))[0] == 200


def _case_identity():
    fixture, _ = _new_fixture()
    assert _post(fixture, "/event", _bootstrap_ready())[0] == 200
    assert _post(fixture, "/event", {"type": "consent_confirmed",
                                     "consent_version": DEFAULT_CONSENT_VERSION})[0] == 200
    assert _post(fixture, "/event", {"type": "account_checked", "identity": "someone_else",
                                     "identity_count": 1,
                                     "navigator_webdriver": False})[1]["error"] == "identity_mismatch"
    fixture, _ = _new_fixture()
    assert _post(fixture, "/event", _bootstrap_ready())[0] == 200
    assert _post(fixture, "/event", {"type": "consent_confirmed",
                                     "consent_version": DEFAULT_CONSENT_VERSION})[0] == 200
    assert _post(fixture, "/event", {"type": "account_checked", "identity": _IDENTITY,
                                     "identity_count": 2,
                                     "navigator_webdriver": False})[1]["error"] \
        == "identity_count_invalid"


def _case_no_retention():
    secret = "sentinel-session-value-do-not-store"
    fixture, machine = _new_fixture()
    _advance_to_capture(fixture)
    status, body = _post(fixture, "/capture", _capture_payload(cookie_value=secret))
    assert status == 200 and body["secret_store_written"] is False
    for holder in (fixture, machine):
        assert secret not in json.dumps(
            {key: repr(value) for key, value in vars(holder).items()}
        ), type(holder).__name__
    # Prove by import graph, not by string search: this file cannot reach the
    # network, the filesystem, a keychain, or another process.
    tree = ast.parse(open(__file__, "r", encoding="utf-8").read())
    imported = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            imported.update(alias.name.split(".")[0] for alias in node.names)
        elif isinstance(node, ast.ImportFrom) and node.module:
            imported.add(node.module.split(".")[0])
    assert imported == {"argparse", "ast", "hmac", "json", "re",
                        "secrets", "sys", "threading", "http"}, sorted(imported)


def run_self_test():
    passed = []
    for name, case in _cases():
        case()
        passed.append(name)
    sys.stdout.write(json.dumps({
        "ok": True, "fixture_version": FIXTURE_VERSION, "protocol_version": PROTOCOL_VERSION,
        "external_connections": 0, "sockets_opened": 0,
        "cases": len(passed), "case_names": passed,
    }, indent=2) + "\n")
    return 0


def main(argv):
    parser = argparse.ArgumentParser(
        description="AlgoLoom V-12 review fixture (stores nothing, contacts nothing).")
    parser.add_argument("--extension-id", help="fixed 32-character Chrome extension ID")
    parser.add_argument("--extension-version", default=DEFAULT_EXTENSION_VERSION)
    parser.add_argument("--consent-version", default=DEFAULT_CONSENT_VERSION)
    parser.add_argument("--self-test", action="store_true",
                        help="run fixed-input checks without opening any socket")
    arguments = parser.parse_args(argv)

    if arguments.self_test:
        return run_self_test()
    if not arguments.extension_id or not EXTENSION_ID_PATTERN.match(arguments.extension_id):
        sys.stderr.write("error: --extension-id must be the fixed 32-character extension ID\n")
        return 2
    try:
        identity = input("AtCoder username you will sign in with: ").strip()
    except EOFError:
        identity = ""
    if not ACCOUNT_PATTERN.match(identity):
        sys.stderr.write("error: enter the AtCoder username you will sign in with\n")
        return 2
    try:
        return run_server(arguments.extension_id, arguments.extension_version,
                          arguments.consent_version, identity)
    except ProtocolError as error:
        sys.stderr.write("error: %s\n" % error.reason)
        return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
