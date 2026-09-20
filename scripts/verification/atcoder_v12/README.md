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
| [`v12e-submission/`](v12e-submission/README.md) | `V-12E`の提出前入力。対象問題、提出先URL、言語、固定source。**AtCoder由来の内容は含まない** |
| `algoloom_v12_review_fixture.py` | **helperの代用部品（review fixture）。** CWS reviewerがhelperなしで拡張機能を確認するための単一ファイルで、同じprotocolと同じ検査を実装する。**本物のhelperと違い、何も保存せず、外部へ接続しない。** 審査期間だけのもので、製品では使わない |
| `prepare.mjs` | リポジトリ外へ拡張ZIP、実行時compile不要のhelper、reviewer受渡し用の再現可能な`.tar.gz`を排他的に生成する |
| `prepare-store-assets.mjs` | clean buildに対応する実同意UIのscreenshot、small promo、iconをリポジトリ外へ生成する |

拡張機能の版は、対象版兼更新元`0.1.1`と更新先`0.1.2`です。**`0.1.0`は2026年9月19日に同意画面のcontent scriptの不具合が見つかったため対象版から外しました**（[ADR-0005](../../../docs/decisions/0005-verify-consent-flow-in-browser-semantics.md)）。helper版は`0.1.0`、protocol版は`1`、template schema版と同意版は`1.0`です。Chrome Web Storeが割り当てる固定IDは、最初の外部操作が承認され、実際のitemを作るまでsourceへ仮置きしません。

## 外部接続なしの事前test

次はAtCoder、Chrome Web Store、Cloudflareへ接続しません。

```console
go test ./...
node --test scripts/verification/test_atcoder_v12.mjs
python3 scripts/verification/atcoder_v12/algoloom_v12_review_fixture.py --self-test
```

Go testは、protocolの状態順序、版・同意不一致、`Host`、接続元、拡張機能origin、Bearer token、JSON本文上限・余剰data、Cookieの一意性、redaction、取消、timeout、browser process終了、profile file lock、template完全性、campaign manifestと結果無効化を固定入力で確認します。Node testは、拡張機能の権限、Cookie取得範囲、ログイン・Turnstile・提出の非自動化、秘密値の非出力、build入力にpublisher credentialがないこと、review fixtureがhelperと同じ検査を満たすこと、quarantine属性が付いた複製でも動くことを確認します。あわせて、同意画面のcontent scriptを**実際に評価**し、動的な待受番号が付くURLでhandlerが登録されて`initialize`が送られること、loopback以外のページでは何も起きないことを確認します。**ソースの文字列検査だけを成立証拠にしません**（[ADR-0005](../../../docs/decisions/0005-verify-consent-flow-in-browser-semantics.md)）。

review fixtureの`--self-test`は、socketを一つも開かず固定入力だけでprotocolを確認します。16のcaseで、`Host`、接続元、拡張機能origin、Bearer token、`Content-Type`、32 KiB上限、状態順序、余剰key、版不一致、自動操作識別値、Cookieの範囲と属性、本人不一致、そして**受け取った値がどこにも残らないこと**を検査します。

## browserで起きることを、手元の想定で代用しない

**これは3度失敗した箇所です。4度目を起こさないための規則です。**

| いつ | 何が起きたか | 決定 |
|---|---|---|
| 2026年9月19日 | 同意画面のcontent scriptが、sourceの検査では正しく見えるのに**実browserで動かなかった。** `location.origin`でloopbackを判定していたため、動的な待受番号が付くURLで常に`return`していた | [ADR-0005](../../../docs/decisions/0005-verify-consent-flow-in-browser-semantics.md) |
| 2026年9月20日 | 提出確認画面のform POSTを**helper自身が拒否した。** 契約testが`Origin`ヘッダーを手で立てていたため、`Referrer-Policy: no-referrer`のページからのPOSTでChromeが`Origin: null`を送ることを一度も通していなかった | [ADR-0012](../../../docs/decisions/0012-serve-submission-page-with-same-origin-referrer-policy.md) |
| 2026年9月21日 | 提出確認画面から提出pageへの**受け渡しが成立しなかった。** helperはPOSTを受理して303を書いたが、browserは提出pageへ着かなかった。**CSPの`form-action`がloopback originだけを許していたため、Chromeがform POSTの後の303遷移を止めていた。** この経路の契約testは`httptest`でhandlerを直接呼ぶため、**browserが応答を受け取って遷移するところを一度も通していない** | [ADR-0013](../../../docs/decisions/0013-find-the-cause-before-fixing-the-v12e-handoff.md)、[ADR-0014](../../../docs/decisions/0014-allow-the-submit-origin-in-the-submission-page-form-action.md) |

