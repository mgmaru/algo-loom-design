# ADR-0007 helperのbuild hashを、履歴ではなく挙動に追従させる

| 項目 | 内容 |
|---|---|
| 状態 | 採用 |
| 日付 | 2026年9月20日 |
| 関連TODO | [`TD-11`](../../TODO.md#td-11-方式a製品形態を実サービスで検証する)、[`TD-45`](../../TODO.md#td-45-campaign-manifestの確定遷移が自分を無効化する不整合を直す) |
| 正本 | [`JudgeAdapter`技術検証計画 §3.1.1](../project/judge-adapter-verification.md#311-v-12-方式a製品形態の検証)、[V-12検証物](../../scripts/verification/atcoder_v12/README.md) |

## 背景と前提

[ADR-0006](0006-profile-contract-establishment-is-not-invalidation.md)の`TD-45`手順3として、**同じ型の取り違えが他のfieldにも残っていないか**を洗い出しました。探したのは「判定に使っている値が、確かめたいものの代理になっていて、代理が本体からずれている」箇所です。

`V-12`の結果無効化規則は「helper版・build hashが変われば`V-12A`〜`V-12E`のすべてを無効にする」と定め、理由を「挙動へ影響するため」としています。helperは5つのsub検証すべての入力projectionに入るため、**一部だけ生き残る切り方がない**ことは規則として妥当です。

問題は条件のほうでした。判断の土台にした事実です。

| # | 前提 | 根拠 |
|---|---|---|
| 1 | `prepare.mjs`のhelper buildは`-trimpath`だけを付けていた | 2026年9月20日時点のソース |
| 2 | Goは`vcs.revision`、`vcs.time`、`vcs.modified`をbinaryへ埋め込む | 2026年9月19日の実測。同じsource treeでも`b03ab6b`と`32e05d4`のbuildで288 byteが異なり、差は`buildinfo`のVCS欄 |
| 3 | 前提1と2により、**helper sourceが1 byteも変わらなくてもcommitが進めばbuild hashが変わる。** 2026年9月20日にも再現した（`726db47e…`→`ab256712…`、source treeは`9d6a27ae…`のまま） | 同日の実測 |
| 4 | campaign manifestはhelperのbinary hashを保持し、`first-login`が実行中の自分自身と照合する | `helper/main.go`の`artifactFileMatches` |
| 5 | 前提3と4により、**文書を1つ直しただけでcampaignが無効になる。** これは「挙動へ影響するため」という理由で説明できない | ― |
| 6 | 拡張機能ZIPはsource treeだけから作るため、commitが進んでも同じbytesになる | 2026年9月19日・20日の実測 |
| 7 | `environments[]`は入力専用である。`representativeEnvironmentMatches`が実行中のOS版とChrome版を**読んで照合する**だけで、manifestへ書き戻さない | `helper/main.go` |
| 8 | `extension.signed_builds`は入力専用である。値はCWS配信bytesの取得（`TD-42`手順5）で決まり、campaign中のsub検証は生成しない | [CWS配布準備 §8.1.9](../verification/judge-adapter/v12-chrome-web-store-preparation.md#819-2026年9月20日の011配信bytes取得記録) |

前提7と8により、[ADR-0006](0006-profile-contract-establishment-is-not-invalidation.md)が扱った「出力が入力を兼ねる」型は`profile.integrity_id`以外に見つかりませんでした。代わりに見つかったのが前提5の「代理がずれている」型です。

## 決定

1. **helperのbuildへ`-buildvcs=false`を付け、source treeから再現できるようにします。** 規則を緩めるのではなく、判定条件のほうを忠実にします。これでhelperのbinary hashは履歴ではなく挙動に追従します
2. **build条件を[`helper-build.mjs`](../../scripts/verification/atcoder_v12/helper-build.mjs)の1箇所に置きます。** `prepare.mjs`と固定入力testが同じ値を読みます。条件が2箇所にあると、片方だけ直したときにtestが実態を見なくなります
3. **buildした実行ファイルのバイト列にcommit idが現れないことを、固定入力testで毎回確認します。** `prepare.mjs`のソースに`-buildvcs=false`という文字列があることを検査するのではなく、**実際にbuildした結果**を見ます
4. `environments[]`と`extension.signed_builds`は入力専用であり、同じ型の取り違えはありません。前提7と8が根拠です

## 検討して採らなかった案

| 案 | 採らなかった理由 |
|---|---|
| `manifest compare`で「VCS欄だけの差」を無視する | **代理がずれたまま、判定側で例外を増やすことになる。** 「binary hashは違うが挙動は同じ」という状態を許すと、`first-login`の自己照合も同じ例外を持つ必要が出る。[ADR-0006](0006-profile-contract-establishment-is-not-invalidation.md)で採らなかった「人が読み替える運用」と同じ形 |
| helperのbinary hashをmanifestから外し、`source_tree_sha256`だけで判定する | **実行した実行ファイルを固定できなくなる。** sourceが同じでもbuild環境が違えば別のbinaryになりうる。証拠がどのバイナリについてのものか言えなくなる |
| `ldflags`でVCS情報を上書きする | 埋め込み自体は残るため、Goの版によって欄の構成が変わればまた差が出る。**埋め込まないほうが前提が少ない** |
| 規則の理由を「挙動へ影響するため」から「配布物が変わるため」へ書き換える | 規則の文言を実装へ合わせるだけで、**無関係な変更でcampaignが無効になる問題は残る。** やり直しの手作業が減らない |

## 理由

2つあります。

第一に、**代理を正確にするほうが、判定側へ例外を足すより前提が少なくて済みます。** `-buildvcs=false`を付ければ、「helperのbinary hashが違う」と「helperの挙動が違う」が一致します。判定側で例外を扱う案は、`manifest compare`と`first-login`の自己照合の両方に同じ例外を持たせる必要があり、片方を直し忘れたときに気づけません。

第二に、**この repository はすでに再現性を成立の根拠にしています。** 拡張機能ZIPは`0.1.1`を4つの独立したpathで同じbytesとして再現し、それを配信物とsourceの照合の土台にしました。helperだけが再現しないのは、同じ基準を一部にだけ適用している状態です。

## 影響と再評価条件

- **helperのbinary hashが変わります。** `9cf93575…`が新しい値です。進行中のcampaignはいずれにせよ[ADR-0006](0006-profile-contract-establishment-is-not-invalidation.md)の修正で無効になっているため、追加の損失はありません
- **`.tar.gz`のSHA-256も変わります。** reviewer受渡し用の経路1で、これまで「どのcommitのbuildか」を添える必要がありましたが、不要になります
- [サポートページ](../verification/judge-adapter/v12-extension-support.md)が固定しているのはreview fixtureのSHA-256だけで、fixtureはPythonの単一fileです。**この値は変わりません**
- **[CWS配布準備 §8.1.6](../verification/judge-adapter/v12-chrome-web-store-preparation.md#816-2026年9月19日の011審査提出記録)の記述は当時の事実として残します。** 「helperの実行ファイルはcommitが変わるとbytesが変わります」は2026年9月19日時点で正しく、記録は書き換えません

次のいずれかが起きたら見直します。

- **`-buildvcs=false`でも再現しない差が見つかったとき。** build環境（Goの版、toolchain）がbinaryへ影響する範囲を調べ、manifestの環境entryへ加えるかを判断します
- **同じ型の取り違えが3件目として見つかったとき。** 個別に直すのをやめ、「判定に使う値が代理になっていないか」を検証物の設計規則として明文化します
