# AlgoLoom Authentication Verification BETA Privacy Policy

Last updated: August 26, 2026

This policy applies only to **AlgoLoom Authentication Verification BETA**, an unlisted Chrome extension used for the `V-12` technical verification. It does not describe a released AlgoLoom product or any future AlgoLoom service.

## Purpose

The extension has one purpose: after the user gives consent and manually signs in to AtCoder, it confirms the visible AtCoder account and transfers exactly one allowlisted AtCoder session cookie to the AlgoLoom verification helper running on the same device. The helper uses the session only to confirm that the same account is authenticated.

The extension and helper do not automate an AtCoder password, Turnstile, sign-in, or submission. They do not submit source code.

## Data processed

The extension and its local helper process the following data:

- **Authentication information:** one cookie named `REVEL_SESSION` for `https://atcoder.jp/`.
- **Website content:** the AtCoder account identifier displayed on `https://atcoder.jp/settings`, used only to confirm that exactly one expected account is signed in.
- **Local protocol data:** a random loopback port, a one-time bearer token, the consent version, the extension version, and the browser tab identifier needed to complete the requested local transfer safely.

The extension does not read the user's AtCoder password, Turnstile token, source code, submissions, browsing history, or other cookies. It does not collect analytics, advertising identifiers, payment information, or location information.

## How data is used and transferred

The account identifier and `REVEL_SESSION` are sent only to the verification helper on the same device through an authenticated connection to IPv4 loopback address `127.0.0.1`. They are not sent to the AlgoLoom publisher or to an AlgoLoom-operated server.

The local helper sends the `REVEL_SESSION` back to `https://atcoder.jp/settings` over HTTPS, using at most two read-only `GET` requests, to verify the authenticated account. It does not make a submission request or any other `POST` request to AtCoder.

The publisher does not sell user data, use it for advertising, credit decisions, profiling, or unrelated purposes, or allow a human to read it. No user data is transferred to another party except when necessary to provide this user-requested authentication verification or when required by applicable law.

## Local storage and retention

- The loopback port, one-time token, consent version, and tab identifier are kept in `chrome.storage.session` only during the active browser session and are removed after a successful transfer or when the session ends.
- After the helper verifies the session, it temporarily stores the `REVEL_SESSION` in a campaign-specific macOS Keychain item on the user's device. It is not written to a plaintext file, normal log, Git repository, or published verification result.
- Unverified session data is discarded on cancellation or failure. A verified Keychain item is retained only until the `V-12` verification campaign ends or the local deletion command is run, whichever comes first.
- Temporary browser profiles used for individual verification cases are destroyed after the case. The baseline profile does not retain AtCoder cookies, history, or form data.

Removing the extension does not by itself invalidate the session held by AtCoder. The user must use AtCoder's own account controls if they also want to invalidate the server-side session.

## User choice and deletion

The local consent page explains this processing before it begins. A user can decline by not selecting **Consent and continue to AtCoder** and by closing Chrome.

To stop an active verification, close the dedicated Chrome window. To delete a verified session from macOS Keychain, follow the local deletion procedure in the [support page](v12-extension-support.md). Do not send a password or session cookie in a support request.

## Security

The local transfer accepts only IPv4 loopback connections and validates the request host, extension origin, one-time bearer token, content type, message size, extension version, consent version, and event order. Communication with AtCoder uses HTTPS. The extension does not execute remotely hosted code.

## Limited Use

