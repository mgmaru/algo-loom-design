// 実際のbrowserが送るrequestヘッダーを測る。
//
// なぜ要るか: 契約testはrequestを手で組み立てるため、browserが実際に何を
// 送るかを確かめられない。2026年9月20日、提出確認画面の`Referrer-Policy:
// no-referrer`が同一originのform POSTの`Origin`を`null`にし、`V-12E`が
// 最後の受け渡しで停止した（ADR-0012）。同じ型の見落としはADR-0005に続いて
// 2度目である。
//
// このscriptはローカルだけで完結する。AtCoder、Chrome Web Store、
// Cloudflareへ接続しない。使い捨てChrome profileはリポジトリ外に作り、
// 終了時に破棄する。
//
//   node scripts/verification/atcoder_v12/browser-request-probe.mjs
//
// 終了コード0で、helperが使う値がbrowserの実挙動と整合している。

import { spawn } from "node:child_process";
import fs from "node:fs";
import http from "node:http";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
const CHROME = process.env.ALGOLOOM_V12_CHROME
  || "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";

// helperのsourceから読む。probeと実装が別々に動いて食い違うのを防ぐ。
function helperReferrerPolicy() {
  const source = fs.readFileSync(path.join(HERE, "helper", "protocol.go"), "utf8");
  const found = source.match(/const submissionReferrerPolicy = "([^"]+)"/);
  if (!found) throw new Error("submission_referrer_policy_not_found");
  return found[1];
}

// 測る値。negative controlを必ず含める。「常に成功」を成立証拠にしない。
function policiesToMeasure(helperPolicy) {
  return [
    { policy: helperPolicy, role: "helperが使う値", originMustSurvive: true },
    { policy: "no-referrer", role: "negative control", originMustSurvive: false },
  ];
}

const PAGE = (policy) => `<!doctype html><html lang="ja"><head><meta charset="utf-8"></head><body>
<form id="f" method="post" action="/measure">
<input type="hidden" name="policy" value="${policy}">
</form>
<script>document.getElementById('f').submit();</script>
</body></html>`;

function startServers(onRecord) {
  const cross = http.createServer((request, response) => {
    const policy = new URL(request.url, "http://127.0.0.1").searchParams.get("policy") ?? "?";
    onRecord(policy, "cross", { referer: request.headers.referer ?? null });
    response.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
    response.end("<!doctype html>done");
  });
  const main = http.createServer((request, response) => {
    const url = new URL(request.url, "http://127.0.0.1");
    if (request.method === "GET" && url.pathname.startsWith("/page/")) {
      const policy = decodeURIComponent(url.pathname.slice("/page/".length));
      // helperが提出確認画面へ付けるのと同じヘッダーにする。form POSTを
      // 人の操作なしで起こすため、CSPだけはscriptを許す。CSPは`Origin`の
      // 決まり方に関与しない。
      response.writeHead(200, {
        "Content-Type": "text/html; charset=utf-8",
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
        "Referrer-Policy": policy,
        "X-Frame-Options": "DENY",
      });
      response.end(PAGE(policy));
      return;
    }
    if (request.method === "POST" && url.pathname === "/measure") {
      let body = "";
      request.on("data", (chunk) => { body += chunk; });
      request.on("end", () => {
        const policy = new URLSearchParams(body).get("policy") ?? "?";
        onRecord(policy, "same", {
          origin: request.headers.origin ?? null,
          contentType: request.headers["content-type"] ?? null,
          host: request.headers.host ?? null,
          fieldCount: [...new URLSearchParams(body).keys()].length,
        });
        // 別originへ303で送り、`Referer`が漏れないことも同時に測る。
        response.writeHead(303, {
          Location: `http://localhost:${cross.address().port}/landing?policy=${encodeURIComponent(policy)}`,
        });
        response.end();
      });
      return;
    }
    response.writeHead(404).end();
  });
  return { main, cross };
}

async function main() {
  const helperPolicy = helperReferrerPolicy();
  const plan = policiesToMeasure(helperPolicy);
  const records = new Map();
  let resolveDone;
  const done = new Promise((resolve) => { resolveDone = resolve; });

  const expected = plan.length * 2;
  const { main: mainServer, cross } = startServers((policy, kind, data) => {
    records.set(`${policy}:${kind}`, data);
    if (records.size >= expected) resolveDone();
  });

  await new Promise((r) => cross.listen(0, "127.0.0.1", r));
  await new Promise((r) => mainServer.listen(0, "127.0.0.1", r));
  const port = mainServer.address().port;

  const profile = fs.mkdtempSync(path.join(os.tmpdir(), "algoloom-v12-probe-"));
  const chrome = spawn(CHROME, [
    `--user-data-dir=${profile}`,
    "--no-first-run",
    "--no-default-browser-check",
    ...plan.map(({ policy }) => `http://127.0.0.1:${port}/page/${encodeURIComponent(policy)}`),
  ], { stdio: "ignore" });

  const timer = setTimeout(() => resolveDone(), 45_000);
  await done;
  clearTimeout(timer);
  chrome.kill("SIGTERM");
  await new Promise((r) => setTimeout(r, 1000));
  mainServer.close();
  cross.close();
  fs.rmSync(profile, { recursive: true, force: true });

  const origin = `http://127.0.0.1:${port}`;
  const findings = [];
  for (const { policy, role, originMustSurvive } of plan) {
    const same = records.get(`${policy}:same`);
    const crossed = records.get(`${policy}:cross`);
    if (!same) {
      findings.push({ policy, role, ok: false, reason: "measurement_missing" });
      continue;
    }
    const survived = same.origin === origin;
    findings.push({
      policy,
      role,
      ok: survived === originMustSurvive,
      origin_sent: same.origin,
      origin_survived: survived,
      content_type_sent: same.contentType,
      form_field_count: same.fieldCount,
      cross_origin_referer: crossed ? crossed.referer : "<未測定>",
    });
  }

  const ok = findings.every((f) => f.ok)
    && findings.some((f) => f.role === "negative control" && f.origin_survived === false);
  console.log(JSON.stringify({
    ok,
    helper_referrer_policy: helperPolicy,
    external_connections: 0,
    findings,
  }, null, 2));
  process.exit(ok ? 0 : 1);
}

main().catch((error) => {
  console.error(String(error && error.message ? error.message : error));
  process.exit(1);
});
