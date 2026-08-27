# AlgoLoom 未決事項一覧

> 作成日: 2026年7月18日
>
> 状態確認日: 2026年8月24日
>
> 目的: リポジトリ内に分散している未決事項を、各正本文書の現在の判断と再評価条件を変えずに確認できるよう集約する。
>
> 範囲: 「未確定」「別途決定」「機能設計で決める」「採用判断」「公式情報の再確認」が明記されている事項、および将来の追加時に再評価が必要と明記されている事項を対象にする。実装順序だけが書かれ、判断が保留されていない項目は含めない。

## ドキュメント概要

本書は、リポジトリ内に分散する未決事項を分野別に集約し、現在の状態、決定済みの範囲、残る判断、出典を追跡する方法を定義します。

## 読み方

- 各項目の「原文」は出典からの抜粋であり、表記・条件・留保を変更していない。
- 「決めること」は原文から分かる確認対象を短く示すための案内であり、決定そのものではない。
- 「決定済みの内容」は、別の関連文書ですでに確定している範囲を示す。元の未決事項全体が解消していない場合は、残る未決範囲も併記する。
- MVPの範囲に関する正本は、引き続き [MVPスコープ](../product/mvp.md) である。この一覧は正本を置き換えない。

### 状態の定義

| 状態 | 意味 |
|---|---|
| 決定済み | 現在の対象範囲では必要な判断が完了している |
| 一部決定済み | 境界または一部仕様は決定しているが、具体値・名称・実装方法等が残っている |
| 条件付き決定 | 現在の方針と再評価条件が決定しており、条件成立後の最終採否だけが残っている |
| 未決 | 採否または具体仕様を決める根拠・設計がまだ揃っていない |

### 状態確認の結果

| 状態 | 項目 |
|---|---|
| 決定済み | 1.3、2.4、2.6、4.4、5.2 |
| 一部決定済み | 1.1、1.2、1.4、1.5、1.6、1.7、1.8、1.9、1.10、2.2、2.3、2.5、3.2、3.4、3.5、4.2、5.3 |
| 条件付き決定 | 1.11、3.1、3.6、3.7、4.1、4.3、5.1、6.2、6.3、7.1、7.2、8.1、8.2 |
| 未決 | 2.1、3.3、6.1、7.3、7.4 |

## 用語

| 用語 | この文書での意味 |
|---|---|
| MVP | 最小限の価値を検証するために、初期版で実装・保証する範囲 |
| Core | 日常利用の基本導線と、その不変条件を担う中核機能 |
| workspace | 問題、解答source、test、metadataを扱う作業単位 |
| context | commandが対象とするworkspace、問題、sourceの組み合わせ |
| Adapter | Coreと外部サービスや交換可能な実装を隔離する境界 |
| BYOC | 利用者自身が所有・管理するCloud接続先を使う方式 |
| fail closed | 安全性を確認できない場合に、許可せず停止する方針 |
| outbox | ローカルで確定した変更を、後から外部へ同期するために保持する記録 |
| Cloud-primary | Cloud側を主要なデータ保存・参照先とする構成 |
| last-push-wins | 後からpushされた状態で同期先を上書きする競合処理 |
| test oracle | 実行結果が正しいかを判定する基準 |
| sandbox | code実行がfilesystem、network、process等へ及ぼす影響を隔離・制限する仕組み |
| threat model | 保護対象、想定する攻撃者、攻撃経路、許容する残余riskを整理したもの |
| HTTP | ウェブ上で要求と応答を交換する通信規約。検証記録では、AtCoderへの通信数を集計する単位として扱う |
| GET | HTTPで情報を取得するための要求方式 |
| POST | HTTPで情報を送信するための要求方式 |
| Cookie | ブラウザーやクライアントがセッション等の状態を保持するための小さなデータ。AtCoder認証では`REVEL_SESSION`だけを対象とする |
| User-Agent | HTTP要求を送るソフトウェアを識別するための要求情報 |
| Set-Cookie | サーバーがCookieの保存、更新、失効を指示するためのHTTP応答ヘッダー |
| `REVEL_SESSION` | AtCoderの認証済みセッションを表すCookie名 |
| CLI | 文字でコマンドを入力して操作する利用者向けの窓口 |
| OS | 端末の基本機能と資源を管理する基本ソフトウェア |
| ID | 対象を他と区別するための識別子 |
| AI | 人工知能を用いた支援機能の総称 |
| URL | ウェブ上の場所を示す文字列 |
| UTC | 地域ごとの時差に依存しない協定世界時 |
| JSON | 項目と値を構造化して表すデータ形式 |
| CSRFトークン | 利用者の意図しない送信を防ぐため、送信元の正当性確認に用いる秘密値 |
| Authorization | HTTP要求で認証情報を伝えるためのヘッダー名 |
| CDP | Chromeを外部プロセスから制御・観測するための開発者向け通信規約 |
| WebDriver | ブラウザーを外部プログラムから操作するための標準的な仕組み |
| Turnstile | Cloudflareが提供する、人による操作と自動アクセスを識別するための仕組み |

## 1. CLI・workspace・出力UX

### 1.1 日常commandの最終仕様

**状態:** 一部決定済み

**決定済みの内容:** 製品名は`AlgoLoom`、正式commandは`aloom`、互換commandは`algoloom`とする。AtCoderの明示ログインは`aloom auth login`とする。`al`は利用者が任意に設定するaliasであり、AlgoLoomは自動登録しない。`loom`は使用しない。aliasはまずshell側へ委ね、将来AlgoLoom内で提供する場合もcanonicalなAlgoLoom commandとargv prefixへの短縮に限定し、組み込みcommandの上書き、再帰、raw shell構文、AlgoLoom外のcommand実行を許可しない。`get`、SolveAttemptの開始・pause・resume・終了、freshな解き直し、`test`、`checkpoint`、`submit`、`log`、`show`、`diff`、問題一覧ページと公式問題・解説ページへの外部参照、`export`の責任分割もMVP契約として定義されている。

**残る未決:** `auth login`以外の各subcommand・引数・optionの最終名称、aliasとcompletionの詳細、各表示例を正式契約にする範囲。

**決めること:** subcommand名、引数、option、alias、および例示されたCLI案をどこまで正式契約にするか。

**原文:** 「本書および関連文書に記載するcommand名、引数、option、対話例、出力例は、明示的にCLI契約として確定したものを除き、機能と責任を説明するための暫定案とする。具体的なCLI設計は、上記原則と実際の利用検証を踏まえて別途決定する。」

