#!/usr/bin/env node
// リポジトリ内のMarkdownの相対リンクとアンカー、および決定記録とTODOの相互リンクを検査する。
// 使い方: node scripts/check-docs.mjs
// 壊れたリンクまたは片側だけの相互リンクが1件でもあれば終了コード1で終わる。

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const SKIP_DIRECTORIES = new Set([".git", "node_modules", "prompts"]);
const LINK_PATTERN = /\[[^\]]*\]\(([^)\s]+)\)/g;
const INLINE_LINK = /\[([^\]]*)\]\([^)]*\)/g;
const DROPPED = /[^\p{L}\p{N}\s\-_]/gu;

function markdownFiles(directory, found = []) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    if (entry.isSymbolicLink()) continue;
    const absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      if (!SKIP_DIRECTORIES.has(entry.name)) markdownFiles(absolute, found);
    } else if (entry.isFile() && entry.name.endsWith(".md")) {
      found.push(absolute);
    }
  }
  return found;
}

// GitHubの見出しID規則に合わせる。リンク記法は表示テキストへ畳み、記号を落とし、空白をハイフンにする。
function slug(heading) {
  return heading
    .replace(/^#+\s*/, "")
    .replace(INLINE_LINK, "$1")
    .replace(/`/g, "")
    .trim()
    .toLowerCase()
    .replace(DROPPED, "")
    .replace(/\s/g, "-");
}

function anchorsOf(filePath) {
  const seen = new Map();
  const anchors = new Set();
  let inFence = false;
  for (const line of fs.readFileSync(filePath, "utf8").split("\n")) {
    if (/^\s*```/.test(line)) inFence = !inFence;
    if (inFence || !line.startsWith("#")) continue;
    const base = slug(line);
    const count = seen.get(base) ?? 0;
    seen.set(base, count + 1);
    anchors.add(count === 0 ? base : `${base}-${count}`);
  }
  return anchors;
}

const files = markdownFiles(ROOT).sort();
const anchorIndex = new Map(files.map((file) => [file, anchorsOf(file)]));
const problems = [];

for (const file of files) {
  const text = fs.readFileSync(file, "utf8");
  for (const match of text.matchAll(LINK_PATTERN)) {
    const url = match[1];
    if (/^(?:https?:|mailto:|#|tel:)/.test(url) && !url.startsWith("#")) continue;
    const [target, fragment] = url.split("#");
    const resolved = target ? path.resolve(path.dirname(file), target) : file;
    const shown = path.relative(ROOT, file);
    if (target && !fs.existsSync(resolved)) {
      problems.push(`${shown} -> ${url} (参照先のファイルがない)`);
      continue;
    }
    if (!fragment) continue;
    const anchors = anchorIndex.get(resolved);
    if (!anchors) {
      problems.push(`${shown} -> ${url} (Markdown以外へのアンカー)`);
    } else if (!anchors.has(decodeURIComponent(fragment))) {
      problems.push(`${shown} -> ${url} (見出しがない)`);
    }
  }
}

// 決定記録（ADR）とTODOの相互リンクを検査する。どちらか片側だけの記載を落とす。
const DECISIONS = path.join(ROOT, "docs", "decisions");
const TODO_ROW = /^\|\s*\[?`(TD-\d+)`\]?[^|]*\|[^|]*\|[^|]*\|[^|]*\|[^|]*\|([^|]*)\|$/gm;
const RELATED_TODO_ROW = /^\|\s*関連TODO\s*\|([^|]*)\|/m;
const ADR_ID = /ADR-\d+/g;

function decisionIndex() {
  const byAdr = new Map();
  if (!fs.existsSync(DECISIONS)) return byAdr;
  for (const name of fs.readdirSync(DECISIONS).sort()) {
    const found = /^(\d{4})-.+\.md$/.exec(name);
    if (!found) continue;
    const text = fs.readFileSync(path.join(DECISIONS, name), "utf8");
    const related = RELATED_TODO_ROW.exec(text);
    byAdr.set(`ADR-${found[1]}`, {
      file: path.join("docs", "decisions", name),
      todos: new Set(related ? [...related[1].matchAll(/TD-\d+/g)].map((m) => m[0]) : []),
    });
  }
  return byAdr;
}

const adrs = decisionIndex();
const todoText = fs.existsSync(path.join(ROOT, "TODO.md"))
  ? fs.readFileSync(path.join(ROOT, "TODO.md"), "utf8")
  : "";
const todoRows = new Map(
  [...todoText.matchAll(TODO_ROW)].map((m) => [m[1], new Set([...m[2].matchAll(ADR_ID)].map((a) => a[0]))]),
);
const crossProblems = [];

for (const [adr, entry] of adrs) {
  for (const todo of entry.todos) {
    const listed = todoRows.get(todo);
    if (!listed) {
      crossProblems.push(`${entry.file} -> ${todo} (TODO.mdの一覧表にその作業がない)`);
    } else if (!listed.has(adr)) {
      crossProblems.push(`${entry.file} -> ${todo} (TODO.mdの「決定」列に${adr}がない)`);
    }
  }
}
for (const [todo, listed] of todoRows) {
  for (const adr of listed) {
    const entry = adrs.get(adr);
    if (!entry) {
      crossProblems.push(`TODO.md -> ${adr} (${todo}の「決定」列が指す決定記録がない)`);
    } else if (!entry.todos.has(todo)) {
      crossProblems.push(`TODO.md -> ${adr} (${entry.file}の「関連TODO」に${todo}がない)`);
    }
  }
}

for (const problem of problems) process.stdout.write(`NG ${problem}\n`);
for (const problem of crossProblems) process.stdout.write(`NG ${problem}\n`);
process.stdout.write(
  `${problems.length === 0 ? "OK" : "NG"} Markdown ${files.length}件を検査し、壊れたリンクは${problems.length}件でした\n`,
);
process.stdout.write(
  `${crossProblems.length === 0 ? "OK" : "NG"} 決定記録 ${adrs.size}件とTODOの相互リンクを検査し、片側だけの記載は${crossProblems.length}件でした\n`,
);
process.exit(problems.length === 0 && crossProblems.length === 0 ? 0 : 1);