3つとも、**確かめたい相手はbrowserなのに、browserの代わりに自分の思い込みを置いていた**ために起きました。テストは通り、実機で止まりました。

規則は4つです。

1. **requestのヘッダーを手で立てる前に、その値をbrowserが決めるのかを判定します。** helperが生成する値（`Authorization`のtoken）や、拡張機能が明示的に付ける値（`Content-Type: application/json`）は手で立ててかまいません。**browserが決める値は代用しません**
2. **browserが決める値は、実挙動を測ってから固定します。** 測り直せる形で残し、「前に確かめた」で済ませません
3. **測る仕組みには必ずnegative controlを入れます。** 壊れた条件を検出できることを先に見せない限り、成功は成立証拠になりません
4. **直す前に、その仮説が症状を説明できることを確かめます。** 2026年9月21日、303が届かない症状に対して「通知が応答より先だから`Shutdown`が先回りする」という仮説を立てましたが、**4条件×20回の再現ですべて303が届き、否定されました**（[ADR-0013](../../../docs/decisions/0013-find-the-cause-before-fixing-the-v12e-handoff.md)）。**確かめずに直していたら、人が15分操作するcampaignで同じ場所に止まっていました。** 測るのに15分もかかりません

### 手で組み立てているrequestの分類

2026年9月20日に洗い出した結果です。

| 箇所 | ヘッダー | browserが決めるか | 扱い |
|---|---|---|---|
| `/event`・`/capture`のbody検査 | `Content-Type: application/json` | **決めない。** `service_worker.js`が明示的に付ける | 手で立ててよい |
| loopback handlerの認可検査 | `Authorization: Bearer <token>` | **決めない。** tokenはhelperが生成し、拡張機能が付ける | 手で立ててよい |
| loopback handlerの認可検査 | `Origin: chrome-extension://<固定ID>` | **決める。** service workerの`fetch`に付く | `V-12D`が3回の実campaignで合格しており、実挙動で裏づけ済み |
| loopback handlerの接続検査 | `Host`、接続元 | **決める** | 同上 |
| 提出確認画面の`proceed`検査 | `Origin: http://127.0.0.1:<待受番号>` | **決める** | **ここが2度目の失敗。** `browser-request-probe.mjs`で測る |
| 提出確認画面の`proceed`検査 | `Content-Type: application/x-www-form-urlencoded` | **決める。** form POSTでbrowserが付ける | 同じprobeで測り、一致を確認した |
| 提出確認画面から提出pageへの303遷移 | CSPの`form-action`の適用範囲 | **決める。** browserがform POSTの後の遷移先も検査する | **ここが3度目の失敗。** [受け渡しのprobe](submission-handoff-probe.mjs)で測る |

### 実挙動を測る

```console
node scripts/verification/atcoder_v12/browser-request-probe.mjs
node scripts/verification/atcoder_v12/submission-handoff-probe.mjs
```

どちらもローカルだけで完結します。**AtCoder、Chrome Web Store、Cloudflareへ接続しません。** 使い捨てChrome profileをリポジトリ外に作り、終了時に破棄します。

1つ目は、提出確認画面からのform POSTで`Origin`が保たれるかを測ります。2つ目は、**押した後に提出pageへ着くか**を測ります。提出pageの代わりに別portのローカル待受を置き、そこへのGETが届いたかどうかで判定します。portが違えばoriginも違うため、AtCoderへの遷移と同じ「別originへの303」になります。**画面は実物の`submission.html`とhelperと同じヘッダーを使い、人の指の代わりにscriptだけを足しています。**

受け渡しのprobeのnegative controlは、**`form-action`をloopback originだけにした状態**（5回目のcampaignの設定）です。これが「着かない」と出ることを先に見てから、現行の設定の結果を成立とします。あわせて「応答を書かずに接続を切る」も測ります。

probeはhelperの`submissionReferrerPolicy`を**sourceから読みます。** 実装とprobeが別々に動いて食い違うことを防ぐためです。あわせて`no-referrer`をnegative controlとして必ず測り、**`Origin`が落ちる条件を検出できることを示してから**、helperが使う値の結果を成立とします。

**これらのprobeはCIで実行しません。** 通常Google Chromeを必要とし、CIの実行環境にないためです。CIが守るのは固定入力testのほうで、`Referrer-Policy`が`Origin`を保つ値から外れたときと、CSPの`form-action`が2つのorigin以外になったときにGo testが落ちます。**probeは「その指定が実機で正しいか」を確かめるもので、campaignの開始前と、この経路へ触れたときに人が実行します。**

