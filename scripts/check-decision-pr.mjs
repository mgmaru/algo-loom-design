#!/usr/bin/env node
// 判断を含むPR（`docs/decisions/`を変更するPR）で、確認項目が埋まっているかを検査する。
// 使い方: CHANGED_FILES="<改行区切りのpath>" PR_BODY="<PRの本文>" node scripts/check-decision-pr.mjs
// 未記入の項目が残っていれば終了コード1で終わる。
//
// 一人で開発するため、GitHubのApproveは使えない（PRの作成者は自分のPRを承認できない）。
// 承認の代わりに、自己確認を機械で強制する。

import process from "node:process";

const SECTION = /^#{2,}\s*判断の確認/;
const HEADING = /^#{1,6}\s/;
const UNCHECKED = /^\s*[-*]\s*\[\s\]\s*(.*)$/;

const changed = (process.env.CHANGED_FILES ?? "")
  .split("\n")
  .map((line) => line.trim())
  .filter(Boolean);
const decisions = changed.filter((file) => file.startsWith("docs/decisions/"));

if (decisions.length === 0) {
  process.stdout.write("OK 判断を含まないPRのため、確認項目の検査は行いません\n");
  process.exit(0);
}

process.stdout.write(`判断を含むPRです。変更された決定記録: ${decisions.join("、")}\n`);

const body = (process.env.PR_BODY ?? "").split("\n");
const start = body.findIndex((line) => SECTION.test(line));
if (start === -1) {
  process.stdout.write("NG PRの本文に「判断の確認」の節がありません。PRテンプレートの項目を残して記入してください\n");
  process.exit(1);
}

const remaining = [];
for (const line of body.slice(start + 1)) {
  if (HEADING.test(line)) break;
  const found = UNCHECKED.exec(line);
  if (found) remaining.push(found[1].trim());
}

for (const item of remaining) process.stdout.write(`NG 未確認: ${item}\n`);
process.stdout.write(
  `${remaining.length === 0 ? "OK" : "NG"} 判断の確認項目のうち、未記入は${remaining.length}件でした\n`,
);
process.exit(remaining.length === 0 ? 0 : 1);
