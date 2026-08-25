#!/usr/bin/env node

import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import process from "node:process";
import zlib from "node:zlib";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const SCRIPT_ROOT = path.dirname(fileURLToPath(import.meta.url));
const REPOSITORY_ROOT = path.resolve(SCRIPT_ROOT, "../../..");
const EXTENSION_SOURCE = path.join(SCRIPT_ROOT, "extension");
const HELPER_SOURCE = path.join(SCRIPT_ROOT, "helper");
const KEYCHAIN_SOURCE = path.join(SCRIPT_ROOT, "keychain", "atcoder_v12_keychain.swift");
const CONSENT_SOURCE = path.join(SCRIPT_ROOT, "consent-v1.0.ja.md");
const EXTENSION_VERSIONS = ["0.1.0", "0.1.1"];
const HELPER_VERSION = "0.1.0";
const FIXED_TIME = new Date("2026-08-26T00:00:00.000Z");
const HASH_PATTERN = /^[0-9a-f]{64}$/;

function fail(reason) {
  if (!/^[a-z0-9_]{1,96}$/.test(reason)) reason = "preparation_failed";
  process.stderr.write(`${JSON.stringify({ ok: false, error: reason })}\n`);
  process.exit(1);
}

function inside(parent, child) {
  const relative = path.relative(parent, child);
  return relative === "" || (!relative.startsWith(`..${path.sep}`) && relative !== "..");
}

function validateOutputRoot(outputRoot) {
  if (!path.isAbsolute(outputRoot) || outputRoot === path.parse(outputRoot).root) {
    fail("output_path_invalid");
  }
  const resolved = path.resolve(outputRoot);
  if (inside(REPOSITORY_ROOT, resolved) || fs.existsSync(resolved)) {
    fail("output_scope_invalid");
  }
  let parent;
  try {
    parent = fs.realpathSync(path.dirname(resolved));
  } catch (_) {
    fail("output_parent_unavailable");
  }
  const info = fs.statSync(parent);
  if (!info.isDirectory() || (info.mode & 0o077) !== 0) {
    fail("output_parent_not_owner_only");
  }
  return resolved;
}

function sha256(data) {
  return crypto.createHash("sha256").update(data).digest("hex");
}

function artifact(filePath, alias, extra = {}) {
  const data = fs.readFileSync(filePath);
  return { alias, ...extra, sha256: sha256(data), bytes: data.length };
}

function sourceTreeHash(root) {
  const files = [];
  const visit = (directory) => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
      const absolute = path.join(directory, entry.name);
      if (entry.isSymbolicLink()) fail("source_symlink_rejected");
      if (entry.isDirectory()) visit(absolute);
      else if (entry.isFile()) files.push(absolute);
    }
  };
  visit(root);
  const digest = crypto.createHash("sha256");
  for (const filePath of files) {
    digest.update(path.relative(root, filePath).split(path.sep).join("/"));
    digest.update("\0");
    digest.update(sha256(fs.readFileSync(filePath)));
    digest.update("\n");
  }
  return digest.digest("hex");
}

function crc32(buffer) {
  let value = 0xffffffff;
  for (const byte of buffer) {
    value ^= byte;
    for (let bit = 0; bit < 8; bit += 1) {
      value = (value >>> 1) ^ ((value & 1) ? 0xedb88320 : 0);
    }
  }
  return (value ^ 0xffffffff) >>> 0;
}

function pngChunk(type, data) {
  const typeBuffer = Buffer.from(type, "ascii");
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  const checksum = Buffer.alloc(4);
  checksum.writeUInt32BE(crc32(Buffer.concat([typeBuffer, data])));
  return Buffer.concat([length, typeBuffer, data, checksum]);
}

function iconPNG() {
  const width = 128;
  const height = 128;
  const rows = [];
  for (let y = 0; y < height; y += 1) {
    const row = Buffer.alloc(1 + width * 4);
    for (let x = 0; x < width; x += 1) {
      const offset = 1 + x * 4;
      const center = x >= 28 && x <= 99 && y >= 24 && y <= 103;
      const thread = Math.abs(x - y) <= 5 || Math.abs((127 - x) - y) <= 5;
      const color = center && thread ? [255, 255, 255] : center ? [37, 99, 235] : [15, 23, 42];
      row.set([...color, 255], offset);
    }
    rows.push(row);
  }
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header.set([8, 6, 0, 0, 0], 8);
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    pngChunk("IHDR", header),
    pngChunk("IDAT", zlib.deflateSync(Buffer.concat(rows), { level: 9 })),
    pngChunk("IEND", Buffer.alloc(0)),
  ]);
}

function copyExtension(stage, version) {
  fs.mkdirSync(stage, { mode: 0o700 });
  const template = JSON.parse(fs.readFileSync(path.join(EXTENSION_SOURCE, "manifest.template.json"), "utf8"));
  template.version = version;
  if (JSON.stringify(template).includes("__VERSION__")) fail("manifest_version_unresolved");
  fs.writeFileSync(path.join(stage, "manifest.json"), `${JSON.stringify(template, null, 2)}\n`, { mode: 0o600 });
  for (const name of ["atcoder.js", "bootstrap.js", "service_worker.js"]) {
    fs.copyFileSync(path.join(EXTENSION_SOURCE, name), path.join(stage, name));
    fs.chmodSync(path.join(stage, name), 0o600);
  }
  fs.writeFileSync(path.join(stage, "icon128.png"), iconPNG(), { mode: 0o600 });
  for (const name of fs.readdirSync(stage)) fs.utimesSync(path.join(stage, name), FIXED_TIME, FIXED_TIME);
}

