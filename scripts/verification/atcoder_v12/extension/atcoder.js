"use strict";

const ACCOUNT_PATTERN = /^[A-Za-z0-9_]{1,64}$/;
const IDENTITY_PATTERN = /var\s+userScreenName\s*=\s*"([A-Za-z0-9_]{1,64})"\s*;/g;

function pageIdentities() {
  const identities = new Set();
  for (const script of document.scripts) {
    for (const match of (script.textContent || "").matchAll(IDENTITY_PATTERN)) {
      identities.add(match[1]);
    }
  }
  return [...identities].sort();
}

(async () => {
  if (location.origin !== "https://atcoder.jp" || location.pathname !== "/settings") return;
  if (document.getElementById("algoloom-v12-status")) return;

  const status = document.createElement("section");
  status.id = "algoloom-v12-status";
  status.style.cssText =
    "position:relative;z-index:2147483647;margin:16px auto;padding:20px;max-width:760px;border:3px solid #2563eb;background:#eff6ff;color:#17202a;font:16px/1.6 system-ui,sans-serif";
  status.textContent = "AlgoLoomが本人アカウントを確認しています。提出は行いません。";
  document.body.prepend(status);

  const identities = pageIdentities();
  if (
    identities.length !== 1 ||
    !ACCOUNT_PATTERN.test(identities[0]) ||
    navigator.webdriver === true
  ) {
    status.textContent = "本人アカウントを一意に確認できません。再試行せずブラウザを閉じてください。";
    return;
  }

  const checked = await chrome.runtime.sendMessage({
    type: "account_observed",
    identity: identities[0],
    identity_count: identities.length,
    navigator_webdriver: navigator.webdriver === true,
  });
  if (!checked?.ok) {
    status.textContent = "期待する本人アカウントと一致しません。認証情報を取り込まずブラウザを閉じてください。";
    return;
  }

  status.textContent = "本人アカウントを確認しました。REVEL_SESSIONだけを端末内ヘルパーへ渡しています。";
  const captured = await chrome.runtime.sendMessage({
    type: "capture_session",
    observed_identity: identities[0],
  });
  status.textContent = captured?.ok
    ? "認証情報を端末内で確認しました。ブラウザを閉じてください。"
    : "認証情報を確認できませんでした。再試行せずブラウザを閉じてください。";
})().catch(() => {
  const status = document.getElementById("algoloom-v12-status");
  if (status) status.textContent = "認証確認を完了できませんでした。再試行せずブラウザを閉じてください。";
});
