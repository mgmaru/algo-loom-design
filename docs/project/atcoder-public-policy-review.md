# AtCoder公開情報に基づく配布判断記録

> 対象: `TD-06`〜`TD-08`で問い合わせを予定していた12項目
>
> 調査基準日: 2026年8月24日
>
> 状態: MVPと限定公開ベータに適用する条件付き決定。AtCoderからの許諾・公認を意味しない
>
> 関連文書:
> - [配布方針ガイド](../operations/algoloom-distribution.md)
> - [AtCoder認証設計](../architecture/atcoder-authentication.md)
> - [AIレビュー安全設計](../features/ai-review-safety-design.md)
> - [外部学習資料参照設計](../features/external-learning-resources.md)
> - [未決事項一覧](unresolved-decisions.md)
> - [`JudgeAdapter`技術検証計画](judge-adapter-verification.md)

## ドキュメント概要

本書は、AtCoderへ問い合わせる予定だった12項目を、2026年8月24日時点の公式公開情報、技術検証で得た事実、AlgoLoom自身が採る保守的な製品判断に分けて再整理した判断記録です。公開情報に明記されていないことを「許可」と解釈せず、MVPを止めずに進めるための条件と停止基準を定義します。

本書は法的助言ではありません。利用規約、個別コンテストルール、Bot対策およびページ構造は変更され得ます。実装開始、限定公開ベータ、各リリースのgateで公式情報を再確認します。

## 0. 結論

12項目について、通常のツールサポート問い合わせは送付せず、公開情報に基づく条件付き判断で先へ進みます。

理由は、AtCoderが2025年4月17日の公式告知で、非公式ツールはサポート対象外であり、問い合わせにも回答しないと明記しているためです。この告知はAlgoLoomの利用許可ではありません。問い合わせ待ちを配布gateにせず、次の3層でリスクを制御します。

1. 公式情報に明記された規則を守る。
2. 明記のない範囲は、1問・1操作・人による最終操作・有限通信・保存最小化というAlgoLoom独自の制約で狭める。
3. 規則、互換性または安全条件を確認できない場合は、外部通信または該当機能だけをfail closedで停止する。

外部へ連絡するのは、AtCoderが新しい公式窓口・API・提携制度を設けた場合、AtCoderから個別に連絡や停止要請を受けた場合、ロゴ利用ガイドに基づく連絡が必要な場合、または商用・法人用途を別途検討する場合に限ります。非公式ツールの通常サポート問い合わせは行いません。

## 1. 事実とプロジェクト判断の区別

| 種別 | 意味 | 本書での扱い |
|---|---|---|
| 公式事実 | AtCoderの規約、告知、ルール、公式ページに記載された内容 | 出典と確認日を記録し、その記載を超えて一般化しない |
| 技術検証事実 | AlgoLoomの限定条件で観測した通信数、画面、Cookie、提出・判定導線等 | 検証時点の成立性と負荷設計の根拠。AtCoderの許諾とは扱わない |
| プロジェクト判断 | 公式情報と検証事実を踏まえ、AlgoLoomが自ら課す製品制約 | 正本文書と仕様へ反映し、再確認gateで見直す |
| 未決・将来判断 | 現在のMVPに含まれず、利用形態の変更時に別途判断する事項 | 現在の公開を止める問い合わせ待ちにしない |

「利用規約に名指しの禁止が見当たらない」「他の非公式ツールが存在する」「技術的に動作した」のいずれも、AlgoLoomへの許諾を意味しません。

## 2. 12項目の再整理

