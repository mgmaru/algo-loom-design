# 秘密情報保管庫の保証範囲を観測する手順

> 対象: [`scripts/verification/secret_store_guarantees.py`](../../../scripts/verification/secret_store_guarantees.py)を、macOS以外のOSで再現するための手順
>
> 作成日: 2026年9月21日
>
> 関連作業: [`TD-47`](../../../TODO.md#td-47-windowsの秘密情報保管庫が保証する範囲を観測する)（Windows）、[`TD-48`](../../../TODO.md#td-48-linuxの検証環境を確保し秘密情報保管庫が保証する範囲を観測する)（Linux）
>
> 記録先: [AtCoder認証設計 §4.1](../../architecture/atcoder-authentication.md#41-保管先)、[未決事項 7.3](../../project/unresolved-decisions.md#73-秘密情報保管庫が保証する範囲と表示文言)

## ドキュメント概要

本書は、**観測を別の端末で再現するための実行手順**です。観測結果の解釈と表示文言の決定は含みません。それらは[`TD-46`](../../../TODO.md#td-46-検証マトリクスを確定しwindowsとlinuxの秘密情報保管庫を観測する)が3つのOSの結果を揃えてから行います。

macOSの観測結果は[認証設計 §4.1.1](../../architecture/atcoder-authentication.md#411-macosで観測した実際の保証範囲)にあります。**同じ観測IDを使うため、OSをまたいで結果を並べて比較できます。**

## 1. 何を確かめるか

AlgoLoomはAtCoderの`REVEL_SESSION`をOSの秘密情報保管庫へ保存します。**利用者へ何を保証できるかは、OSが実際に何を守るかで決まります。** 保証していないことを保証すると書けば、それは嘘になります。

確かめるのは次の5つです。**同じ利用者として動く他のプロセスから読めるかどうか**が中心です。

| 観測ID | 内容 | なぜ見るか |
|---|---|---|
| `same-process` | 作成したプロセス自身から読めるか | 保存と読み出しが成立する前提の確認 |
| `same-interpreter` | 同じインタプリタの別スクリプトから読めるか | **AlgoLoomを更新したときに再認可を求められるか**に相当する |
| `other-executable` | 別の実行ファイルから読めるか | **同じ利用者の他のプロセスから読めるか。** ここが表示文言を左右する |
| `delete` | 読み出しの認可なしで削除できるか | 削除の約束が成立するか |
| `residue` | probe項目が残っていないか | 後始末の確認 |

**どの結果も「正解」ではありません。** macOSと違う結果が出ること自体が観測事実であり、そのまま記録します。

## 2. 安全境界

**この観測は外部サービスへ接続しません。人の明示承認は要りません**（[作業ガイド §4](../../../CLAUDE.md#4-外部操作には明示承認が必要)の対象外）。

| 項目 | 内容 |
|---|---|
| 外部接続 | **0件。** AtCoder、Chrome Web Store、その他のどこへも接続しない |
| 扱う値 | 固定のダミー値だけ。**実アカウント、Cookie、password、実際の認証情報を一切使わない** |
| 保管庫への書き込み | `algoloom-secret-store-probe-`で始まる使い捨て項目1件のみ。観測後に削除する |
| 安全弁 | probe専用の名前以外は扱わない。名前の形式を正規表現で検査してから保管庫へ触れる |
| 管理者権限 | **不要。** 通常の利用者として実行する |
| 出力 | 端末名、利用者名、ホームディレクトリ配下の絶対pathを観測物が伏せる |

## 3. 前提

| 項目 | 内容 |
|---|---|
| Python | 3.9以上。追加パッケージは要らない（標準ライブラリだけを使う） |
| git | cloneに使う。**リポジトリは公開のため認証は要らない** |
| Windowsで追加に要るもの | PowerShell（Windows標準のもので可）。`other-executable`の観測に使う |
| Linuxで追加に要るもの | **観測物のLinux経路は未実装。** [`TD-48`](../../../TODO.md#td-48-linuxの検証環境を確保し秘密情報保管庫が保証する範囲を観測する)で実装してから行う |

**WSLはnative Linuxの代替になりません**（[認証設計 §6.2.7](../../architecture/atcoder-authentication.md#627-確認手段を用意できない組み合わせ)）。WSL上で実行した結果を、Linuxの観測結果として記録しません。

### 3.1. シェルはどれでもよい

**Windowsでは、コマンドプロンプトとPowerShellのどちらでも動きます。** 観測物はshell文字列を組み立てず、子プロセスをargvで起動するためです。これは製品側の原則と同じ考え方です（[Core契約](../../architecture/core-contracts.md#24-設定と実行commandの信頼境界)の「`LanguageProfile`はshell文字列ではなくargv、working directory、source、artifact、timeout区分等からなるBuildPlan / RunPlanを返す」）。

**`other-executable`の観測結果も、親のシェルによって変わりません。** この観測は観測物自身が`powershell.exe`を子プロセスとして起動して行うため、書き込むのが`python.exe`、読み出すのが新しい`powershell.exe`という関係は、どのシェルから起動しても同じです。

Pythonの呼び出し方だけ、環境によって違います。**先に版を確認します。**

```console
py -3 --version
```

これが動かない環境では`python --version`を使い、以降の`py -3`を`python`へ読み替えます。**PowerShellで`python`と打ってMicrosoft Storeが開く場合は、Pythonが入っていません。**

## 4. 手順

### 4.1. 取得

```console
git clone https://github.com/mgmaru/algo-loom-design.git
cd algo-loom-design
```

### 4.2. 自己testを先に実行する

**保管庫へ触れず、安全弁と伏せ字と出力形式だけを確認します。** ここが通らない状態で本体を実行しません。

```console
py -3 scripts\verification\secret_store_guarantees.py --self-test
```

`py -3`が使えない環境では`python`に読み替えます。macOSとLinuxでは`python3 scripts/verification/secret_store_guarantees.py --self-test`です。

### 4.3. 観測を実行する

```console
py -3 scripts\verification\secret_store_guarantees.py
```

Markdownの表が標準出力へ出ます。**そのまま貼れる形になっています。**

### 4.4. 結果を渡す

次の4つを渡します。

1. 自己testの出力
2. 観測本体の出力（Markdownの表全体）
3. Pythonの版（`py -3 --version`）とOSの版
4. 失敗した場合は、**エラーの全文**

## 5. 期待する出力の形

先頭にOS、CPU architecture、保管庫の名前、使ったインタプリタが並び、続いて5行の表が出ます。

```text
# 秘密情報保管庫の保証範囲の観測

- OS: <OS名とversion>
- CPU architecture: <architecture>
- 保管庫: <保管庫の名前>
- 観測に使ったインタプリタ: <実行ファイル名>

| 観測 | 内容 | 結果 | 詳細 |
|---|---|---|---|
| `same-process` | … | … | … |
| `same-interpreter` | … | … | … |
| `other-executable` | … | … | … |
| `delete` | … | … | … |
| `residue` | … | … | … |
```

**`residue`が「削除済み」以外になった場合だけ、[§7](#7-後始末)を実行します。**

`結果`の欄には次のいずれかが入ります。**`判定できなかった`は保管庫の保護ではありません。** 観測物が値を取り出せなかったという意味で、[§6](#6-うまくいかないとき)のとおり不具合として報告します。

| 結果 | 意味 |
|---|---|
| `読めた` / `読めなかった` | 保管庫の保護として観測できた |
| `確認画面が出た（対話が要る）` | 読み出しに利用者の確認が要る |
| **`判定できなかった`** | **観測物が値を取り出せなかった。保管庫の結果として記録しない** |
| `実施できず` | 子プロセスを起動できない等で観測を行えなかった |
| `削除できた` / `残っていない` | `delete`と`residue`の結果 |

## 6. うまくいかないとき

**失敗したこと自体を記録に残します。** 消して無かったことにしません。

| してよいこと | してはいけないこと |
|---|---|
| エラーの全文を報告する | **管理者権限で実行し直す** |
| Pythonの版を変えて再実行し、結果を両方報告する | **別の保管庫や別の実装へ切り替える** |
| 観測物の不具合を直してから再実行する | **動かなかった観測を「未実施」として省く** |
| 実行環境（OS版、Python版）を報告する | **出力へ端末名、利用者名、絶対pathを書き足す** |
| **`判定できなかった`が出たら、不具合として報告する** | **`判定できなかった`を「読めなかった」として記録する** |

管理者権限で実行しない理由は、**製品が管理者権限を前提にしないためです。** 通常の利用者として読めるかどうかが、そのまま利用者への説明になります。

**`判定できなかった`を「読めなかった」として扱わない理由も同じです。** 実際には読めるものを読めないと記録すれば、その上に過大な表示文言を置くことになります。2026年9月21日のWindowsの観測で実際に起きました（[認証設計 §4.1.2.1](../../architecture/atcoder-authentication.md#4121-最初の実行が反対の結果を報告したこと)）。

## 7. 後始末

観測物は自分で削除まで行いますが、途中で失敗した場合はprobe項目が残ることがあります。**残っていないことを別の手段で確認します。**

Windowsでは資格情報マネージャーを確認します。

```console
cmdkey /list | findstr algoloom
```

`algoloom-secret-store-probe-`で始まる項目が出た場合だけ、その名前を指定して削除します。

```console
cmdkey /delete:algoloom-secret-store-probe-XXXXXXXX
```

PowerShellでは`cmdkey /delete:"algoloom-secret-store-probe-XXXXXXXX"`と引用符を付けても構いません。

**`algoloom`を含まない項目へ触れません。** 観測物と同じく、probe専用の名前だけを対象にします。

## 8. 観測後にAlgoLoom側で行うこと

**この手順の実行者は、観測結果をリポジトリへコミットしません。** 記録の形と置き場所が決まっているため、次はowner側で行います。

1. 結果を[認証設計 §4.1](../../architecture/atcoder-authentication.md#41-保管先)配下へ、macOSの節と同じ並びで記録する
2. [未決事項 7.3](../../project/unresolved-decisions.md#73-秘密情報保管庫が保証する範囲と表示文言)の観測結果へ足す
3. [作業ガイド §3](../../../CLAUDE.md#3-変更したら必ず実行する)の5つの検査を通してPRを出す

**表示文言はここで決めません。** 3つのOSが揃ってから[`TD-46`](../../../TODO.md#td-46-検証マトリクスを確定しwindowsとlinuxの秘密情報保管庫を観測する)で決めます。
