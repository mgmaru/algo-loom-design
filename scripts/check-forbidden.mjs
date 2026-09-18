#!/usr/bin/env node
// リポジトリへ書いてはいけない値が混ざっていないかを検査する。
// 使い方: node scripts/check-forbidden.mjs
// 検出が1件でもあれば終了コード1で終わる。
//
// 対象は、gitが追跡しているfileと、追跡されていないが`.gitignore`で除外されていないfile。
// 検査できるのは**形式で判別できるもの**だけである。実名そのものは検出できない。
// 検知パターンへ実名を書けば、そのパターン自体が禁止事項違反になるためである。
// 規約は[作業ガイド §5](../CLAUDE.md)を参照する。

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const SKIP_EXTENSIONS = new Set([
  ".png", ".jpg", ".jpeg", ".gif", ".ico", ".pdf", ".zip", ".gz", ".tar", ".crx", ".woff", ".woff2",
]);

// 同じ文字の繰り返しは、文書中のplaceholderとして扱う。
const isPlaceholder = (value) => new Set(value).size === 1;

const RULES = [
  {
    name: "端末の絶対path",
    pattern: /\/(?:Users|home)\/[A-Za-z0-9._-]+\//g,
    hint: "利用者名を含むpathを書かない。`<HOME>`等のplaceholderにする",
  },
  {
    name: "Chrome拡張機能の固定ID",
    pattern: /\b[a-p]{32}\b/g,
    allow: (found) => isPlaceholder(found),
    hint: "固定IDはowner専用領域へ置く。文書では同じ文字の繰り返しをplaceholderにする",
  },
  {
    name: "秘密鍵",
    pattern: /-----BEGIN [A-Z ]*PRIVATE KEY-----/g,
    hint: "秘密鍵をリポジトリへ置かない",
  },
  {
    name: "access token",
    pattern: /\b(?:gh[pousr]_[A-Za-z0-9]{36}|xox[baprs]-[A-Za-z0-9-]{10,}|AIza[0-9A-Za-z_-]{35})\b/g,
    hint: "tokenをリポジトリへ置かない。失効させてから取り除く",
  },
  {
    name: "session cookieの値",
    pattern: /REVEL_SESSION\s*=\s*[^\s"'`<>|)]{32,}/g,
    hint: "Cookieの値を書かない。testでは短いplaceholderを使う",
  },
  {
    name: "小文字のuser指定",
    pattern: /--(?:user|username|account|handle)[= ]([a-z][A-Za-z0-9._-]*)/g,
    hint: "CLI例のplaceholderは大文字にする（`--user USER`）。実account名を書かない",
  },
];

function targetFiles() {
  const listed = execFileSync("git", ["ls-files", "--cached", "--others", "--exclude-standard"], {
    cwd: ROOT,
    encoding: "utf8",
    maxBuffer: 32 * 1024 * 1024,
  });
  return listed
    .split("\n")
    .filter(Boolean)
    .filter((relative) => !SKIP_EXTENSIONS.has(path.extname(relative).toLowerCase()))
    .filter((relative) => fs.existsSync(path.join(ROOT, relative)));
}

const files = targetFiles();
const problems = [];

for (const relative of files) {
  const absolute = path.join(ROOT, relative);
  if (fs.statSync(absolute).isDirectory()) continue;
  const lines = fs.readFileSync(absolute, "utf8").split("\n");
  for (const [index, line] of lines.entries()) {
    for (const rule of RULES) {
      rule.pattern.lastIndex = 0;
      for (const match of line.matchAll(rule.pattern)) {
        const found = match[1] ?? match[0];
        if (rule.allow?.(found)) continue;
        problems.push(`${relative}:${index + 1} ${rule.name} (${rule.hint})`);
      }
    }
  }
}

for (const problem of problems) process.stdout.write(`NG ${problem}\n`);
process.stdout.write(
  `${problems.length === 0 ? "OK" : "NG"} ${files.length}件を走査し、禁止事項の検出は${problems.length}件でした\n`,
);
process.exit(problems.length === 0 ? 0 : 1);