`Referrer-Policy`は見た目の設定ではなく、`Origin`の検査と対になっています。提出確認画面は`same-origin`で配信します。同一originへは正しい`Origin`を送り、AtCoderへは`Referer`を送りません。**`no-referrer`へ戻すと固定入力testが落ちます。**

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

`.tar.gz`はuid・gid・modeとmtimeを固定したustar形式で組み立てます。`prepare.mjs`は生成のたびに再生成して一致を確認し、一致しなければ`review_bundle_not_reproducible`で停止します。

**再現の単位はsource treeです。** すべての配布物が、同じsourceから同じbytesになります。

これは2026年9月20日に変えました。それまではGoが`vcs.revision`、`vcs.time`、`vcs.modified`をbinaryへ埋め込むため、**helper sourceが1 byteも変わらなくてもcommitが変われば実行ファイルのbytesが変わっていました。** 2026年9月19日の実測では、同じhelper source treeでも`b03ab6b`と`32e05d4`のbuildで288 byteが異なり、差は`buildinfo`のVCS欄でした。

campaign manifestはhelperのhashを「挙動が変わったか」の判定に使うため、この埋め込みがあると**無関係なコミットでcampaignが無効になります。** 判定の代理が本体からずれている状態でした。そこで[`helper-build.mjs`](helper-build.mjs)へ`-buildvcs=false`を置き、build条件を`prepare.mjs`と固定入力testで共有しています。判断の経緯は[ADR-0007](../../../docs/decisions/0007-make-helper-build-hash-track-behaviour.md)を参照します。

**source treeが同じなら、commitが進んでも作業treeがdirtyでも同じbytesになります。** 2026年9月20日に、別々のcommitからのbuildが同じ`9cf93575…`になることを実測しました。固定入力testは、buildした実行ファイルにcommit idが現れないことを毎回確認します。

経路1の`.tar.gz`のSHA-256を外部へ示す場合も、source treeで再現できます。

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

### エラー名の方針