| # | 対象 | 公式情報から確認できること | AlgoLoomの判断 | 必須制約と再確認条件 |
|---:|---|---|---|---|
| 1 | 公開sampleを1問ずつ保存するCLI | コンテンツの権利はAtCoderまたは権利者に帰属する。非公式ツールはサポート対象外 | **条件付きで進める** | 利用者が選んだ終了済み過去問1件の公開sampleだけを実行時取得する。問題文・画像・解説を配布せず、一括取得しない。規約または取得経路が変われば停止して再確認 |
| 2 | browser sessionをCLIの提出補助へ使う方式 | CAPTCHAがsubmit等へ導入され、非公式ツールによる提出が難しくなることが告知されている。非公式ツールへのサポートはない | **非サポート経路として条件付きで進める** | 空の可視専用browserで人がlogin・Turnstile・最後の提出操作を行う。`REVEL_SESSION`だけをOS secret storeへ保存し本人照合する。既存profile参照、Bot対策回避、無人submit、直接HTTP POSTへのfallbackを禁止。成立しなければ提出だけ停止 |
| 3 | アクセス間隔と再試行 | 一般的な正確な推奨間隔は公式公開情報から確認できない。serverの待機指示やアクセス拒否は尊重する必要がある | **プロジェクト側で保守値を定める** | 初期下限は同一originへの開始間隔2秒。これはAtCoder公式値ではない。`Retry-After`等のserver指示を優先し、429・403・challenge・通信不明では自動再試行しない。有限polling、一括取得・常時監視なし。実測変更は工程7で決定 |
| 4 | User-Agent | RFC 9110は、必要な製品識別子だけを送るよう求め、詳細すぎる識別を避ける考え方を示す。AtCoder固有の指定は確認できない | **最小識別子を固定する** | 直接HTTP通信は`AlgoLoom/<version> (+https://github.com/mgmaru/algo-loom-design)`形式を初期値とする。account、端末、OS、メールアドレス等を含めない。可視browserのUser-Agentは変更しない |
| 5 | 公開sampleのlocal cache | #1と同じ権利・非サポート境界 | **条件付きで進める** | 利用者が取得した問題の作業用cacheに限定し、配布・共有・Cloud同期・公開exportへ含めない。再取得を減らすために使い、明示削除手段を設ける |
| 6 | AIレビューの開催中判定 | 現行のABC・ARC・AGC向け生成AIルールは開催中に適用され、終了済み過去問の練習には適用されない。rule-basedなsample file作成toolは明示的に許容される | **rule-version profileで進める** | AI reviewはMVP対象外。実装時はcontest種別、開催状態、正規問題ID、ルール版を確認し、判定不能なら拒否。AIを使わないsample取得・testまで停止しない。各releaseでルールを再確認 |
| 7 | ADTで再利用中の過去問のAIレビュー | ADTはABC過去問による練習virtual contestで、開催中の解法・感想投稿や配信が認められる一方、過去の自分の提出からのcopy & pasteは禁止される。生成AIの直接許可は記載されていない | **注意表示付き許可をプロジェクト判断とする** | AtCoderの許諾とは表示しない。AIなしで練習する`contest_mode`を案内する。ADTルールまたは一般AIルールが変わる、種別を確認できない場合は拒否へ戻す |
| 8 | 「AtCoder対応」という文章表記 | ロゴガイドは、条件に従うロゴ利用自体は許諾不要とする一方、media・serviceでの利用時は連絡を求め、公式と誤認させる利用を禁じる | **文字による事実説明だけを使う** | 「AtCoderの終了済み過去問に対応する非公式CLI」と非公式・非提携・非公認表示を併記する。製品名にAtCoderを含めず、ロゴと公式siteの意匠を使わない。将来ロゴを使う場合だけガイドに従って連絡 |
| 9 | 無料OSS版と有料版 | 企業・団体による過去問の採用試験・coding試験・採用関連利用には公式の制限がある。無料OSSと有料版の一般的な区分は確認できない | **MVPは無料OSSに限定し、有料版は別判断** | 採用・査定・採用活動への利用をサポートしない。課金、法人研修、managed service等は、事業形態を具体化して法務・事業面を別途判断する。将来判断を通常のツールサポート問い合わせに置き換えない |
| 10 | 問題別解説ページをbrowserで開く | 問題別の公式URLが提供されている。コンテンツの権利境界は引き続き適用される | **MVPで進める** | 正規IDから公式URLを構成し、default browserへ1 URLだけ渡す。本文、HTML、画像、PDF、sample codeをAlgoLoomへ取得・保存・再表示しない。終了確認不能なら開かない |
| 11 | filter付き提出一覧をbrowserで開く | 公式学習資料は、終了後の学習で他者の提出を見ることを有用な方法として案内する。`robots.txt`はsubmission系のcrawler accessを制限している | **MVP後にbrowser-onlyで進められる** | code本文、HTML、author、submission ID、CookieをAlgoLoomへ取得・保存しない。browserの認証とUIへ委譲し、crawlしない。終了状態とspoilerを確認し、参加中または不明なら開かない |
| 12 | 可視専用browserから`REVEL_SESSION`だけを取り込む方式 | #2と同じ。公式APIまたはサポートされた連携方式は確認できない | **方式Aを条件付き候補として維持する** | 利用者が明示開始し、人がlogin・Turnstileを操作する。Cookie名・hostを固定し、本人accountを照合後だけsecret storeへ保存する。既存profile、他Cookie、password、CDP/WebDriverによる保護page制御を禁止。ブラウザ互換性が崩れたら回避せず停止 |

