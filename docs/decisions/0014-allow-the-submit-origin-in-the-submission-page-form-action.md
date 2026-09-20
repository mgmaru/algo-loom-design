# ADR-0014 提出確認画面のCSPの`form-action`へ提出先originを含める

| 項目 | 内容 |
|---|---|
| 状態 | 採用 |
| 日付 | 2026年9月21日 |
| 関連TODO | [`TD-11`](../../TODO.md#td-11-方式a製品形態を実サービスで検証する)、[`TD-53`](../../TODO.md#td-53-提出確認画面から提出pageへの受け渡しが成立しない原因を特定する)、[`TD-54`](../../TODO.md#td-54-helperの成功報告が到達していないことを覆い隠さないようにする) |
| 正本 | [`JudgeAdapter`技術検証計画 §3.1.1](../project/judge-adapter-verification.md#311-v-12-方式a製品形態の検証) |

## 背景と前提

[ADR-0013](0013-find-the-cause-before-fixing-the-v12e-handoff.md)で、`V-12E`の受け渡しは**原因を特定してから直す**と決めました。2026年9月21日に特定できたため、その結果と直し方を記録します。

**原因は、提出確認画面へ付けている`Content-Security-Policy`の`form-action`でした。** helperはこの画面のCSPで`form-action`を自分のloopback originだけに限っていました。**Chromeは、form POSTのあとの遷移先も`form-action`で検査します。** そのため、AtCoderの提出pageへの303がChromeに止められ、browserは提出pageへ着きませんでした。

判断の土台にした事実です。

| # | 前提 | 根拠 |
|---|---|---|
| 1 | 提出確認画面は`default-src 'none'; style-src 'unsafe-inline'; form-action <loopback origin>; frame-ancestors 'none'`で配信していた | `helper/protocol.go`の`serveSubmission` |
| 2 | **`form-action`にloopback originだけを許すと、別originへの303遷移でbrowserは提出pageへ着かない。** 遷移先も許すと着く | 2026年9月21日に実測（[§実測](#実測)） |
| 3 | **原因は待受を閉じる時刻ではない。** 応答を書いた後に待受を閉じるのを5秒遅らせても、結果は変わらない | 同じ実測の切り分けcase |
| 4 | ADR-0013の前提4のとおり、通知と応答の順序も原因ではない。Goの`http.Server.Shutdown`は処理中のrequestを待つ | 2026年9月21日の再現（4条件×20回） |
| 5 | この経路の契約testは`httptest`でhandlerを直接呼ぶため、**browserが応答を受け取って遷移するところ**を通っていなかった | `helper/helper_test.go` |
| 6 | `form-action`は、押した先を限る指定である。`'none'`や広すぎる指定にすると、「この画面のformはどこへ送られるか」という検査そのものが意味を失う | CSPの仕様と、[ADR-0012](0012-serve-submission-page-with-same-origin-referrer-policy.md)決定1の考え方 |
| 7 | 提出先URLは実行時の入力（`v12e-submission/input.json`）から決まり、AtCoderのURLであることを`validateAtCoderURL`が検査している | `helper/submission.go`、`helper/protocol.go` |
| 8 | helperのbuild hashが変われば`V-12A`〜`V-12E`のすべてが無効になる | [ADR-0007](0007-make-helper-build-hash-track-behaviour.md)、検証計画の無効化表 |

### 実測

2026年9月21日、macOS 26.5・Chrome 153.0.8010.48で、[受け渡しのprobe](../../scripts/verification/atcoder_v12/submission-handoff-probe.mjs)を使って測りました。**AtCoderにもChrome Web Storeにも接続していません。** 提出pageの代わりに別portのローカル待受を置いています。portが違えばoriginも違うため、AtCoderへの遷移と同じ「別originへの303」になります。**画面は実物の`submission.html`とhelperと同じヘッダーを使い、人の指の代わりにscriptだけを足しています。**

| 条件 | 提出pageへ到達したか |
|---|---|
| `form-action`がloopback originだけ（5回目のcampaignの状態） | **着かない** |
| `form-action`へ遷移先originも許す | **着く** |
| CSPを付けない | 着く |
| 応答を書いた後、待受を閉じるのを5秒遅らせる | **着かない**（修正前の設定のため） |
| 応答を書かずに接続を切る（negative control） | 着かない |

**差がついたのは`form-action`だけです。** 待受を閉じる時刻を変えても結果は動きませんでした。

## 決定

1. **提出確認画面のCSPの`form-action`へ、loopback originと提出先originの2つだけを許します**（[`TD-53`](../../TODO.md#td-53-提出確認画面から提出pageへの受け渡しが成立しない原因を特定する)）。提出先originは実行時の提出先URLから作ります
2. **CSPの他の指定は変えません。** `default-src 'none'`、`style-src 'unsafe-inline'`、`frame-ancestors 'none'`はそのままにし、`script-src`を足しません
3. **契約testを、`form-action`が2つのoriginだけを許していることを検査する形にします。** 広すぎる指定（`*`）と、遷移先を欠いた指定の両方で落ちることを確かめます
4. **受け渡しのprobeを検証物へ置きます。** negative controlとして「`form-action`をloopback originだけにした状態」を必ず測り、**壊れた条件を検出できることを示してから**成立と見なします
5. **helperの結果JSONは、到達を`true`で埋めません**（[`TD-54`](../../TODO.md#td-54-helperの成功報告が到達していないことを覆い隠さないようにする)、[ADR-0013](0013-find-the-cause-before-fixing-the-v12e-handoff.md)決定3）。`observed_scope`で観測の範囲を述べ、`submit_page_arrival`は`unobserved_by_helper`にします

## 検討して採らなかった案

| 案 | 採らなかった理由 |
|---|---|
| CSPの`form-action`を外す（指定しない） | 実測では着くが、**この画面のformがどこへ送られるかの検査を失う。** 前提6のとおり、`form-action`は受け渡しの経路を限るための指定である |
| `form-action`へ`*`を許す | 外すのと同じ帰結。**許す先を限っていないため、指定が残っているように見えて何も守らない** |
| CSPそのものを付けない | `default-src 'none'`と`frame-ancestors 'none'`まで失う。原因と関係のない部分を弱める理由がない |
| 提出確認画面にJavaScriptを入れ、`location.assign`で遷移する | [ADR-0011](0011-add-submit-entry-before-rerunning-v12.md)が画面をJavaScriptなしに設計した理由が残っている。加えて`script-src`を許すことになり、**原因と関係のない緩めを増やす** |
| helperが303ではなく200で「このリンクを押してください」と出す | 手動のページ探索とcopy-and-pasteを要求しない、という`V-12E`の条件に近づかない。一往復で進めるかどうかが確かめたい性質である |
| 待受を閉じるのを遅らせる | 前提3の実測で、結果が変わらないことを確かめた。**症状に効かない** |

## 理由

2つあります。

第一に、**`form-action`は受け渡しの経路そのものを表しています。** この画面のformが送られてよい先は、押下を受けるhelper自身と、その後に開く提出先の2つだけです。**2つを書けば、指定は実際の経路と一致し、意味を保ったまま通ります。** 外す・`*`にするという直し方は、通すために検査を捨てることになります。[ADR-0012](0012-serve-submission-page-with-same-origin-referrer-policy.md)で`Origin: null`を受け付けないと決めたのと同じ考え方です。

第二に、**この不具合は「browserが応答を受け取って遷移するところ」を一度も通していなかったために起きました**（前提5）。[ADR-0005](0005-verify-consent-flow-in-browser-semantics.md)、[ADR-0012](0012-serve-submission-page-with-same-origin-referrer-policy.md)に続いて3度目の同じ型です。そのため決定3と4で、**固定入力testと実browserのprobeの両方**を置きます。testは`form-action`の中身を、probeは実際に着くかどうかを見ます。

## 影響と再評価条件

- **helperのbuild hashが変わるため、5回目のcampaignの合格は取り消されます**（前提8）。[ADR-0013](0013-find-the-cause-before-fixing-the-v12e-handoff.md)決定5のとおりです
- **人の操作が要る`V-12B → V-12D`を、6回目でもう一度15分やり直します**
- **Chrome Web Storeの再審査は発生しません。** 拡張機能のsourceを変えないためです
- **製品実装がloopback画面からform POSTで別originへ遷移する形を採る場合、同じ制約が当てはまります。** `Referrer-Policy`（[ADR-0012](0012-serve-submission-page-with-same-origin-referrer-policy.md)）と合わせて[`TD-38`](../../TODO.md#td-38-認証配布物とテンプレートのライフサイクル契約を確定する)へ渡します

次のいずれかが起きたら見直します。

- **6回目のcampaignで、probeが通るのに実機で着かないとき。** probeが実機と違う条件を測っていることになるため、probeの条件（画面、ヘッダー、profile、拡張機能の有無）を実機へ寄せます
- **`form-action`の扱いがbrowserの版で変わったとき。** 対応版の範囲を[`TD-31`](../../TODO.md#td-31-実行環境の組み合わせを固定する)へ回します
- **提出先originが実行時に複数になりうる設計へ変わったとき。** 許す先の作り方を決め直します
