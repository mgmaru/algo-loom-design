# AlgoLoom設計リポジトリの作業ガイド

このファイルは、作業を再開するときに最初に読む運用ガイドです。製品の設計判断は書きません。設計の正本は[README](README.md)から辿ります。

## 1. 最初に見る場所

1. **[`TODO.md`](TODO.md) の「外部の応答を待っている作業」** — 外部の返答待ちで止まっている作業がここに集約されています。**触ってはいけないファイルもここに書いてあります。**
2. `node scripts/todo-status.mjs` — 着手可能な作業と、それぞれが解放する下流の件数が出ます。

```console
node scripts/todo-status.mjs
```

## 2. このリポジトリの性質

設計文書のリポジトリです。実装コードは置きません。

| 場所 | 役割 |
|---|---|
| [`docs/`](docs/) | 設計判断の**正本**。製品思想、設計理由、判断経緯、調査記録 |
| [`spec/`](spec/) | `docs/`から作る規範的な**投影**。独立した正本ではない |
| [`scripts/verification/`](scripts/verification/) | 技術検証の支援物。**製品コードではなく、製品へ流用しない** |

`docs/`と`spec/`が矛盾したら、実装で一方を選ばず、同じ変更で両方を整合させます。

## 3. 変更したら必ず実行する

```console
node scripts/check-docs.mjs                          # 相対リンクとアンカー
node --test scripts/verification/test_atcoder_v12.mjs # 検証物の固定入力test
python3 scripts/verification/atcoder_v12/algoloom_v12_review_fixture.py --self-test
cd scripts/verification/atcoder_v12/helper && go test ./...
```

`check-docs.mjs`は壊れたリンクがあれば終了コード1で終わります。**見出しを変えたらリンクも同じ変更で直します。**

## 4. 外部操作には明示承認が必要

**次はいずれも、対象と影響を提示して人間の明示承認を得るまで実行しません。**

- 外部サービスへの登録、契約、支払い
- Chrome Web Storeへのupload、審査提出、公開、unpublish
- GitHubのリリース作成、ファイルのupload
- 問い合わせの送信
- AtCoderを含む外部サービスへの接続を伴う検証の実行

「TD-xxを進めてよい」という一般的な依頼を、これらの承認へ読み替えません。承認記録はリポジトリ外のowner専用領域で管理します。

コミットとpushは、利用者が求めたときだけ行います。

## 5. リポジトリへ書かないもの

- credential、password、Cookie、token、秘密鍵
- 実際のAtCoderアカウント名、Chrome Web Storeの拡張機能の固定ID、publisher情報
- 端末の絶対path、ホスト名、利用者名
- AtCoder由来の問題文、画像、解説、他ユーザーのコード
- ビルド生成物。`prepare.mjs`はリポジトリ内への出力を拒否します

## 6. 書き方

[表記規則](docs/project/writing-conventions.md)に従います。要点は次の2つです。

- 一般語は日本語で書く。コード上の名前だけ英語で残し、バッククォートで囲む
- 既存文書を編集するときは、**編集した範囲を表記規則へ合わせる**

見出しへMarkdownのリンクを入れません。アンカーが読みにくくなります。

## 7. コミット

`docs:`、`feat:`等の接頭辞を付け、本文は日本語で書きます。**何を決めたかと、なぜそう決めたか**を残します。末尾に次を付けます。

```text
Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

## 8. 判断の分担

[notes.mdの振り返り](notes.md)で確認された方針です。

- **製品の契約が変わる判断は人間が関与します。** 保存する内容、利用者へ示す約束、外部作用の範囲が変わるものは、決める前に提示して確認を得ます
- 実装方法、内部構成、文書の構成は、契約を変えない範囲で進めて構いません

判断に迷ったら、[製品契約と実装判断の境界](docs/project/product-contract-and-implementation-boundary.md)を参照します。
