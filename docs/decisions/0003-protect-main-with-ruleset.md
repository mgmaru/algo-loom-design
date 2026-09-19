# ADR-0003 `main`の保護をRulesetで行い、bypassを設定しない

| 項目 | 内容 |
|---|---|
| 状態 | 採用 |
| 日付 | 2026年9月19日 |
| 関連TODO | ―（TODOに属さない判断） |
| 正本 | GitHubのRuleset設定（リポジトリのRules画面）、[作業ガイド §4.1](../../CLAUDE.md) |

## 背景と前提

[ADR-0002](0002-ci-and-branch-policy.md)で「`main`への直pushを禁止する」「管理者のbypassを当面許可する」と決めましたが、**実現する仕組みは決めていませんでした。** GitHubにはブランチを守る仕組みが2つあります。

> Multiple rulesets can apply to the same branch at the same time, while only one branch protection rule applies.
>
> Rulesets and branch protection rules can both protect branches in a repository. They work alongside each other, and all applicable rules are enforced.
> — [About rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets)（2026年9月19日確認）

判断の土台にした事実です。

| # | 前提 | 根拠 |
|---|---|---|
| 1 | 2つの仕組みは併存し、両方のルールが適用される | 上記の公式説明 |
| 2 | branch protection ruleは、一時的に外すには**削除するしかない** | 公式説明に`Disabled`相当の状態がない。Rulesetには`Active`と`Disabled`がある |
| 3 | Rulesetはread権限があれば誰でも読める | 公式説明の「broader visibility」 |
| 4 | このリポジトリでRulesetを作成できる | 2026年9月19日に実際に作成して確認した。個人所有・公開 |
| 5 | 一人で開発するためApproveは使えない | [ADR-0002](0002-ci-and-branch-policy.md)の前提4 |
| 6 | Rulesetの`pull_request`ルールは、`require_extra_approval_for_unattributed_changes`が既定で`true`になる | 2026年9月19日、作成時のAPI応答で観測 |

## 決定

1. **`main`の保護をRulesetで行います。** branch protection ruleは使わず、併用もしません
2. **bypass actorsを設定しません。** 緊急時は、Enforcementを`Disabled`へ切り替え、作業後に`Active`へ戻します
3. ルールは次のとおりです

| ルール | 値 |
|---|---|
| pull request | 必須。**承認は0件**。mergeは`rebase`のみ |
| required status checks | `文書と検証物`、`判断の確認`。strictを有効にし、`main`が進んだら追従してから merge |
| linear history | 必須 |
| force push | 禁止 |
| ブランチの削除 | 禁止 |
| `require_extra_approval_for_unattributed_changes` | **無効**（既定の`true`から変更） |

## 検討して採らなかった案

| 案 | 採らなかった理由 |
|---|---|
| branch protection rule（classic）を使う | 前提2のとおり、一時的に緩めるには削除するしかなく、設定を失う。管理者にしか見えないため、前提3の透明性も得られない |
| 両方を併用する | 前提1のとおり両方のルールが適用される。**PRが通らないときに、どちらが原因かの切り分けが難しくなる** |
| bypass actorsへ管理者を入れる（ADR-0002の当初案） | bypassは**保護を素通りする**運用で、緩めた事実が残らない。`Disabled`への切り替えなら設定の変更として残る |
| `require_extra_approval_for_unattributed_changes`を既定の`true`のままにする | **承認できない環境で、満たせない要求を残すことになる。** このリポジトリのcommitは`Co-Authored-By`を付けるため、該当してmergeを塞ぐ可能性がある |

## 理由

3つあります。

第一に、**一時的に緩める操作を安全にできます。** 削除して作り直す方式では、設定内容を人が覚えておく必要があり、戻し忘れや設定違いが起きます。

第二に、**保護の内容が誰からも見えます。** 「第三者がこのリポジトリを見て開発を継続できる」という[ADR-0001](0001-adopt-decision-records.md)の目的に沿います。

第三に、**満たせない要求を残しません。** 承認を使わないと決めた以上、承認を要求する設定は、既定値であっても外します。

## 影響と再評価条件

影響は次のとおりです。

- **[ADR-0002](0002-ci-and-branch-policy.md)の「管理者のbypassを当面許可する」は、この決定で置き換えます。** 以後bypassは使わず、`Disabled`への切り替えで対応します
- 作成者自身を含め、誰も`main`へ直接pushできません。2026年9月19日に実測で確認しました

```text
remote: error: GH013: Repository rule violations found for refs/heads/main.
remote: - Changes must be made through a pull request.
remote: - 2 of 2 required status checks are expected.
```

**運用上の落とし穴が1つあります。** required status checksは**job名の文字列**で指定しています。[workflow](../../.github/workflows/checks.yml)のjob名を変えると、Rulesetは報告されないcheckを待ち続け、**PRが永久にmergeできなくなります。** job名を変えるときは、同じ変更でRulesetの`contexts`も直します。同じ理由で、jobへ`paths`フィルタを足すと、対象外のPRでcheckが報告されず止まります。

次のいずれかが起きたら見直します。

- **開発者が増えたとき。** 承認の要求と`require_extra_approval_for_unattributed_changes`を再検討します
- **merge方式を変えるとき。** `allowed_merge_methods`を同じ変更で直します
- **`Disabled`への切り替えが常態化したとき。** ルールが実態に合っていないため、ルールのほうを見直します
