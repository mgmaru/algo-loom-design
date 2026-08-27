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

## For Chrome Web Store reviewers

The extension hands one AtCoder session cookie to a companion program running on
the same device, so a local companion must be running for the core flow. Two ways
are provided. **Neither asks you to bypass a macOS security warning**, and neither
needs a compiler, developer mode, or shared AtCoder credentials. Use your own
AtCoder account.

Both files are published on the
[releases page](https://github.com/mgmaru/algo-loom-design/releases). Each release
lists the SHA-256 of every file.

### Route A — single-file fixture (recommended)

One Python file. It works no matter how you download it.

```console
curl -fsSLO <release-url>/algoloom_v12_review_fixture.py
shasum -a 256 algoloom_v12_review_fixture.py     # compare with the release notes
python3 algoloom_v12_review_fixture.py --self-test
python3 algoloom_v12_review_fixture.py --extension-id <EXTENSION_ID>
```

The fixture prints a `http://127.0.0.1:<port>/bootstrap` URL. Open it in the
browser where the extension is installed, consent, and sign in to AtCoder as
yourself. Press Ctrl-C at any time to stop.

The fixture speaks the same authenticated loopback protocol as the production
helper and applies the same checks. **It stores nothing**: it validates the
cookie's name, host, path, and security attributes, reports the result, and
discards the value. It writes no keychain item, no file, and no log entry, and it
never contacts atcoder.jp. `--self-test` verifies this offline, without opening a
socket. Requires Python 3.9 or later.

### Route B — prebuilt helper (optional)

If you would rather run the actual macOS helper, download the bundle **with
`curl`**:

```console
curl -fsSLO <release-url>/algoloom-v12-review-darwin-arm64.tar.gz
shasum -a 256 algoloom-v12-review-darwin-arm64.tar.gz
tar xzf algoloom-v12-review-darwin-arm64.tar.gz
cd algoloom-v12-review && shasum -a 256 -c SHA256SUMS
```

**Please use `curl` rather than a browser download for this route.** `tar`
propagates the download quarantine attribute to the files it extracts, and macOS
then terminates the extracted helper without an explanatory dialog. `curl` does
not set that attribute, so the helper runs normally. Route A is unaffected either
way. If Route B is blocked for any reason, use Route A instead of overriding any
warning.

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

## Chrome ウェブストア審査担当者の方へ

本拡張機能は、AtCoderのセッションクッキー1件を同じ端末上の相棒プログラムへ渡します。中核の動作にはローカルの相棒プログラムが必要です。2つの経路を用意しています。**どちらもmacOSの警告の回避を求めません。** コンパイラ、デベロッパーモード、共有のAtCoder認証情報も不要です。ご自身のAtCoderアカウントをお使いください。

どちらのファイルも[リリースページ](https://github.com/mgmaru/algo-loom-design/releases)で公開しています。各リリースにすべてのファイルのSHA-256を記載しています。

### 経路A: 単一ファイルのフィクスチャ（推奨）

Pythonファイル1つです。取得方法にかかわらず動作します。

```console
curl -fsSLO <release-url>/algoloom_v12_review_fixture.py
shasum -a 256 algoloom_v12_review_fixture.py     # リリースノートの値と照合します
python3 algoloom_v12_review_fixture.py --self-test
python3 algoloom_v12_review_fixture.py --extension-id <EXTENSION_ID>
```

フィクスチャが`http://127.0.0.1:<port>/bootstrap`を表示します。拡張機能を追加したブラウザで開き、同意してご自身のAtCoderアカウントでログインしてください。Ctrl-Cでいつでも停止できます。

フィクスチャは製品相当ヘルパーと同じ認証付き折返し通信を話し、同じ検査を行います。**何も保存しません。** クッキーの名前、ホスト、パス、安全属性を検査して結果を返し、値を破棄します。Keychain項目、ファイル、ログのいずれにも書かず、atcoder.jpへ接続しません。`--self-test`はソケットを開かずにこれを確認します。Python 3.9以上が必要です。

### 経路B: 事前ビルド済みヘルパー（任意）

実際のmacOSヘルパーを動かす場合は、**`curl`で**取得してください。

```console
curl -fsSLO <release-url>/algoloom-v12-review-darwin-arm64.tar.gz
shasum -a 256 algoloom-v12-review-darwin-arm64.tar.gz
tar xzf algoloom-v12-review-darwin-arm64.tar.gz
cd algoloom-v12-review && shasum -a 256 -c SHA256SUMS
```

**この経路ではブラウザではなく`curl`をお使いください。** `tar`はダウンロード時のquarantine属性を展開したファイルへ伝播し、macOSは展開されたヘルパーを説明なく終了させます。`curl`はこの属性を付けないため、ヘルパーは通常どおり動作します。経路Aはどちらの取得方法でも影響を受けません。経路Bが動かない場合は、警告を回避せず経路Aをお使いください。

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
