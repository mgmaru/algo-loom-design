import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import vm from "node:vm";
import { HELPER_BUILD_ENV, HELPER_BUILD_FLAGS } from "./atcoder_v12/helper-build.mjs";

const ROOT = path.join(path.dirname(fileURLToPath(import.meta.url)), "atcoder_v12");
const EXTENSION = path.join(ROOT, "extension");
const MANIFEST = JSON.parse(fs.readFileSync(path.join(EXTENSION, "manifest.template.json"), "utf8"));
const WORKER = fs.readFileSync(path.join(EXTENSION, "service_worker.js"), "utf8");
const ATCODER = fs.readFileSync(path.join(EXTENSION, "atcoder.js"), "utf8");
const BOOTSTRAP = fs.readFileSync(path.join(EXTENSION, "bootstrap.js"), "utf8");
const HELPER_SOURCES = fs.readdirSync(path.join(ROOT, "helper"))
  .filter((name) => name.endsWith(".go"))
  .map((name) => fs.readFileSync(path.join(ROOT, "helper", name), "utf8"))
  .join("\n");
const CONSENT_PAGE = fs.readFileSync(path.join(ROOT, "helper", "consent.html"), "utf8");
const STORE_ASSET_PREPARATION = fs.readFileSync(path.join(ROOT, "prepare-store-assets.mjs"), "utf8");
const PREPARATION = fs.readFileSync(path.join(ROOT, "prepare.mjs"), "utf8");
const REVIEW_FIXTURE_PATH = path.join(ROOT, "algoloom_v12_review_fixture.py");
const REVIEW_FIXTURE = fs.readFileSync(REVIEW_FIXTURE_PATH, "utf8");

function fixtureSelfTest(scriptPath) {
  const output = execFileSync("python3", [scriptPath, "--self-test"], {
    encoding: "utf8", env: {}, stdio: ["ignore", "pipe", "pipe"],
  });
  return JSON.parse(output);
}


// 同意画面のcontent scriptを実際に評価する。helperは動的な待受番号を使うため、
// bootstrap URLには必ずポートが付く。文字列検査ではこの条件を確認できない。
function runBootstrapAt(url) {
  const target = new URL(url);
  class HTMLButtonElement {
    constructor() { this.disabled = false; this.textContent = "同意してAtCoderへ進む"; this.onClick = null; }
    addEventListener(type, handler) { if (type === "click") this.onClick = handler; }
  }
  const button = new HTMLButtonElement();
  const meta = {
    'meta[name="algoloom-loopback-token"]': { content: "t".repeat(64), remove() {} },
    'meta[name="algoloom-consent-version"]': { content: "1.0", remove() {} },
  };
  const navigated = [];
  const sent = [];
  const sandbox = {
    HTMLButtonElement,
    location: {
      protocol: target.protocol, hostname: target.hostname, origin: target.origin,
      pathname: target.pathname, port: target.port,
      replace: (to) => navigated.push(to),
    },
    document: {
      body: {},
      querySelector: (selector) => meta[selector] ?? null,
      getElementById: (id) => (id === "algoloom-consent" ? button : null),
    },
    navigator: { webdriver: false },
    chrome: {
      runtime: {
        sendMessage: async (message) => { sent.push(message); return { ok: true }; },
        getManifest: () => ({ version: "0.1.0" }),
      },
    },
  };
  vm.createContext(sandbox);
  vm.runInContext(BOOTSTRAP, sandbox);
  return { button, navigated, sent };
}

test("V-12 extension has one purpose and the exact minimal permission set", () => {
  assert.equal(MANIFEST.manifest_version, 3);
  assert.equal(MANIFEST.version, "__VERSION__");
  assert.deepEqual(MANIFEST.permissions, ["cookies", "storage"]);
  assert.deepEqual(MANIFEST.host_permissions, [
    "https://atcoder.jp/*",
    "http://127.0.0.1/*",
  ]);
  const serialized = JSON.stringify(MANIFEST);
  for (const forbidden of [
    "debugger", "webRequest", "tabs", "scripting", "nativeMessaging", "<all_urls>",
  ]) assert.equal(serialized.includes(`"${forbidden}"`), false);
});