The use of information received from Chrome APIs complies with the [Chrome Web Store User Data Policy, including the Limited Use requirements](https://developer.chrome.com/docs/webstore/program-policies/limited-use). Data is used only to provide the extension's disclosed single purpose.

## Changes and contact

Material changes to the data processed, purpose, transfer destination, or retention period require an updated policy and new consent before the changed behavior is used. The revision date at the top identifies the current policy.

For privacy questions, use the verified publisher contact shown on the Chrome Web Store listing. For non-sensitive technical questions, use the [support page](v12-extension-support.md). Never include a password, session cookie, one-time token, real account identifier, or other secret in a public issue.

---

# AlgoLoom Authentication Verification BETA プライバシーポリシー

最終更新日: 2026年8月26日

このポリシーは、`V-12`技術検証だけに用いる限定公開Chrome拡張機能 **AlgoLoom Authentication Verification BETA** に適用されます。正式公開されたAlgoLoom製品や将来のAlgoLoomサービスを対象とするものではありません。

## 目的

この拡張機能の目的は一つです。利用者が同意し、自身でAtCoderへログインした後、表示されたAtCoderアカウントを確認し、許可したAtCoderセッションCookieを厳密に1件だけ同じ端末上のAlgoLoom検証用helperへ渡します。helperは、同じアカウントが認証済みであることの確認だけにそのセッションを使います。

拡張機能とhelperは、AtCoderのパスワード、Turnstile、ログインまたは提出を自動化せず、ソースコードを提出しません。

## 処理するデータ

拡張機能と端末内helperは、次のデータを処理します。

- **認証情報:** `https://atcoder.jp/`の`REVEL_SESSION`という名前のCookie 1件
- **ウェブサイトのコンテンツ:** `https://atcoder.jp/settings`に表示されたAtCoderアカウント識別子。期待するアカウントが1件だけログインしていることの確認に限って使用
- **端末内プロトコル情報:** 安全な端末内転送に必要な、ランダムなloopback port、一回限りのBearer token、同意版、拡張機能版、browser tab識別子

AtCoderのパスワード、Turnstile token、ソースコード、提出、閲覧履歴、他のCookieは読みません。analytics、広告識別子、支払情報、位置情報も収集しません。

## データの利用と転送

アカウント識別子と`REVEL_SESSION`は、認証付きのIPv4 loopback address `127.0.0.1`を通して、同じ端末上の検証用helperだけへ送ります。AlgoLoomのpublisherまたはAlgoLoomが運用するserverへは送信しません。

端末内helperは、認証済みアカウントを確認するため、`REVEL_SESSION`をHTTPSで`https://atcoder.jp/settings`へ送り返し、読取り専用の`GET`を最大2回行います。提出requestやその他の`POST` requestは行いません。

publisherは、利用者データの販売、広告、credit判断、profiling、目的外利用、人による閲覧を行いません。利用者が要求した認証確認の提供に必要な場合、または適用法令上必要な場合を除き、データを他者へ転送しません。

## 端末内の保存と保持期間

- loopback port、一回限りのtoken、同意版、tab識別子は、実行中のbrowser sessionに限り`chrome.storage.session`へ保持し、転送成功後またはsession終了時に削除されます。
- helperがsessionを確認した後、`REVEL_SESSION`を利用者の端末にあるcampaign専用macOS Keychain項目へ一時保存します。平文file、通常log、Git repository、公開する検証結果には保存しません。
- 確認できなかったsession dataは、取消または失敗時に破棄します。確認済みKeychain項目は、`V-12`検証campaign終了時または端末内の削除command実行時の、いずれか早い時点まで保持します。
- 各検証caseの一時browser profileはcase終了後に破棄します。基準profileにはAtCoderのCookie、履歴、form dataを残しません。

拡張機能を削除しても、それだけではAtCoder側のsessionは失効しません。server側sessionも失効したい場合は、利用者がAtCoder自身のaccount管理機能を使用する必要があります。

## 利用者の選択と削除

端末内の同意画面で、処理を始める前にこの内容を説明します。同意しない場合は「同意してAtCoderへ進む」を選択せず、Chromeを閉じることで拒否できます。

実行中の検証を止めるには、専用Chrome windowを閉じます。確認済みsessionをmacOS Keychainから削除する方法は、[サポートページ](v12-extension-support.md)を参照してください。サポート依頼へパスワードまたはsession Cookieを記載しないでください。

## セキュリティ

端末内転送ではIPv4 loopback接続だけを受け付け、request host、拡張機能origin、一回限りのBearer token、content type、message size、拡張機能版、同意版、event順序を検査します。AtCoderとの通信にはHTTPSを使用します。拡張機能はremote codeを実行しません。

## Limited Use

Chrome APIから受け取った情報の利用は、[Chromeウェブストア ユーザーデータポリシーのLimited Use要件](https://developer.chrome.com/docs/webstore/program-policies/limited-use)に従います。データは、開示した拡張機能の単一目的を提供するためだけに使用します。

## 変更と問い合わせ

処理するデータ、目的、転送先または保持期間を実質的に変更する場合は、変更後の動作を使用する前にこのポリシーを更新し、新たな同意を求めます。冒頭の最終更新日が現在の版を示します。

privacyに関する問い合わせには、Chromeウェブストアの掲載情報に表示される確認済みpublisher連絡先を使用してください。機微でない技術的な問い合わせは[サポートページ](v12-extension-support.md)を使用できます。公開issueへ、パスワード、session Cookie、一回限りのtoken、実account識別子、その他の秘密情報を記載しないでください。