出典: [プロダクトビジョン §3.5](../product/vision.md#35-標準ツールとの責任境界)、[MVPスコープ §1.2](../product/mvp.md#12-本書で決めないこと)

### 1.2 workspace metadataとcontext指定

**状態:** 一部決定済み

**決定済みの内容:** metadataは問題directoryとともに移動できる通常fileとし、問題ID、judge、schema version等の宣言的情報だけを持つ。絶対pathとdirectory名を恒久IDにしない。現在directory・明示source・親directoryから安全に一意な場合だけcontextを推測し、複数候補・欠損・矛盾時は暗黙に選ばない。

**残る未決:** file名・形式・Schema versionの具体値、探索上限・除外directory・規模上限、明示option名、引数省略条件。

**決めること:** metadata fileの名称・形式・Schema version、探索範囲・除外directory・規模上限、明示option名、引数を省略できる条件。

**原文:** 「metadata fileの名称と形式、探索上限、明示option名は機能設計で決める。」

**原文:** 「問題directoryと一緒に移動するmetadata fileの名称、形式、Schema version」「同じ問題を検出するworkspace探索範囲と、除外directory、規模上限」「workspace、問題、source contextを明示指定するoptionの具体名」「引数を省略できる条件」

出典: [Core契約 §2.3](../architecture/core-contracts.md#23-workspaceとcontext)、[ストレスフリーUX設計 §11](../quality/stress-free-ux-design.md#11-現時点で確定しないこと)

### 1.3 `get`の補助動作と途中失敗の回復

**状態:** 決定済み

**決定済みの内容:** 機能設計で、`get`は公式ページ確認、公開sample取得、template・test作成、ユーザーDB保存、公式問題ページのbrowser表示の順に処理すると決定している。再実行では編集済みsourceを上書きせず、完了段階を識別し、重複fileや破損contextを増やさない。browser起動だけの失敗はworkspace作成失敗にせず、既存directoryを勝手にmerge・rename・削除しない。

**残る実装設計:** 各完了段階を識別するmarker、file作成とDB transactionの具体的な順序、atomic writeとcleanupの実装方法。

**原文:** 「問題ページをbrowserで開くか、templateをいつ作るか等は機能設計で決める。」

**原文:** 「問題取得は、公式確認、sample取得、directory作成、template作成、DB保存、browser起動という複数の副作用を持つ。次の途中状態が未定義である。」

出典: [Core契約 §3.2](../architecture/core-contracts.md#32-冪等性と部分失敗)、[問題選択・カタログ設計 §6.3](../features/problem-selection-and-catalog.md#63-getの処理)、[ストレスフリーUX設計 §3.3](../quality/stress-free-ux-design.md#33-問題取得の途中失敗と再実行)

### 1.4 履歴・表示・診断の細部

**状態:** 一部決定済み

**決定済みの内容:** MVPの`show`はterminal上のplain text、`diff`はunified diffをfallbackとする。成功、部分成功、回復可能errorの情報順序は「主結果、追加失敗、影響を受けないもの、次の行動」とし、詳細情報は既定で大量表示せず確認方法を示す。Editor save、明示checkpoint、提出前の必須submission snapshot、FocusIntervalを別の保存境界として扱い、AlgoLoomはEditorの未保存bufferを取得・保存しない。MVP後の自動checkpointはopt-inかつevent駆動を優先し、失敗しても成功済みのpause、test、milestone、提出を変更しない。非公式・非公認であることは、バージョン表示と最上位のヘルプで示し、サブコマンドのヘルプ、通常のコマンド出力、初回起動では繰り返さない。

**残る未決:** 同一sourceへの重複操作の具体表示、色・spinner・table layout、進捗の見た目、診断command名、Viewer fallbackの表示量。MVP後の自動checkpointを採用するか、採用する場合の対象event、保持期間、容量上限、重複表示、無効化、export・同期範囲。

**決めること:** 同一sourceへの重複操作の見せ方、terminal表示の色・spinner・table layout、進捗表示、統一診断入口、Viewer fallbackの表示量、および利用者検証後の自動checkpoint採否と保持契約。

**原文:** 「同じsourceに対する重複操作をどう見せるかは機能設計で決めるが、無断で別のsourceへ差し替えない。」

**原文:** 「terminal表示の色、spinner、table layout」

**原文:** 「進捗表示の具体的な見た目」「統一診断入口の具体名」「Viewer fallbackの具体的な表示量」

出典: [Core契約 §5.3](../architecture/core-contracts.md#53-checkpoint)、[MVPスコープ §1.2](../product/mvp.md#12-本書で決めないこと)、[ストレスフリーUX設計 §11](../quality/stress-free-ux-design.md#11-現時点で確定しないこと)

### 1.5 exit codeとmachine-readable出力

**状態:** 一部決定済み

**決定済みの内容:** success、利用者入力error、環境error、外部サービスerror、状態不明を区別する。通常出力と診断出力を分離し、timeout後の状態を示し、内部traceは既定表示しない。

**残る未決:** 各状態へ割り当てる具体的なexit codeとmachine-readable Schema。

**決めること:** exit codeの体系とmachine-readable出力の詳細。

**原文:** 「exit codeとmachine-readable出力の詳細は機能設計で決める。」

出典: [Core契約 §2.6](../architecture/core-contracts.md#26-出力とerror)

### 1.6 任意機能の具体的な導線

**状態:** 一部決定済み

**決定済みの内容:** AI接続先の初期設定は、接続先、処理方式、接続先URL・認証情報の取得元、実行場所・課金、提供能力、モデル、送信範囲・送信先、同意、接続確認の順とする。クラウド同期は通常コマンドで宣伝せず、複数端末利用を求めたとき、`sync`を実行したとき、または明示的にヘルプを開いたときだけ案内する。AtCoderの明示ログインcommandは`aloom auth login`とし、CLIから可視専用browserへ1回移り、完了後にCLIへ戻る。初回だけ利用委譲と署名済み拡張機能の追加を求める。`submit`で再認証が必要な場合は、browserを開く理由と中止方法を表示して自動起動し、別commandや追加のYes/No確認を要求しない。利用者がログインとTurnstileを手動で完了し、必要なセッションだけをOSの秘密情報保管庫へ保存する。初回提出前と各提出前に本人アカウントを確認し、失敗時もローカルテストと履歴閲覧を止めない。方式AのmacOS・ChromeにおけるCookie限定取得、本人照合、秘密情報保管庫への保存、新規プロセスでの再照合、後始末は[`p0-23`](../verification/judge-adapter/results/2026-08-12-p0-23.md)で成立を確認した。

認証失敗について、[`p1-02`](../verification/judge-adapter/results/2026-08-13-p1-02.md)では、未認証、サーバーが示した明示期限を過ぎた状態、AtCoder・Cloudflare側の拒否、ページ構造の変更、通信障害を分類できた。期限が示されていないログイン誘導から期限切れを推測せず、いずれの失敗も提出前に停止できる。[`p1-03`](../verification/judge-adapter/results/2026-08-13-p1-03.md)では、Cookie更新への追従、サーバーが示した明示期限の扱い、世代番号を用いた原子的な置換、新規プロセスでの本人再照合が成立した。明示失効、更新競合、秘密情報保管庫の読取・書込障害も、平文保存へ切り替えず提出前に停止できる。

**残る未決:** 対話型画面の有無、AtCoder認証の状態確認・削除に使うcommand名と具体表示、方式Aで正式対応するブラウザーと版の組み合わせ、方式Aの配布境界、AI画面の最終的な階層表現、別名・入力補完の詳細。例示されたAI・同期コマンド名もコマンドライン最終設計までは暫定案である。`auth login`、一往復の認証UX、`submit`からの透明な再認証、`p1-02`と`p1-03`で確定した失敗分類、期限非推測、Cookie更新、原子的置換、再起動後の本人照合、提出前停止は未決に含めない。

**決めること:** 対話型画面の有無、AtCoder認証の状態確認・削除commandと表示、対応ブラウザーと配布境界、AI接続先選択画面の階層、クラウド同期を案内するタイミング、シェルの入力補完の詳細。

**原文:** 「interactive UIの有無」「AtCoder認証を確認する具体的な操作」「AI Provider選択画面の具体的な階層」「Cloud同期を案内するタイミング」「aliasとshell completionの詳細」

出典: [AtCoder認証設計](../architecture/atcoder-authentication.md)、[ストレスフリーUX設計 §11](../quality/stress-free-ux-design.md#11-現時点で確定しないこと)、[Review Backend設計 §6](../features/llm-provider-design.md#6-セットアップux)、[ローカル利用とCloud同期の段階的設計 §8](../features/local-and-cloud-sync-design.md#8-cli設計)

### 1.7 ユーザーカスタマイズの具体仕様

**状態:** 一部決定済み

**決定済みの内容:** 設定なしでcanonicalなCore導線を成立させる。カスタマイズは、表示、反復入力の既定値、AlgoLoomが利用する既存外部toolの参照とprocess-localな呼出方法等、Coreの意味を変えないuser-levelの差分に限定する。外部tool本体や永続設定はカスタマイズ対象にしない。commandの意味、状態遷移、データの権威、履歴の不変条件、安全性、privacy、外部作用への同意は変更できない。明示指定、user-level preference、製品既定値の順に解決し、workspace metadataを汎用設定層にしない。新項目は既存設定の変更を要求せず、安全な既定値を適用する。任意機能の設定不備は影響する機能へ局所化する。通常commandはAlgoLoom所有領域と明示workspace以外の永続状態を変更せず、alias、completion、Editor連携は外部設定の直接編集より設定断片・手順の生成を優先する。

**残る未決:** user preferenceの保存場所・file形式・Schema version、最初に採用する設定項目、環境変数を設定経路として採用する範囲、設定の参照・変更・reset・無効化・migrationを行う具体的なCLI、AlgoLoom自身の設定fileを自動更新する場合のbackupとcomment保持方針、外部設定へ作用する専用setup helperを将来採用するか。

**決めること:** 利用者検証で反復的な摩擦を確認したうえで、設定Schema、設定操作、migration、診断、最初の設定項目を具体化する。外部設定へ作用するsetup helperは、設定断片の生成では解決できない実需と、差分、backup、冪等性、rollbackを保証できる場合だけ採否を判断する。

出典: [プロダクトビジョン §3.4](../product/vision.md#34-シンプルさとユーザーの自由)、[Core契約 §2.4](../architecture/core-contracts.md#24-設定と実行commandの信頼境界)、[ストレスフリーUX設計 §7.7](../quality/stress-free-ux-design.md#77-ユーザーカスタマイズ)

### 1.8 学習時間計測のCLIと時計異常からの回復

**状態:** 一部決定済み

**決定済みの内容:** 時間計測はMVPへ含めるが任意とし、利用者の明示操作でSolveAttemptを開始する。`get`、file保存、最初の`test`から暗黙に開始せず、pauseを除いたFocusIntervalからactive durationを計算する。最初の公開sample通過、初回提出、初ACを別のmilestoneとして記録し、初ACのdurationには判定polling時間を加えない。`status`相当を現在状態とactive durationの正規の確認経路とし、通常画面で秒単位timerを常時再描画しない。`test`やcheckpoint後の補助表示は計測中だけ短く表示できる。常駐daemon、Editor plugin、全local test event、自動checkpointはMVPへ含めない。時間を他者rankまたは単一skill scoreへ変換しない。

**残る未決:** start / pause / resume / status / finish / abandonの最終command名と引数、`get --start`相当の明示optionを提供するか、表示精度と丸め、補助表示を既定にする操作、activeな別試行がある場合の具体的な対話、時計の後退・飛躍・未終了intervalに対する確認・訂正UX、AC後に追加作業を続ける場合の既定導線、MVP後に`status --watch`やmachine-readable連携を採用するか。

**決めること:** 明示性を保ちながら記録忘れを増やさない最終CLI、時計異常時に履歴を黙って書き換えない回復方法、通常表示で時間をどの粒度と強さで示すか。

出典: [プロダクトビジョン §3.3](../product/vision.md#33-自己比較を中心とする学習)、[MVPスコープ §3.1](../product/mvp.md#31-mvpに含める能力)、[Core契約 §5.6](../architecture/core-contracts.md#56-solveattemptと学習時間)、[ストレスフリーUX設計 §4.8](../quality/stress-free-ux-design.md#48-時間計測による焦りと記録忘れ)

### 1.9 外部学習資料のCLIとspoiler確認

**状態:** 一部決定済み

**決定済みの内容:** MVPではAtCoder公式問題ページと問題別解説ページをdefault browserで開く。解説本文、画像、PDF、動画、sample codeはAlgoLoomへ取得・保存しない。他ユーザーのAC提出一覧はMVP後のPhase 2候補とし、他ユーザーのcode本文・author・submission IDや、Cookie、browser profileを取得・保存しない。`ReferenceLinkProvider`と`BrowserLauncher`を分け、browser起動失敗はCore履歴を変更しない。未AC時はspoilerを確認し、contest終了を確認できない場合は開かない。`browse`は問題発見、`open`相当はcurrent problemの資料参照として責任を分ける。一覧ページをブラウザで開く`browse`相当はMVPへ含め、カタログの取得・保存とターミナル内検索はMVP対象外とする。一覧の提供元が非公式サービスである旨を開く前に示す。

**残る未決:** `open problem / editorial / submissions`相当の最終command・action・option名、未AC時の対話文、non-interactiveで明示確認を表すoption、初回のADT・virtual contest注意表示、URL変更時のfallback、browserがない環境でURLをどこまで表示するか。

**決めること:** 外部本文を取得しない境界を維持したまま、spoilerの誤操作と確認疲れを両方抑えるCLI、表示、fallbackを利用者検証で確定する。

出典: [外部学習資料参照設計](../features/external-learning-resources.md)、[Core契約 §3.4](../architecture/core-contracts.md#34-外部学習資料への参照)、[配布方針ガイド §4.5](../operations/algoloom-distribution.md#45-解説と他ユーザーの提出code)

### 1.10 freshな解き直しのCLIと回復

**状態:** 一部決定済み

**決定済みの内容:** 同じ問題の解き直しは新しいSolveAttemptとし、前回履歴を上書きしない。freshな解き直しでは、同じ正規問題IDを持つ新しいsibling checkoutと組み込みtemplateを既定で作り、前回sourceを自動copyしない。`abc300_a--02`等は候補名であり恒久IDではない。検証済みlocal sampleは新しい`test/`へ安全にcopyできるが、symlink・hard linkを既定にしない。自分のsnapshotから開始する場合は明示指定し、外部codeを開始元にしない。active / paused attemptを暗黙に変更しない。

**残る未決:** `redo`相当の最終command名、fresh・snapshot・in-placeを選ぶoption、候補directory名の具体形式、checkoutのstable local identity、fileとDBの作成順序、重複操作key、途中状態marker、active attemptがある場合の具体的な対話。

**決めること:** 既存sourceを壊さず、同じ操作の再実行でcheckout・SolveAttemptを重複させないCLI、metadata、transaction、回復UXを機能設計で確定する。

出典: [解き直しworkflow設計](../features/revisit-workflow.md)、[Core契約 §3.3](../architecture/core-contracts.md#33-解き直し用problem-checkout)、[言語・実行環境の可搬性設計 §7.2](../architecture/language-and-platform-portability.md#72-同じ問題をfreshに解き直す場合)

### 1.11 外部資料のターミナル内表示手段

**状態:** 条件付き決定

**決定済みの内容:** ターミナルから離れずに資料を読めるよう、利用者が導入したターミナル内で動作する表示手段へURLを渡す構成をMVP後の候補として採用する。AlgoLoomはURLと起動要求だけを渡し、資料の本文を取得・保存しない。表示手段が未導入または利用不能でも、既定のブラウザへの委譲とCore操作を停止させない。起動の共通要件として、表示言語の指定、一操作あたり一URL、非対話実行と非TTYでの自動起動の抑止、利用者が指定した表示手段の尊重、ターミナルを占有し得る手段を補助動作に使わないこと、起動失敗時のURL提示を定めている。

**残る判断:** 表示崩れの少ない表示手段の選定と、対応OSごとの保証範囲。

**決めること:** 数式、表、図、日英併記を含む実際の問題ページで表示品質を確認し、検証したOSの組み合わせだけを正式対応と表示する。

出典: [外部学習資料参照設計 §4.5](../features/external-learning-resources.md#45-起動の共通要件)、[同 §4.6](../features/external-learning-resources.md#46-ターミナル内での表示)、[ロードマップ §3](../product/roadmap.md#3-phase-2-core安定化近接拡張)

## 2. Core実装・性能パラメータ

### 2.1 実装技術の最終形

**状態:** 未決

**現在の確定境界:** 実装言語はPython、MVPのローカルDBは標準`sqlite3`である。ただし、この項目に挙げた内部実装の具体仕様は決定されていない。

**決めること:** CLI framework、dependency injection手法、class・module・table・column名、metadata/export fileの最終形式。

**原文:** 「CLI frameworkやdependency injection手法」「class、module、table、columnの最終名称」「metadata fileとexport fileの最終形式」

出典: [MVPスコープ §1.2](../product/mvp.md#12-本書で決めないこと)

### 2.2 言語profileとuser-level実行設定

**状態:** 一部決定済み

**決定済みの内容:** MVPはC++、Python、Go、Rustを正式対象とし、単一sourceと標準toolchainを初期保証範囲にした組み込み`LanguageProfile`を提供する。profileはshell文字列ではなくBuildPlan / RunPlanを返し、native macOS、native Linux、native Windowsの`HostPlatform`がprocessを実行する。workspace metadataに実行command、credential、endpointを持たせず、workspaceから任意commandを定義させない。個別profile同士と個別OS Adapter同士を依存させず、canonical language IDをlocal toolchainとAtCoder上の言語IDから分離する。Coreと履歴の意味は具体的なcompiler/runtime versionへ依存させず、local toolchain observationと提出時のjudge language resolutionを別々に扱う。将来のuser-level実行設定も、利用者が既に導入したexecutableの参照とchild processの安全なargvに限定し、toolchainのinstall、update、設定file、永続的な`PATH`・環境変数を変更しない。

**残る未決:** user-level設定による拡張子・template・実行file・引数変更を採用するか、およびその具体契約。compiler/runtime/OS release/CPU architectureの正確なversion matrixは2.3の実測対象として残る。

**決めること:** user-level設定から拡張子、template、実行file、引数を変更可能にするかと、その安全な契約。

**原文:** 「将来、user-level設定から拡張子、template、compile/run commandを変更できる構成を検討する。」

出典: [アーキテクチャ概要 §3](../architecture/overview.md#3-解答言語host-os開発環境設定管理)、[言語・実行環境の可搬性設計 §5.3](../architecture/language-and-platform-portability.md#53-言語compilerversion非依存の境界)、[Core契約 §2.4](../architecture/core-contracts.md#24-設定と実行commandの信頼境界)

### 2.3 実行・保持・性能の具体値

**状態:** 一部決定済み

**決定済みの内容:** compileとrunのtimeoutを分け、stdout/stderrに上限を設け、timeout時は各`HostPlatform`がprocess treeを終了する。MVPの`test`はcompile時間とsampleごとのlocal run時間を可能な限りmonotonic clockで計測して表示するが、全local test eventは保存しない。AtCoderが返すjudge execution time / memoryはnullableなremote観測として保存し、local値と同一条件のbenchmarkにしない。local peak memoryはOSごとの意味と取得範囲を検証してからMVP後に段階導入する。DB lockは有限時間とし、外部通信ではconnection・request・polling全体の上限を分ける。AtCoder側との互換性確認は、release時のmaintainer確認を主とし、実行時は提出に必要なpageで必須条件を確認する。hash確認だけの利用者ごとの追加GETと無期限の常駐監視は行わない。foregroundとbackgroundはasync APIの採否ではなく利用者の主目的と制御返却点で分け、中断して破棄できない処理は必要な状態を先に耐久保存する。初期段階ではbackground化のための常駐daemonを導入しない。初期性能目標として`log` p95 100ms、`show` p95 150ms、`diff` p95 250ms等の計測仮説が定義されている。

#### 技術検証から得た初期値候補

次の値は、終了済み過去問1問を対象とするmacOS上の限られた検証から得た観測値と、検証支援コードで実動した上限である。実装時の初期値候補として扱い、対応OS、回線、混雑状況、履歴量を含む代表環境で再計測するまでは確定値または性能保証にしない。

| 対象 | 初期値候補・観測値 | 出典 |
|---|---|---|
| 判定確認の開始 | 提出IDの取得から最初の判定確認開始まで13ミリ秒 | [`p0-22`](../verification/judge-adapter/results/2026-08-12-p0-22.md) |
| 判定確認の間隔 | サーバー指定の1,000ミリ秒より長い、最小2,000ミリ秒を採用。応答完了後に1,999〜2,000ミリ秒の待機を5回適用 | [`p0-22`](../verification/judge-adapter/results/2026-08-12-p0-22.md) |
| 判定確認の回数と所要時間 | 判定待ち5回と最終判定1回の計6回で確定。最初の判定待ち応答完了から最終判定応答完了まで10,542ミリ秒 | [`p0-22`](../verification/judge-adapter/results/2026-08-12-p0-22.md) |
| 通信と判定確認の上限 | 接続5秒、1要求20秒、判定確認のGET最大10回、判定確認全体120秒を別々に設定して実動 | [`p0-22`](../verification/judge-adapter/results/2026-08-12-p0-22.md) |
| 認証確認の回数と間隔 | 本人確認のGETを最大3回とし、各開始を2秒以上空ける。実サービスでは前の応答完了から2,000ミリ秒、2,159ミリ秒を空けて3回実行 | [`p1-03`](../verification/judge-adapter/results/2026-08-13-p1-03.md) |
| ジャッジ実行時間の保存単位 | 整数ミリ秒。実サービス表示の11ミリ秒を11ミリ秒として正規化 | [`p2-01`](../verification/judge-adapter/results/2026-08-13-p2-01.md) |
| ジャッジ最大メモリの保存単位 | 整数バイト。実サービス表示の9,172キビバイトを9,392,128バイトとして正規化 | [`p2-01`](../verification/judge-adapter/results/2026-08-13-p2-01.md) |

これらは、下記の「実測後に固定する」という未決範囲を変更しない。既存の`log`、`show`、`diff`等の初期性能目標も、引き続き計測仮説である。

**残る未決:** 実測後に固定するtimeout・出力量・保持期間・resource上限・SLO、互換性の利用時の再確認間隔と保守確認の頻度、local durationの表示精度と丸め、対応OSごとのpeak memory取得方法・子process範囲・強制memory/process制限、compiler/runtime version matrix。

**決めること:** timeout、出力量、保持期間、compile/runのresource上限、互換性の再確認間隔と保守確認の頻度、local計測の表示精度、OS別peak memoryの対応可否、性能SLOを固定する値と計測対象環境。

**原文:** 「timeout、出力量、保持期間等の具体値」「compilerとruntimeの細かなversion matrix」

**原文:** 「数値は対応OS、DB規模、sample数、Providerにより変わるため、次は実装開始時の仮説である。固定の約束にする前に、代表的な端末と履歴件数でp50/p95を計測する。」

出典: [MVPスコープ §1.2](../product/mvp.md#12-本書で決めないこと)、[パフォーマンスと待機体験の設計 §1.3](../quality/performance-and-waiting-design.md#13-非同期化と制御返却点)、[同 §5](../quality/performance-and-waiting-design.md#5-性能待機の初期契約)、[MVP機能設計 §8.5](../../spec/features.md#85-atcoder側の変更検知と互換性確認f-submit-05)

### 2.4 DB保守の実行規約

**状態:** 決定済み

**決定済みの内容:** transactionは短くし、競合時は有限時間だけ待機または安全に再試行する。backup/exportは明示操作に分離し、通常閲覧時に実行しない。migration中は通常commandを停止し、完了またはrollback後に再開する。checkpointを閲覧経路で実行しない。lock上限の具体秒数だけは2.3の計測対象として残る。

**決めること:** DB lock、migration、checkpoint、backupをいつ・どのような排他と回復表示で実行するか。

**原文:** 「DB lock、migration、checkpoint、backupの実行規約を確定する。」

出典: [パフォーマンスと待機体験の設計 §7](../quality/performance-and-waiting-design.md#7-実装順序と非目標)

### 2.5 ローカルテストの比較方式

**状態:** 一部決定済み

**決定済みの内容:** 標準出力と公開入出力例の出力を比較する。コンパイルエラー、実行時エラー、タイムアウト、シグナルによる終了、出力量超過を不一致と混同せず、別の分類として示す。どの入出力例がどの理由で失敗したかを確認できる。AtCoderのジャッジを再現できない形式は、近似判定であることを表示するか、未対応として停止する。

**残る未決:** 空白と改行の正規化規則、浮動小数の既定の許容誤差、近似判定の表示文言。

**決めること:** 上記が確定するまでは、近似判定を「AtCoderのジャッジと一致する」と表示しない。

出典: [Core契約 §4.1](../architecture/core-contracts.md#41-testが保証すること)、[MVP機能設計 §6.2](../../spec/features.md#62-比較方式)

### 2.6 ツールチェーン観測を履歴へ保存するか

**状態:** 決定済み

**決定済みの内容:** 2026年8月27日に**保存すると決定した。** ただし7項目と並ぶ8番目の履歴項目としてではなく、**SolveAttemptに付随する観測**として保存する。保存するのはtoolchainの種類、自己申告のversion文字列、host OS名とversion、CPU architectureに限る。compiler/runtimeの絶対path、hostname、利用者名、環境変数、`PATH`は保存しない。canonical language IDを識別子とし、ツールチェーン観測を識別子や結合keyにしない。compilerとruntimeの詳細なversion matrixは維持しない。私的exportへ含め、公開用solution bundleへ含めない。MVPでは端末ローカルに保存し、Cloud同期での扱いは[可搬性設計 §9.1](../architecture/language-and-platform-portability.md#91-共有可能な論理データと端末固有データ)の「条件付き」を維持して同期採用時に再確認する。

**判断の理由:** 3点ある。第一に、可搬性設計 §9.1 が既に「診断・再現性の補助情報」として分類しており、[解き直しworkflow設計](../features/revisit-workflow.md)の「同じalgorithmの別言語」比較が既にtoolchain観測へ依存している。第二に、[初回起動・環境診断](../../spec/features.md#31-初回起動環境診断f-base-01)で既に観測しているため追加の外部作用がない。第三に、**ツールチェーンが変わったことを示せないと、環境差による実行時間の変化を成長と誤読させる。** 保存は「時間短縮だけを成長と断定しない」原則を支える。

**不整合の解消方法:** Core契約 §5.1 の7項目は「利用者が何をしたか」の記録であり、ツールチェーン観測は「観測が生まれた文脈」で性質が異なる。8番目として並べるのでも §4.2 から外すのでもなく、性質の違いを明記して両文書を一致させた。

出典: [Core契約 §5.1](../architecture/core-contracts.md#51-mvpで保存する履歴)、[可搬性設計 §4.2](../architecture/language-and-platform-portability.md#42-境界ごとの責任)、[同 §9.1](../architecture/language-and-platform-portability.md#91-共有可能な論理データと端末固有データ)

## 3. MVP後の機能採否

### 3.1 AI reviewを正式に採用するか

**状態:** 条件付き決定

**決定済みの内容:** AI reviewはMVPへ含めず、Core完了後の独立した採用判断とする。採用後もCoreへ組み込まず、安全判定、過去問識別、送信・費用・保持方針の明示、sourceを自動編集・実行・提出しない権限制約をすべて満たした場合だけ正式なoptional Capabilityとして提供する。依存方向はAI reviewからCoreのsnapshot・verdict・diff参照契約への一方向とし、CoreはProvider、prompt、review設定・状態へ依存しない。reviewはCore tableの任意列ではなく、安定IDを参照する追記型revisionとして保存する。

**残る判断:** 条件を実装・検証した後に正式採用するか。

**決めること:** AI reviewを正式なoptional Capabilityとして採用するか。採否は、ルールのversion確認、過去問識別、送信・費用・保持方針の確認、review-only制約を満たした後に判断する。

**原文:** 「AI reviewはMVP後の独立した採用判断とする。採用後もCoreへ組み込まず、次をすべて満たした場合だけ正式なoptional Capabilityとして提供する。」

出典: [ロードマップ §4.1](../product/roadmap.md#41-ai-review)、[Repair Lab 将来構想 §0](../future/repair-lab-future-design.md#0-結論)

### 3.2 追加Review Backendを採用するか

**状態:** 一部決定済み

**決定済みの内容:** 初期local Model API候補はOllamaとLM Studioとする。BYOK Cloud APIは各Providerを個別Adapterとして評価する。Codex Agent Bridgeは公式app-server、外部runtime所有の認証、一時directory、tool禁止、session破棄、response再検証を満たす場合だけ導入する。Claude Code・Gemini CLIのsubscription credentialは、第三者組み込みの明示許可があるまで再利用しない。

**残る未決:** 各Cloud API・Agent Bridgeの実検証後の採否と、Phase 6以降の追加Backend。

**決めること:** BYOK Cloud API、Codex Agent Bridge、追加Backendごとの採否。追加Backendは需要、公式組み込み経路、利用規約、認証、料金、data retention、SDK依存、contract/security testで評価する。

**原文:** 「Phase 6: 追加Backend」「ユーザー需要を確認する。」「公式の組み込み経路、利用規約、認証方式、料金、data retention、SDK依存を評価する。」

出典: [Review Backend・LLM Provider選択・実行基盤設計 §11](../features/llm-provider-design.md#11-段階的なreview-backend対応)

### 3.3 近接拡張の採否・優先順位

**状態:** 未決

**決定済みの境界:** native Windows、Go、RustはMVPへ昇格済みである。Editor / IDE非依存のCore互換性はMVP契約であり、保存済みの通常fileとCLIだけで主要導線を成立させる。個別のEditor / Diff Viewer Adapter、plugin、machine-readable出力はCore互換性と分離したMVP後の候補である。その他の候補もMVP対象外であり、追加してもMVPのinstall、日常command、offline履歴を複雑にしないことは決定している。

**残る未決:** 各候補を実際に採用するか、および候補間の優先順位。

**決めること:** WSL、追加言語、project build、Editor / Diff Viewer、catalog、問題・解法タグ、local peak memory、opt-in自動checkpoint、継続timer表示、既存履歴import、自動backup/restore、machine-readable出力を、Core安定後にどの順で採用するか。

**原文:** 「Phase 2: Core安定化・近接拡張」

出典: [ロードマップ §3](../product/roadmap.md#3-phase-2-core安定化近接拡張)

### 3.4 CLI問題選択のインタラクション

**状態:** 一部決定済み

**決定済みの内容:** 初期版の主経路はAtCoder Problemsで発見し、`get`で開始する。一覧ページをブラウザで開く導線はMVPへ含め、カタログ取得はPhase 2、`pick`はPhase 3とする。Phase 3の`pick`も独自のworkspace作成処理を持たず、選択後に共通`get`を呼ぶ。local statusはAtCoder全体の提出状態と明確に区別する。

**残る未決:** `pick`を正式採用するか、インタラクティブUIの実装方式、fzfがない場合のfallback選択。

**決めること:** Phase 3の`pick`で採るインタラクティブ方式と、fzf非導入環境でのfallback（番号選択・一覧のみ・`--no-interactive`）。

**原文:** 「fzfがない環境では、次のいずれかへフォールバックする。」

出典: [問題選択・カタログ設計 §7.3](../features/problem-selection-and-catalog.md#73-fzf連携)

### 3.5 問題タグ・SolveAttempt解法タグ

**状態:** 一部決定済み

**決定済みの内容:** タグは製品Phase 2候補とし、単一category列ではなく複数付与できる関係として扱う。問題一般の分類であるProblemTagと、今回実際に使った解法を表すSolveAttempt解法タグを分離する。user、external curated、将来のAI suggestionのsourceを保持し、利用者が入力したタグを外部更新で上書きしない。外部のalgorithm tagは未AC時のspoilerになり得るため通常導線で既定表示せず、明示したタグで分野別練習を行う場合も他の解法タグを自動展開しない。タグを安全判定の正本、苦手分野の断定、単一scoreへ使用しない。

**残る未決:** 初期controlled vocabularyとstable tag ID、custom tag・alias・mergeの具体UX、最終command名、問題とSolveAttemptのscope指定、user tag removeのrevision・tombstone・同時競合、外部tag Providerを採用するか、Provider revisionとcache更新、未ACからACへ変化したときの表示、タグ検索を履歴と`pick`のどちらから先に提供するか。

**決めること:** user tagの最小導線と語彙を利用者検証で確定し、外部タグは精度、出典、継続性、spoiler制御を満たすProviderだけを個別採用する。

出典: [問題選択・カタログ設計 §10.4](../features/problem-selection-and-catalog.md#104-問題タグとsolveattempt解法タグ)、[ロードマップ §3](../product/roadmap.md#3-phase-2-core安定化近接拡張)

### 3.6 公開用solution bundleを正式に採用するか

**状態:** 条件付き決定

**決定済みの境界:** GitHub等へのrepository作成、認証、commit、push、visibility変更はAlgoLoomの責任にしない。MVPの完全版`export`は学習履歴の可搬性・退避を目的とし、公開用成果物として扱わない。公開支援に実需がある場合は、MVP後のPhase 2候補として、一問・一source、provider非依存、networkなし、allowlist方式のlocal bundleを先に検証する。source originとhash、含有file、除外data、contest状態を生成前に確認可能にし、sample、問題文、他者code、履歴、内部ID、絶対path、credentialを既定で含めない。開催中・状態不明・個別ruleを解釈できない場合はfail closedとし、既存fileを上書きせず原子的に生成する。

**残る未決:** 公開候補bundleを正式採用するか。最終command名と完全版`export`からのCLI分離、source候補の提示順、README・manifestの既定、secret検査範囲、contest状態の判定元、license案内、複数問題bundleを将来提供するか。

**決めること:** Core安定後に、利用者が手動で自分の解答を公開するときの誤混入、反復作業、必要metadataを検証する。AlgoLoom固有の切り出し支援が必要と確認できた場合だけlocal bundleを試作し、内容、privacy、contest rule、filesystem、失敗回復の受け入れ条件に合格した後に正式採用を判断する。GitHub APIや自動pushはAlgoLoomの製品範囲へ含めず、local bundleで解決できない需要も公式の外部toolと手順、または別applicationで満たす。

**原文:** 「公開を支援する設計は考慮するが、GitHub連携ではなく、選択した自作sourceから公開候補物をlocalに作り、標準toolへ引き渡す境界までとする。」

出典: [公開用solution bundle将来設計](../features/public-solution-bundle-design.md)、[製品ビジョン §3.3](../product/vision.md#33-自己比較を中心とする学習)、[Core契約 §7.3](../architecture/core-contracts.md#73-export)、[ロードマップ §3](../product/roadmap.md#3-phase-2-core安定化近接拡張)

### 3.7 統合開発環境を採用するか

**状態:** 条件付き決定

**決定済みの内容:** エディタ、ファイルマネージャ、ブラウザをAlgoLoom自身へ内蔵しない。内蔵エディタを既定にすると利用者が選んだ編集環境を中心にする原則に反し、任意にすれば内蔵する意味が薄い。ターミナルはHTMLを描画しないため内蔵ブラウザはテキストブラウザ相当に留まり、実際のレンダリングにはデスクトップアプリ化が必要で、配布方針と環境非侵襲性が成立しなくなる。一つの画面へまとめたい需要には、利用者が既に導入したツールの構成案内で応える。

**残る判断:** 長期候補から製品フェーズへ昇格させるか。

**決めること:** 利用者検証で環境構築の失敗が主要な離脱理由だと確認でき、既存のエディタとIDEに対する優位を説明でき、エディタ非依存の原則を見直す判断を行った場合にだけ再評価する。

出典: [製品ビジョン §3.1](../product/vision.md#31-エディタ非依存)、[可搬性設計 §8.6](../architecture/language-and-platform-portability.md#86-一つの画面へまとめる場合の案内)、[ロードマップ §5](../product/roadmap.md#5-長期候補)

## 4. Cloud同期・データ共有

### 4.1 同期Adapterの採用

**状態:** 条件付き決定

**決定済みの内容:** Cloud同期はMVPへ含めない。MVP後はTurso Syncを第一候補として同一論理Schema・Adapter契約で試作する。必須検証に合格すれば同期Beta候補へ採用し、不合格なら公開を延期する。ローカル履歴をCloudなしで参照できない方式は採用しない。

**残る判断:** SDK、耐久性、競合、強制終了、bootstrap、wheel導入の実測後にTurso Syncを正式採用するか。不合格かつ実需がある場合だけEmbedded Replica + outboxまたは別Adapterを評価する。

**決めること:** MVP後の同期BetaでTurso Syncを採用するか、公開を延期するか、Embedded Replicaまたは別Adapterを評価するか。

**原文:** 「試作と正式な設計判断が完了するまでは、同期機能を正式公開しない。Turso Syncの必須検証に合格した場合は第一候補として採用し、不合格の場合は同期公開を延期する。」

出典: [Turso移行互換性設計 §10](../integrations/turso-migration-compatibility-design.md#10-同期方式の採用判断)、[Turso設計ガイド §0](../integrations/turso-design-guide.md#0-結論)、[ローカル利用とCloud同期の段階的設計 §14](../features/local-and-cloud-sync-design.md#14-段階的な実装計画)

### 4.2 マネージド同期（AlgoLoom Cloud）の提供

**状態:** 一部決定済み

**決定済みの内容:** 初期OSS版の同期方式はユーザー所有のTursoを使うBYOCとし、AlgoLoom Cloudは必須要件にしない。Cloud同期自体もローカル利用へ追加する任意機能で、既定OFFとする。

**残る未決:** BYOC利用状況と離脱理由を確認した後、AlgoLoom Cloudを別サービスとして提供するか。

**決めること:** BYOCの利用状況・離脱理由、運用費、認証、法務、privacyを評価した上で、AlgoLoom Cloudを別サービスとして提供するか。

**原文:** 「初期OSS版の必須要件にはせず、BYOCの利用状況を確認した後に別サービスとして判断する。」

出典: [ローカル利用とCloud同期の段階的設計 §13](../features/local-and-cloud-sync-design.md#13-byocと将来のマネージド同期)

### 4.3 Cloud-primary構成を再検討するか

**状態:** 条件付き決定

**決定済みの内容:** 評価順は標準SQLiteのCore、Turso Sync、必要時のEmbedded Replica + outboxとする。Supabase、Neon、Crunchy Bridge等は現在の採用候補ではなく、将来Cloud-primary構成を再検討する場合だけ比較対象へ戻す。どの場合もローカル永続化・読み取り契約を維持する。

**残る判断:** Turso系の候補が条件を満たさず、Cloud-primaryを再検討する必要が生じた場合の方式選択とローカル層の設計。

**決めること:** Turso Syncが不適合な場合に、Embedded Replica + outbox以外（Supabase、Neon、Crunchy Bridge等）を比較対象に戻すか。その場合のローカル永続化・読み取り層。

**原文:** 「Supabase、Neon、Crunchy Bridge等は、将来Cloud-primary構成を再検討する場合の比較対象として残す。その場合も、Cloudへの直接問い合わせで履歴表示を遅くしないためのローカル永続化・読み取り層を別途設計する。」

出典: [DB候補比較メモ §6](../research/db-comparison.md#6-あなたのケースでの推奨)

### 4.4 共同編集・削除を扱う場合の競合戦略

**状態:** 決定済み

**決定済みの内容:** 現在想定する個人または調整可能な少人数、同一データを複数端末で同時編集しない、追記中心という前提ではlast-push-winsを正式に許容する。手動merge UIやfield単位の複雑な競合解決は実装しない。大人数、共同編集、リアルタイム編集、削除競合、監査要件が生じた場合だけ再評価する。

**決めること:** 利用規模や共同編集・削除・監査要件が変わった場合に、last-push-winsを維持するか、楽観ロック、競合UI、CRDT/OT、tombstone、不変ログなどを導入するか。

**原文:** 「次のいずれかが発生した場合は、last-push-winsを再評価する。」

出典: [Turso設計ガイド §5.5.3](../integrations/turso-design-guide.md#553-再検討が必要になる条件)

### 4.5 学習データアクセス基盤・Hosted APIの具体仕様

**状態:** 一部決定済み

**決定済みの内容:** 学習データアクセスはMVP対象外の長期候補とする。MVPのversion付きexport、MVP後のmachine-readable出力、共通Query・Analytics契約、local data access、公式dashboard、Hosted APIの順に責任を分けるが、local data accessとdashboardは実需に応じて前後または並行できる。公式dashboardは唯一の表示経路ではなく、共通契約を利用するreference clientとする。APIは本人の観測事実と説明可能な複数の自己振り返り指標を提供し、単一skill score、他者rank、DB Schema、secret、端末固有path、外部所有本文を提供しない。初期APIはread-onlyとし、集計・履歴metadataとsource・review本文を別scopeにする。localの基本経路はCloud accountなしで無料、Hosted APIの本人readも無料提供を目標とするが、rate limitとfair-useを設けられる。

**残る未決:** local interfaceをlibrary、subprocess protocol、localhost HTTP APIのどれにするか。Hosted APIのREST / GraphQL等の形式、endpoint、認証、OAuth flow、scope名、CORS、token lifetime、API version support期間、source accessの既定、指標の最小母数・統計定義、無料枠の具体上限、公式dashboardの配置、第三者client登録、AlgoLoom Cloudとの提供順序。

**決めること:** exportとmachine-readable出力の利用実績を確認した上で、必要なRead Model、指標定義、local interfaceを確定する。Hosted APIはCloud上のデータ権威、認証・ownership、privacy、運用費、rate limit、service終了時のexportを検証してから正式採用を判断する。

出典: [学習データアクセス・可視化API将来設計](../features/learning-data-access-api-design.md)、[ロードマップ §5](../product/roadmap.md#5-長期候補)、[セキュリティ設計ガイド §6.8](../quality/security-design.md#68-将来のweb-uiと保存型xss)

## 5. AIレビューのルール適用

### 5.1 開催中AHCの専用プロファイル

**状態:** 条件付き決定

**決定済みの内容:** 初期版では開催中AHCのAI reviewを拒否する。2026年8月29日以降の開催中短期AHCは、専用の生成AI利用ルールにより生成AIが原則禁止され、対話的なreview・debug・testも禁止対象になるため、単発操作だけを理由に解禁しない。長期AHCを含む他のAHCも、rule種別・version・適用期間を確認できる専用profileがない間は拒否する。

**残る判断:** 将来の公式ruleが許容し、rule種別・version・適用期間を安全に識別できるAHCについてだけ、専用profileを有効化するか。

**決めること:** 対象AHCの公式ruleが許容する場合に限り、rule profileの実装と再確認手順を経て有効化するか。短期AHCの現行禁止を、単発review等の製品側制限だけで上書きしない。

**判断の更新理由:** 2026年8月18日に短期AHC向けの新ruleが公開され、2026年8月29日から適用されるため。

出典: [AIレビュー安全設計 §3.2](../features/ai-review-safety-design.md#32-ahcをrule別に扱う理由)

### 5.2 ADTのAIレビュー判定

**状態:** 決定済み

**決定済みの内容:** AlgoLoom内の初期判定は、ADTで再利用中の過去問を「許可・注意表示」とする。コンテスト種別を特定してから正規問題IDを照合し、AIなしで参加したい利用者には`contest_mode`を案内する。

**留保:** 「許可・注意表示」は、ADTの公開ruleと一般の生成AI ruleを組み合わせたAlgoLoomのプロジェクト判断であり、AtCoderの許諾・公認ではない。ADTまたは生成AI ruleの変更、contest種別・適用ruleの確認不能時はfail closedで拒否へ戻す。

**決めたこと:** 現在の公開情報に基づき、ADTで再利用中の過去問は「許可・注意表示」とする。AIなしで練習したい利用者には`contest_mode`を案内する。

**判断記録:** [AtCoder公開情報に基づく配布判断記録 §2 項目7](atcoder-public-policy-review.md#2-12項目の再整理)

出典: [AIレビュー安全設計 §3.3](../features/ai-review-safety-design.md#33-adtをabc系と同一扱いしない理由)、[配布方針ガイド §15](../operations/algoloom-distribution.md#15-公開情報から整理した12項目)

### 5.3 AWC Beta・個別コンテストルールへの対応

**状態:** 一部決定済み

**決定済みの内容:** AWC Betaは対象回の問題ID、Beta種別、AI利用の明示許可、有効なcache期限をすべて確認できた場合だけ注意表示付きで許可し、それ以外はfail closedとする。対象回の明示ルールは毎回確認する。

**残る未決:** AHCを含む個別コンテストルールを、rule種別・version・適用期間とともにどの更新・version管理方法で実装するか。

**決めること:** AWC Betaの対象回で明示許可をどう確認・期限管理するか、個別コンテストルールをどのプロファイル・更新手順で扱うか。

**原文:** 「AWC Beta用プロファイルは対象回の明示許可を毎回確認する。」「個別コンテストルールへの対応方法を追加する。」

出典: [AIレビュー安全設計 §3.4](../features/ai-review-safety-design.md#34-awc-betaは対象回の明示ルールを確認する)、[AIレビュー安全設計 §12](../features/ai-review-safety-design.md#12-段階的な実装)

## 6. Repair Lab

### 6.1 機能の名称・操作・評価方法

**状態:** 未決

**決定済みの境界:** `Repair Lab`は仮称であり、AtCoder Coreとは別の恒常modeや専用command体系を作らず、patch速度や想定解との文字列一致だけで評価しない。

**残る未決:** 正式名称、既存commandへ統合する具体的操作、画面、採点方式。

**決めること:** Repair Labの正式名称、command、画面、採点方式。

**原文:** 「本書ではこの構想を仮に**Repair Lab**と呼ぶ。名称、command、画面、採点方式は確定事項ではない。」

出典: [Repair Lab 将来構想 §0](../future/repair-lab-future-design.md#0-結論)

### 6.2 学習価値と共通UXへの統合

**状態:** 条件付き決定

**決定済みの内容:** 正式設計へ進む前に、手作り教材で仮説・根拠・予測・検証の学習価値を確認する。複数の正しいpatchを受け入れ、LLMを唯一の正しさの根拠にしない。AtCoder Coreと同じ基本導線へ統合できない場合は、AlgoLoomへ追加せず別applicationとして扱う。

**残る判断:** 小規模検証で学習価値と共通UXへの統合可能性を確認できた後に正式機能化するか。

**決めること:** 仮説を先に記録する手順の学習価値、複数の正しいpatchを受け入れる判定方法、共通UXへ統合できるか、統合不能時に別applicationとするか。

**原文:** 「commandや画面を確定する前に、仮説を先に記録する手順が実際の学習に役立つか確認する。」

**原文:** 「複数の正しいpatchを受け入れる判定方法を検証する。」「共通UXを保てない場合は、AlgoLoomへ無理に追加せず別applicationとして扱う。」

出典: [Repair Lab 将来構想 §8](../future/repair-lab-future-design.md#8-段階的な実装判断)、[Repair Lab 将来構想 §10](../future/repair-lab-future-design.md#10-実装開始の判断条件)

### 6.3 教材生成・個別化・外部コードの採用

**状態:** 条件付き決定

**決定済みの内容:** 初期検証では手作りの検証済み教材を優先する。mutationはtest oracleで期待どおりの失敗を確認できる場合、LLM生成codeは人間または決定的な検証工程を通した場合、第三者codeはlicense・帰属・privacy・再配布条件を確認できる場合だけ利用する。利用者投稿教材はmoderation等の設計後の候補とする。

**残る判断:** 各教材種別の検証結果を踏まえた採用範囲、傾向分析に必要な件数・精度の具体基準、利用者投稿機能を採用するか。

**決めること:** mutationとLLMを教材候補生成へ利用する範囲、傾向分析・個別練習候補に必要な件数と精度、利用者投稿教材のmoderation・license・削除手順。

**原文:** 「十分な件数と精度が得られた行動だけを、根拠付きの練習候補へ利用する。」

**原文:** 「利用者投稿教材 | moderation、license、悪意あるcode、品質保証、削除手順を設計した後の候補とする」

出典: [Repair Lab 将来構想 §4.1](../future/repair-lab-future-design.md#41-教材の段階的な採用)、[Repair Lab 将来構想 §8](../future/repair-lab-future-design.md#8-段階的な実装判断)

## 7. セキュリティ境界の拡張

### 7.1 未信頼code実行の隔離方式

**状態:** 条件付き決定

**決定済みの内容:** MVPは利用者自身のcodeだけを対象とし、完全sandboxは必須にしない。ただしcompile/run timeout、出力量上限、`HostPlatform`によるprocess tree終了、secretを除いた環境等はMVPで必須とする。他人・共有・Cloud取得・LLM生成codeを実行対象にする場合は、OS sandbox、containerまたはVMによる隔離とresource制限を必須候補として再評価する。

**残る判断:** 未信頼code実行を採用するときの具体的なsandbox方式、threat model、対応OSと保証範囲。

**決めること:** 他人・共有・Cloud取得・LLM生成のcodeを実行対象に加える場合のOS sandbox、container、VM等の方式と、filesystem/network/process/CPU/memory等の制限。

**原文:** 「次の機能を追加する場合は、OS sandbox、container、VM等による隔離を必須候補として再評価する。」

**原文:** 「具体的なsandbox方式とthreat modelは、実装判断時に[セキュリティ設計ガイド](../quality/security-design.md)へ追加する。」

出典: [セキュリティ設計ガイド §6.9](../quality/security-design.md#69-testによる任意コード実行とresource制限)、[Repair Lab 将来構想 §7](../future/repair-lab-future-design.md#7-安全性)

### 7.2 Web UI・複数ユーザー化の安全設計

**状態:** 条件付き決定

**決定済みの内容:** Web UIはMVP対象外である。追加時に認証・認可とownership検証、保存型XSS対策、CSP/security header、request size、rate limit、session管理、複数ユーザー向け監査logを必須とすることは決定している。

**残る判断:** Web UI自体を採用するか、および採用時の具体的な認証基盤、policy、制限値、実装方式。

**決めること:** Web UIを追加する場合の認証・認可、ownership検証、XSS/CSP/security header、rate limit、session、監査ログ。

**原文:** 「Phase 3: 外部コード実行・Web UI追加時に必須」

出典: [セキュリティ設計ガイド §8](../quality/security-design.md#8-段階的な実装)

### 7.3 秘密情報保管庫が保証する範囲と表示文言

**状態:** 未決

**決定済みの内容:** AtCoderセッションはOSの秘密情報保管庫（macOS Keychain、WindowsのOS保護領域、Linux Secret Service）へAlgoLoom名前空間で保存し、利用できない場合も平文ファイルへ切り替えず提出だけを停止する。「セッションを平文ファイルとして残さない」「OSのロック解除状態に従う」「削除対象と削除方法が一意に定まる」を保証し、「AlgoLoomだけが読める」とは表示しない。

**残る判断:** 3つのOSそれぞれで実際に成立する保護範囲と、利用者へ示す表示文言。

**決めること:** macOS Keychainの項目のアクセス制御は呼び出し元の実行ファイルへ紐づくが、[AtCoder認証設計 §3.6](../architecture/atcoder-authentication.md#36-配布物と実行ファイル署名の境界)の契約により認証ヘルパーはPythonの実行環境の上で動作するため、信頼される実行ファイルはインタプリタになる。専用の実行ファイルを用意しても、ad-hoc署名では再ビルドのたびに識別子が変わり、更新のたびにアクセス制御が失効する。3つのOSで「同じ利用者として動作する他のプロセスから読めるか」「更新時に再認可を求められるか」を確認し、過大でない表示文言を確定する。

**原文:** 「秘密情報保管庫が保証する範囲を次のとおり限定し、これを超える表示を行いません。」

出典: [AtCoder認証設計 §4.1](../architecture/atcoder-authentication.md#41-保管先)、[`TD-12`](../../TODO.md#td-12-3つのosの認証検証マトリクスを作る)

### 7.4 Chrome拡張機能の版と可用性を制御できないことへの対処

**状態:** 未決

**決定済みの内容:** 方式Aは認証用Chrome拡張機能とAlgoLoom本体の2つで成立し、拡張機能はChrome Web Storeで配布する。Firefox + AMO自己配布は、利用者へ普段使わないブラウザの新規インストールを求めるため採用しない。拡張機能のソースはWebExtensions共通仕様の範囲で書き、Chrome固有APIへ依存させない。2026年8月26日に、拡張機能へはブラウザの中でしか実現できない能力だけを置くこと（緩和策H）と、拡張機能が使えない場合の縮退運転を持つこと（緩和策D）を決定した。

**残る判断:** 採用した2つの緩和策の実装粒度と、版の固定・交渉の分離、配布停止手順。

**決めること:** 要素指定の受け渡し形式と拡張機能側の検証規則、固定する対象（拡張機能識別子、権限セット、配布元）と交渉する対象（通信protocol版）の分離、後方互換の責任の所在、縮退運転を開始するCLI導線と表示文言、利用者が残っている状態での配布停止手順。

**原文:** 「最も頻繁に更新が必要な部品が、最も制御できない経路に乗っています。」

出典: [Chrome拡張機能配布の運用リスクと緩和策](chrome-extension-distribution-risks.md)、[拡張機能の責任境界と提出の縮退運転設計](../features/extension-boundary-and-degraded-submission.md)、[`TD-38`](../../TODO.md#td-38-認証配布物とテンプレートのライフサイクル契約を確定する)

## 8. 配布・AtCoder公式情報の再確認

### 8.1 公開情報に基づく配布判断と再確認gate

**状態:** 条件付き決定

**決定済みの内容:** 2026年8月24日に12項目を公式公開情報で再調査し、通常のサポート問い合わせは送らず、条件付きの配布判断と再確認gateへ置き換えた。AtCoderは非公式toolをサポートせず問い合わせにも回答しないと公式告知している。この告知も、技術検証の成立も、AlgoLoomへの許諾とは扱わない。項目ごとの判断、制約、再確認条件は[AtCoder公開情報に基づく配布判断記録](atcoder-public-policy-review.md)を正とする。

**残る確認:** 実装開始前、限定公開Beta前、各release前に、規約、非公式tool告知、AI rule、ADT、logo guide、`robots.txt`、認証・提出互換性を再確認する。方式Aの製品形態と3 OSの成立性は別TODOで検証する。

**決めたこと:** 12項目は「公式事実」「検証事実」「AlgoLoomの判断」を分けて設計へ反映する。一般的な正確な頻度等が公式情報から得られない項目は、有限通信、自動再試行なし、人による最終操作、fail closedという保守的な既定値を置き、実測で調整する。

#### 配布判断に用いた検証事実

##### AtCoderへの要求数

次の表は、各実行記録で検証支援コードがAtCoderへ直接発行したと確定できる、方式別の要求数である。可視ブラウザー内の副資源、リダイレクト、ログイン、Turnstile、フォーム送信に伴う内部通信は、Cookieや送信本文を受け取らない安全境界を優先したため総数を観測しておらず、表へ推測で加えていない。`p0-01`と`p0-02`も内部リダイレクト数が不明なため、合計は通信路上の正確な交換数ではなく、方式を確定できたアプリケーション単位の要求数である。

| 実行記録 | GET | POST | 実行ごとの内訳・補足 |
|---|---:|---:|---|
| [`p0-01`](../verification/judge-adapter/results/2026-08-11-p0-01.md) | 5 | 1 | 認証確認2、公開サンプル取得1、ログインGET 1・POST 1、自動アクセス対策診断1。内部リダイレクト数は不明 |
| [`p0-02`](../verification/judge-adapter/results/2026-08-11-p0-02.md) | 5 | 1 | コマンドラインログインGET 3・POST 1、認証確認1、自動アクセス対策診断1。内部の交換数は不明 |
| [`p0-03`](../verification/judge-adapter/results/2026-08-11-p0-03.md) | 2 | 0 | 空セッションと方式Cで認証確認を各1回 |
| [`p0-04`](../verification/judge-adapter/results/2026-08-11-p0-04.md) | 2 | 0 | 空セッションと方式Cで認証確認を各1回 |
| [`p0-05`](../verification/judge-adapter/results/2026-08-11-p0-05.md) | 3 | 0 | 空セッション、方式C、提出ページを各1回 |
| [`p0-06`](../verification/judge-adapter/results/2026-08-11-p0-06.md) | 3 | 0 | 空セッション、方式C、提出ページを各1回 |
| [`p0-07`](../verification/judge-adapter/results/2026-08-12-p0-07.md) | 3 | 0 | 空セッション、方式C、提出ページを各1回 |
| [`p0-08`](../verification/judge-adapter/results/2026-08-12-p0-08.md) | 0 | 0 | 支援コードからの直接要求なし。ブラウザーで対象提出URLへ1回移動し、内部通信数は不明 |
| [`p0-09`](../verification/judge-adapter/results/2026-08-12-p0-09.md) | 0 | 0 | 支援コードからの直接要求なし。ブラウザーでログインURLへ1回移動し、内部通信数は不明 |
| [`p0-10`](../verification/judge-adapter/results/2026-08-12-p0-10.md) | 0 | 0 | AtCoderへ接続せず、ローカル診断とCloudflareの対照確認だけを実施 |
| [`p0-11`](../verification/judge-adapter/results/2026-08-12-p0-11.md) | 0 | 0 | ローカル準備で停止し、AtCoderへ未到達 |
| [`p0-12`](../verification/judge-adapter/results/2026-08-12-p0-12.md) | 0 | 0 | 支援コードからの直接要求なし。ブラウザーで設定、提出、提出前一覧を各1ページ観測し、内部通信数は不明 |
| [`p0-13`](../verification/judge-adapter/results/2026-08-12-p0-13.md) | 0 | 0 | ローカル準備で停止し、AtCoderへ未到達 |
| [`p0-14`](../verification/judge-adapter/results/2026-08-12-p0-14.md) | 0 | 0 | 支援コードからの直接要求なし。POST方式フォームの送信イベント1回はHTTP到達不明 |
| [`p0-15`](../verification/judge-adapter/results/2026-08-12-p0-15.md) | 0 | 0 | 公開情報の確認とローカルテストだけを実施 |
| [`p0-16`](../verification/judge-adapter/results/2026-08-12-p0-16.md) | 0 | 0 | 支援コードからの直接要求なし。3回目のブラウザー操作でPOST方式フォームから提出1件の受理を確認したが、内部通信数は不明 |
| [`p0-17`](../verification/judge-adapter/results/2026-08-12-p0-17.md) | 4 | 0 | 2実行で本人確認と対象提出の判定確認を各2回 |
| [`p0-18`](../verification/judge-adapter/results/2026-08-12-p0-18.md) | 0 | 0 | ローカルテストだけを実施 |
| [`p0-19`](../verification/judge-adapter/results/2026-08-12-p0-19.md) | 2 | 0 | 本人確認1、対象提出の判定確認1 |
| [`p0-20`](../verification/judge-adapter/results/2026-08-12-p0-20.md) | 0 | 0 | ローカルテストだけを実施 |
| [`p0-21`](../verification/judge-adapter/results/2026-08-12-p0-21.md) | 0 | 0 | 方針変更とローカルテストだけを実施 |
| [`p0-22`](../verification/judge-adapter/results/2026-08-12-p0-22.md) | 7 | 0 | 本人確認1、対象提出の判定確認6。別枠のPOST方式フォーム送信イベント1回で提出1件の受理を確認 |
| [`p0-23`](../verification/judge-adapter/results/2026-08-12-p0-23.md) | 7 | 0 | 5実行で結果記録から確認できる本人確認の累計。1件は開始有無を確定できず、推測加算していない |
| [`p1-01`](../verification/judge-adapter/results/2026-08-13-p1-01.md) | 2 | 0 | 本人確認1、同じ提出IDの判定再照合1 |
| [`p1-02`](../verification/judge-adapter/results/2026-08-13-p1-02.md) | 1 | 0 | Cookieなしの未認証対照1 |
| [`p1-03`](../verification/judge-adapter/results/2026-08-13-p1-03.md) | 3 | 0 | 本人確認とCookie更新確認を3回 |
| [`p2-01`](../verification/judge-adapter/results/2026-08-13-p2-01.md) | 8 | 0 | 4実行で本人確認と、固定条件を付けた本人提出一覧の先頭1ページ取得を各4回 |
| **方式を確定できた合計** | **57** | **2** | POST 2回はいずれもログイン試行。ブラウザー内部の通信、外部条件確認、AtCoder以外への通信を含めない |

##### 間隔、待機指示、再試行

| 観測項目 | 配布判断に用いる事実 | 主な出典 |
|---|---|---|
| 通常の最小間隔 | 直接要求を制御できる実行では、原則として前の応答完了から次の要求開始まで2秒以上を空けた。実測は2,000〜2,006ミリ秒を中心とし、`p0-23`の合格実行では2,340ミリ秒だった | `p0-03`〜`p0-07`、`p0-17`、`p0-19`、`p0-23`、`p1-01`、`p1-03`、`p2-01` |
| サーバーの待機指示 | `p0-22`の判定確認では、サーバー指定の1,000ミリ秒より長い最小2,000ミリ秒を選び、5回の待機へ適用した | [`p0-22`](../verification/judge-adapter/results/2026-08-12-p0-22.md) |
| 間隔を確定できない範囲 | `p0-02`のCLI内部と可視ブラウザー内部の通信は未計測。推測値を置かない | [`p0-02`](../verification/judge-adapter/results/2026-08-11-p0-02.md)、`p0-08`以後の可視ブラウザー記録 |
| 再試行 | 検証支援コードの自動再試行は全実行で0回。外部情報確認用の別基盤で`p0-01`が429応答を1件観測した際も再試行しなかった | [`p0-01`](../verification/judge-adapter/results/2026-08-11-p0-01.md)ほか全実行記録 |

##### 提出件数

| 区分 | 件数 | 内容 |
|---|---:|---|
| AtCoderによる受理を確認できた提出 | **2件** | `p0-16`と`p0-22`で各1件。いずれも終了済み過去問、本人アカウント、事前に対象と上限を明示した操作 |
| 送信状態不明 | 1回 | `p0-14`のブラウザー送信イベント。HTTP到達と受理を確認できず、提出件数2件には含めない |
| 自動再送 | **0回** | 送信後の結果が不明な場合も再送しなかった |
| 同一承認内の再提出 | **0回** | 送信開始相当へ到達した時点で承認枠を消費し、同じ承認を再利用しなかった |

##### AtCoder以外への通信

| 区分 | 集計上の扱い |
|---|---|
| パッケージと公開情報の確認 | PyPI、GitHub、AtCoderInfo、公式文書等の確認は、検証支援コードと別の実行基盤で行った。各実行記録で送信先を分離し、上記57回・2回へ含めていない。全記録を通じた正確な要求数は記録がないため推測しない |
| Cloudflare公式互換性診断 | 可視ブラウザーで人が確認した通信としてAtCoder通信から分離した。ブラウザー内部の総数は取得していない |
| ローカル通信と固定入力 | `127.0.0.1`の認証付き通信、固定入力、ローカルテストはAtCoderへの通信へ含めない |

##### 保存する内容と保存しない内容

[技術検証の実施手順 §5](../verification/judge-adapter/README.md#5-観測と保存の境界)の許可リストに従い、配布判断では次の保存境界を維持する。

| 保存する内容 | 保存しない内容 |
|---|---|
| UTCの取得時刻と単調増加時計で測った処理時間 | Cookie、パスワード、セッショントークン、CSRFトークン、認証用URL |
| 操作の入口、終了コード、成功・失敗・状態不明の分類 | `Authorization`、`Cookie`、`Set-Cookie`を含む生のHTTPヘッダー |
| 許可リストへ正規化した項目名、型、単位、欠損 | 問題文、生のHTML、完全なJSON応答、入出力例の実値 |
| HTTP方式、送信先の分類、応答状態の分類、要求数、待機時間 | 提出用ソースコード、標準入力、標準出力、標準エラー出力 |
| 入出力例の件数と、入力・期待出力の対応関係を確認できた事実 | 実際のアカウント名、メールアドレス、ホスト名、利用者名、不要な絶対パス |
| 言語候補数と、一意に解決した識別子、表示名、処理系、版 | 外部ツールの生ログ、例外の完全な呼出履歴 |
| 判定待ちと最終判定の取得時刻付き観測 | 実際の提出ID、ブラウザープロファイル、他のCookie、他利用者のコード |

検証終了時に一時Cookie、専用ブラウザープロファイル、提出用ソース、匿名化前の一時結果、検証用の秘密情報保管庫項目を後始末した。ローカルの項目削除をAtCoder側のセッション失効とは扱っていない。

##### 方式Aの認証手順

方式Aでは、既存プロファイルと分離した空の可視専用ブラウザーを起動し、利用者本人がログインとTurnstileを手動操作する。CDP、WebDriver、遠隔デバッグ、ヘッドレス化、検知隠蔽、User-Agent上書きは使用しない。ログイン後に`https://atcoder.jp`、パス`/`の`REVEL_SESSION`だけを取得し、本人アカウントとの一致を確認してから、AlgoLoom名前空間のOS秘密情報保管庫へ保存する。新規プロセスでも同じ本人を再照合し、終了時は専用ブラウザー、子プロセス、一時プロファイル、一時実行ファイル、ローカル通信、秘密情報保管庫の検証項目を後始末する。既存のブラウザープロファイル、他のCookie、パスワードは参照または保存しない。この原理はmacOSとChromeで成立したが、製品として正式対応するブラウザーと配布形態は未決である。製品の提出補助では可視browser上の最後の提出操作も利用者へ委ね、成立しない場合にsessionを使った直接HTTP POSTへfallbackしない。

##### 12項目との対応

| 項目 | 判断に用いた事実 |
|---:|---|
| 1 | `p0-01`では終了済み過去問1問の公開サンプル3組をGET 1回で取得し、対応関係だけを確認した。実値は成果物へ保存していない |
| 2、12 | 上記の方式Aで、利用者の手動ログイン、`REVEL_SESSION`限定取得、OS秘密情報保管庫、新規プロセスでの本人再照合、後始末が成立した |
| 3 | 直接制御できる要求は原則2秒以上空け、サーバー指定がある場合はより長い間隔を採用し、自動再試行を0回とした |
| 4 | 薄いHTTPクライアントは検証用クライアントと分かる固定User-Agentを送り、可視ブラウザーではUser-Agentを変更しなかった。製品はRFC 9110に沿う最小識別子と公開support URLを使う |
| 5 | 公開サンプルの実値は検証成果物へ保存せず、リポジトリ外の一時データも成果物確定後に破棄した。製品でも利用者が選んだ問題単位の作業用cacheに限定する |
| 10 | 検証では解説本文を取得・保存していない。製品も正規問題IDから公式URLを構成し、本文を取得せず既定ブラウザーへ渡す境界である |
| 11 | 他利用者のコード、アカウント名、提出IDは取得・保存していない。`p2-01`で取得したのは本人の提出一覧を条件で絞った先頭1ページだけで、他利用者の一覧表示機能とは別である |

**再調査した12項目:**

1. 「ユーザー操作により、公開サンプル入出力を1問ずつローカル保存するCLIの配布可否」
2. 「ユーザー自身がbrowserでloginしたsessionを、同一端末のlocal CLIがOSのsecret storeへ保存し、明示操作による提出補助へ使う方式の可否」
3. 「推奨されるアクセス間隔と再試行方法」
4. 「User-Agentへ記載すべき情報」
5. 「公開サンプルのローカルキャッシュ可否」
6. 「AIレビュー要求時に開催状況・種別・正規問題IDを照合し、ABC・ARC・AGCの開催中問題だけを拒否する設計で問題ないか」
7. 「ADTで再利用中の過去問をAIレビュー対象とする解釈で問題ないか」
8. 「READMEやPyPIで『AtCoder対応』と文章表記する場合の注意点」
9. 「無料OSS版と有料版で確認事項が異なるか」
10. 「正規問題IDからAtCoderの問題別解説ページを構成し、本文を取得せずdefault browserで開く機能の可否」
11. 「MVP後に、問題・AC・言語filter付きのAtCoder提出一覧を、code本文・Cookieを取得せずdefault browserで開く機能の可否」
12. 「可視の専用browserで利用者がTurnstileを手動操作し、CLIが`REVEL_SESSION`だけを取り込む方式で遵守すべき追加条件の有無」

判断結果: [AtCoder公開情報に基づく配布判断記録 §2](atcoder-public-policy-review.md#2-12項目の再整理)

### 8.2 商用・法人向け配布の可否と条件

**状態:** 条件付き決定

**決定済みの内容:** 現在は個人利用から無料OSS公開までを先に検証し、商用・法人向けはOSS公開とは別判断とする。採用試験・coding試験・査定・採用活動へのAtCoder過去問利用をサポートしない。進む場合は事業形態を具体化し、必要な公式の事業窓口があれば利用し、専門家への相談、利用規約・privacy policy・support範囲の整備を行う。

**残る判断:** OSS公開後に商用・法人向けへ進むか。

**決めること:** OSS公開後に商用・法人向け配布へ進むか。進む場合の公式な事業上の確認経路、専門家への相談、利用規約、privacy policy、support範囲、採用・査定用途の禁止。

**現在の方針:** 「OSS公開とは別の判断を行う。」「採用試験・査定用途を禁止する。」「事業形態に対応する公式窓口・partner制度があれば確認し、通常の非公式tool support問い合わせとは分ける。」

出典: [配布方針ガイド §16](../operations/algoloom-distribution.md#16-段階的な配布計画)

## 確認時の注意

- この一覧にない将来機能でも、MVPの範囲を変更する場合は [MVPスコープ §7](../product/mvp.md#7-変更管理) の変更管理に従う。
- 外部サービスの規約、SDK、料金、配布wheelは変わり得る。各設計文書にあるとおり、実装開始時・リリース前に公式情報を再確認する。
