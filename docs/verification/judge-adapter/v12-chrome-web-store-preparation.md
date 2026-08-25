# V-12 Chrome Web Store配布準備

> 確認日: 2026年8月26日
>
> 状態: `TD-37`のローカル準備は完了し、外部操作を扱う`TD-39`は作業停止中。CWSではdeveloper登録、新規itemへの`0.1.0` upload、Store listing・Privacy・Distributionの保存まで完了した。Distribution設定は料金なし・`Unlisted`・日本のみで、item自体はdraftのままである。test instructionsは未入力・未保存で、審査提出、公開、CWS署名済みbuild取得、`0.1.1` uploadは未実施。account名、実address、developer dashboard URLは記録していない
>
> 対象: `TD-39`のCWS審査用helperと限定公開版。製品の正式公開手順ではない

## 0. 結論

`V-12`ではChrome Web Store（CWS）の`Unlisted`を使います。ここでいう限定公開は「listing URLを知る利用者が追加できる」範囲であり、指定accountだけに制限する`Private`ではありません。`Unlisted`もCWS policyと審査の対象です。[公式のvisibility説明](https://developer.chrome.com/docs/webstore/cws-dashboard-distribution)

ローカルbuildと事前testを扱う`TD-37`は完了しました。CWS審査と限定公開を扱う`TD-39`は、reviewerへhelperを安全に渡す方法が確定していないため停止しています。次の外部操作は、対象、費用、公開範囲を読める形で人間が明示承認するまで行いません。

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

reviewer用macOS arm64 helper候補はad-hoc署名です。`codesign --verify --strict`には合格しますが、`spctl --assess --type execute`は拒否し、このMacには有効なcode-signing identityがありません。ownerはApple Developer Programへ加入していません。このままではreviewerに通常のmacOS保護を回避させずhelperを再現してもらえるとは確認できないため、test instructionsは確定・保存しません。

Test instructionsではAtCoder usernameとpasswordを空欄に保ち、共有AtCoder account、password、Cookieを作成・提供しません。追加手順は、helperの安全な受渡し方法が確定した後に500文字以内で作成し、reviewer自身が利用を許可されたaccountを使うこと、sign-in・Turnstile・submissionを自動化しないこと、developer modeとGatekeeper回避を使わないことを明記します。reviewerがfull flowを確認できない場合はcredential共有や回避経路を追加せず、明示承認後にCWS supportへ確認します。local fixtureだけで審査可能かは公式回答または審査feedbackに従い、推測で判定しません。

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

次は未実施です。

- reviewer用helper受渡し方法の確定
- test instructionsの確定・保存
- pre-submission testと最終入力の確認
- deferred publishingによる審査提出と審査通過
- 審査とは別の明示承認に基づく`Unlisted`公開
- CWS配信済み`0.1.0` bytesの取得、hash固定、最終campaign manifestへの反映
- 標準Chromeの標準追加画面でdeveloper modeなしに追加できることの確認
- `TD-11`の`V-12C`で使う`0.1.1`のupload・審査

この分離により、ローカル成果物を完了条件とする`TD-37`は完了です。上記の外部作業とcampaign manifestの`signed_builds`が未完了であるため、`TD-39`は保留、`T-11`と`V-12`は未完了です。

## 8. 作業停止点と再開手順

作業はtest instructionsの入力前で停止しました。CWS itemはdraftで、「審査のため送信」は押しておらず、公開もしていません。

### 8.1. 停止している問題

次の問題が連鎖しているため、test instructionsを確定できず、審査へ提出できません。

1. 拡張機能のcore flowにはlocal helperが必要ですが、実際のCWS test instructions画面にはhelperの添付欄またはreviewer専用URL欄がなく、公式な受渡し方法が分かりません。
2. 現在のhelperはad-hoc署名で、`codesign --verify --strict`には合格する一方、Gatekeeperの通常評価である`spctl --assess --type execute`には拒否されます。この状態で配布するとreviewerへGatekeeper回避を要求する可能性があり、安全な審査手順にできません。
3. このMacにはDeveloper ID署名とnotarizationを行えるidentityがありません。その取得候補であるApple Developer Programには、新しい契約、年額費用、自動更新の判断が伴います。
4. AtCoderの共有credentialは安全上提供しない方針です。reviewer自身のaccountまたはlocal fixtureでどこまで認証機能を審査できるか、CWSの公式確認が必要です。

このため、helperの公式な受渡し方法と認証機能の審査方法が判明し、安全な署名済みhelperを用意できるか別方式を選ぶまでは、test instructions保存、審査提出、公開へ進みません。

### 8.2. 再開手順

再開時の推奨する最初の操作は、CWS supportへ次の3点を問い合わせることです。ただし、問い合わせ送信自体が外部操作であるため、文面と送信先を提示して明示承認を得てから行います。

1. dashboardに添付欄がない場合、companion helperをreviewerへ安全に渡す公式経路は何か。
2. 共有AtCoder credentialを提供せず、reviewer自身のaccountまたはlocal fixtureで審査できるか。
3. macOS helperにはDeveloper ID署名とnotarizationが必要か、CWSに別の公式要件があるか。

回答によりDeveloper ID署名とnotarizationが必要だと確認できた場合は、Apple Developer Programの現在の契約、年額、税、自動更新、加入名義をowner自身が確認し、加入・支払いを別途明示承認してから進みます。費用を使わない場合は、reviewerや利用者にGatekeeper回避、実行時compile、developer mode、CDP、WebDriverまたは共有credentialを要求しない設計へ戻します。

helper配布方法の確定後は、`TD-39`の手順どおり、test instructions案の承認と保存、pre-submission確認、deferred publishingを指定した審査提出の承認、審査対応、別承認による限定公開、CWS署名済みbytesと最終campaign manifestの固定を順に行います。`0.1.1`は`TD-11`の更新testまでuploadしません。
