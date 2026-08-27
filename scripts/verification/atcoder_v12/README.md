# AtCoder V-12 製品相当検証物

このディレクトリは、[`V-12`](../../../docs/project/judge-adapter-verification.md#311-v-12-方式a製品形態の検証)の成立性だけを確認する検証支援物です。製品コード、正式release、一般利用者向け配布物、CI用認証手段ではありません。検証物のコードを製品実装へ複製せず、`V-12`で観測できた契約だけを設計へ戻します。

現在の対象環境は、代表環境であるApple silicon搭載macOS、通常Google Chrome、macOS Keychainです。Windows Credential ManagerとLinux Secret Serviceは、後続の`TD-12`で検証マトリクスを決める前に「対応済み」と扱いません。

## 構成

| 場所 | 役割 |
|---|---|
| `extension/` | Chrome Web Storeへ提出するManifest V3拡張機能のsource。単一目的、最小権限、固定したCookie範囲を持つ |
| `helper/` | protocol、AtCoder本人照合、Keychain adapter呼出し、profileの確定・複製・破棄、campaign manifest検査を行うGo helper。認証ヘルパーの役割は[AtCoder認証設計 §1.2](../../../docs/architecture/atcoder-authentication.md#12-認証ヘルパーとは何か)を参照 |
| `keychain/` | 実行前に配布物へcompileするmacOS Security Framework adapter |
| `consent-v1.0.ja.md` | 初回画面と対応付ける同意文面の正本 |
| `fixtures/` | 秘密値を含まないmanifest例。実campaignへ流用しない |
| `algoloom_v12_review_fixture.py` | **helperの代用部品（review fixture）。** CWS reviewerがhelperなしで拡張機能を確認するための単一ファイルで、同じprotocolと同じ検査を実装する。**本物のhelperと違い、何も保存せず、外部へ接続しない。** 審査期間だけのもので、製品では使わない |
| `prepare.mjs` | リポジトリ外へ拡張ZIP、実行時compile不要のhelper、reviewer受渡し用の再現可能な`.tar.gz`を排他的に生成する |
| `prepare-store-assets.mjs` | clean buildに対応する実同意UIのscreenshot、small promo、iconをリポジトリ外へ生成する |

拡張機能の版は、対象版兼更新元`0.1.0`と更新先`0.1.1`です。helper版は`0.1.0`、protocol版は`1`、template schema版と同意版は`1.0`です。Chrome Web Storeが割り当てる固定IDは、最初の外部操作が承認され、実際のitemを作るまでsourceへ仮置きしません。

## 外部接続なしの事前test

次はAtCoder、Chrome Web Store、Cloudflareへ接続しません。

```console
go test ./...
node --test scripts/verification/test_atcoder_v12.mjs
python3 scripts/verification/atcoder_v12/algoloom_v12_review_fixture.py --self-test
```

Go testは、protocolの状態順序、版・同意不一致、`Host`、接続元、拡張機能origin、Bearer token、JSON本文上限・余剰data、Cookieの一意性、redaction、取消、timeout、browser process終了、profile file lock、template完全性、campaign manifestと結果無効化を固定入力で確認します。Node testは、拡張機能の権限、Cookie取得範囲、ログイン・Turnstile・提出の非自動化、秘密値の非出力、build入力にpublisher credentialがないこと、review fixtureがhelperと同じ検査を満たすこと、quarantine属性が付いた複製でも動くことを確認します。

review fixtureの`--self-test`は、socketを一つも開かず固定入力だけでprotocolを確認します。16のcaseで、`Host`、接続元、拡張機能origin、Bearer token、`Content-Type`、32 KiB上限、状態順序、余剰key、版不一致、自動操作識別値、Cookieの範囲と属性、本人不一致、そして**受け取った値がどこにも残らないこと**を検査します。

テスト成功は、Chrome Web Storeの審査、標準追加、AtCoder実サービス、Keychainへの実session保存、`V-12`全体の合格証拠ではありません。

## 隔離build

所有者だけがアクセスできるリポジトリ外の親ディレクトリを作り、その配下のまだ存在しないpathを指定します。

```console
node scripts/verification/atcoder_v12/prepare.mjs \
  /absolute/owner-only/v12-preparation/build-01
```

`prepare.mjs`は次を生成します。

- CWS upload用`0.1.0`、`0.1.1` ZIP
- `darwin/arm64`用Go helper実行ファイル
- `darwin/arm64`用Keychain helper実行ファイル
- reviewer受渡し用`algoloom-v12-review-darwin-arm64.tar.gz`（helper、Keychain adapter、review fixture、`SHA256SUMS`、`README.txt`）
- 単体で配れるreview fixture
- source revision、source tree hash、各配布物のSHA-256とbytesを持つ`build-index.json`

`.tar.gz`はuid・gid・modeとmtimeを固定したustar形式で組み立て、同じsourceからbyte単位で再現します。`prepare.mjs`は生成のたびに再生成して一致を確認し、一致しなければ`review_bundle_not_reproducible`で停止します。

拡張ZIPとindexは`0600`、実行ファイルは`0700`です。GoとSwiftはこの準備時にだけcompileし、`V-12B`〜`V-12E`の実行時にはcompileしません。作業treeがdirtyならindexの`campaign_ready`は`false`になり、そのbuildをCWS uploadまたはcampaign manifestへ使いません。

`build-index.json`の`signed_extension_artifacts`は意図的に空です。CWSの署名済み配布物を、標準追加の前にローカルbuildで代用してはいけません。承認後にCWSから配信された対象版の正確なbytesを取得・hash照合できた場合だけ、campaign manifestの`signed_builds`を埋めます。取得できない場合は`V-12A`を不合格として停止し、インストール済みdirectoryを「署名済みbuild」と読み替えません。

### Store listing asset

cleanなsource revisionから隔離buildを作った後、そのbuild rootを指定してlisting assetを生成します。

```console
node scripts/verification/atcoder_v12/prepare-store-assets.mjs \
  /absolute/owner-only/v12-preparation/build-01
```

`listing/`へ、ZIP内と同じ128×128 PNG icon、440×280 PNG small promo、1280×800 PNG screenshot、各hashとcapture条件を持つ`listing-index.json`を作ります。screenshotはhelperへ埋め込む実際の同意HTMLを、外部account・拡張機能実行・AtCoder接続なしの`file:`表示でcaptureしたものです。これはStore listing用assetであり、通常Chrome、標準追加、一往復UXまたは`V-12`の合格証拠にはしません。small promoはtextを含まないbrand図形で、実機能を追加示唆しません。

## reviewerへの受渡し

CWS reviewerへは2経路を用意します。どちらもGatekeeperの警告を無効化・上書きさせません。

| 経路 | 内容 | 実測した性質 |
|---|---|---|
| 経路1 | `.tar.gz`を`curl`で取得して`tar`で展開し、事前build済みhelperを実行する | `curl`はquarantine属性を付けないため、ad-hoc署名のhelperがそのまま動く |
| 経路2 | review fixtureを`python3`で実行する | **取得方法にかかわらず動く。** scriptにはGatekeeperが適用されない |

2026年8月27日にmacOS 26.5・Apple siliconで実測した結果は次のとおりです。

| 取得方法 | 展開後のquarantine属性 | helper（Mach-O） | fixture（script） |
|---|---|---|---|
| `curl` + `tar` | 付かない | 実行できる | 実行できる |
| ブラウザ等でダウンロード + `tar` | **付く。`tar`が展開先へ伝播する** | **`SIGKILL`で停止** | **実行できる** |

**`tar`はアーカイブのquarantine属性を展開したfileへ伝播します。** したがって経路1は取得方法に依存し、経路2だけが取得方法に依存しません。

### なぜ「フィクスチャ」と呼ぶか

`fixture`は、テストや検証のために用意する、**内容または振る舞いが固定されたもの**を指します。このリポジトリでは架空の問題データのような**固定データ**にも使いますが、ここでは**本物の代わりに置く代用部品**の意味です。

- **「スクリプト」は実装形式**を指します。Pythonで書いてあり、compileせずに実行できるという意味です
- **「フィクスチャ」は役割**を指します。本物のhelperの代わりに置く代用品だという意味です

同じファイルが両方に当てはまります。矛盾しません。

確かめたいのは**拡張機能**であり、fixtureはそのために必要な相手役です。拡張機能から見ると、同じprotocolで同じ応答を返すため本物と区別がつきません。

fixtureは受け取ったCookieを検査した後に破棄します。秘密情報保管庫、file、logのいずれへも書きません。AtCoderを含む外部hostへ接続しません。製品相当helperが行う本人照合とKeychain保存は、拡張機能の審査範囲外のため実装していません。**審査期間だけのものであり、製品のhelperをPythonで配るという意味ではありません。**

## helperの公開command

以下の例の`V12_HELPER`は事前build済みのGo helperです。実値をshell履歴へ貼る手順は採用しません。

### `V-12B → V-12D`の分断しない初回導線

`first-login`が、CWS標準追加から新process再照合までを一つのhelper invocationで管理します。実行前に`manifest validate --subtest V-12A`で得たcanonical hashを固定します。期待accountは引数・環境変数ではなくstdinで1行だけ渡します。

```console
"$V12_HELPER" first-login \
  --manifest /absolute/owner-only/campaign-manifest-r1.json \
  --expected-manifest-sha256 aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --listing-url https://chromewebstore.google.com/detail/algoloom/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --extension-id aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --extension-version 0.1.0 \
  --consent-version 1.0 \
  --template-schema-version 1.0 \
  --keychain-helper /absolute/owner-only/algoloom-v12-keychain-darwin-arm64 \
  --keychain-service io.algoloom.verification.v12.00000000000000000000000000000000.session \
  --chrome "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --setup-profile /absolute/owner-only/new-v12b-setup-profile \
  --template /absolute/owner-only/baseline-template \
  --runtime /absolute/owner-only/runtime-profile-normal \
  --repository-root /absolute/path/to/algo-loom-design
```

commandは、manifestの固定ID・対象版・listing・同意・template schema・helper版・protocol版、代表環境のmacOS・architecture・Chrome版・secret storeと、実行中のhelper・Keychain helperのbytes/hashを照合してから通常ChromeをCWS listingへ開きます。利用者が標準追加を完了すると固定IDで自動検出し、terminalへ`extension_installation_detected_close_chrome`だけを表示します。利用者がChromeを完全終了すると、同じprocessがsetup profileから基準templateを一度だけ確定し、setup profileを削除し、runtime cloneを作り、local同意画面を開いて`serve`を続けます。

正常時は新process再照合とChrome完全終了を確認してruntime cloneを削除し、秘密でないtemplate完全性IDをJSONで返します。基準templateと確認済みKeychain項目だけを`V-12E`まで保持します。取消、timeout、標準追加なしのChrome終了、manifest・版・hash不一致、profile lockでは安全側に停止し、作成済みの専用marker付きprofileだけをcleanupします。

### manifest

```console
"$V12_HELPER" manifest validate \
  --file /absolute/owner-only/campaign-manifest.json \
  --subtest V-12A

"$V12_HELPER" manifest compare \
  --before /absolute/owner-only/manifest-r1.json \
  --after /absolute/owner-only/manifest-r2.json
```

`validate`は未知fieldを拒否し、固定ID、版、限定公開CWS URL、正規化した権限、source・build hash、代表環境、template状態を検査します。`V-12A`だけは`profile.status=pending_v12b`を許可します。`V-12B`以降は`fixed`と完全性IDが必要です。sub結果へはcanonical manifest hashとsubtest projection hashだけを記録し、絶対pathを記録しません。

### profile

```console
"$V12_HELPER" profile inspect \
  --root /absolute/owner-only/disposable-setup-profile \
  --extension-id aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --extension-version 0.1.0

"$V12_HELPER" profile finalize \
  --source /absolute/owner-only/new-v12b-setup-profile \
  --template /absolute/owner-only/baseline-template \
  --repository-root /absolute/path/to/algo-loom-design \
  --extension-id aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --extension-version 0.1.0 \
  --schema-version 1.0

"$V12_HELPER" profile clone \
  --template /absolute/owner-only/baseline-template \
  --runtime /absolute/owner-only/runtime-profile-case-a \
  --repository-root /absolute/path/to/algo-loom-design

"$V12_HELPER" profile destroy \
  --runtime /absolute/owner-only/runtime-profile-case-a \
  --repository-root /absolute/path/to/algo-loom-design
```

`inspect`と`finalize`は固定ID・版・権限が一致し、Chromeの`SingletonLock`等がなく完全終了している場合だけ成功します。`finalize`はChrome account・sync状態がないことを検査し、履歴、Cookie、login data等のbrowser stateを除去してから完全性IDを作ります。sourceはCWSの標準追加だけを行った新規profileとし、ChromeへのGoogle account追加・syncが必要なら停止し、AtCoderを一度でも開いたprofileを入力にしません。

`clone`は完全性を再計算し、専用markerを持つ新規runtimeだけを作ります。`destroy`は所有者専用・リポジトリ外・専用runtime marker・lockなしをすべて確認してから、その正確なruntime rootだけを削除します。利用者の既存profile、広いpath、symlink、markerなしdirectoryは削除しません。

準備確認に使った`disposable-setup-profile`は破棄し、`V-12B`の基準templateへ流用しません。基準templateは`TD-11`の`V-12B → V-12D`を分断しない初回導線の中で、新しいprofileから一度だけ確定します。

### 認証付きloopbackとKeychain

`serve`は`first-login`が内部で使う低水準commandです。`V-12C`の独立case、`V-12E`、またはcampaign manifest検査とruntime cloneを済ませた検証harnessだけが直接呼びます。期待accountは引数・環境変数・fileではなくstdinで1行だけ渡します。

```console
"$V12_HELPER" serve \
  --extension-id aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --extension-version 0.1.0 \
  --consent-version 1.0 \
  --keychain-helper /absolute/owner-only/algoloom-v12-keychain-darwin-arm64 \
  --keychain-service io.algoloom.verification.v12.00000000000000000000000000000000.session \
  --chrome "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --profile /absolute/owner-only/runtime-profile-case-a
```

helperは`127.0.0.1`の動的portへbindし、一回限りの64桁token、正確な`Host`、IPv4 loopback接続元、`chrome-extension://<固定ID>` origin、Bearer認証、`application/json`、32 KiB上限、余剰fieldと状態順序を検査します。tokenはAtCoderへ送らず、通常log、JSON結果、campaign evidenceへ出しません。

利用者はlocal同意画面で版1.0の要点を読み、「同意してAtCoderへ進む」を押します。その後は同じ通常ChromeでAtCoderのloginと必要なTurnstileだけを操作します。拡張機能は`/settings`の本人を自動照合し、`REVEL_SESSION`を厳密に1件だけhelperへ渡します。helperは`GET /settings`で本人を確認し、応答に正しい同Cookie更新が1件あればその値を採用してKeychainへ保存します。新しいhelper processがKeychainから読み、2秒以上空けた2回目の`GET /settings`で再照合します。

Cookieと実account名は、URL、argv、環境変数、通常log、公開JSONへ含めません。パスワード、Turnstile token、他のCookieを読みません。CDP、WebDriver、remote debugging、headless、stealth、User-Agent偽装を使いません。POST、自動再試行、提出は実装していません。

正常完了後も、利用者がChromeを完全終了するまでhelperは成功を返しません。取消、全体timeout、Chrome先行終了、helper終了では安全側に停止します。確認済みのKeychain項目はcampaign終了まで保持するため、正確なservice IDをmanifest外のstore用一時情報で管理し、終了時は次で削除します。

```console
"$V12_HELPER" secret delete \
  --keychain-helper /absolute/owner-only/algoloom-v12-keychain-darwin-arm64 \
  --keychain-service io.algoloom.verification.v12.00000000000000000000000000000000.session
```

## campaignと無効化

manifestの例は[`fixtures/campaign-manifest.example.json`](fixtures/campaign-manifest.example.json)です。すべての値がfixtureであり、実campaignへ複製しません。実manifestは、CWS固定IDとlisting URL、upload ZIP、CWS配信済み対象版、事前build済みhelper、計画revision、同意文面hash、実行環境を観測してリポジトリ外へ新規作成します。

| 変更 | 無効化 | 新campaign |
|---|---|---|
| 対象拡張の固定ID・版・配布元・権限・source・対象版build | `V-12A`〜`V-12E` | 必須 |
| 更新testの版組・更新先artifactだけ | `V-12A`、`V-12C` | 不要 |
| helper・protocol | `V-12A`〜`V-12E` | 必須 |
| Chrome・OS環境 | `V-12B`〜`V-12E` | 不要 |
| template schema・完全性 | `V-12B`〜`V-12E` | 不要 |
| 同意版・同意文面hash | `V-12A`〜`V-12E` | 必須 |

判定は`manifest compare`のJSONを記録します。旧結果を新revisionへ付け替えません。再利用するsub結果は、入力projection hash一致、旧campaign ID、旧証拠、再承認を新しいsub結果へ明記します。

## `TD-11`での順序と後始末

1. 最終manifest入力を固定し、外部通信0件で`V-12A`を再実行する。
2. `V-12B → V-12D`を同じharness processから開始する。新規profileでCWS標準追加、固定ID検出、Chrome完全終了、基準template確定、runtime複製、同意、AtCoder login、Cookie受領、本人照合、Keychain保存、新process再照合まで分断しない。
3. `V-12D`後にruntime profile、未確認session、loopback、子process、file lockを回収する。基準templateと確認済みKeychain項目は隔離して保持する。
4. `V-12C`をcaseごとの新しいruntimeとKeychain namespaceで実行する。template不要の版・権限不一致、取消、timeout、process終了、file lock、redactionは事前testと対応付ける。template・環境更新caseだけは基準template確定後の使い捨て複製または隔離snapshotで行う。
5. `V-12E`直前にlocal Keychain項目だけを削除し、別の`submit`相当入口から同じ基準templateを使って再認証し、同じChromeの提出確認画面まで進む。最後の提出操作は行わない。
6. 各case後にruntime、未確認session、一回限りtoken、loopback、process、file lockを0件にする。campaign終了時に基準template、store用一時情報、正確なKeychain項目を削除する。

CWS itemを残す場合のowner、限定公開範囲、停止方法は成果物へ記録します。外部操作の承認境界とlisting・privacy準備は[`v12-chrome-web-store-preparation.md`](../../../docs/verification/judge-adapter/v12-chrome-web-store-preparation.md)を正とします。
