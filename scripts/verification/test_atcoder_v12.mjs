import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

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