test("V-12 extension only reads one scoped AtCoder session", () => {
  assert.match(WORKER, /chrome\.cookies\.getAll\(\{/);
  assert.match(WORKER, /url: "https:\/\/atcoder\.jp\/"/);
  assert.match(WORKER, /name: "REVEL_SESSION"/);
  assert.match(WORKER, /path: "\/"/);
  assert.match(WORKER, /secure: true/);
  assert.match(WORKER, /candidates\.length !== 1 \|\| allowed\.length !== 1/);
  assert.doesNotMatch(WORKER, /chrome\.cookies\.(?:set|remove|getAllCookieStores)/);
  assert.doesNotMatch(WORKER, /console\.(?:log|error|warn|debug)/);
});

test("V-12 extension does not automate login, Turnstile, or submission", () => {
  for (const source of [WORKER, ATCODER, BOOTSTRAP]) {
    assert.doesNotMatch(source, /requestSubmit\s*\(|\.submit\s*\(|\.click\s*\(/);
    assert.doesNotMatch(source, /cf-turnstile-response|HTMLFormElement|remote-debugging|webdriver\s*=\s*false/i);
  }
  assert.deepEqual(MANIFEST.content_scripts[1].matches, ["https://atcoder.jp/settings*"]);
  assert.doesNotMatch(JSON.stringify(MANIFEST.content_scripts), /\/login/);
});

test("loopback protocol keeps the Cookie out of URL, argv, environment, and public output", () => {
  assert.doesNotMatch(WORKER, /cookie_value.*console|URLSearchParams.*cookie/i);
  assert.match(HELPER_SOURCES, /command\.Stdin = bytes\.NewReader\(input\)/);
  assert.match(HELPER_SOURCES, /command\.Env = \[\]string\{\}/);
  assert.match(HELPER_SOURCES, /SecretValuesInOutput.*json:"secret_values_in_output"/);
  assert.doesNotMatch(HELPER_SOURCES, /fmt\.(?:Print|Printf|Println)\([^\n]*CookieValue/);
});

test("build preparation has no publisher credential or signing secret input", () => {
  const prepare = fs.readFileSync(path.join(ROOT, "prepare.mjs"), "utf8");
  assert.doesNotMatch(prepare, /client_secret|private_key|publisher_password|refresh_token/i);
  assert.match(prepare, /signed_extension_artifacts: \[\]/);
  assert.match(prepare, /campaign_ready: !git\.dirty/);
  assert.match(prepare, /output_parent_not_owner_only/);
});

test("first-login keeps standard installation and authentication in one helper invocation", () => {
  assert.match(HELPER_SOURCES, /case "first-login":/);
  assert.match(HELPER_SOURCES, /manifest\.Extension\.ListingURL != \*listingURL/);
  assert.match(HELPER_SOURCES, /detectInstalledExtension\(\*setupRoot/);
  assert.match(HELPER_SOURCES, /finalizeTemplate\(\*setupRoot/);
  assert.match(HELPER_SOURCES, /cloneTemplate\(\*templateRoot/);
  assert.match(HELPER_SOURCES, /serveArguments := \[\]string/);
  assert.doesNotMatch(HELPER_SOURCES, /--load-extension|remote-debugging|--headless/);
});

test("listing screenshot source is the embedded consent UI and contains no account data", () => {
  assert.match(HELPER_SOURCES, /go:embed consent\.html/);
  assert.match(CONSENT_PAGE, /algoloom-loopback-token/);
  assert.match(CONSENT_PAGE, /algoloom-consent-version/);
  assert.match(CONSENT_PAGE, /id="algoloom-consent"/);
  assert.match(CONSENT_PAGE, /TECHNICAL VERIFICATION BETA/);
  assert.doesNotMatch(CONSENT_PAGE, /fixture_account|REVEL_SESSION=/);
});

test("store asset preparation is local, fixed-size, and bound to a clean build", () => {
  assert.match(STORE_ASSET_PREPARATION, /index\.campaign_ready/);
  assert.match(STORE_ASSET_PREPARATION, /git", \["status", "--porcelain"\]/);
  assert.match(STORE_ASSET_PREPARATION, /1280, 800/);
  assert.match(STORE_ASSET_PREPARATION, /440, 280/);
  assert.match(STORE_ASSET_PREPARATION, /index\.versions\?\.extension_target/);
  assert.match(STORE_ASSET_PREPARATION, /extension-upload-\$\{targetVersion\}/);
  assert.match(STORE_ASSET_PREPARATION, /file-only rendering; no extension execution and no external account/);
  assert.doesNotMatch(STORE_ASSET_PREPARATION, /https:\/\/atcoder\.jp|chrome-extension:\/\//);
});

test("review fixture speaks the helper protocol under fixed inputs and stores nothing", () => {
  const result = fixtureSelfTest(REVIEW_FIXTURE_PATH);
  assert.equal(result.ok, true);
  assert.equal(result.protocol_version, 1);
  assert.equal(result.external_connections, 0);
  assert.equal(result.sockets_opened, 0);
  assert.ok(result.cases >= 16, `expected the full case set, saw ${result.cases}`);
  for (const required of [
    "host_header_must_match_the_bound_port",
    "client_must_be_ipv4_loopback",
    "origin_must_be_the_fixed_extension",
    "bearer_token_must_match",
    "body_over_32_kib_is_rejected",
    "state_order_is_enforced",
    "cookie_scope_and_attributes_are_enforced",
    "session_value_is_not_retained_anywhere",
  ]) assert.ok(result.case_names.includes(required), required);
});

test("review fixture still runs when the file carries the download quarantine attribute", (context) => {
  if (process.platform !== "darwin") return context.skip("macOS only");
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "v12-quarantine-"));
  try {
    const quarantined = path.join(directory, "algoloom_v12_review_fixture.py");
    fs.copyFileSync(REVIEW_FIXTURE_PATH, quarantined);
    const stamp = `0001;68aa6f00;Google Chrome;${crypto.randomUUID()}`;
    execFileSync("xattr", ["-w", "com.apple.quarantine", stamp, quarantined], { stdio: "pipe" });
    const attributes = execFileSync("xattr", [quarantined], { encoding: "utf8", stdio: "pipe" });
    assert.match(attributes, /com\.apple\.quarantine/);
    assert.equal(fixtureSelfTest(quarantined).ok, true);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});

test("review fixture cannot reach the network, the filesystem, or another process", () => {
  const imported = new Set();
  for (const match of REVIEW_FIXTURE.matchAll(/^import\s+([\w.]+)|^from\s+([\w.]+)\s+import/gm)) {
    imported.add((match[1] || match[2]).split(".")[0]);
  }
  assert.deepEqual([...imported].sort(), [
    "argparse", "ast", "hmac", "http", "json", "re", "secrets", "sys", "threading",
  ]);
  assert.doesNotMatch(REVIEW_FIXTURE, /urllib|requests|socket\.socket|keyring|Keychain|subprocess/);
  assert.doesNotMatch(REVIEW_FIXTURE, /open\(.*["']w/);
  assert.match(REVIEW_FIXTURE, /"secret_store_written": False/);
  assert.match(REVIEW_FIXTURE, /hmac\.compare_digest/);
});

test("build preparation ships both reviewer routes from one reproducible bundle", () => {
  assert.match(PREPARATION, /review_bundle_not_reproducible/);
  assert.match(PREPARATION, /algoloom-v12-review-darwin-arm64\.tar\.gz/);
  assert.match(PREPARATION, /route: "curl-and-tar"/);
  assert.match(PREPARATION, /route: "python3-single-file"/);
  assert.match(PREPARATION, /quarantine_safe: true/);
  assert.match(PREPARATION, /review_delivery: reviewDelivery/);
  assert.match(PREPARATION, /review_fixture: sha256\(fs\.readFileSync\(REVIEW_FIXTURE_SOURCE\)\)/);
  assert.match(PREPARATION, /FIXED_UNIX_TIME/);
  assert.doesNotMatch(PREPARATION, /xattr|spctl|codesign|--deep/);
});

test("V-12 consent page reacts on the dynamic loopback port, not only on the default port", async () => {
  for (const url of [
    "http://127.0.0.1:53673/bootstrap",
    "http://127.0.0.1:1024/bootstrap",
    "http://127.0.0.1/bootstrap",
  ]) {
    const { button, navigated, sent } = runBootstrapAt(url);
    await new Promise((resolve) => setImmediate(resolve));
    assert.notEqual(button.onClick, null, `handlerが付かない: ${url}`);
    await button.onClick();
    assert.deepEqual(navigated, ["https://atcoder.jp/settings"], `遷移しない: ${url}`);
    assert.equal(sent.length, 1);
    assert.equal(sent[0].type, "initialize");
    assert.equal(sent[0].port, Number(new URL(url).port));
  }
});

test("V-12 consent page stays inert outside the loopback bootstrap page", async () => {
  for (const url of [
    "https://atcoder.jp/bootstrap",
    "http://127.0.0.2:53673/bootstrap",
    "http://127.0.0.1:53673/settings",
  ]) {
    const { button, navigated, sent } = runBootstrapAt(url);
    await new Promise((resolve) => setImmediate(resolve));
    assert.equal(button.onClick, null, `handlerが付いてしまう: ${url}`);
    assert.deepEqual(navigated, []);
    assert.deepEqual(sent, []);
  }
});

test("helper build embeds no commit id, so its hash tracks behaviour not history", () => {
  // campaign manifestはhelperのhashを「挙動が変わったか」の判定に使う。
  // commit idが埋まると、helperのsourceが同じでもhashが変わり、判定が代理として
  // 成り立たなくなる。ここではソースの文字列ではなく、実際にbuildしたバイト列を見る。
  let revision;
  try {
    revision = execFileSync("git", ["rev-parse", "HEAD"], { cwd: ROOT, encoding: "utf8" }).trim();
  } catch {
    return; // gitのないtarball展開では判定できない
  }
  assert.match(revision, /^[0-9a-f]{40}$/);

  const workspace = fs.mkdtempSync(path.join(os.tmpdir(), "algoloom-helper-build-"));
  try {
    const output = path.join(workspace, "helper");
    execFileSync("go", [
      "build", ...HELPER_BUILD_FLAGS,
      "-ldflags", "-s -w -X main.helperVersion=0.0.0-test",
      "-o", output, ".",
    ], {
      cwd: path.join(ROOT, "helper"),
      env: { ...process.env, ...HELPER_BUILD_ENV },
      stdio: "pipe",
    });
    const binary = fs.readFileSync(output);
    for (const form of [revision, revision.slice(0, 12)]) {
      assert.equal(binary.includes(Buffer.from(form, "utf8")), false,
        `built helper must not contain the commit id ${form}`);
    }
  } finally {
    fs.rmSync(workspace, { recursive: true, force: true });
  }
});

// atcoder.jsを実際に評価する。ソースに文字列が現れるかではなく、
// どのページで何を送るかを見る。bootstrap.jsと同じ環境依存（origin、pathname、
// message passing）を持つため、同じ扱いにする。判断の経緯はADR-0008を参照する。
function runAtCoderScriptAt(href, options = {}) {
  const { identities = ["verifieraccount"], webdriver = false, identityOk = true, captureOk = true } = options;
  const target = new URL(href);
  const sent = [];
  const nodes = new Map();
  const makeNode = () => ({ id: "", style: { cssText: "" }, textContent: "" });
  const sandbox = {
    URL,
    location: { origin: target.origin, pathname: target.pathname },
    navigator: { webdriver },
    document: {
      scripts: identities.map((name) => ({ textContent: `var userScreenName = "${name}";` })),
      createElement: () => makeNode(),
      getElementById: (id) => nodes.get(id) ?? null,
      body: { prepend: (node) => nodes.set(node.id, node) },
    },
    chrome: {
      runtime: {
        sendMessage: async (message) => {
          sent.push(message);
          if (message.type === "account_observed") return { ok: identityOk };
          if (message.type === "capture_session") return { ok: captureOk };
          return { ok: false };
        },
      },
    },
  };
  vm.createContext(sandbox);
  vm.runInContext(ATCODER, sandbox);
  return { sent, status: () => nodes.get("algoloom-v12-status") ?? null };
}

const settle = () => new Promise((resolve) => setImmediate(() => setImmediate(resolve)));

test("V-12 settings script confirms the account and asks for the session, in that order", async () => {
  const run = runAtCoderScriptAt("https://atcoder.jp/settings");
  await settle();
  assert.deepEqual(run.sent.map((message) => message.type), ["account_observed", "capture_session"]);
  assert.equal(run.sent[0].identity, "verifieraccount");
  assert.equal(run.sent[0].identity_count, 1);
  assert.equal(run.sent[0].navigator_webdriver, false);
  assert.equal(run.sent[1].observed_identity, "verifieraccount");
  assert.match(run.status().textContent, /端末内で確認しました/);
});

test("V-12 settings script stays inert outside the settings page", async () => {
  for (const url of [
    "https://atcoder.jp/home",
    "https://atcoder.jp/settings/extra",
    "https://example.invalid/settings",
    "http://atcoder.jp/settings",
  ]) {
    const run = runAtCoderScriptAt(url);
    await settle();
    assert.deepEqual(run.sent, [], `動いてしまう: ${url}`);
    assert.equal(run.status(), null, `DOMへ書き込む: ${url}`);
  }
});

test("V-12 settings script never asks for the session when identity is not unique", async () => {
  for (const options of [
    { identities: ["one", "two"] },
    { identities: [] },
    { webdriver: true },
    { identityOk: false },
  ]) {
    const run = runAtCoderScriptAt("https://atcoder.jp/settings", options);
    await settle();
    assert.equal(run.sent.some((message) => message.type === "capture_session"), false,
      `capture_sessionを送ってしまう: ${JSON.stringify(options)}`);
  }
});

// service_worker.jsを実際に評価する。動的な待受番号を持つ送信元を受け付けるか、
// それ以外を拒むかは、bootstrap.jsで止まったのと同じ種類の判定である。
function runServiceWorker() {
  const listeners = [];
  const storage = new Map();
  const requests = [];
  const sandbox = {
    URL,
    fetch: async (url, options) => {
      requests.push({ url, options });
      return { ok: true, status: 200, json: async () => ({ ok: true }) };
    },
    chrome: {
      runtime: {
        onMessage: { addListener: (listener) => listeners.push(listener) },
        getManifest: () => ({ version: "0.1.1" }),
      },
      storage: {
        session: {
          get: async (key) => (storage.has(key) ? { [key]: storage.get(key) } : {}),
          set: async (values) => { for (const [key, value] of Object.entries(values)) storage.set(key, value); },
          remove: async (keys) => { for (const key of [].concat(keys)) storage.delete(key); },
          clear: async () => storage.clear(),
        },
      },
      cookies: { getAll: async () => [] },
    },
  };
  vm.createContext(sandbox);
  vm.runInContext(WORKER, sandbox);
  assert.equal(listeners.length, 1, "message listenerが登録されない");
  const send = (message, sender) => new Promise((resolve) => listeners[0](message, sender, resolve));
  return { send, requests, storage };
}

test("V-12 service worker initializes from a dynamic loopback port and refuses other senders", async () => {
  for (const port of [53673, 1024, 65535]) {
    const worker = runServiceWorker();
    const answer = await worker.send({
      type: "initialize", port, token: "a".repeat(64), consent_version: "1.0", extension_version: "0.1.1",
    }, { url: `http://127.0.0.1:${port}/bootstrap` });
    assert.equal(answer.ok, true, `port ${port}で初期化できない: ${answer.error}`);
    assert.equal(worker.requests[0].url, `http://127.0.0.1:${port}/event`);
    assert.equal(worker.storage.get("v12Loopback").port, port);
  }
});

test("V-12 service worker refuses senders and values that do not match the loopback", async () => {
  const valid = { type: "initialize", port: 53673, token: "a".repeat(64), consent_version: "1.0", extension_version: "0.1.1" };
  const cases = [
    ["送信元がloopbackでない", valid, { url: "https://atcoder.jp/bootstrap" }, "initializer_origin_invalid"],
    ["送信元のpathが違う", valid, { url: "http://127.0.0.1:53673/settings" }, "initializer_origin_invalid"],
    ["送信元のportと申告portが違う", valid, { url: "http://127.0.0.1:1024/bootstrap" }, "initializer_value_invalid"],
    ["拡張機能の版が違う", { ...valid, extension_version: "0.1.0" }, { url: "http://127.0.0.1:53673/bootstrap" }, "initializer_value_invalid"],
    ["tokenが64桁の16進でない", { ...valid, token: "short" }, { url: "http://127.0.0.1:53673/bootstrap" }, "initializer_value_invalid"],
    ["知らないmessage type", { type: "capture_everything" }, { url: "http://127.0.0.1:53673/bootstrap" }, "message_type_invalid"],
  ];
  for (const [label, message, sender, expected] of cases) {
    const worker = runServiceWorker();
    const answer = await worker.send(message, sender);
    assert.equal(answer.ok, false, `通ってしまう: ${label}`);
    assert.equal(answer.error, expected, `停止理由が違う: ${label}`);
    assert.deepEqual(worker.requests, [], `外部へ出てしまう: ${label}`);
  }
});
