"use strict";

const LOOPBACK_KEY = "v12Loopback";
const ACCOUNT_TAB_KEY = "v12AccountTab";
const TOKEN_PATTERN = /^[0-9a-f]{64}$/;
const VERSION_PATTERN = /^\d+(?:\.\d+){1,3}$/;
const ACCOUNT_PATTERN = /^[A-Za-z0-9_]{1,64}$/;
const SERVER_ERROR_PATTERN = /^[a-z0-9_]{1,96}$/;

function validLoopback(value) {
  return value !== null &&
    typeof value === "object" &&
    Number.isInteger(value.port) &&
    value.port >= 1024 &&
    value.port <= 65535 &&
    typeof value.token === "string" &&
    TOKEN_PATTERN.test(value.token) &&
    typeof value.consent_version === "string" &&
    VERSION_PATTERN.test(value.consent_version);
}

async function getLoopback() {
  const stored = await chrome.storage.session.get(LOOPBACK_KEY);
  const value = stored[LOOPBACK_KEY];
  if (!validLoopback(value)) throw new Error("loopback_not_initialized");
  return value;
}

async function loopbackFetch(path, options = {}) {
  const loopback = await getLoopback();
  const response = await fetch(`http://127.0.0.1:${loopback.port}${path}`, {
    ...options,
    cache: "no-store",
    headers: {
      Authorization: `Bearer ${loopback.token}`,
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
  });
  if (!response.ok) {
    let reason = `loopback_http_${response.status}`;
    try {
      const body = await response.json();
      if (SERVER_ERROR_PATTERN.test(body?.error || "")) reason = body.error;
    } catch (_) {
      // Keep the bounded status-only fallback.
    }
    throw new Error(reason);
  }
  return await response.json();
}

function bootstrapSender(sender) {
  try {
    const value = new URL(sender.url || "about:blank");
    return value.protocol === "http:" &&
      value.hostname === "127.0.0.1" &&
      value.pathname === "/bootstrap";
  } catch (_) {
    return false;
  }
}

function settingsSender(sender) {
  try {
    const value = new URL(sender.url || "about:blank");
    return value.origin === "https://atcoder.jp" &&
      value.pathname === "/settings" &&
      Number.isInteger(sender.tab?.id);
  } catch (_) {
    return false;
  }
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  (async () => {
    if (message === null || typeof message !== "object") throw new Error("message_invalid");

    if (message.type === "initialize") {
      if (!bootstrapSender(sender)) throw new Error("initializer_origin_invalid");
      const senderUrl = new URL(sender.url);
      const loopback = {
        port: message.port,
        token: message.token,
        consent_version: message.consent_version,
      };
      if (
        !validLoopback(loopback) ||
        Number(senderUrl.port) !== loopback.port ||
        message.extension_version !== chrome.runtime.getManifest().version
      ) throw new Error("initializer_value_invalid");

      await chrome.storage.session.clear();
      await chrome.storage.session.set({ [LOOPBACK_KEY]: loopback });
      await loopbackFetch("/event", {
        method: "POST",
        body: JSON.stringify({
          type: "bootstrap_ready",
          protocol_version: 1,
          extension_version: message.extension_version,
          consent_version: loopback.consent_version,
          navigator_webdriver: message.navigator_webdriver === true,
        }),
      });
      return await loopbackFetch("/event", {
        method: "POST",
        body: JSON.stringify({
          type: "consent_confirmed",
          consent_version: loopback.consent_version,
        }),
      });
    }

    if (message.type === "account_observed") {
      if (
        !settingsSender(sender) ||
        !ACCOUNT_PATTERN.test(message.identity || "") ||
        message.identity_count !== 1 ||
        message.navigator_webdriver === true
      ) throw new Error("account_observation_invalid");

      await loopbackFetch("/event", {
        method: "POST",
        body: JSON.stringify({
          type: "account_checked",
          identity: message.identity,
          identity_count: message.identity_count,
          navigator_webdriver: false,
        }),
      });
      await chrome.storage.session.set({ [ACCOUNT_TAB_KEY]: sender.tab.id });
      return { ok: true };
    }

    if (message.type === "capture_session") {
      if (!settingsSender(sender)) throw new Error("capture_sender_invalid");
      const stored = await chrome.storage.session.get(ACCOUNT_TAB_KEY);
      if (stored[ACCOUNT_TAB_KEY] !== sender.tab.id) throw new Error("account_gate_missing");

      const candidates = await chrome.cookies.getAll({
        url: "https://atcoder.jp/",
        name: "REVEL_SESSION",
        path: "/",
        secure: true,
      });
      const allowed = candidates.filter((cookie) =>
        cookie.name === "REVEL_SESSION" &&
        new Set(["atcoder.jp", ".atcoder.jp"]).has(cookie.domain) &&
        cookie.path === "/" &&
        cookie.secure === true &&
        cookie.partitionKey === undefined
      );
      if (candidates.length !== 1 || allowed.length !== 1) {
        throw new Error("cookie_scope_not_unique");
      }

      const cookie = allowed[0];
      const response = await loopbackFetch("/capture", {
        method: "POST",
        body: JSON.stringify({
          candidate_count: candidates.length,
          cookie_name: cookie.name,
          cookie_domain: cookie.domain,
          cookie_path: cookie.path,
          cookie_secure: cookie.secure,
          cookie_http_only: cookie.httpOnly,
          cookie_host_only: cookie.hostOnly,
          cookie_session: cookie.session,
          cookie_partitioned: cookie.partitionKey !== undefined,
          cookie_value: cookie.value,
          observed_identity: message.observed_identity,
        }),
      });
      await chrome.storage.session.remove([LOOPBACK_KEY, ACCOUNT_TAB_KEY]);
      return response;
    }

    throw new Error("message_type_invalid");
  })()
    .then((value) => sendResponse({ ok: true, value }))
    .catch((error) => sendResponse({
      ok: false,
      error: error instanceof Error && SERVER_ERROR_PATTERN.test(error.message)
        ? error.message
        : "extension_failure",
    }));
  return true;
});
