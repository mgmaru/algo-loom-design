import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
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
  assert.match(STORE_ASSET_PREPARATION, /file-only rendering; no extension execution and no external account/);
  assert.doesNotMatch(STORE_ASSET_PREPARATION, /https:\/\/atcoder\.jp|chrome-extension:\/\//);
});
