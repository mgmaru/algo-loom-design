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

### Route A — single-file fixture (recommended)

One Python file, served straight from the public source repository at a pinned
commit, so the contents cannot change under you. The extension ID to pass is in
the Test instructions field of this item.

```console
curl -fsSLO \
  https://raw.githubusercontent.com/mgmaru/algo-loom-design/0f17861f4a0037bc6376f5bf0300fd62e29f257a/scripts/verification/atcoder_v12/algoloom_v12_review_fixture.py
shasum -a 256 algoloom_v12_review_fixture.py
python3 algoloom_v12_review_fixture.py --self-test
python3 algoloom_v12_review_fixture.py --extension-id <EXTENSION_ID>
```

Expected SHA-256 of the fixture:

```text
18b3be7b4023dffd2db06c66e96b86bb3f7f1697c744fbcf7e0836eb43f29153
```

It works no matter how you download it, including a plain browser download.

The fixture prints a `http://127.0.0.1:<port>/bootstrap` URL. Open it in the
browser where the extension is installed, consent, and sign in to AtCoder as
yourself. Press Ctrl-C at any time to stop.

The fixture speaks the same authenticated loopback protocol as the production
helper and applies the same checks. **It stores nothing**: it validates the
cookie's name, host, path, and security attributes, reports the result, and
discards the value. It writes no keychain item, no file, and no log entry, and it
never contacts atcoder.jp. `--self-test` verifies this offline, without opening a
socket. Requires Python 3.9 or later.

### Route B — prebuilt helper (on request)

Route A is enough to exercise every behaviour of the extension. If you would
rather run the actual compiled macOS helper instead of the fixture, ask through
the publisher contact shown on this item's Chrome Web Store listing and a signed-
off bundle will be published for you.

That bundle must be fetched with `curl` rather than a browser download. `tar`
propagates the download quarantine attribute to the files it extracts, and macOS
then terminates the extracted helper without an explanatory dialog. `curl` does
not set that attribute, so the helper runs normally. Route A is unaffected either
way. **Never override a macOS security warning to run either route.** If something
is blocked, use Route A or contact the publisher.

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

### 経路A: 単一ファイルのフィクスチャ（推奨）

Pythonファイル1つです。公開ソースリポジトリの固定したコミットから直接配信するため、内容が後から変わることはありません。指定する拡張機能IDは、本itemのtest instructions欄に記載しています。

```console
curl -fsSLO \
  https://raw.githubusercontent.com/mgmaru/algo-loom-design/0f17861f4a0037bc6376f5bf0300fd62e29f257a/scripts/verification/atcoder_v12/algoloom_v12_review_fixture.py
shasum -a 256 algoloom_v12_review_fixture.py
python3 algoloom_v12_review_fixture.py --self-test
python3 algoloom_v12_review_fixture.py --extension-id <EXTENSION_ID>
```

フィクスチャの期待するSHA-256:

```text
18b3be7b4023dffd2db06c66e96b86bb3f7f1697c744fbcf7e0836eb43f29153
```

ブラウザーでのダウンロードを含め、取得方法にかかわらず動作します。

フィクスチャが`http://127.0.0.1:<port>/bootstrap`を表示します。拡張機能を追加したブラウザで開き、同意してご自身のAtCoderアカウントでログインしてください。Ctrl-Cでいつでも停止できます。

フィクスチャは製品相当ヘルパーと同じ認証付き折返し通信を話し、同じ検査を行います。**何も保存しません。** クッキーの名前、ホスト、パス、安全属性を検査して結果を返し、値を破棄します。Keychain項目、ファイル、ログのいずれにも書かず、atcoder.jpへ接続しません。`--self-test`はソケットを開かずにこれを確認します。Python 3.9以上が必要です。

### 経路B: 事前ビルド済みヘルパー（要望があれば提供）

拡張機能の挙動はすべて経路Aで確認できます。フィクスチャではなく実際にコンパイルしたmacOSヘルパーを動かしたい場合は、本itemのChromeウェブストア掲載情報に表示されるpublisher連絡先からご連絡ください。承認のうえ配布物を公開します。

その配布物は、ブラウザーではなく`curl`で取得してください。`tar`はダウンロード時のquarantine属性を展開したファイルへ伝播し、macOSは展開されたヘルパーを説明なく終了させます。`curl`はこの属性を付けないため、ヘルパーは通常どおり動作します。経路Aはどちらの取得方法でも影響を受けません。**どちらの経路でも、macOSの警告を回避して実行しないでください。** 動かない場合は経路Aをお使いいただくか、publisherへご連絡ください。

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
