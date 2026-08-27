#!/usr/bin/env node
// TODO.mdの一覧表から、着手可能な作業と、それぞれが解放する下流の件数を出す。
// 使い方: node scripts/todo-status.mjs

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const ROW = /^\|\s*\[?`(TD-\d+)`\]?[^|]*\|([^|]*)\|([^|]*)\|([^|]*)\|\s*(\S+)\s*\|$/gm;

const text = fs.readFileSync(path.join(ROOT, "TODO.md"), "utf8");
const tasks = new Map();
for (const match of text.matchAll(ROW)) {
  const [, id, category, work, dependsOn, status] = match;
  tasks.set(id, {
    id,
    category: category.trim(),
    work: work.trim(),
    dependsOn: [...dependsOn.matchAll(/`(TD-\d+)`/g)].map((m) => m[1]),
    status: status.trim(),
  });
}
if (tasks.size === 0) {
  process.stderr.write("TODO.mdの一覧表を読めませんでした\n");
  process.exit(1);
}

const done = new Set([...tasks.values()].filter((t) => t.status === "完了").map((t) => t.id));
const open = [...tasks.values()].filter((t) => t.status !== "完了");
const unknown = [...tasks.values()].flatMap((t) => t.dependsOn).filter((d) => !tasks.has(d));

function downstream(rootId) {
  const reached = new Set([rootId]);
  for (let changed = true; changed; ) {
    changed = false;
    for (const task of open) {
      if (!reached.has(task.id) && task.dependsOn.some((d) => reached.has(d))) {
        reached.add(task.id);
        changed = true;
      }
    }
  }
  reached.delete(rootId);
  return reached;
}

const ready = open
  .filter((t) => t.dependsOn.every((d) => done.has(d)))
  .map((t) => ({ ...t, blocks: downstream(t.id).size }))
  .sort((a, b) => b.blocks - a.blocks);

process.stdout.write(`完了 ${done.size} / 全 ${tasks.size}\n\n着手可能な作業（下流を多く解放する順）\n`);
for (const task of ready) {
  const state = task.status === "完了" ? "" : `[${task.status}] `;
  process.stdout.write(`  ${task.id.padEnd(6)} 下流${String(task.blocks).padStart(2)}件  ${state}${task.work}\n`);
}
const waiting = open.filter((t) => t.status === "進行中" || t.status === "保留");
if (waiting.length > 0) {
  process.stdout.write("\n進行中・保留（外部待ちの可能性。TODO.md「外部の応答を待っている作業」を確認）\n");
  for (const task of waiting) process.stdout.write(`  ${task.id.padEnd(6)} [${task.status}] ${task.work}\n`);
}
if (unknown.length > 0) process.stdout.write(`\n未定義の依存: ${[...new Set(unknown)].join(", ")}\n`);