function buildExtension(outputRoot, version) {
  const stage = path.join(outputRoot, "stage", `extension-${version}`);
  copyExtension(stage, version);
  const output = path.join(outputRoot, "artifacts", `algoloom-v12-extension-${version}.zip`);
  const names = fs.readdirSync(stage).sort();
  execFileSync("zip", ["-X", "-q", output, ...names], { cwd: stage, env: {}, stdio: "pipe" });
  fs.chmodSync(output, 0o600);
  return artifact(output, `extension-upload-${version}`);
}

function buildHelper(outputRoot) {
  const output = path.join(outputRoot, "artifacts", "algoloom-v12-helper-darwin-arm64");
  execFileSync("go", [
    "build", "-trimpath", "-ldflags", `-s -w -X main.helperVersion=${HELPER_VERSION}`,
    "-o", output, ".",
  ], {
    cwd: HELPER_SOURCE,
    env: { ...process.env, CGO_ENABLED: "0", GOOS: "darwin", GOARCH: "arm64" },
    stdio: "pipe",
  });
  fs.chmodSync(output, 0o700);
  return artifact(output, "helper-darwin-arm64", { os: "darwin", arch: "arm64" });
}

function buildKeychainHelper(outputRoot) {
  const output = path.join(outputRoot, "artifacts", "algoloom-v12-keychain-darwin-arm64");
  execFileSync("xcrun", ["swiftc", KEYCHAIN_SOURCE, "-framework", "Security", "-O", "-o", output], {
    env: process.env,
    stdio: "pipe",
  });
  fs.chmodSync(output, 0o700);
  return artifact(output, "keychain-darwin-arm64", { os: "darwin", arch: "arm64" });
}

function gitObservation() {
  const revision = execFileSync("git", ["rev-parse", "HEAD"], {
    cwd: REPOSITORY_ROOT, encoding: "utf8", env: process.env,
  }).trim();
  const dirty = execFileSync("git", ["status", "--porcelain"], {
    cwd: REPOSITORY_ROOT, encoding: "utf8", env: process.env,
  }).trim() !== "";
  return { revision, dirty };
}

function main() {
  if (process.argv.length !== 3) fail("usage_invalid");
  const outputRoot = validateOutputRoot(process.argv[2]);
  fs.mkdirSync(outputRoot, { mode: 0o700 });
  fs.mkdirSync(path.join(outputRoot, "artifacts"), { mode: 0o700 });
  fs.mkdirSync(path.join(outputRoot, "stage"), { mode: 0o700 });
  try {
    const git = gitObservation();
    const uploadPackages = EXTENSION_VERSIONS.map((version) => buildExtension(outputRoot, version));
    const helperArtifacts = [buildHelper(outputRoot), buildKeychainHelper(outputRoot)];
    const helperSourceHash = sourceTreeHash(HELPER_SOURCE);
    const keychainSourceHash = sha256(fs.readFileSync(KEYCHAIN_SOURCE));
    const index = {
      schema_version: 1,
      preparation_scope: "V-12-local-build",
      campaign_ready: !git.dirty,
      source_revision: git.revision,
      source_tree_sha256: {
        extension: sourceTreeHash(EXTENSION_SOURCE),
        helper: helperSourceHash,
        keychain: keychainSourceHash,
        helper_bundle: sha256(Buffer.from(`${helperSourceHash}\0${keychainSourceHash}`, "utf8")),
      },
      versions: {
        extension_target: EXTENSION_VERSIONS[0],
        extension_update_from: EXTENSION_VERSIONS[0],
        extension_update_to: EXTENSION_VERSIONS[1],
        helper: HELPER_VERSION,
        protocol: 1,
        template_schema: "1.0",
        consent: "1.0",
        consent_sha256: sha256(fs.readFileSync(CONSENT_SOURCE)),
      },
      extension_upload_packages: uploadPackages,
      helper_artifacts: helperArtifacts,
      signed_extension_artifacts: [],
      secrets_present: false,
    };
    const encoded = `${JSON.stringify(index, null, 2)}\n`;
    if (encoded.includes("REVEL_SESSION=") || !uploadPackages.every((item) => HASH_PATTERN.test(item.sha256))) {
      fail("build_index_secret_or_hash_invalid");
    }
    fs.writeFileSync(path.join(outputRoot, "build-index.json"), encoded, { mode: 0o600, flag: "wx" });
    fs.rmSync(path.join(outputRoot, "stage"), { recursive: true });
    process.stdout.write(`${JSON.stringify({
      ok: true,
      campaign_ready: index.campaign_ready,
      extension_packages: uploadPackages.length,
      helper_artifacts: helperArtifacts.length,
      signed_extension_artifacts: 0,
    })}\n`);
  } catch (error) {
    fail(/^[a-z0-9_]{1,96}$/.test(error?.message || "") ? error.message : "build_failed");
  }
}

main();
