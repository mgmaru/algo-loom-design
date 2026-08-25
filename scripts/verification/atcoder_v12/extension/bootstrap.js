"use strict";

(async () => {
  if (location.origin !== "http://127.0.0.1" || location.pathname !== "/bootstrap") return;

  const tokenNode = document.querySelector('meta[name="algoloom-loopback-token"]');
  const consentNode = document.querySelector('meta[name="algoloom-consent-version"]');
  const token = tokenNode?.content || "";
  const consentVersion = consentNode?.content || "";
  tokenNode?.remove();
  consentNode?.remove();

  const consentButton = document.getElementById("algoloom-consent");
  if (!(consentButton instanceof HTMLButtonElement)) {
    document.body.textContent = "同意画面を確認できません。ブラウザを閉じてください。";
    return;
  }

  consentButton.addEventListener("click", async () => {
    consentButton.disabled = true;
    consentButton.textContent = "認証を準備しています…";
    try {
      const response = await chrome.runtime.sendMessage({
        type: "initialize",
        port: Number(location.port),
        token,
        consent_version: consentVersion,
        extension_version: chrome.runtime.getManifest().version,
        navigator_webdriver: navigator.webdriver === true,
      });
      document.querySelector('meta[name="algoloom-loopback-token"]')?.remove();

      if (!response?.ok) throw new Error("initialization_failed");
      location.replace("https://atcoder.jp/settings");
    } catch (_) {
      document.body.textContent = "認証の初期化に失敗しました。ブラウザを閉じてください。";
    }
  }, { once: true });
})().catch(() => {
  document.body.textContent = "認証の初期化に失敗しました。ブラウザを閉じてください。";
});