## 3. 分野別の適用条件

### 3.1. コンテンツ取得とcache（#1、#5）

- 対象は利用者が明示的に選択した終了済み過去問1件に限定します。
- 取得するのはローカルテストに必要な公開sampleの入力・期待出力だけです。
- 問題文、画像、解説、動画、PDF、他ユーザーのcode、非公開testを取得・保存・配布しません。
- 配布物、demo、test fixture、screenshot、Cloud同期、公開exportへAtCoder由来のsampleを含めません。
- cacheは再取得を減らす作業用データであり、AlgoLoomのコンテンツカタログにはしません。
- 問題の一括取得、account全体の履歴収集、background crawl、先読みを実装しません。

### 3.2. 認証と提出補助（#2、#12）

方式Aは、AtCoderがサポートまたは公認する連携方式ではありません。次の条件をすべて満たす間だけ、AlgoLoomの限定機能として扱います。

1. 空の可視専用browserを使い、利用者の既存profileを探索・複製しない。
2. 利用者本人がlogin情報を入力し、Turnstileと最後の提出buttonを操作する。
3. AlgoLoomは`https://atcoder.jp`の`REVEL_SESSION`だけを取り込み、同じsessionで本人accountを一意に照合する。
4. sessionはOSのsecret storeだけへ保存し、DB、workspace、log、telemetry、Cloud、exportへ含めない。
5. submit先、問題、言語、source hashを表示し、提出操作の明示確認を得る。
6. 提出button押下後に受理を確認できない場合は`REMOTE_STATUS_UNKNOWN`として自動再提出しない。
7. CAPTCHA・Turnstileの自動操作、headless化、stealth、指紋偽装、CDP/WebDriverによる保護page制御を行わない。
8. 可視browser経路が成立しない場合、sessionを使った直接HTTP POSTや旧画面の推測をfallbackにせず、提出だけを停止する。

HTTP Adapterは問題取得、account確認、受理済み提出の判定確認等の許可された限定責任に使えますが、人の最終提出操作を置き換えません。

### 3.3. アクセスと識別（#3、#4）

- 2秒はAlgoLoomが置く保守的な初期下限であり、AtCoderが推奨または許可した値ではありません。
- serverがより長い待機を示した場合はserver指示を優先します。
- 429、403、challenge、構造変更、送信状態不明では自動再試行せず、該当操作を停止します。
- 5xx、timeout、TLS、名前解決等も同じ操作内で自動再試行しません。利用者が状態を確認した後の新しい明示操作としてのみ再開します。
- 判定確認は提出ID 1件に対する有限pollingとし、最大待機・最大回数を持たせます。
- 起動時の通信、常時監視、session維持通信、問題や提出の一括取得を行いません。
- `contest.js`の内容hash確認はrelease時のmaintainer確認を主とします。実行時は提出に必要な既存pageで必須条件を確認し、hash確認だけを目的とした追加GETを各提出に課しません。
- 直接HTTP Adapterだけに製品識別User-Agentを設定し、利用者が操作するbrowserのUser-Agentを変更しません。

### 3.4. AIレビュー（#6、#7）

2025年10月3日版の生成AI対策ルールは、開催中のABC・ARC・AGCへ適用され、過去問練習には適用されないと明記しています。一方、2026年8月18日に公開された短期AHC向けルールは2026年8月29日から適用され、開催中の短期AHCでは生成AI利用を原則禁止します。対話的なreview、debug、testも禁止対象になるため、従来の「AHCでは単発reviewなら将来解禁し得る」という条件だけでは不十分です。

