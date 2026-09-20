// 提出確認画面から提出pageへの受け渡しを、実際のbrowserで測る。
//
// なぜ要るか: 2026年9月21日の5回目のcampaignで、`V-12E`が最後の受け渡しに
// 到達しなかった。helperはPOSTを受理して303を書いたのに、browserは提出page
// へ着かず`ERR_CONNECTION_REFUSED`を表示した（ADR-0013）。契約testは
// `httptest`でhandlerを直接呼ぶため、**browserが応答を受け取り遷移するところ**
// を一度も通していない。
//
// このscriptはローカルだけで完結する。AtCoder、Chrome Web Store、
// Cloudflareへ接続しない。提出pageの代わりに別portのローカル待受を置く。
// portが違えばoriginも違うため、AtCoderへの遷移と同じ「別originへの303」に
// なる。使い捨てChrome profileはリポジトリ外に作り、終了時に破棄する。
//
//   node scripts/verification/atcoder_v12/submission-handoff-probe.mjs
//
// 終了コード0で、helperが使う受け渡しがbrowserの実挙動と整合している。

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

// helperが提出確認画面へ付けるCSPの雛形をsourceから読む。probeが自分で
// 書いた文字列を測ると、実装との食い違いに気づけない。
function helperCSPTemplate() {
  const source = fs.readFileSync(path.join(HERE, "helper", "protocol.go"), "utf8");
  const declaration = source.match(/const submissionCSPTemplate = ((?:\s*"[^"]*"\s*\+?)+)/);
  if (!declaration) throw new Error("submission_csp_template_not_found");
  const parts = declaration[1].match(/"[^"]*"/g);
  if (!parts) throw new Error("submission_csp_template_unreadable");
  const template = parts.map((part) => part.slice(1, -1)).join("");
  if (!template.includes("form-action %s %s")) {
    throw new Error("submission_csp_template_shape_changed");
  }
  return template;
}

// 雛形の`%s`を埋める。Goの`fmt.Sprintf`と同じ並びにする。
function fillCSP(template, ...values) {
  let index = 0;
  return template.replace(/%s/g, () => values[index++]);
}

// 実物の画面を使う。代用の画面で測ると、確かめたい相手がまた入れ替わる。
function realSubmissionPage(proceedToken) {
  const template = fs.readFileSync(path.join(HERE, "helper", "submission.html"), "utf8");
  if (template.includes("<script")) throw new Error("real_page_unexpectedly_has_script");
  return template
    .replace("{{PROBLEM_ID}}", "abc300_a")
    .replace("{{PROBLEM_URL}}", "https://example.invalid/tasks/abc300_a")
    .replace("{{SUBMIT_URL}}", "https://example.invalid/submit")
    .replace("{{LANGUAGE}}", "Python (CPython 3.11.4)")
    .replace("{{SOURCE_NAME}}", "source.py")
    .replace("{{SOURCE_BYTES}}", "369")
    .replace("{{SOURCE_LINES}}", "13")
    .replace("{{SOURCE_SHA256}}", "0".repeat(64))
    .replace("{{SOURCE_PREVIEW}}", "print(&#39;probe&#39;)")
    .replace("{{PROCEED_TOKEN}}", proceedToken);
}

// 人の指の代わり。画面そのものはJavaScriptを持たないため、probeのときだけ
// scriptを足し、CSPも`script-src`だけ緩める。`form-action`は変えない。
function withClickScript(page, mode) {
  const scripts = {
    immediate: "form.submit();",
    delayed: `setTimeout(function(){ form.submit(); }, ${12_000});`,
    double: "form.submit(); setTimeout(function(){ try { form.submit(); } catch (e) {} }, 150);",
  };
  const script = `<script>var form = document.querySelector('form'); ${scripts[mode]}</script>`;
  return page.replace("</body>", `${script}\n</body>`);
}

const CASES = [
  {
    key: "current_handoff",
    name: "現行の受け渡し（実物の画面・実物のヘッダー）",
    submit: "immediate",
    csp: "helper",
    shutdown: "after_response",
    mustArrive: true,
    role: "helperが使う経路",
  },
  {
    key: "form_action_loopback_only",
    name: "`form-action`をloopback originだけにする（5回目のcampaignの状態）",
    submit: "immediate",
    csp: "loopback_only",
    shutdown: "after_response",
    mustArrive: false,
    role: "negative control",
  },
  {
    key: "no_csp",
    name: "CSPを付けない",
    submit: "immediate",
    csp: "none",
    shutdown: "after_response",
    mustArrive: true,
    role: "切り分け",
  },
  {
    key: "shutdown_deferred",
    name: "待受を閉じるのを5秒遅らせる",
    submit: "immediate",
    csp: "helper",
    shutdown: "deferred",
    mustArrive: true,
    role: "切り分け",
  },
  {
    key: "double_click",
    name: "2回押す（1回目の遷移が終わる前に）",
    submit: "double",
    csp: "helper",
    shutdown: "after_response",
    mustArrive: true,
    informational: true,
    role: "観測のみ。利用者がやりうる操作",
  },
  {
    key: "no_response",
    name: "応答を書かずに接続を切る",
    submit: "immediate",
    csp: "helper",
    shutdown: "destroy",
    mustArrive: false,
    role: "negative control",
  },
];

const CASE_TIMEOUT_MS = 45_000;

function runCase(testCase, policy) {
  return new Promise((resolve) => {
    const observation = {
      case: testCase.name,
      key: testCase.key,
      role: testCase.role,
      page_shown: false,
      proceed_received: 0,
      proceed_origin_ok: null,
      redirect_written: false,
      arrived_at_submit_page: false,
    };

    const target = http.createServer((request, response) => {
      observation.arrived_at_submit_page = true;
      response.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
      response.end("<!doctype html>submit page stand-in");
      finish();
    });

    // portは待受を閉じた後も参照するため、listen時に控える。
    let origin = "";
    let targetOrigin = "";

    const main = http.createServer((request, response) => {
      const url = new URL(request.url, "http://127.0.0.1");
      if (request.method === "GET" && url.pathname === "/submission") {
        observation.page_shown = true;
        const headers = {
          "Content-Type": "text/html; charset=utf-8",
          "Referrer-Policy": policy,
          "X-Frame-Options": "DENY",
        };
        if (testCase.csp !== "none") {
          const template = helperCSPTemplate();
          // `loopback_only`は5回目のcampaignの状態を再現する。遷移先を
          // 許さない指定にすると、browserが303を止めることを確かめる。
          const base = testCase.csp === "loopback_only"
            ? fillCSP(template, origin, origin)
            : fillCSP(template, origin, targetOrigin);
          // probeの自動送信のためだけにscriptを許す。form-actionは変えない。
          headers["Content-Security-Policy"] = base.replace(
            "default-src 'none';", "default-src 'none'; script-src 'unsafe-inline';");
        }
        response.writeHead(200, headers);
        response.end(withClickScript(
          realSubmissionPage("0123456789abcdef".repeat(4)), testCase.submit));
        return;
      }
      if (request.method === "POST" && url.pathname === "/submission/proceed") {
        observation.proceed_received += 1;
        if (observation.proceed_received === 1) {
          observation.proceed_origin_ok = request.headers.origin === origin;
        } else {
          response.writeHead(409).end();
          return;
        }
        if (testCase.shutdown === "destroy") {
          shutdown();
          request.socket.destroy();
          return;
        }
        response.writeHead(303, { Location: `${targetOrigin}/submit-page` });
        response.end(() => {
          observation.redirect_written = true;
          if (testCase.shutdown === "deferred") {
            setTimeout(shutdown, 5_000);
          } else {
            shutdown();
          }
        });
        return;
      }
      response.writeHead(404).end();
    });

    let shuttingDown = false;
    function shutdown() {
      if (shuttingDown) return;
      shuttingDown = true;
      // Goの`server.Shutdown`に相当する。待受を閉じ、空いた接続を落とす。
      main.close();
      main.closeIdleConnections();
    }

    let finished = false;
    let chrome;
    let profile;
    function finish() {
      if (finished) return;
      finished = true;
      clearTimeout(timer);
      setTimeout(() => {
        if (chrome) chrome.kill("SIGTERM");
        main.close();
        main.closeIdleConnections();
        target.close();
        setTimeout(() => {
          if (profile) fs.rmSync(profile, { recursive: true, force: true });
          resolve(observation);
        }, 500);
      }, 300);
    }

    const timer = setTimeout(finish, CASE_TIMEOUT_MS);

    target.listen(0, "127.0.0.1", () => {
      targetOrigin = `http://127.0.0.1:${target.address().port}`;
      main.listen(0, "127.0.0.1", () => {
        origin = `http://127.0.0.1:${main.address().port}`;
        profile = fs.mkdtempSync(path.join(os.tmpdir(), "algoloom-v12-handoff-"));
        chrome = spawn(CHROME, [
          `--user-data-dir=${profile}`,
          "--no-first-run",
          "--no-default-browser-check",
          `${origin}/submission`,
        ], { stdio: "ignore" });
      });
    });
  });
}

async function main() {
  const policy = helperReferrerPolicy();
  const findings = [];
  for (const testCase of CASES) {
    const observation = await runCase(testCase, policy);
    observation.expected_to_arrive = testCase.mustArrive;
    observation.ok = observation.arrived_at_submit_page === testCase.mustArrive;
    if (testCase.informational) observation.informational = true;
    findings.push(observation);
  }

  const judged = findings.filter((f) => !f.informational);
  const negatives = judged.filter((f) => f.role === "negative control");
  const ok = judged.every((f) => f.ok)
    && negatives.length > 0
    && negatives.every((f) => f.arrived_at_submit_page === false);

  console.log(JSON.stringify({
    ok,
    helper_referrer_policy: policy,
    external_connections: 0,
    findings,
  }, null, 2));
  process.exit(ok ? 0 : 1);
}

main().catch((error) => {
  console.error(String(error && error.message ? error.message : error));
  process.exit(1);
});
