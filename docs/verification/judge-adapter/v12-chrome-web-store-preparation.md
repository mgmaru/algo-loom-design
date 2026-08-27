# V-12 Chrome Web Store配布準備

> 確認日: 2026年8月26日
>
> 状態: `TD-37`のローカル準備は完了。`TD-39`はreviewer用helperの受渡し方式を確定できず一度停止したが、2026年8月26日の再調査で停止理由が誤りであることを確認し、費用を伴わない方式を確定して再開した（[§4.1](#41-macos側の配布要件の再調査結果)、[§4.2](#42-採用するreviewer用helperの受渡し方式)）。CWSではdeveloper登録、新規itemへの`0.1.0` upload、Store listing・Privacy・Distributionの保存まで完了した。Distribution設定は料金なし・`Unlisted`・日本のみで、item自体はdraftのままである。test instructionsは未入力・未保存で、審査提出、公開、CWS署名済みbuild取得、`0.1.1` uploadは未実施。account名、実address、developer dashboard URLは記録していない
>
> 対象: `TD-39`のCWS審査用helperと限定公開版。製品の正式公開手順ではない

## 用語

本書はChrome Web Storeの画面表記と対応付けるため英語表記を多く残します。主な語を次のとおり対応付けます。

| 本書の表記 | 意味 |
|---|---|
| CWS | Chrome Web Store。Chrome拡張機能を審査、署名、配布するGoogleの公式基盤 |
| **reviewer** | **CWSの審査を行うGoogle側の担当者。** AlgoLoomの利用者でも開発者でもない第三者。拡張機能を実際に動かして判断するため、本拡張機能の場合は端末内helperの役割を渡す必要がある |
| publisher | CWSへitemを登録した主体。本検証では検証owner |
| owner | 本検証の所有者。外部操作を承認し実行する人 |
| item | CWS上の登録単位。1つの拡張機能に1つの固定IDが割り当てられる |
| listing | itemの掲載情報。名称、説明、画像、カテゴリ等 |
| visibility | itemの公開範囲。`Public`、`Unlisted`（限定公開）、`Private`のいずれか |
| helper | 端末内で動く認証ヘルパー。専用ブラウザーの起動、同意画面の配信、拡張機能からのCookie受領、本人照合、秘密情報保管庫への保存を行う。役割の詳細は[AtCoder認証設計 §1.2](../../architecture/atcoder-authentication.md#12-認証ヘルパーとは何か) |
| fixture | テストや検証のために用意する、内容または振る舞いが固定されたもの。固定データと代用部品の両方を指す。本書で扱うのは後者 |
| **review fixture** | **helperの代用部品。** reviewerがhelperなしで拡張機能を確認するための単一fileで、同じprotocolと同じ検査を実装する。**本物のhelperとは違い、受け取った値を保存せず、外部hostへ接続しない。** 審査期間だけのものであり、製品では使わない。Pythonで書いてあるのは実装形式の話で、「スクリプト」は形式、「フィクスチャ」は役割を指す |
| quarantine属性 | macOSがダウンロードしたfileへ付ける`com.apple.quarantine`。Gatekeeperはこの印を見て未notarizeの実行ファイルを止める |
| test instructions | CWS dashboardでreviewerへ手順とcredentialを伝える欄。username・password各100文字と追加手順500文字 |
| deferred publishing | 審査通過だけでは自動公開せず、公開を別操作にする設定 |

登場人物の全体像と、誰がいつどうhelperを入手するかは[拡張機能とヘルパーの配布：ステークホルダーと注意点](../../research/extension-and-helper-distribution.md)を参照します。

## 0. 結論

`V-12`ではChrome Web Store（CWS）の`Unlisted`を使います。ここでいう限定公開は「listing URLを知る利用者が追加できる」範囲であり、指定accountだけに制限する`Private`ではありません。`Unlisted`もCWS policyと審査の対象です。[公式のvisibility説明](https://developer.chrome.com/docs/webstore/cws-dashboard-distribution)

本書でいうhelperは、専用Chromeの起動、拡張機能から受け取った1件のCookieの本人照合、macOS Keychainへの保存、後始末を行う端末内の実行ファイルです。役割の詳しい説明は[AtCoder認証設計 §1.2](../../architecture/atcoder-authentication.md#12-認証ヘルパーとは何か)を参照します。

ローカルbuildと事前testを扱う`TD-37`は完了しました。CWS審査と限定公開を扱う`TD-39`は、reviewerへhelperを安全に渡す方法が確定していないため一度停止しましたが、2026年8月26日の再調査で受渡し方式を確定して再開しました。次の外部操作は、対象、費用、公開範囲を読める形で人間が明示承認するまで行いません。

1. publisher用Google accountのdeveloper登録と登録費用の支払い（2026年8月26日完了）
2. 新しいCWS itemの作成とZIP upload（2026年8月26日完了）
3. 審査への提出
4. 審査通過後の`Unlisted`公開
5. 更新test用`0.1.1`のupload・審査提出・反映
6. campaign終了後にitemを非公開化する操作

一つの承認で複数を許可する場合も、対象item、版、支払画面に表示された金額・通貨・税、`Unlisted`の意味、停止方法、実行者を承認記録へ明記します。「`TD-39`を進めてよい」という一般的な依頼を、問い合わせ送信、契約、支払い、審査提出または公開の承認へ読み替えません。

## 1. 2026年8月26日時点の公式要件

| 観点 | 確認した事実 | V-12での扱い |
|---|---|---|
| developer登録 | CWS developer登録には一回限りの登録費用があり、developer emailは後から変更できないため、監視できるaccountを使うよう案内されている。[登録手順](https://developer.chrome.com/docs/webstore/register/) | 個人の一時accountを推測で選ばない。2026年8月26日の登録画面では一回限り`5 USD`と表示された。税の別表示は観測していない。この日付付き観測を将来の固定価格保証にせず、支払い直前にも表示額を再確認する |
| publisher情報 | publisher表示名と確認済みcontact emailが必要。[account設定](https://developer.chrome.com/docs/webstore/set-up-account) | 検証ownerと連絡先を人が決める。credentialやrecovery情報はGitへ置かない |
| package | ZIPのrootに`manifest.json`を置き、versionを更新ごとに増やす。[package準備](https://developer.chrome.com/docs/webstore/prepare) | `prepare.mjs`の`0.1.0`、`0.1.1` ZIPを使い、再生成時はhashを再固定する |
| dashboard入力 | package、Store listing、Privacy、Distribution、必要ならtest instructionsを埋めて提出する。[公開手順](https://developer.chrome.com/docs/webstore/publish/) | dashboardの最終入力を画面で再確認し、原稿との差を承認記録へ残す |
| visibility | `Unlisted`はlisting URLを知る者が追加でき、検索やcategoryには表示されない。visibilityにかかわらずpolicyとreviewは同じ。[Distribution設定](https://developer.chrome.com/docs/webstore/cws-dashboard-distribution) | URLを秘密のaccess controlとして扱わない。URLを成果物に記す場合も拡散可能性を前提にする |
| review | すべてのitemがreview対象で、権限・変更内容等により時間が延び得る。[review process](https://developer.chrome.com/docs/webstore/review-process) | 審査時間を固定値にせず、通過前は`TD-11`を開始しない |
| pre-submission test | 2026年8月の更新で、自動installation test等を審査提出前に表示する仕組みが案内されている。[2026年review更新](https://developer.chrome.com/blog/cws-review-updates-2026) | dashboardに表示されたtest結果を人が確認し、不合格を回避して提出しない |
| 単一目的・最小権限 | extensionは狭く理解可能な単一目的を持ち、必要最小限のpermissionだけを要求する。[Program Policies](https://developer.chrome.com/docs/webstore/program-policies/policies) | AtCoder sessionを端末内helperへ渡す目的だけに限定し、`cookies`、`storage`、AtCoder、loopbackだけを要求する |
| privacy入力 | single purpose、各permissionの理由、remote code、data利用を申告し、必要なprivacy policyを示す。[Privacy欄](https://developer.chrome.com/docs/webstore/cws-dashboard-privacy) | §3の原稿を使う。認証情報とaccount識別情報の取扱いを過少申告しない |
| data disclosure | 2026年のpolicy更新は、収集するuser dataの正確な開示を要求し、2026年8月1日から既存itemにも適用すると案内している。[2026年policy更新](https://developer.chrome.com/blog/cws-policy-updates-2026) | 「端末内だけだから申告不要」と推測しない。提出時のdashboard設問に合わせ、認証情報とwebsite contentを開示する |
| listing画像 | 128×128 PNG icon、少なくとも1枚のscreenshot、440×280のsmall promo tileが必須。screenshotは1280×800または640×400で実際の体験を示す。[画像仕様](https://developer.chrome.com/docs/webstore/images) | iconはbuild時生成する。`prepare-store-assets.mjs`で実同意UIの1280×800 screenshotとtextなしの440×280 promoをclean buildへ対応付け、placeholderまたは架空のAtCoder画面で提出しない |
| test instructions | test手順は公開の必須条件ではないが、制限付きcredentialまたは有料accountが全機能に必要な場合、reviewerへcredentialと手順を伝えるために使える。[公式説明](https://developer.chrome.com/docs/webstore/cws-dashboard-test-instructions) | 本拡張はAtCoder認証を扱うため手順を記載する。共有AtCoder account、password、Cookieは作成・提供せず、事前build済みhelperと、reviewer自身が利用を許可されたaccountでの手動操作だけを案内する。これで審査不能ならcredentialを共有せずCWS supportへ確認する |
| macOS向け実行ファイル | CWSへ拡張機能を配布することにApple側の要件はない。Apple Developer Program（年額99 USD）が必要になるのは、Developer ID署名とnotarizationを行う場合、すなわちquarantine属性が付く経路でmacOS向け実行ファイルを配布する場合だけである。[Developer ID](https://developer.apple.com/support/developer-id) | reviewer用helperをquarantine属性が付かない経路で渡し、Developer ID署名とnotarizationを必要としない。詳細と実測結果は[§4.1](#41-macos側の配布要件の再調査結果) |
| update | 新しいZIPはversionを増やしてuploadし、再reviewを受ける。CWSは配布packageへ署名する。[更新手順](https://developer.chrome.com/docs/webstore/update) | `0.1.0 → 0.1.1`だけを更新testの版組とし、CWS以外の署名・配布経路へ切り替えない |
| 停止 | dashboardからunpublishでき、再公開には新しいversionとreviewが必要になる場合がある。[unpublishの説明](https://developer.chrome.com/docs/webstore/account-deletion/) | 新規標準追加を止める手段をunpublishに固定する。実行前承認の対象にし、ローカルsession削除と混同しない |

公式文書が「一回限りの登録費用」と説明していても、実際の金額、通貨、税、支払者はdashboardの現在表示でしか確定しません。2026年8月26日のpublisher候補accountの登録画面では「5ドルの登録料を支払う」「アカウントの登録には、1回限りの登録料が必要」と表示されました。税の表示は確認できていません。承認記録がない契約同意または支払いは行いません。

同じ画面で、Google Chrome ウェブストア デベロッパー契約、Googleプライバシーポリシー、Chromeウェブストア デベロッパー プログラム ポリシーへのlinkと、「デベロッパー契約とプライバシー ポリシーを確認し、内容に同意します」という必須同意項目を確認しました。その後、owner本人の判断と操作で同意、支払い、登録を完了しています。

## 2. publisherと配布範囲

publisherは次を満たす一つのGoogle accountに固定します。

- ownerが明確で、検証期間中とitemを残す期間に通知を監視できる
- contact emailを確認できる
- 必要な多要素認証を有効にできる
- credential、recovery code、支払情報をrepository、build入力、campaign manifestへ渡さない
- accountを削除・移管する場合のownerが決まっている

publisher表示名、contact email、Google account自体はこの公開repositoryへ記録しません。匿名化済み成果物には`publisher-A`、`owner-A`等のcampaign内aliasだけを残します。

distributionは`Unlisted`です。`Public`、domain限定`Private`、group限定`Private`、self-hosted、enterprise policy、registry、外部拡張設定を代替経路にしません。標準CWS画面から開発者向け設定なしで追加できなければ停止します。

## 3. Store listingとPrivacy原稿

### 3.1. listing原稿

| field | 原稿 |
|---|---|
| Name | `AlgoLoom Authentication Verification BETA` |
| Summary (`manifest.json`の`description`) | `Transfers one AtCoder session to the local AlgoLoom verification helper after consent.` |
| Single purpose | `After the user explicitly consents and signs in to AtCoder, transfer exactly one AtCoder REVEL_SESSION cookie to the AlgoLoom verification helper running on the same device so the helper can verify the same account and store the verified session in the OS secret store.` |
| Category | dashboardで選べる現在の候補から`Developer Tools`相当を人が選び、表示名を記録する |
| Language | Englishを主、同意と検証UIは日本語。必要なlocalizationは提出前にdashboard上で確認する |

Long description案:

> This unlisted beta extension is a companion for the AlgoLoom V-12 technical verification. It runs only on the local consent page and the AtCoder settings page. After the user consents and manually completes any AtCoder sign-in or Turnstile step, it verifies that one account is visible, reads exactly one allowlisted REVEL_SESSION cookie, and transfers it once to the authenticated helper on 127.0.0.1. It does not automate sign-in, Turnstile, or submissions. It does not request debugger, webRequest, tabs, scripting, nativeMessaging, or broad website access. The helper verifies the same account with GET /settings and stores the verified session in macOS Keychain for the isolated verification campaign.

`BETA`表示を外す、一般利用者向け機能を示唆する、AtCoder公式または提携製品と誤認させる表現を追加する変更は、同じ原稿の軽微修正として扱いません。

### 3.2. permission justification

| permission / host | dashboard用説明 |
|---|---|
| `cookies` | Read exactly one cookie named `REVEL_SESSION` for `https://atcoder.jp/` after consent and a unique account check. The extension does not enumerate other cookie stores and does not set or delete cookies. |
| `storage` | Keep the one-time loopback port, bearer token, consent version, and account-gate tab ID in `chrome.storage.session` only for the active browser session. |
| `https://atcoder.jp/*` | Run the account-check content script only on `/settings` and read the allowlisted AtCoder session cookie. It does not run on the login page or submission pages. |
| `http://127.0.0.1/*` | Receive the local consent bootstrap and send authenticated protocol events to the helper on a dynamic loopback port. No non-loopback HTTP host is allowed. |

Remote codeは`No`です。extension package外のJavaScript、WebAssembly、`eval`、remote scriptを読み込みません。

### 3.3. data disclosure

提出時のdashboardの現在のcategory名に合わせ、少なくとも次を「扱う」と申告します。

| data | 取扱い |
|---|---|
| Personally identifiable information | AtCoder `/settings`に表示されたusername／account識別子を、期待する本人との一致確認に使う。公開結果には値を残さない |
| Authentication information | `REVEL_SESSION`をextension memoryから同じ端末のloopback helperへ一度だけ渡す。helperが本人確認後にcampaign専用macOS Keychainへ一時保存する |
| Website content | AtCoder `/settings`に埋め込まれたaccount識別情報を一意性・期待値一致の確認に使う。公開結果には値を残さない |

売却、広告、credit判定、目的外transfer、人による閲覧を行いません。helperからAtCoderへの`GET /settings`は利用者が要求した認証確認という単一目的だけです。permission justificationとdata-use certificationのdashboard文言がこの説明より広い、または狭い場合は、都合よく選ばず差分を承認者へ戻します。

認証情報を扱うため、公開到達可能なprivacy policy URLを提出前に用意します。[検証用拡張機能のprivacy policy](v12-extension-privacy-policy.md)と[サポートページ](v12-extension-support.md)の原稿には、対象、処理data、目的、端末内transfer、Keychain保持、保持期間、削除方法、第三者提供、問い合わせ先、変更日を明記しています。local pathをURLとして入力せず、公開先の表示と原稿revisionが一致することを確認します。

## 4. 審査用assetとtest instructions

準備するassetは次のとおりです。

- buildに含む128×128 PNG icon
- local同意画面を正確に示す1280×800または640×400 screenshot 1枚以上
- `/settings`上の本人確認statusを正確に示すscreenshot
- 440×280 small promo tile

screenshotへ実account名、Cookie、profile path、Keychain service ID、publisher情報を写しません。合成画面を実動screenshotと偽らず、UI変更後は撮り直します。

2026年8月26日に実際のdashboardで確認できた入力欄は、username（100文字）、password（100文字）、追加手順（500文字）だけでした。helper bundleの添付欄またはreviewer専用URL欄は確認できませんでした。3欄とも未入力・未保存です。

### 4.1. macOS側の配布要件の再調査結果

> 再調査日: 2026年8月26日

作業を一度停止した当初の理由は、「reviewer用helperがad-hoc署名であり、Gatekeeperに拒否されるため、Apple Developer Program（年額99 USD）への加入が必要になる」というものでした。同日、公式資料の確認とこのMac（macOS 26.5、Apple silicon）での実測により、**この理解が誤りであること**を確認しました。

**CWSへ拡張機能を配布することに、Apple側の要件はありません。** 拡張機能はHTML、CSS、JavaScriptのファイル群であり、Chromeが自身のプロファイルへ展開するため、macOSの実行ファイル保護の対象になりません。Apple Developer Programが必要になるのは、Developer ID署名とnotarizationを行う場合、すなわち**quarantine属性が付く経路でmacOS向け実行ファイルを配布する場合**だけです。[Developer ID](https://developer.apple.com/support/developer-id)、[notarizationの説明](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution)

同一のad-hoc署名実行ファイルについて、quarantine属性の有無だけを変えて実測した結果は次のとおりです。

| 条件 | `spctl --assess --type execute` | 実行結果 |
|---|---|---|
| ad-hoc署名・quarantine属性なし | 拒否 | **成功** |
| ad-hoc署名・quarantine属性あり（`0001`、`0081`、`0083`） | 拒否 | **`SIGKILL`で停止** |
| quarantine属性付きのシェルスクリプト | ― | 成功 |
| quarantine属性付きの`.py`を`python3`で実行 | ― | 成功 |
| `curl`で取得し`tar`で展開した実行ファイル | 拒否 | **成功**。付与される拡張属性は`com.apple.provenance`だけで、`com.apple.quarantine`は付かない |

さらに、この端末へ導入済みの`uv`、`go`、`node`はいずれもad-hoc署名で、`spctl --assess --type execute`に拒否されますが、通常どおり動作しています。

したがって、`spctl --assess --type execute`の拒否は「配布できない」という意味ではなく、パッケージマネージャ経由で導入したコマンドラインツール一般の通常状態です。実行を実際に阻害するのはGatekeeperそのものではなく、**ダウンロード時にブラウザ等が付与するquarantine属性**です。当初の停止理由は「reviewerへブラウザでダウンロードさせる」という前提の下でだけ成立していました。

この観測は2026年8月26日のmacOS 26.5・Apple siliconのものです。将来の固定保証とは扱わず、helperの配布方法を変える際に再確認します。

### 4.2. 採用するreviewer用helperの受渡し方式

費用を伴わない次の2経路を採用します。どちらもreviewerへGatekeeperの警告を無効化・上書きさせません。`xattr`によるquarantine属性の削除、右クリックからの実行、「このまま開く」の選択、`spctl`の無効化をいずれも求めません。

| 経路 | 内容 | 位置付け |
|---|---|---|
| 経路2 | 認証付き折返し通信の同じプロトコルを話す単一ファイルのreview用フィクスチャを、`python3`で実行する | **主経路**（2026年8月27日決定）。取得方法に依存しない。スクリプトにはGatekeeperが適用されない。要件はPython 3.9以上 |
| 経路1 | 公開URLから`curl`で取得し`tar`で展開し、事前build済みhelperを実行する | 任意。製品相当helperを実際に動かしたいreviewer向け。`curl`で取得する必要がある |

主経路を経路2としたのは、下の実測により**経路1がreviewerの取得方法に依存する**ためです。reviewerがブラウザで取得した場合、helperは説明のない`SIGKILL`で停止し、reviewerには壊れたツールとして見えます。経路2は1ファイル1コマンドで完結し、500文字のtest instructionsにも収まります。

#### 配布先（2026年8月27日決定）

主経路である経路2は、**新たな外部操作を必要としません。** フィクスチャは既にこの公開repositoryへコミット済みで、commit SHAで固定した`raw.githubusercontent.com`のURLからそのまま取得できます。同日、実際に取得してリポジトリ内のファイルとbyte一致し、`--self-test`が合格することを確認しました。

| 経路 | 配布先 | 新たな外部操作 |
|---|---|---|
| 経路2（主） | 公開repositoryのcommit SHA固定`raw`URL | **不要。コミット済みで到達確認済み** |
| 経路1（任意） | GitHubリリース | 必要。**reviewerから要望があった場合にだけ**、明示承認を得て作成する |

commit SHAで固定するため内容は不変で、reviewerはSHA-256を照合できます。経路1を先に用意しないのは、主経路が既に成立している以上、承認を要する外部操作を先に増やす理由がないためです。

公開する場合の注意を2点、記録します。

- **拡張機能の固定IDをリリースへ記載しません。** reviewerへはtest instructionsでだけ渡します。限定公開URLを知る者が不必要に増えないようにします
- **製品リリースと誤認させません。** `V-12`の検証支援物であって正式配布物ではないことを、タグ名と説明で明示します

2026年8月27日に、公開対象へ秘密情報が含まれないことを確認しました。helperとKeychain adapterの実行ファイルには、ホームpath、利用者名、repositoryの絶対pathがいずれも含まれていません（Goは`-trimpath`）。フィクスチャは既に公開済みのソースです。

経路2はhelperの代替であり、**拡張機能側のソースコードは一切変更しません**。reviewer専用の分岐、隠し機能、条件付きコードを拡張機能へ追加しません。フィクスチャは受け取ったCookieを検査した後に破棄し、秘密情報保管庫、ファイル、ログのいずれへも書かず、外部ホストへ接続しません。

#### 2026年8月27日の実測: `tar`はquarantine属性を伝播する

同じ`.tar.gz`を取得方法だけ変えて展開し、実行した結果です（macOS 26.5、Apple silicon）。

| 取得方法 | 展開後のquarantine属性 | helper（Mach-O実行ファイル） | フィクスチャ（スクリプト） |
|---|---|---|---|
| `curl` + `tar` | 付かない | 実行できる | 実行できる |
| ブラウザ等でダウンロード + `tar` | **付く。`tar`が展開先のファイルへ伝播する** | **`SIGKILL`で停止する** | **実行できる** |

**`tar`はアーカイブのquarantine属性を展開したファイルへ伝播します。** したがって経路1はreviewerの取得方法に依存し、reviewerがブラウザで取得した場合は説明のない`SIGKILL`になります。経路2だけが取得方法に依存しません。

この観測は2026年8月27日のmacOS 26.5・Apple siliconのものです。将来の固定保証とは扱いません。

「ブラウザ以外の経路で取得してもらうこと」をGatekeeper回避とは扱いません。ここでいう回避とは、表示された警告を利用者に無効化・上書きさせることを指します。経路1・経路2はいずれも警告を発生させず、`uv`やHomebrewと同じ通常の導入手順です。この解釈は2026年8月26日にownerが判断し、承認記録へ残します。

Developer ID署名とnotarizationは、経路1・経路2のどちらもreviewerが実行できず、かつCWS supportがDeveloper ID署名を必須と回答した場合にだけ再検討します。その場合も、契約、年額、税、自動更新、名義をowner自身が確認し、別途明示承認を得ます。

### 4.3. test instructionsの方針

test instructionsではAtCoder usernameとpasswordを空欄に保ち、共有AtCoder account、password、Cookieを作成・提供しません。追加手順は500文字以内で、次を明記します。

- §4.2の経路2でフィクスチャを取得して実行する手順と、照合するSHA-256
- 製品相当helperを動かしたい場合の経路1と、`curl`で取得する必要があること
- reviewer自身が利用を許可されたaccountを使うこと
- sign-in、Turnstile、submissionを自動化しないこと
- developer modeとGatekeeper回避を使わないこと

reviewerがfull flowを確認できない場合はcredential共有や回避経路を追加せず、明示承認後にCWS supportへ確認します。local fixtureだけで審査可能かは公式回答または審査feedbackに従い、推測で判定しません。

## 5. 外部操作の承認記録

外部操作前に、リポジトリ外の作業記録へ次を記入します。credential、実account名、支払情報は記録しません。

| field | 記録内容 |
|---|---|
| approval ID | campaign内alias |
| approved at | UTC時刻 |
| approver / executor | 匿名化可能なalias |
| operation | 登録、支払い、item作成、upload、review提出、Unlisted公開、update、unpublishの正確な組 |
| item | `extension-A`等のalias。固定ID取得後はcampaign manifestで対応付ける |
| version / ZIP hash | 対象版とSHA-256 |
| fee | dashboardに表示された金額・通貨・税と支払者の確認結果。成果物では機微でない集計へ射影する |
| distribution | `Unlisted`と、URLを知る者が追加できることへの同意 |
| listing / privacy revision | 承認した原稿hashとprivacy policy URL |
| stop owner / trigger | 誰が、いつ、何を根拠にunpublishするか |
| expires | 承認の有効期限。未記入なら当該操作一回で消費 |

画面表示の費用、permission、data disclosure、公開範囲、ZIP hashのいずれかが記録と違えば承認を消費せず停止します。upload後に固定IDを取得したらmanifestへ記録します。審査通過後、通常Chromeの標準追加で固定ID・対象版・配布元・権限を検出し、準備確認profileは破棄します。

## 6. 停止とcampaign終了

新規追加を止めるときは、承認済みのownerがCWS dashboardから対象itemをunpublishし、実行時刻と結果を記録します。CWS itemのunpublishと次を別々に扱います。

- Chromeへ既に追加済みのcopyの状態
- local Keychain項目の削除
- runtime profileと基準templateの削除
- AtCoder server側sessionの失効

local CookieまたはKeychain項目の削除を、CWS配布停止やAtCoder側失効の証拠にしません。campaign終了時は基準template、store用一時情報、検証用Keychain項目を削除し、限定公開itemを残すならowner、目的、visibility、停止方法、次回確認日を成果物へ残します。

## 7. 現在地

2026年8月26日時点で、次まで完了しています。

- 契約同意、一回限り5米ドルの支払い、developer登録
- 非取引業者の自己申告、Publisher nameの保存、contact emailの検証
- 対象版`0.1.0`（SHA-256 `b0a8d07812abd8661630689e57c8c241aaeb223312bafbbc58877a4fa4dbbe78`）による新規item作成とupload、固定IDのowner-only記録
- privacy policyとsupport pageの公開URL、実UIのscreenshot、small promo tileの準備と到達確認
- Store listing、個人を特定できる情報・認証情報・website contentを開示したPrivacy、料金なし・`Unlisted`・日本のみのDistribution設定の保存

reviewer用helperの受渡し方式は[§4.2](#42-採用するreviewer用helperの受渡し方式)で確定しました。次は未実施です。

- 経路1の`.tar.gz`生成とSHA-256のbuild indexへの記録
- 経路2のreview用フィクスチャの作成と固定入力testへの追加
- サポートページへのhelper取得手順の追記と公開URLでの到達確認
- test instructionsの確定・保存
- pre-submission testと最終入力の確認
- deferred publishingによる審査提出と審査通過
- 審査とは別の明示承認に基づく`Unlisted`公開
- CWS配信済み`0.1.0` bytesの取得、hash固定、最終campaign manifestへの反映
- 標準Chromeの標準追加画面でdeveloper modeなしに追加できることの確認
- `TD-11`の`V-12C`で使う`0.1.1`のupload・審査

この分離により、ローカル成果物を完了条件とする`TD-37`は完了です。上記の作業とcampaign manifestの`signed_builds`が未完了であるため、`TD-39`は未着手、`TD-11`と`V-12`は未完了です。

## 8. 判断記録と次の作業

### 8.1. 2026年8月26日の判断

作業はtest instructionsの入力前で一度停止しました。CWS itemはdraftで、「審査のため送信」は押しておらず、公開もしていません。この状態は変わっていません。

停止時に記録した4つの問題のうち、2番目と3番目は誤った前提に基づいていたため、次のとおり訂正します。

| # | 停止時の記録 | 2026年8月26日の判断 |
|---|---|---|
| 1 | test instructions画面にhelperの添付欄またはreviewer専用URL欄がない | 事実として維持する。500文字の追加手順欄へ、公開URLと照合用SHA-256を書いて渡す |
| 2 | helperがad-hoc署名のためGatekeeperの通常評価で拒否され、渡すとGatekeeper回避を要求しかねない | **訂正する。** 実行を阻害するのはquarantine属性であり、`spctl`の拒否そのものではない（[§4.1](#41-macos側の配布要件の再調査結果)）。quarantine属性が付かない経路で渡せば、回避を求めずに再現できる |
| 3 | Developer ID署名とnotarizationのidentityがなく、Apple Developer Programの契約・年額費用・自動更新の判断が伴う | **訂正する。** CWSへの拡張機能配布にApple側の要件はない。[§4.2](#42-採用するreviewer用helperの受渡し方式)の方式では署名もnotarizationも必要としないため、加入・支払いの判断は発生しない |
| 4 | 共有AtCoder credentialを提供しない方針のため、審査可能範囲のCWS公式確認が必要 | 事実として維持する。ただし問い合わせは先行させず、reviewer自身のaccountで審査を試みて、拒否された場合にだけ行う |

これにより、`TD-39`の停止は解除されます。**Apple Developer Programへの加入と年額99 USDの支払いは行いません。** CWS supportへの問い合わせも、[§8.2](#82-次に行う作業)の段階2で審査が成立しなかった場合の後段の手段へ移します。

なお、Developer ID署名を付けても、この設計の認証・認可の強度はほとんど上がりません。方式Aの安全性を支えているのは、拡張機能の固定ID・版・配布元・権限の照合、認証付き折返し通信の一回限りトークン・`Host`・送信元・origin・本文上限・状態順序の検査、および本人照合後にだけ保存する契約です。helperの真正性は、利用者自身が導入したAlgoLoomが起動した子プロセスであることで担保します。詳細は[AtCoder認証設計 §3.6](../../architecture/atcoder-authentication.md#36-配布物と実行ファイル署名の境界)を参照します。

### 8.2. 次に行う作業

作業は3段階に分かれます。段階1は外部操作を含まないため、追加の承認なしで着手できます。段階2以降は操作ごとに明示承認が必要です。

#### 段階1: 受渡し方式の実装（外部操作なし・承認不要）

| # | 作業 | 完了の目安 |
|---|---|---|
| 1 | 本書と[TODO.md](../../../TODO.md)の`TD-39`の停止理由を§4.1の再調査結果へ合わせて訂正する | 「Apple Developer Programが必要」という記述が残っていない（2026年8月26日完了） |
| 2 | `prepare.mjs`へ、helperとKeychain adapterをまとめた`.tar.gz`の生成と、そのSHA-256・bytesの`build-index.json`への記録を追加する | 隔離buildから経路1の配布物を再現でき、ハッシュが固定されている |
| 3 | 経路2のreview用フィクスチャを1ファイルで作る。プロトコル、一回限りトークン、`Host`検査、送信元検査、拡張機能origin検査、本文上限、状態順序をhelperと同じ条件で満たす | 拡張機能のソースを変更せずに同意画面から本人照合まで到達できる |
| 4 | 固定入力testへ、quarantine属性が付いた状態でも経路2が成立することを確認するケースを追加する | `go test ./...`と`node --test`が合格する |
| 5 | helperとフィクスチャの取得手順、SHA-256、停止方法を[サポートページ](v12-extension-support.md)へ追記し、公開URLで到達確認する | reviewerが500文字のtest instructionsからたどれる。**2026年8月27日完了。** 主経路の配布先は公開repositoryのcommit SHA固定`raw`URLで、到達確認と`--self-test`合格を確認済み |
| 6 | 経路1のGitHubリリースを先に作らない | reviewerから実際のhelperを動かしたいという要望があった場合にだけ、明示承認を得て作成する。**2026年8月27日決定** |

`TD-11`は未着手のため、この段階でソースリビジョンが変わっても無効化される`V-12`のsub結果はありません。**訂正と追加を行うなら現時点が最も影響が小さくなります。**

#### 段階2: 審査提出（操作ごとに明示承認が必要）

6. [§4.3](#43-test-instructionsの方針)に従いtest instructionsの文面を確定し、入力内容とhelper受渡し先を提示して**明示承認**を得てから保存する。
7. dashboardのpre-submission test、要求権限、data disclosure、listing、Privacy、Distribution、version、ZIPハッシュを再確認する。不合格、差分、helper再現不能があれば審査へ送信しない。
8. 対象item、version、ZIPハッシュ、料金なし、`Unlisted`・日本のみ、test instructions、審査通過だけでは自動公開しないことを提示して**明示承認**を得る。承認後、deferred publishingを選んで審査へ送信する。
9. 審査結果を記録する。不承認なら理由を記録して停止し、credential共有、Gatekeeper回避、別配布元、手動読込で迂回しない。helperを再現できないことが理由の場合にだけ、文面と送信先を提示して**明示承認**を得たうえでCWS supportへ問い合わせる。

#### 段階3: 限定公開と`TD-11`への引き渡し（別の明示承認が必要）

10. 対象item、version、限定公開URL、停止方法を再提示し、公開の**明示承認**を別に得る。公開後、通常Chromeの標準追加画面でdeveloper modeなしに追加できる事前状態だけを確認し、`TD-11`の基準templateはまだ作らない。
11. CWS配信済み`0.1.0`の正確なbytesを取得できた場合だけハッシュを固定し、固定ID、listing URL、helper、protocol、source、build、Chrome・OS、template schema、同意版とともに最終campaign manifestへ記録する。取得できない場合は`TD-11`へ進まない。
12. `0.1.1`は`TD-11`の`V-12C`で明示承認を得るまでuploadしない。`0.1.0`の初回標準追加を確認する前にcurrent versionを置き換えない。

**この後に人が判断する必要があるのは、段階2の#6・#8・#9と段階3の#10だけです。** 段階1に外部影響はありません。