**エラー名は「次に取るべき行動」で分けます。** 安全側で停止するという結果が同じでも、読んだ人が次にやることが違うなら別の名前にします。2026年9月20日に`first_login_secret_namespace_not_empty`が、secret storeに項目が残っている場合と設定が不正な場合の両方で返り、**残っていない項目を消そうとする**誤った対処へ誘導しました（[`TD-44`](../../../TODO.md#td-44-helperのエラーが原因を一意に指せない箇所を洗い出して直す)）。

分けたものです。

| 停止の原因 | エラー名 | 次に取る行動 |
|---|---|---|
| campaign manifestを正規化できない | `manifest_hash_unavailable` | manifestを直す |
| `--expected-manifest-sha256`が64桁の16進でない | `expected_manifest_hash_invalid` | 引数を直す |
| hashが一致しない | `manifest_hash_mismatch` | 参照しているrevisionを確かめる |
| `profile.status`が`pending_v12b`でない | `first_login_profile_not_pending` | 初回導線用のrevisionを使う |
| 固定ID・対象版・listingが引数と違う | `first_login_extension_mismatch` | 対象itemを確かめる |
| 同意版・template schemaが違う | `first_login_consent_or_template_mismatch` | 同意文面かschemaを合わせる |
| helper版・protocol版が違う | `first_login_helper_contract_mismatch` | buildし直す |
| 実行中のhelper自身のhashが違う | `first_login_self_hash_mismatch` | manifestのbuildで実行する |
| Keychain adapterのhashが違う | `first_login_keychain_helper_hash_mismatch` | 渡したpathを確かめる |
| service IDの書式、または実行ファイルのpathが不正 | `first_login_verifier_configuration_invalid` | **引数を直す。項目を消しに行かない** |
| secret storeに項目が残っている | `first_login_secret_namespace_not_empty` | `secret delete`で消す |
| 対象版が導入されていない | `extension_version_not_installed` | 対象itemと版を確かめる |
| 同じ版が複数導入されている | `extension_installation_not_unique` | 重複した導入を片付ける |
| 削除したのに項目が残っている | `secret_store_item_still_present` | 渡したadapterとservice IDを確かめる |
| 削除できたか判定できない | `secret_store_deletion_unverifiable` | Keychain adapterが動くかを確かめる |
| secret storeを参照できず判定できない | `first_login_secret_store_unavailable` | Keychain adapterが動くかを確かめる |
| 子processの出力が上限を超えた | `first_login_child_output_too_large` | **秘密値の混入を疑う** |
| 子processの出力を読めない | `first_login_output_undecodable` | helperの不具合として扱う |
| 認証が成立しなかった | `first_login_authentication_not_ok` | 子processの停止理由を見る |

**分けなかったものもあります。** `*_arguments_invalid`（引数の解析失敗、余剰引数、timeoutの範囲外）、`manifest_size_invalid`（空、上限超過）、`manifest_identity_invalid`（schema版、revision、campaign ID）は、原因が違っても**次に取る行動が「渡した値を直す」で同じ**なので、一つの名前のままにしています。

`keychainItemAbsent`は「項目なし」「項目あり」「判定できなかった」の3つを返します。**判定できなかったものを「項目あり」と報告しません。** 終了コード44が「なし」、0が「あり」、それ以外は判定できていない、という対応です。

### `V-12E`の`submit`相当の入口

`submit-entry`は、保存済みlocal sessionが無い状態から`V-12E`の一連を実行します。**製品の`aloom submit`ではありません。** 提出は行わず、提出formも操作しません。

```console
printf '%s\n' "$EXPECTED_ACCOUNT" | "$V12_HELPER" submit-entry \
  --manifest /absolute/owner-only/campaign-manifest-r2.json \
  --expected-manifest-sha256 aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --extension-id aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
  --extension-version 0.1.1 \
  --consent-version 1.0 \
  --template-schema-version 1.0 \
  --keychain-helper /absolute/owner-only/algoloom-v12-keychain-darwin-arm64 \
  --keychain-service io.algoloom.verification.v12.00000000000000000000000000000000.session \
  --chrome "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --template /absolute/owner-only/baseline-template \
  --runtime /absolute/owner-only/runtime-profile-v12e \
  --repository-root /absolute/path/to/algo-loom-design \
  --submission-input /absolute/path/to/algo-loom-design/scripts/verification/atcoder_v12/v12e-submission/input.json
```

順序と、どこを人が操作するかです。

| # | 場所 | 誰が | 何が起きるか |
|---|---|---|---|
| 1 | CLI | helper | manifest・自分自身・Keychain adapterのhashと、基準templateの完全性IDを照合する |
| 2 | CLI | helper | **保存済みlocal sessionが無いことを確認する。** 残っていれば`submit_entry_local_session_present`で停止し、消しに行かない |
| 3 | CLI | helper | **再認証の理由、中止方法、対象問題・言語・sourceのhashとbytes、提出先を表示する。** 質問はしない |
| 4 | browser | 人 | 同意画面（既存）で「同意してAtCoderへ進む」 |
| 5 | browser | 人 | AtCoderへログインする。Turnstileが出たら操作する |
| 6 | browser | 拡張機能とhelper | `/settings`で本人照合し、`REVEL_SESSION`を1件だけ受け取り、`GET /settings`1回で確認してsecret storeへ保存する |
| 7 | browser | helper | 同じChromeで**提出確認画面**（`http://127.0.0.1:<待受番号>/submission`）を開く |
| 8 | browser | 人 | 対象問題・言語・sourceを見て「AtCoderの提出画面へ進む」を押す |
| 9 | browser | helper | 303応答で同じtabをAtCoderの提出pageへ送る。**formへは何も入れない** |
| 10 | CLI | 人 | Chromeを完全終了する。helperがruntime複製を破棄して結果JSONを返す |

提出確認画面は同意画面と同じ仕組みで配信しますが、**JavaScriptを持ちません。** 押されたことは同じoriginへのform POSTで伝わり、遷移は303応答で起きます。拡張機能のcontent scriptは`/bootstrap`でしか動かないため、この画面では何もしません。

**helperが観測できるのは303を返したところまでです。** browserが提出pageへ着いたことは、提出ページで拡張機能を動かさない限り観測できません。それは`V-12`の範囲外です（[`TD-40`](../../../TODO.md#td-40-提出ページのcontent-scriptとturnstileの共存を検証する)）。

そのため結果JSONは**到達を埋めません。**

| field | 意味 |
|---|---|
| `observed_scope` | `helper_observable_only`。**helperが観測できた範囲だけが成立したという意味である** |
| `submit_page_redirect_issued` | 303を書いた。書いたというだけで、届いたことも着いたことも含まない |
| `submit_page_arrival` | `unobserved_by_helper`。**真偽値にしていない。** `true`か`false`だと、helperが見ていない結果を見たように読める |

**`ok: true`は`V-12E`の合格を意味しません。** 到達は人が画面で見た事実として実行記録へ書きます。2026年9月21日の5回目のcampaignでは、browserが着いていないのにhelperが`ok: true`を返し、結果JSONだけを見ると成立したと読めました（[ADR-0013](../../../docs/decisions/0013-find-the-cause-before-fixing-the-v12e-handoff.md)）。

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

`secret delete`は、削除したあとに**項目が無いことを別の観測で確かめてから**成功を返します。削除commandの終了コード0だけを成功と読みません。Keychain adapterでない実行ファイルを渡しても0で終わりうるためです。

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
