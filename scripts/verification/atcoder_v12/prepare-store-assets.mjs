#!/usr/bin/env node

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { execFileSync, spawnSync } from "node:child_process";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_ROOT = path.dirname(fileURLToPath(import.meta.url));
const REPOSITORY_ROOT = path.resolve(SCRIPT_ROOT, "../../..");
const CONSENT_TEMPLATE = path.join(SCRIPT_ROOT, "helper", "consent.html");
const PROMO_SOURCE = path.join(SCRIPT_ROOT, "listing", "small-promo.html");
const CHROME = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
const TOKEN_PLACEHOLDER = "{{TOKEN}}";
const CONSENT_PLACEHOLDER = "{{CONSENT_VERSION}}";
const HASH_PATTERN = /^[0-9a-f]{64}$/;
const REVISION_PATTERN = /^[0-9a-f]{40,64}$/;

function fail(reason) {
  if (!/^[a-z0-9_]{1,96}$/.test(reason)) reason = "store_asset_preparation_failed";
  process.stderr.write(`${JSON.stringify({ ok: false, error: reason })}\n`);
  process.exit(1);
}

function inside(parent, child) {
  const relative = path.relative(parent, child);
  return relative === "" || (!relative.startsWith(`..${path.sep}`) && relative !== "..");
}

function sha256(data) {
  return crypto.createHash("sha256").update(data).digest("hex");
}

function checkedBuildRoot(argument) {
  if (!path.isAbsolute(argument) || argument === path.parse(argument).root) fail("build_root_invalid");
  const resolved = fs.realpathSync(argument);
  const info = fs.statSync(resolved);
  if (!info.isDirectory() || inside(REPOSITORY_ROOT, resolved) || (info.mode & 0o077) !== 0) {
    fail("build_root_scope_invalid");
  }
  const index = JSON.parse(fs.readFileSync(path.join(resolved, "build-index.json"), "utf8"));
  const revision = execFileSync("git", ["rev-parse", "HEAD"], {
    cwd: REPOSITORY_ROOT, encoding: "utf8", env: process.env,
  }).trim();
  const dirty = execFileSync("git", ["status", "--porcelain"], {
    cwd: REPOSITORY_ROOT, encoding: "utf8", env: process.env,
  }).trim() !== "";
  if (!index.campaign_ready || dirty || index.source_revision !== revision || !REVISION_PATTERN.test(revision)) {
    fail("build_revision_not_clean");
  }
  return { resolved, index, revision };
}

function pngDimensions(data) {
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
  if (data.length < 24 || !data.subarray(0, 8).equals(signature)) fail("asset_png_invalid");
  return { width: data.readUInt32BE(16), height: data.readUInt32BE(20) };
}

function artifact(filePath, alias, expectedWidth, expectedHeight) {
  const data = fs.readFileSync(filePath);
  const dimensions = pngDimensions(data);
  if (dimensions.width !== expectedWidth || dimensions.height !== expectedHeight) fail("asset_dimensions_invalid");
  return {
    alias,
    file: path.basename(filePath),
    width: dimensions.width,
    height: dimensions.height,
    sha256: sha256(data),
    bytes: data.length,
  };
}

function capture(source, destination, width, height, profile) {
  fs.mkdirSync(profile, { mode: 0o700 });
  const result = spawnSync(CHROME, [
    "--headless=new",
    "--disable-background-networking",
    "--disable-component-update",
    "--disable-default-apps",
    "--disable-extensions",
    "--disable-gpu",
    "--disable-sync",
    "--hide-scrollbars",
    "--no-default-browser-check",
    "--no-first-run",
    "--force-device-scale-factor=1",
    `--user-data-dir=${profile}`,
    `--window-size=${width},${height}`,
    `--screenshot=${destination}`,
    pathToFileURL(source).href,
  ], { env: {}, stdio: "pipe", timeout: 10_000, killSignal: "SIGTERM" });
  const timedOutAfterCapture = result.error?.code === "ETIMEDOUT" && fs.existsSync(destination);
  if ((result.error && !timedOutAfterCapture) || (!timedOutAfterCapture && result.status !== 0)) {
    fail("chrome_capture_failed");
  }
  fs.chmodSync(destination, 0o600);
}