- AI reviewはMVPへ含めません。
- 実装する場合は`contest kind + rule version + effective period`を一体のrule profileとして管理します。
- 開催中ABC・ARC・AGCと、2026年8月29日以降の開催中短期AHCの同一問題は拒否します。
- 長期AHC、AWC、企業contest、その他の個別ruleは、対応profileと有効な公式根拠を確認できる場合だけ判定し、不明なら拒否します。
- 終了済み過去問の練習は、開催中の別contestの対象問題と一致しないことを確認できればreview候補にできます。
- ADTの`ALLOW_WITH_NOTICE`は公開情報を組み合わせたAlgoLoomの判断であり、AtCoderによる許可とは表示しません。
- `contest_mode = true`では問題照合前に全AI機能を止めます。

### 3.5. 表示と事業範囲（#8、#9）

公開面では、次の趣旨をREADME、配布page、top-level help、version表示に掲載します。

> AlgoLoomは、AtCoderの終了済み過去問に対応する非公式の第三者製CLIです。AtCoder株式会社とは提携しておらず、保証・公認を受けていません。

AtCoderロゴを使いません。将来ロゴを使う場合は、ロゴガイドが求める条件と連絡をその時点で行います。

MVPと初期公開は無料OSSです。有料版、法人研修、managed service、採用関連機能はMVPの判断に含めません。特に、AtCoder過去問を採用試験、coding試験、査定または採用活動へ利用する機能は提供・推奨しません。

### 3.6. 外部学習資料（#10、#11）

- 問題・解説・提出一覧は公式URLをdefault browserで開きます。
- AlgoLoomは内容を取得、cache、preview、要約、検索、diff、copyまたはexportしません。
- 問題別解説はMVPに含め、提出一覧はMVP後の候補とします。
- 提出一覧はbrowser-onlyとし、`robots.txt`で制限されたsubmission領域をHTTP Adapterでcrawlしません。
- `robots.txt`はcrawlerへの指示であり、アクセス許諾を与える文書ではありません。AlgoLoomはbrowser委譲の許可根拠として使わず、crawlしない設計の補助根拠としてだけ参照します。
- 未AC、contest開催中、virtual contest参加中、終了確認不能ではspoilerを安全側に扱います。

## 4. 技術検証事実の位置付け

[`JudgeAdapter`技術検証](judge-adapter-verification.md)と[匿名化済み実行記録](../verification/judge-adapter/results/)は、1操作の通信数、間隔、保存項目、認証・提出・判定の成立条件を確認しました。これらは次の用途に限定します。

- AlgoLoomの通信上限、保存許可list、停止条件を設計する根拠
- 配布前に想定外のbackground通信や秘密情報保存がないことを照合する基準
- AtCoder側の変更を無期限の互換性保証と誤認しないための日付付き観測

実サービスで成立したこと、57 GET・2 POSTという集計、`contest.js`のhash等は、AtCoderの推奨値、許諾または将来の互換性保証ではありません。生のCookie、header、HTML、source、account名、提出IDは判断記録へ残しません。

## 5. 再確認gate

### 5.1. gateの時点

| 時点 | 必須確認 | 不成立時の扱い |
|---|---|---|
| 実装開始前 | 対象機能の公式source、方式Aのbrowser境界、直接HTTPの送信先・method・上限、仕様への投影 | 該当機能の実装を開始しない |
| 限定公開ベータ前 | 利用規約、非公式ツール告知、AI rule、ADT、logo guide、`robots.txt`、認証・提出互換性、3 OSのsecret store | local機能を残して、取得・提出・AI・外部参照の該当機能だけを停止 |
| 各release前 | 前回確認日以降の公式変更、`contest.js`と必須条件、User-Agent、通信実測、配布物内容 | 未確認の変更を自動受入れせずreleaseを止める |
| 公式情報の変更検知時 | 変更されたruleの適用日、対象contest、既存設計への影響 | 対応profileを更新するまでfail closed |

### 5.2. 毎回確認する公式source

