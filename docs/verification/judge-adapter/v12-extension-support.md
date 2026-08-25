# AlgoLoom Authentication Verification BETA Support

Last updated: August 26, 2026

This page supports only **AlgoLoom Authentication Verification BETA**, the unlisted extension used for the `V-12` technical verification. It is not a general release of AlgoLoom.

## Supported environment and scope

- Apple silicon Mac
- Current stable Google Chrome installed in its standard location
- macOS Keychain
- Installation from the extension's unlisted Chrome Web Store URL
- Manual AtCoder sign-in and any required Turnstile interaction by the user

Windows, Linux, Chrome policies, developer-mode installation, unpacked extensions, WebDriver, CDP, remote debugging, and automated sign-in are outside this verification's support scope.

## Before requesting support

1. Confirm that the extension was installed from the unlisted Chrome Web Store URL supplied by the verification owner.
2. Confirm that the extension version and requested permissions match the version recorded for the verification campaign.
3. Use the dedicated temporary Chrome profile created by the verification helper. Do not use a personal everyday Chrome profile.
4. If the page tells you to stop, close the dedicated Chrome window. Do not retry automatically or work around the stop condition.

## Stop and delete local data

Close the dedicated Chrome window to stop an active verification. The helper is designed to remove an unverified session, its loopback listener, one-time token, and temporary runtime profile on cancellation or failure.

A verified AtCoder session is temporarily retained in a campaign-specific macOS Keychain item. The verification owner deletes the exact item at campaign completion with:

```console
"$V12_HELPER" secret delete \
  --keychain-helper /absolute/owner-only/algoloom-v12-keychain-darwin-arm64 \
  --keychain-service io.algoloom.verification.v12.<campaign-namespace>.session
```

The placeholders must be replaced only with the exact paths and service identifier recorded outside the public campaign manifest. The helper rejects broad or unrelated Keychain targets. Removing the Chrome extension does not delete this Keychain item and does not invalidate the server-side AtCoder session.

If you also want to invalidate the AtCoder session, use AtCoder's own account controls. Do not send the session cookie to the extension publisher or include it in an issue.

## Report a problem

For a non-sensitive question or reproducible problem, open an issue in the [AlgoLoom design repository](https://github.com/mgmaru/algo-loom-design/issues). Include only:

- extension version
- Chrome version
- macOS version and hardware architecture
- the public, value-free error identifier shown by the helper
- the step at which the process stopped

The issue tracker is public. **Never include an AtCoder password, `REVEL_SESSION`, one-time bearer token, real AtCoder account identifier, Chrome profile contents, Keychain contents, or other personal or secret data.**

For a privacy or security matter that cannot be reported publicly, use the verified publisher contact displayed on the Chrome Web Store listing.

See the [privacy policy](v12-extension-privacy-policy.md) for details about processed data, purpose, transfer, retention, and user choices.

---

# AlgoLoom Authentication Verification BETA サポート

最終更新日: 2026年8月26日

このページは、`V-12`技術検証で使用する限定公開拡張機能 **AlgoLoom Authentication Verification BETA** だけを対象とします。AlgoLoomの一般公開版に対するサポートページではありません。

## 対応環境と範囲

- Apple silicon Mac
- 標準の場所へinstallされた現在のstable版Google Chrome
- macOS Keychain
- 検証ownerから渡されたChromeウェブストアの限定公開URLによる追加
- 利用者自身によるAtCoder loginと必要なTurnstile操作

Windows、Linux、Chrome policy、developer modeによる追加、unpacked extension、WebDriver、CDP、remote debugging、自動loginは、この検証のsupport対象外です。

## 問い合わせ前の確認

1. 検証ownerから渡されたChromeウェブストアの限定公開URLから追加したことを確認します。
2. 拡張機能版と要求権限が、検証campaignに記録された版と一致することを確認します。
3. 検証用helperが作った専用の一時Chrome profileを使用します。日常利用する個人profileは使用しません。
4. 画面に停止指示が出た場合は、専用Chrome windowを閉じます。自動再試行や停止条件の回避は行いません。

## 停止と端末内データの削除

実行中の検証を止めるには、専用Chrome windowを閉じます。helperは、取消または失敗時に、未確認session、loopback listener、一回限りのtoken、一時runtime profileを回収する設計です。

確認済みAtCoder sessionは、campaign専用のmacOS Keychain項目へ一時保持されます。検証ownerはcampaign終了時に、次のcommandで正確な項目を削除します。

```console
"$V12_HELPER" secret delete \
  --keychain-helper /absolute/owner-only/algoloom-v12-keychain-darwin-arm64 \
  --keychain-service io.algoloom.verification.v12.<campaign-namespace>.session
```

placeholderは、公開campaign manifestの外で管理する正確なpathとservice識別子だけに置き換えます。helperは広い範囲や無関係なKeychain項目を拒否します。Chrome拡張機能を削除しても、このKeychain項目は削除されず、AtCoder server側のsessionも失効しません。

AtCoder側のsessionも失効したい場合は、AtCoder自身のaccount管理機能を使用してください。session Cookieをpublisherへ送ったり、issueへ記載したりしないでください。

## 問題を報告する

機微でない質問や再現可能な問題は、[AlgoLoom設計repositoryのissue](https://github.com/mgmaru/algo-loom-design/issues)へ報告できます。記載する情報は次に限ります。

- 拡張機能版
- Chrome版
- macOS版とhardware architecture
- helperが表示した、値を含まない公開error識別子
- 停止した手順

issue trackerは公開されています。**AtCoder password、`REVEL_SESSION`、一回限りのBearer token、実AtCoder account識別子、Chrome profile内容、Keychain内容、その他の個人情報または秘密情報を絶対に記載しないでください。**

公開できないprivacyまたはsecurity上の問題は、Chromeウェブストアの掲載情報に表示される確認済みpublisher連絡先を使用してください。

処理するデータ、目的、転送、保持期間、利用者の選択については[プライバシーポリシー](v12-extension-privacy-policy.md)を参照してください。
