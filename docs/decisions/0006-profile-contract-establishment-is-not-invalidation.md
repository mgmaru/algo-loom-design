# ADR-0006 基準templateの一度きりの確定を、結果の無効化として扱わない

| 項目 | 内容 |
|---|---|
| 状態 | 採用 |
| 日付 | 2026年9月20日 |
| 関連TODO | [`TD-11`](../../TODO.md#td-11-方式a製品形態を実サービスで検証する)、[`TD-45`](../../TODO.md#td-45-campaign-manifestの確定遷移が自分を無効化する不整合を直す) |
| 正本 | [`JudgeAdapter`技術検証計画 §3.1.1](../project/judge-adapter-verification.md#311-v-12-方式a製品形態の検証)、[V-12検証物](../../scripts/verification/atcoder_v12/README.md) |

## 背景と前提

2026年9月20日、campaign `v12-2026-09-20-01`で`V-12A`・`V-12B`・`V-12D`が合格しました（[実行記録](../verification/judge-adapter/results/2026-09-20-v12-01.md)）。続けて`V-12C`へ進もうとしたところ、**`V-12`が原理的に完了できない**ことが分かりました。

判断の土台にした事実です。

| # | 前提 | 根拠 |
|---|---|---|
| 1 | `first-login`（`V-12B → V-12D`を実行する唯一の入口）は`profile.status`が`pending_v12b`であることを要求する | `helper/main.go`の`first_login_manifest_mismatch`判定 |
| 2 | `manifest validate --subtest`は、`V-12A`以外では`profile.status`が`fixed`であることを要求する | `helper/manifest.go`の`profile_not_fixed_for_*` |
| 3 | 基準templateの完全性IDは`V-12B`が作る。manifestへ記録すると新しいrevisionになる | [検証物README](../../scripts/verification/atcoder_v12/README.md)の順序 |
| 4 | `manifest compare`は`profile`が変われば`profile_contract_changed`として`V-12B`・`V-12C`・`V-12D`・`V-12E`を無効化していた | 同日に`{"invalidated":["V-12B","V-12C","V-12D","V-12E"],"reasons":["profile_contract_changed"]}`を観測 |
| 5 | `V-12B`の入力projectionは完全性IDに依存する | 同日の実測。IDだけ変えると`05042d80…`から`aa3a3e93…`へ変わる |
| 6 | `V-12A`の入力projectionは`profile`に依存しない | 同日の実測。revision 1と2で`4f400704…`のまま |
| 7 | `V-12`全体は、5つすべてが**同じ最終campaign manifest revision**で合格した場合だけ合格になる | [検証計画](../project/judge-adapter-verification.md#311-v-12-方式a製品形態の検証) |

前提1と2により、`V-12B`は「`V-12B`として検証できないmanifest」でしか実行できません。実行後に前提3の記録を行うと前提4が働き、たった今合格した`V-12B`と`V-12D`が無効になります。前提5により、検証計画が用意している「入力projectionのhash一致による再承認」も使えません。前提7とあわせると、**5つが揃う状態に到達できません。**

**この不整合は`TD-37`から存在していました。** `V-12B`を実機で最後まで通したのが2026年9月20日が初めてで、それまで`manifest compare`をこの遷移へ適用する場面がなかったためです。

## 決定

1. **`pending_v12b`から`fixed`への一度きりの遷移を、結果の無効化として扱いません。** 完全性IDが未設定から設定へ変わり、schema版が変わらない場合だけを対象とします。`manifest compare`はこれを`profile_contract_established`として報告し、無効化する結果を0件にします
2. **それ以外の`profile`の差は、従来どおり`profile_contract_changed`として`V-12B`〜`V-12E`を無効にします。** 完全性IDの差し替え、`fixed`から`pending_v12b`への逆行、schema版の変更が該当します
3. **この変更を検出するtestを、修正前のコードで落ちることを確認したうえで入れます。** 確定の遷移、IDの差し替え、逆行、schema変更の4caseを固定入力で判定します

## 検討して採らなかった案

| 案 | 採らなかった理由 |
|---|---|
| 完全性IDをmanifestから外し、sub結果側にだけ記録する | manifestが`V-12C`・`V-12E`の入力として基準templateを固定できなくなる。**どのtemplateを前提にした結果かを後から辿れない。** 無効化規則の「template schema・完全性規則」の行が機能しなくなる |
| `V-12B`の検証を`pending_v12b`でも通す | 同じ問題が残る。`V-12C`と`V-12E`がどのtemplateを前提にしたかを追えず、`V-12B`の証拠と後続の入力が結び付かない |
| `V-12B`の入力projectionから完全性IDを外す | 一見筋が通るが、`V-12D`は同じ導線の中でそのtemplateの複製を使う。`V-12D`のprojectionからも外すと、**`V-12D`が壊れたtemplateで動いても検出できなくなる** |
| 手作業で「この遷移は無効化に当たらない」と判断して記録する | 判断が実行者ごとに揺れる。**機械が無効と報告しているものを人が有効と読み替える運用**になり、無効化規則そのものが形骸化する |

## 理由

2つあります。

第一に、**無効化規則が「契約が変わった」と「契約がこのsub検証によって初めて埋まった」を区別していませんでした。** `profile`の差分をJSONの等価比較だけで判定していたためです。完全性IDは`V-12B`の**出力**であり、`V-12C`・`V-12E`の**入力**です。出力を記録した瞬間に、それを生んだsub検証が無効になるのは、規則の取り違えであって検証結果の問題ではありません。

第二に、**機械の報告を人が読み替える運用にしないためです。** 手作業で例外扱いにすれば今日は進めますが、次に`profile_contract_changed`が出たとき、それが本物の契約変更なのか同じ例外なのかを判断する根拠が残りません。[ADR-0005](0005-verify-consent-flow-in-browser-semantics.md)で問題にしたのと同じ型の取り違えを、運用側に作ることになります。

## 影響と再評価条件

- **campaign `v12-2026-09-20-01`は無効になります。** helperのbuild hashが変わるため、無効化規則の「helper版・build hash」の行により`V-12A`〜`V-12E`のすべてが対象です。`V-12A`・`V-12B`・`V-12D`の合格は取り消し、新しいcampaign IDでやり直します。**2026年9月20日の実行記録は残します。** 導線が成立した事実と、この不整合を見つけた経緯を消さないためです
- **やり直しの前に、検証支援物へ触る作業をまとめます。** [`TD-43`](../../TODO.md#td-43-検証支援物の実行経路をbrowser相当で確認する範囲を決める)と[`TD-44`](../../TODO.md#td-44-helperのエラーが原因を一意に指せない箇所を洗い出して直す)も同じ無効化を起こすため、別々に直すとそのたびに`V-12B`・`V-12D`の手作業をやり直すことになります
- **`V-12C`の必須caseの範囲は、本ADRでは決めていません。** 別の判断として残っています

次のいずれかが起きたら見直します。

- **同じ型の不整合が`environment_changed`や`extension_update_pair_changed`でも見つかったとき。** 決定1を`profile`だけでなく、出力が入力を兼ねる他のfieldへ広げます
- **基準templateを一つのcampaignで複数回確定する必要が出たとき。** 決定1は「一度きり」を前提にしています。前提が変われば、確定の回数を数える形へ変えます