function main() {
  if (process.argv.length !== 3) fail("usage_invalid");
  if (!fs.existsSync(CHROME)) fail("chrome_unavailable");
  const { resolved: buildRoot, index, revision } = checkedBuildRoot(process.argv[2]);
  const listingRoot = path.join(buildRoot, "listing");
  if (fs.existsSync(listingRoot)) fail("listing_output_exists");
  fs.mkdirSync(listingRoot, { mode: 0o700 });
  const stage = path.join(listingRoot, ".capture-stage");
  fs.mkdirSync(stage, { mode: 0o700 });

  try {
    const consentVersion = index.versions?.consent;
    if (!/^\d+(?:\.\d+){1,3}$/.test(consentVersion || "")) fail("consent_version_invalid");
    const template = fs.readFileSync(CONSENT_TEMPLATE, "utf8");
    const rendered = template
      .replaceAll(TOKEN_PLACEHOLDER, "0".repeat(64))
      .replaceAll(CONSENT_PLACEHOLDER, consentVersion);
    if (rendered.includes(TOKEN_PLACEHOLDER) || rendered.includes(CONSENT_PLACEHOLDER)) fail("consent_template_unresolved");
    const preview = path.join(stage, "consent-preview.html");
    fs.writeFileSync(preview, rendered, { mode: 0o600, flag: "wx" });

    const screenshot = path.join(listingRoot, "screenshot-consent-1280x800.png");
    capture(preview, screenshot, 1280, 800, path.join(stage, "chrome-consent"));
    const promo = path.join(listingRoot, "small-promo-440x280.png");
    capture(PROMO_SOURCE, promo, 440, 280, path.join(stage, "chrome-promo"));

    const targetVersion = index.versions?.extension_target;
    const upload = index.extension_upload_packages?.find((item) => item.alias === `extension-upload-${targetVersion}`);
    if (!upload || !HASH_PATTERN.test(upload.sha256)) fail("target_upload_missing");
    const uploadPath = path.join(buildRoot, "artifacts", `algoloom-v12-extension-${targetVersion}.zip`);
    const icon = execFileSync("unzip", ["-p", uploadPath, "icon128.png"], { env: {}, stdio: ["ignore", "pipe", "pipe"] });
    const iconPath = path.join(listingRoot, "icon128.png");
    fs.writeFileSync(iconPath, icon, { mode: 0o600, flag: "wx" });

    const assets = [
      artifact(iconPath, "extension-icon", 128, 128),
      artifact(promo, "small-promo", 440, 280),
      artifact(screenshot, "consent-screenshot", 1280, 800),
    ];
    const chromeVersion = execFileSync(CHROME, ["--version"], { encoding: "utf8", env: {} }).trim();
    const listingIndex = {
      schema_version: 1,
      preparation_scope: "V-12-cws-listing-assets",
      source_revision: revision,
      extension_version: targetVersion,
      extension_upload_sha256: upload.sha256,
      capture: {
        chrome_version: chromeVersion,
        source: "file-only rendering; no extension execution and no external account",
        consent_ui_sha256: sha256(fs.readFileSync(CONSENT_TEMPLATE)),
        promo_source_sha256: sha256(fs.readFileSync(PROMO_SOURCE)),
      },
      assets,
      secrets_present: false,
    };
    fs.writeFileSync(path.join(listingRoot, "listing-index.json"), `${JSON.stringify(listingIndex, null, 2)}\n`, {
      mode: 0o600, flag: "wx",
    });
    fs.rmSync(stage, { recursive: true });
    process.stdout.write(`${JSON.stringify({ ok: true, assets: assets.length, listing_root: listingRoot })}\n`);
  } catch (error) {
    fail(/^[a-z0-9_]{1,96}$/.test(error?.message || "") ? error.message : "store_asset_preparation_failed");
  }
}

main();
