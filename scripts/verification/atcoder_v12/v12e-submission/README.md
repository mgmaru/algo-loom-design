# `V-12E`の提出前入力

[`V-12E`](../../../../docs/project/judge-adapter-verification.md#311-v-12-方式a製品形態の検証)の入力である「終了済み過去問1件の提出前入力」です。`submit-entry`が読み、提出確認画面へ対象・言語・sourceを表示します。

**提出は行いません。** `source.py`は対象問題の解答ではなく、提出対象を表示できることだけを確かめるための固定の最小コードです。解答である必要はありません。

AtCoder由来の問題文、入出力例、他の利用者のコードは含みません。含めてはいけません（[作業ガイド §5](../../../../CLAUDE.md#5-リポジトリへ書かないもの)）。

| file | 役割 |
|---|---|
| `input.json` | 対象問題、提出先URL、言語の表示名、sourceのfile名 |
| `source.py` | 提出対象として表示する固定source |

`input.json`の`submit_url`は`https://atcoder.jp/contests/`配下だけを許します。`source_file`は同じディレクトリのfile名だけを許し、path区切りを含められません。
