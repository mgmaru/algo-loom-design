# ADR-0012 提出確認画面の`Referrer-Policy`を`same-origin`へ変え、5回目のcampaignでやり直す

| 項目 | 内容 |
|---|---|
| 状態 | 採用 |
| 日付 | 2026年9月20日 |
| 関連TODO | [`TD-11`](../../TODO.md#td-11-方式a製品形態を実サービスで検証する)、[`TD-51`](../../TODO.md#td-51-提出確認画面のform-postがbrowserで拒否される問題を直す)、[`TD-52`](../../TODO.md#td-52-browser由来のrequestを手で組み立てている契約testを洗い出す) |
| 正本 | [`JudgeAdapter`技術検証計画 §3.1.1](../project/judge-adapter-verification.md#311-v-12-方式a製品形態の検証) |

## 背景と前提

2026年9月20日、4回目のcampaign `v12-2026-09-20-03`で`V-12A`・`V-12B`・`V-12C`の必須case・`V-12D`が合格し、`V-12E`へ進みました。再認証は成立し、**提出確認画面まで到達しました。** そこで利用者が「AtCoderの提出画面へ進む」を押したところ、helperが`authentication_rejected`を返して停止しました。

提出確認画面は[ADR-0011](0011-add-submit-entry-before-rerunning-v12.md)決定2により、拡張機能を変えずにhelper側で実装したものです。**JavaScriptを持たず、押されたことを同じoriginへのform POSTで伝えます。**

原因を実測で切り分けました。判断の土台にした事実です。

| # | 前提 | 根拠 |
|---|---|---|
| 1 | helperは、POST `/submission/proceed`の`Origin`ヘッダーが自分のoriginと一致しなければ`authentication_rejected`を返す | `helper/protocol.go`の`serveSubmission` |
| 2 | 提出確認画面は`Referrer-Policy: no-referrer`付きで配信している | 同上 |
| 3 | **`Referrer-Policy: no-referrer`のページからの同一originへのform POSTで、Chromeは`Origin: null`を送る** | 2026年9月20日に実測（[§実測](#実測)） |
| 4 | 同意画面も`no-referrer`だが影響を受けない。POSTを送るのが**拡張機能のcontent scriptの`fetch`**であり、`Origin`が`chrome-extension://<固定ID>`になるため | `helper/protocol.go`の`serveBootstrap`、`extension/bootstrap.js`。`V-12D`が4回目も合格している |
| 5 | この経路の契約testは`request.Header.Set("Origin", origin)`と**手でヘッダーを立てている。** browserが実際に何を送るかを一度も通っていない | `helper/helper_test.go` |
| 6 | helperのbuild hashが変われば`V-12A`〜`V-12E`のすべてが無効になり、新しいcampaignが必要になる | [ADR-0007](0007-make-helper-build-hash-track-behaviour.md)、検証計画の無効化表 |
| 7 | 拡張機能のsourceを変えると、Chrome Web Storeへの再提出と審査が発生する | [ADR-0008](0008-scope-of-execution-based-checks-for-verification-artifacts.md)の前提7 |
| 8 | `no-referrer`を置いた目的は、303遷移でloopbackのURLがAtCoderへ漏れるのを防ぐことである | 提出確認画面はAtCoderの提出pageへ303で送る唯一の画面である |

### 実測

2026年9月20日、macOS 26.5・Chrome 153.0.8010.48で、リポジトリ外の使い捨てprofileとローカルの観測用サーバーだけを使って測りました。**AtCoderにもChrome Web Storeにも接続していません。** 別originの受け側にはもう一つのローカルportを使いました。

| ページの`Referrer-Policy` | 同一originへのform POSTの`Origin` | 別originへの303遷移の`Referer` |
|---|---|---|
| `no-referrer`（現行） | **`null`** | 送られない |
| `same-origin` | **正しいorigin** | **送られない** |
| `strict-origin-when-cross-origin` | 正しいorigin | origin部分が送られる |
| `origin` | 正しいorigin | origin部分が送られる |

**`same-origin`だけが、同一originの`Origin`を正しく送りながら、別originへは何も送りません。** 前提8の目的を保ったまま前提3の問題が消えます。

## 決定

1. **提出確認画面の`Referrer-Policy`を`no-referrer`から`same-origin`へ変えます**（[`TD-51`](../../TODO.md#td-51-提出確認画面のform-postがbrowserで拒否される問題を直す)）。`Origin`の検査は緩めません
2. **変更は提出確認画面の応答だけに閉じます。** 同意画面の`no-referrer`は変えません。前提4のとおり影響を受けておらず、変える理由がないためです
3. **browser由来のrequestを手で組み立てている契約testを洗い出し、実際に送られるヘッダーで確かめる形にします**（[`TD-52`](../../TODO.md#td-52-browser由来のrequestを手で組み立てている契約testを洗い出す)）。前提5の穴は、この1箇所だけとは限りません
4. **修正は、5回目のcampaignを始める前にまとめて行います。** 前提6により、個別に直すとそのたびにcampaignが無効になります
5. **campaign `v12-2026-09-20-03`は、この修正をもって無効になります。** 5回目で`V-12A`からやり直します。4回目の記録は消さずに残します

## 検討して採らなかった案

| 案 | 採らなかった理由 |
|---|---|
| helperが`Origin: null`も受け付ける | **検査が弱くなる。** `null`はsandboxed iframe、`data:`、redirectを経たrequestなど複数の出所から来るため、「自分が出した画面から来た」ことの根拠にならない。前提1の検査は、拡張機能のoriginではなく自分のoriginで確かめる点に意味がある |
| `Origin`の検査をやめ、一回限りの`proceed_token`だけで判定する | tokenは画面のHTMLに埋まっており、**画面が一度でも他所へ渡れば再現できる。** originの検査は「同じbrowserの、helperが出した画面から」という`V-12E`の観測対象そのものを支えている。確かめたい性質の担保を1本減らすことになる |
| `Referrer-Policy`を外す（何も設定しない） | 実測のとおり`Origin`は正しくなるが、**別originへ完全なURLの`Referer`が送られる。** 前提8の目的を失い、loopbackの待受番号をAtCoderへ渡すことになる |
| `strict-origin-when-cross-origin`を使う | `Origin`は正しくなるが、別originへorigin部分が送られる。`same-origin`なら何も送らずに同じ結果が得られるため、**弱いほうを選ぶ理由がない** |
| 提出確認画面にJavaScriptを持たせ、`fetch`でPOSTする | 画面をJavaScriptなしにしたのは[ADR-0011](0011-add-submit-entry-before-rerunning-v12.md)の設計である。`fetch`にすると同意画面と同じくCSPとtoken管理が要り、**確かめたい「押したら遷移する」という単純な経路に部品が増える。** 遷移も303ではなくJavaScriptの責務になる |
| 人が手でAtCoderの提出pageを開いて`V-12E`を合格とする | **停止条件そのものである**（[ADR-0011](0011-add-submit-entry-before-rerunning-v12.md)で同じ案を却下している） |
| `TD-51`だけ先に直し、`TD-52`の洗い出しを後回しにする | 前提6により、後から別の修正が出ればcampaignがもう一度無効になる。`TD-49`と`TD-50`をまとめた[ADR-0011](0011-add-submit-entry-before-rerunning-v12.md)決定3と同じ理由 |

## 理由

2つあります。

第一に、**検査を弱めて通すのは、確かめたい性質を捨てることになります。** `V-12E`が観測しているのは「同じbrowserで、helperが出した画面から、一往復で提出確認まで進めるか」です。`Origin`の検査はその「helperが出した画面から」を支えています。`null`を受け付ける、あるいは検査をやめるという直し方は、**画面が出て遷移したという結果だけを残し、それがどこから来たかを確かめない記録**を作ります。実測の結果、性質を落とさずに直せる値があるため、落とす理由がありません。

第二に、**これは[ADR-0005](0005-verify-consent-flow-in-browser-semantics.md)と同じ型の見落としです。** ADR-0005では、同意画面のcontent scriptがsourceの検査では正しく見えるのに実browserで動きませんでした。今回は、契約testが`Origin`ヘッダーを手で立てていたため、**browserが実際に送る値を一度も通していませんでした**（前提5）。同じ型が2度出た以上、この1箇所を直すだけでは足りません。だから決定3で洗い出しを別に起票します。

## 影響と再評価条件

- **4回目のcampaignの`V-12A`・`V-12B`・`V-12C`・`V-12D`の合格は、修正をもって取り消されます。** 記録は[`v12-03`](../verification/judge-adapter/results/2026-09-20-v12-03.md)に残します
- **人の操作が要る`V-12B → V-12D`を、もう一度15分やり直すことになります。** これで3回目です
- **Chrome Web Storeの再審査は発生しません。** 決定2により拡張機能のsourceを変えないためです
- **`V-12E`は不合格ではありません。** 再認証と提出確認画面までは成立しており、確かめられていないのは最後の受け渡しだけです。方式Aの成立性そのものへの否定的な証拠ではありません
- 観測した`Referrer-Policy`と`Origin`の関係は、**製品実装がloopback画面からform POSTを使う場合に同じく当てはまります。** 検証物のコードは流用しませんが、この契約は[`TD-38`](../../TODO.md#td-38-認証配布物とテンプレートのライフサイクル契約を確定する)へ渡します

次のいずれかが起きたら見直します。

- **`same-origin`でも`Origin`が正しく送られないChromeの版が現れたとき。** 対応版の範囲を[`TD-31`](../../TODO.md#td-31-実行環境の組み合わせを固定する)へ回し、form POST以外の受け渡しを設計し直します
- **`TD-52`の洗い出しで、browserの実挙動と食い違う契約testが他にも見つかったとき。** 個別に直さず、`V-12`の検証層の定義（[ADR-0008](0008-scope-of-execution-based-checks-for-verification-artifacts.md)）から見直します
