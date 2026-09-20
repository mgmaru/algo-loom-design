# ADR-0008 検証支援物のうち、実行して評価する範囲を拡張機能の3つのentry pointに定める

| 項目 | 内容 |
|---|---|
| 状態 | 採用 |
| 日付 | 2026年9月20日 |
| 関連TODO | [`TD-32`](../../TODO.md#td-32-テスト方針の骨格を決める)、[`TD-43`](../../TODO.md#td-43-検証支援物の実行経路をbrowser相当で確認する範囲を決める) |
| 正本 | [V-12検証物](../../scripts/verification/atcoder_v12/README.md)、[固定入力test](../../scripts/verification/test_atcoder_v12.mjs) |

## 背景と前提

[ADR-0005](0005-verify-consent-flow-in-browser-semantics.md)は`bootstrap.js`について、ソースの文字列検査ではなく実際に評価するtestを入れました。**どこまで広げるかは決めていませんでした。** `TD-43`はその範囲を決める作業です。

2026年9月20日に検証支援物のentry pointを列挙し、各testが何を見ているかを分類しました。

| entry point | 環境依存 | 2026年9月20日より前のtest |
|---|---|---|
| `bootstrap.js` | URL、動的な待受番号、message passing | **評価**（[ADR-0005](0005-verify-consent-flow-in-browser-semantics.md)で追加） |
| `atcoder.js` | origin、pathname、DOM、message passing | ソースの文字列検査だけ |
| `service_worker.js` | 送信元URL、動的な待受番号、storage、`fetch` | ソースの文字列検査だけ |
| helperのHTTP handler | loopback、`Host`、origin、token、状態順序 | **評価**（Goの`httptest`で14件） |
| review fixture | 同じprotocol、quarantine属性 | **評価**（実際に起動して`--self-test`） |
| `prepare.mjs` | Go・Swiftのtoolchain、git、ファイルシステム | ソースの文字列検査だけ |
| `prepare-store-assets.mjs` | 画像生成 | ソースの文字列検査だけ |

判断の土台にした事実です。

| # | 前提 | 根拠 |
|---|---|---|
| 1 | `atcoder.js`と`service_worker.js`は、`bootstrap.js`が止まったのと**同じ種類の判定**（URLのどの部分を見るか）を持つ | 2026年9月20日のソース確認 |
| 2 | `service_worker.js`の送信元判定は動的な待受番号を含むURLを受ける。`bootstrap.js`の不具合が起きた条件と同じ | `bootstrapSender`が`hostname`で判定している |
| 3 | 両者は`vm`とmockで評価でき、実browserもCDPも要らない | 同日、実際にtestを書いて確認した |
| 4 | **同じ型の不具合を入れると、追加したtestが落ちる。** `atcoder.js`の判定へportを足し、`service_worker.js`の送信元判定を`origin`比較へ変えたところ、3件が落ちた | 同日の実測 |
| 5 | CIは`ubuntu-latest`で動く。`prepare.mjs`はSwiftの`xcrun swiftc`を使うため、CIで最後まで実行できない | [workflow](../../.github/workflows/checks.yml) |
| 6 | `prepare.mjs`の生成物が壊れれば、campaign manifestの検証（`validate`、`artifactFileMatches`）が停止する。**検証の実行時に必ず通る関門がある** | `helper/main.go` |
| 7 | 拡張機能のsourceを変えると`V-12A`〜`V-12E`が無効になり、CWSへの再提出と審査が要る | [検証計画の無効化表](../project/judge-adapter-verification.md#311-v-12-方式a製品形態の検証) |

## 決定

1. **拡張機能の3つのentry pointすべてを、実行して評価します。** `bootstrap.js`に加えて`atcoder.js`と`service_worker.js`を移しました。検査するのは「どのページ・どの送信元で動くか」「何を送るか」「送らないか」です
2. **`atcoder.js`と`service_worker.js`のソースは変更しません。** 前提7のとおり、変更すればCWSへの再提出と審査が発生します。今回入れたのはtestだけです。**testが落ちないことを確認したのであって、不具合が見つかったのではありません**
3. **`prepare.mjs`と`prepare-store-assets.mjs`は文字列検査のままにします。** 前提5により全体をCIで実行できず、前提6により実行時の関門が別にあるためです。ただし、**特定の性質が重要な場合は、その性質だけを実行して確かめるtestを足します。** 2026年9月20日に追加したhelper buildの再現性test（[ADR-0007](0007-make-helper-build-hash-track-behaviour.md)）がその形です
4. **移した各testは、同じ型の不具合を入れると落ちることを確認してから入れます。** 落ちないtestは、その不具合を検出できていません

## 検討して採らなかった案

| 案 | 採らなかった理由 |
|---|---|
| すべてのentry pointを実行評価へ移す | `prepare.mjs`の全体実行はCIで成立しない（前提5）。macOS runnerを足せば動くが、**CIが外部サービスへ接続しない**という現在の性質と、実行時間・維持費の増加に見合わない |
| 実browserを使って評価する | CDP、WebDriver、headlessは`V-12`の停止条件である。[ADR-0005](0005-verify-consent-flow-in-browser-semantics.md)の再評価条件のとおり、Bot対策の回避なしに用意できるようになってから引き上げる |
| `atcoder.js`の判定も`bootstrap.js`と同じ`protocol`・`hostname`・`pathname`へ揃える | **拡張機能のsourceが変わり、CWSへの再提出と審査が発生する**（前提7）。`atcoder.js`の`origin`比較は443番でポートが付かないため現に成立しており、**成立している配信物を書き換える理由がない。** 揃えること自体は望ましいので、次に拡張機能を作り直す機会に行う |
| 文字列検査を全部消す | 評価testと重複する範囲はあるが、**消す作業自体が検出漏れを作りうる。** 重複は費用が小さい |

## 理由

2つあります。

第一に、**`bootstrap.js`の不具合が起きた条件が、他の2つのentry pointにもそのまま存在します。** URLのどの部分を見るかという判定を持ち、動的な待受番号が絡みます。前提4のとおり、同じ壊し方を入れると新しいtestは落ちます。裏返すと、**移す前はこの型の不具合を検出できていませんでした。**

第二に、**関門があるかどうかで線を引けます。** `prepare.mjs`の生成物は、検証の実行時にmanifest検証とhash照合という別の関門を必ず通ります（前提6）。一方、拡張機能のentry pointは**実際にbrowserで動かすまで誰も通らない関門**でした。だから3週間以上気づけませんでした。関門のないものを優先します。

## 影響と再評価条件

- `TD-32`（テスト方針の骨格）へ、この線引きを引き渡します。**製品実装でも「関門のない経路を優先して実行評価へ移す」を判断基準に使えます**
- 追加したtestは`vm`とmockだけで動き、外部接続もCDPも使いません。CIの実行時間は1秒未満の増加です
- **`atcoder.js`と`service_worker.js`のソースは変えていないため、配信中の`0.1.1`との対応は保たれます**

次のいずれかが起きたら見直します。

- **`prepare.mjs`の生成物の不具合が、実行時の関門をすり抜けて見つかったとき。** 決定3の前提が崩れるため、build scriptも実行評価へ移します
- **実browserでの自動確認を、Bot対策の回避なしに用意できるようになったとき。** 決定1のtestを評価から実browserの実行へ引き上げます
- **mockと実際のChromeの挙動がずれていたと分かったとき。** `vm`で置いた`chrome.*`とDOMの仮定を見直します
