# ADR-0005 同意画面のcontent scriptをbrowser相当で検証し、修正版を`0.1.1`として配信し直す

| 項目 | 内容 |
|---|---|
| 状態 | 採用 |
| 日付 | 2026年9月19日 |
| 関連TODO | [`TD-11`](../../TODO.md#td-11-方式a製品形態を実サービスで検証する)、[`TD-42`](../../TODO.md#td-42-修正版011を公開しv-12の再実行条件を整える)、[`TD-43`](../../TODO.md#td-43-検証支援物の実行経路をbrowser相当で確認する範囲を決める) |
| 正本 | [`JudgeAdapter`技術検証計画 §3.1.1](../project/judge-adapter-verification.md#311-v-12-方式a製品形態の検証)、[V-12検証物](../../scripts/verification/atcoder_v12/README.md) |

## 背景と前提

2026年9月19日、`TD-11`の`V-12B → V-12D`を初めて実機で実行しました。標準追加、固定IDによる導入完了の自動検出、Chromeの完全終了、基準templateの確定までは成立しましたが、**同意画面の「同意してAtCoderへ進む」を押しても何も起きませんでした。**

原因は`bootstrap.js`の先頭1行です。

```javascript
if (location.origin !== "http://127.0.0.1" || location.pathname !== "/bootstrap") return;
```

判断の土台にした事実です。

| # | 前提 | 根拠 |
|---|---|---|
| 1 | helperは`127.0.0.1`の**動的な待受番号**へbindする | [検証計画](../project/judge-adapter-verification.md#311-v-12-方式a製品形態の検証)の`V-12A`合格条件。実行時の観測でも`127.0.0.1:53673`だった |
| 2 | URLにポートが付くと`location.origin`はポートを含む | 2026年9月19日に実測。`new URL("http://127.0.0.1:53673/bootstrap").origin`は`http://127.0.0.1:53673` |
| 3 | 前提1と2により、この条件は**常に真**になり、content scriptは即座に`return`する。ボタンにclick handlerが付かない | 同日、`bootstrap.js`をmockした`location`で評価して再現した |
| 4 | 既存のNode testは`bootstrap.js`を**文字列として**検査していた | `assert.doesNotMatch`等による正規表現照合のみで、評価していない |
| 5 | Chromeのmatch patternはポートを含まないため、`http://127.0.0.1/*`は動的ポートにも一致する | content script自体は注入されていた。止まっていたのはscript内の判定 |
| 6 | 同じ拡張機能の`service_worker.js`は`value.hostname === "127.0.0.1"`と**ホスト名で**判定している | 同一拡張機能内で書き方が揃っていなかった |
| 7 | `atcoder.js`の`location.origin !== "https://atcoder.jp"`は、443番でポートが付かないため影響を受けない | 同日確認 |
| 8 | CWSが配信中の`0.1.0`は、リポジトリのsourceと`update_url`の1行と署名metadataを除いてbyte単位で一致する | 同日の再照合。9月18日の[§8.1.4](../verification/judge-adapter/v12-chrome-web-store-preparation.md#814-2026年9月18日の配信bytes取得記録)と同じ結果 |
| 9 | 拡張機能のsourceが変わると`V-12A`〜`V-12E`が無効になり、新campaignが必要になる | [検証計画の無効化表](../project/judge-adapter-verification.md#311-v-12-方式a製品形態の検証) |
| 10 | 標準追加はCWSの配信物からしか行わない。developer modeと手動読込は`V-12B`の停止条件 | [実施手順 §6.5](../verification/judge-adapter/README.md#65-v-12-方式a製品形態の実行gate) |

**この不具合は2026年8月26日の`TD-37`から存在し、CWSの審査と限定公開を通過していました。** 固定入力testが合格し続けていたのは前提4のためです。reviewer用フィクスチャも同じ動的ポートを使うため、審査担当者もこの導線を最後まで動かせなかった可能性が高いと考えます（これは推測で、実測していません）。

## 決定

1. **content scriptは、実際に評価してhandlerの登録と遷移先を確認するtestを持ちます。** ソースの文字列検査だけを成立証拠にしません。`bootstrap.js`については、動的ポート付きURLでhandlerが付き`initialize`が送られること、loopback以外のページでは何も起きないことをtestにしました
2. **`bootstrap.js`の判定を`protocol`・`hostname`・`pathname`へ変更します。** 前提6の`service_worker.js`と同じ書き方に揃えます
3. **修正版を`0.1.1`、更新testに使う版を`0.1.2`とします。** `0.1.1`はまだuploadしていないため、未使用の版を修正版へ充てます
4. **campaign `v12-2026-09-19-01`は無効とし、同campaignの`V-12A`の合格も取り消します。** 前提9の無効化規則に従い、修正版を配信した後に新しいcampaign IDで`V-12A`からやり直します
5. **回避しません。** 手動読込、developer mode、固定ポートへの変更で導線だけを先に確認することはしません

## 検討して採らなかった案

| 案 | 採らなかった理由 |
|---|---|
| 手動読込（unpacked）で同意画面から先の導線だけ先に確認する | 標準追加の導線が成立することは`V-12B`の合格条件そのものである。手動読込は停止条件であり、確認しても`V-12`の証拠にならない。**不具合の修正を確かめたいだけなら、browser相当のtestで足りる** |
| helperを固定ポート（80番）にして`location.origin`の比較を残す | 動的な待受番号は`V-12A`の合格条件であり、一回限りの秘密値と組み合わせた保護の一部である。80番のbindには特権も要る。**検証条件のほうを曲げることになる** |
| 修正版を`0.2.0`とし、更新testは`0.1.1`のままにする | 更新testが`0.2.0`→`0.1.1`の逆行になり、CWSの更新導線として成立しない。結局`0.1.1`側も取り直しになる |
| `bootstrap.js`で`location.origin.startsWith("http://127.0.0.1")`とする | `http://127.0.0.1.example.com`のようなホスト名に前方一致してしまう。ホスト名での完全一致のほうが安全側 |

## 理由

2つあります。

第一に、**テストが確認していた対象と、確認したかった対象がずれていました。** 確かめたかったのは「同意画面から認証へ進めること」で、実際に確かめていたのは「ソースに特定の文字列が現れないこと」です。前者は後者から導けません。今回はその差が、審査を通過して配信された成果物の中に3週間以上残りました。

第二に、**版を新しく起こすほうが、記録の対応関係を保てます。** `0.1.1`は未uploadで、`TD-39`の完了条件も「未uploadのまま保持」としていました。ここへ修正版を充てれば、CWS上の版と配信物のhashが一対一のまま増えます。

## 影響と再評価条件

- **`TD-39`の完了条件のうち「`0.1.1`が未uploadのまま保持され、`TD-11`の更新test前に対象版`0.1.0`を置き換えていない」は、この決定で前提が変わりました。** `TD-39`の本文と完了条件は書き換えません。現在の版の割り当ての正本は本ADRと[CWS配布準備](../verification/judge-adapter/v12-chrome-web-store-preparation.md)です
- CWSへの`0.1.1` uploadと審査提出は**外部操作**であり、[作業ガイド §4](../../CLAUDE.md#4-外部操作には明示承認が必要)により別の明示承認が要ります。本ADRはその承認を含みません
- [サポートページ](../verification/judge-adapter/v12-extension-support.md)と[privacy policy](../verification/judge-adapter/v12-extension-privacy-policy.md)が固定しているreviewer用フィクスチャのSHA-256は、フィクスチャを変更していないため有効なままです。ただし`0.1.1`を審査へ出す期間は、この2ファイルへ再び凍結を適用します
- `V-12B`のうち、標準追加、固定IDによる自動検出、Chrome完全終了、基準templateの一度だけの確定は**実機で成立することを観測しました。** この観測は新campaignの合格判定には使いませんが、方式Aの成立性に不利な事実は得られていません
- **`TD-11`の1回目の実行記録は残し、やり直しに必要な作業を別のTODOとして起票しました。** 修正版の公開は[`TD-42`](../../TODO.md#td-42-修正版011を公開しv-12の再実行条件を整える)、testの適用範囲の決定は[`TD-43`](../../TODO.md#td-43-検証支援物の実行経路をbrowser相当で確認する範囲を決める)です。失敗した実行の記録を消すと、同じ失敗を繰り返す余地が残るためです。`TD-11`は`TD-42`の完了まで保留とします

次のいずれかが起きたら見直します。

- **同じ種類の不具合が他のcontent scriptでも見つかったとき。** 決定1をcontent scriptだけでなく拡張機能の全entry pointへ広げます
- **実browserでの自動確認手段を、Bot対策の回避なしに用意できるようになったとき。** 決定1のtestをbrowser相当の評価から実browserの実行へ引き上げます。現時点ではCDPとWebDriverが`V-12`の停止条件であるため採れません