- AtCoder利用規約
- 非公式ツールとCAPTCHAに関する公式告知
- ABC・ARC・AGC向け生成AI対策ルール
- 短期AHC向け生成AI利用ルール、および対象となる他のAHC rule
- ADTと対象contestの個別rule
- AtCoderロゴ利用ガイドライン
- 企業・団体におけるAtCoder利用条件
- AtCoder `robots.txt`
- 提出form、Turnstile、account照合、提出ID、判定状態の必須条件

確認日、確認者、source URL、適用版、差分、判断、停止または再開した機能をrelease記録へ残します。

## 6. 例外的に外部連絡を検討する条件

- AtCoderが非公式tool向けのAPI、登録制度、partner programまたは専用窓口を公開した。
- AtCoderからAlgoLoomへ個別の連絡、条件提示、修正または停止要請があった。
- AtCoderロゴをmedia・serviceへ使用し、公式ガイドが求める連絡を行う。
- 有料、法人研修、managed service等の事業形態を具体化し、通常のtool supportではない事業・法務上の確認が必要になった。
- security incident等で、利用者保護のためAtCoderとの調整が必要になった。

この条件に該当しない限り、問い合わせへの回答を実装・限定公開ベータ・OSS公開の前提にしません。

## 7. 公式・一次資料

| 資料 | 本判断で確認した要点 |
|---|---|
| [AtCoder利用規約](https://atcoder.jp/tos?lang=ja) | コンテンツ権利、account共有禁止、包括的な禁止事項。2026年6月29日改定版を2026年8月24日に確認 |
| [非公式ツールとCAPTCHAに関する告知](https://atcoder.jp/posts/1456?lang=ja) | submit・custom testへのCAPTCHA導入、非公式toolはサポート対象外、問い合わせへ回答しない旨 |
| [AtCoder生成AI対策ルール](https://info.atcoder.jp/entry/llm-rules-ja) | 2025年10月3日版。開催中ABC・ARC・AGCへの適用、過去問練習の除外、rule-based sample file作成toolの扱い |
| [AHC生成AI利用ルール](https://info.atcoder.jp/entry/ahc-llm-rules-ja) | 2025年6月16日版。多数候補の生成・評価等に関するAHC固有の制限。短期AHCの新ruleへ一般化しない |
| [短期AHCにおける生成AI利用ルール](https://info.atcoder.jp/entry/short-ahc-llm-rules-ja) | 2026年8月18日公開、2026年8月29日から適用。開催中短期AHCの生成AI利用を原則禁止 |
| [AtCoder Daily Training](https://atcoder.jp/contests/adt_top?lang=ja) | ABC過去問による練習、開催中の投稿・配信、過去提出からのcopy & paste禁止 |
| [AtCoderロゴ利用ガイドライン](https://info.atcoder.jp/logoguide) | 公式誤認の禁止、media・serviceでロゴを使う場合の連絡 |
| [企業・団体におけるAtCoderの利用](https://info.atcoder.jp/utilize/school/riyou) | 過去問の採用試験・coding試験・採用関連利用に関する制限 |
| [AtCoder Problemsの説明](https://info.atcoder.jp/more/contents/problems) | AtCoder Problemsが有志による別serviceで、AtCoderの問い合わせ対象外であること |
| [APG4b: 他の人の提出を見る](https://atcoder.jp/contests/apg4b/tasks/APG4b_al?lang=ja) | 他者の提出を学習へ活用する公式案内 |
| [問題別解説ページの例](https://atcoder.jp/contests/abc416/tasks/abc416_a/editorial) | 正規contest・problem IDから到達できる公式解説page |
| [AtCoder `robots.txt`](https://atcoder.jp/robots.txt) | submission、standings等のcrawler制限。2026年8月24日に確認 |
| [RFC 9110: HTTP Semantics](https://www.rfc-editor.org/info/rfc9110/) | User-Agentに必要なproduct identifierだけを送る原則 |
| [RFC 9309: Robots Exclusion Protocol](https://www.rfc-editor.org/info/rfc9309/) | `robots.txt`はcrawler向け規約で、access authorizationではないこと |

## 8. 変更対象への投影

本判断の規範的な要点は、配布、認証、AI review、外部資料、`JudgeAdapter`互換性、ライブラリ選定、開発用仕様へ反映します。検証当時の事実を保存する`docs/verification/`の実行記録は、履歴として改変しません。
