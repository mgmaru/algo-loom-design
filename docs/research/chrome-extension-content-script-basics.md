# Chrome拡張機能のcontent script入門：2026年9月19日の不具合から学ぶ

> 目的：`V-12`の初回導線が止まった原因を題材に、Chrome拡張機能のcontent scriptがどう動くかと、URLの構成要素の違いを整理する。
>
> 位置付け：本書は**学習用の整理メモ**であり、設計を決める正本ではない。不具合そのものの判断記録は[ADR-0005](../decisions/0005-verify-consent-flow-in-browser-semantics.md)、検証物の仕様は[V-12検証物](../../scripts/verification/atcoder_v12/README.md)を正本とする。
>
> 作成日：2026年9月19日 ／ Chromeの仕様は変わり得るため、挙動に依存する判断を行う前に公式資料と実測で再確認すること。

---

## ドキュメント概要

本書は、ジュニアエンジニア向けに次を説明します。

1. Chrome拡張機能のcontent scriptが「注入される」とはどういうことか
2. URLの`origin`、`hostname`、`port`の違い
3. 2026年9月19日に起きた不具合が、この2つの理解のどこでつまずいた結果だったか
4. 検討中に出た誤解と、その正しい形

実際に手元で確かめられるコマンドも添えます。**読むだけでなく動かしてください。**

## 1. 何が起きたか（30秒の要約）

AtCoderの認証を行うChrome拡張機能を作り、審査を通して配信していました。利用者が同意画面のボタンを押すとAtCoderへ進む、という導線です。

**そのボタンを押しても、何も起きませんでした。**

原因は拡張機能の`bootstrap.js`の1行です。

```javascript
if (location.origin !== "http://127.0.0.1" || location.pathname !== "/bootstrap") return;
```

同意画面のURLは`http://127.0.0.1:53673/bootstrap`でした。**ポートが付いているため`location.origin`は`http://127.0.0.1:53673`になり、比較対象の`http://127.0.0.1`と一致しません。** その結果、処理が始まる前に`return`していました。

## 2. 登場人物

まず、誰がどこで動いているかを押さえます。**ここが今回いちばん誤解しやすい部分です。**

| 名前 | 実体 | 動く場所 | 役割 |
|---|---|---|---|
| helper | Goで作った実行ファイル | 利用者の端末 | `127.0.0.1`の空きポートで待受し、同意画面のHTMLを配る |
| `consent.html` | HTML | helperの中に埋め込み | 同意画面の見た目。ボタンの`<button>`要素がある |
| `bootstrap.js` | 拡張機能のcontent script | **ブラウザの中** | 配られたページに注入され、ボタンに動作を付ける |
| `service_worker.js` | 拡張機能のbackground script | ブラウザの中 | Cookieを読み、helperへ渡す |

**helperはサーバ、`bootstrap.js`はクライアントです。** 名前が似ていて同じ機能のために作られていますが、動く場所が違います。

## 3. content scriptは「注入」される

### 3.1. 普通のWebページの場合

通常、ページで動くJavaScriptは**そのページ自身が指定したもの**です。

```text
  サーバ                          ブラウザ
    │                               │
    │  HTML + <script src="app.js">  │
    ├──────────────────────────────▶│
    │                               │ app.js を読み込んで実行
    │                               │
```

サーバが「このJavaScriptを読み込んでください」と書いているから実行されます。**サイトが自分で決めた経路**です。

### 3.2. content scriptの場合

拡張機能のcontent scriptは、この経路の外から入ります。

```text
  helper                          ブラウザ
    │                               │
    │  HTML だけ（scriptタグなし）    │
    ├──────────────────────────────▶│
    │                               │ ページを表示
    │                               │      ▲
    │                               │      │ Chromeが拡張機能の
    │                               │      │ bootstrap.js を差し込む
    │                               │      │
    │                               │  ┌───┴──────────┐
    │                               │  │ 拡張機能     │
    │                               │  │ (利用者が    │
    │                               │  │  入れたもの) │
    │                               │  └──────────────┘
```

実際に確かめられます。helperが配るHTMLには`<script>`タグが**1つもありません**。

```console
$ grep -c "<script" scripts/verification/atcoder_v12/helper/consent.html
0
```

では誰が`bootstrap.js`を動かしているのか。**Chromeです。** 拡張機能の`manifest.json`に、こう申請してあります。

```json
{
  "matches": ["http://127.0.0.1/*"],
  "js": ["bootstrap.js"],
  "run_at": "document_end"
}
```

意味は「`http://127.0.0.1`のページが開かれたら、そのページの中で`bootstrap.js`を実行してください」です。

**ページを作った側の意思とは無関係に、利用者が入れた拡張機能の都合でコードが後から入ってくる。** だから「注入（inject）」と呼びます。

### 3.3. だから自己確認が要る

`http://127.0.0.1/*`に一致するページなら、helperの同意画面でなくても注入されます。ローカルで動かしている無関係な開発サーバのページでも注入されます。

そのためcontent scriptは、動き出す前に**自分がいま乗っているページが想定どおりか**を確かめるのが定石です。`bootstrap.js`の1行目はそのための確認でした。

**その確認が、本物の同意画面を「違う」と判断してしまった**というのが今回の不具合です。

### 3.4. isolated world

content scriptは、ページのDOM（要素）には触れますが、**ページ自身のJavaScriptとは変数の世界が分かれています**。isolated worldと呼ばれる仕組みです。

| できること | できないこと |
|---|---|
| `document.getElementById`で要素を掴む | ページ側のJSが持つ変数を読む |
| 要素にイベントを付ける | ページ側の関数を呼ぶ |
| DOMを書き換える | ページ側から自分の変数を読ませる |

ページ側のコードと名前がぶつかって壊れないようにするための分離です。これも「外から来ている」ことの表れです。

## 4. URLの構成要素

もう一つの土台がURLの分解です。今回はここの取り違えが直接の原因でした。

```text
  http://127.0.0.1:53673/bootstrap
  └─┬──┘ └───┬───┘└─┬──┘└───┬────┘
    │        │      │       │
 protocol  hostname port  pathname
    │        │      │
    └────────┴──────┘
         origin
```

| プロパティ | 値 | 説明 |
|---|---|---|
| `location.protocol` | `http:` | 通信方式。**末尾にコロンが付く** |
| `location.hostname` | `127.0.0.1` | ホスト名。**ポートを含まない** |
| `location.port` | `53673` | ポート番号。**文字列**。既定ポートのときは空文字 |
| `location.host` | `127.0.0.1:53673` | ホスト名とポート |
| `location.origin` | `http://127.0.0.1:53673` | protocol + host。**ポートを含む** |
| `location.pathname` | `/bootstrap` | パス |

**要点は`origin`がポートを含むことです。** 手元で確かめられます。

```console
$ node -e 'const u = new URL("http://127.0.0.1:53673/bootstrap");
  console.log("origin  :", u.origin);
  console.log("hostname:", u.hostname);
  console.log("port    :", u.port);'
origin  : http://127.0.0.1:53673
hostname: 127.0.0.1
port    : 53673
```

### 4.1. 既定ポートのときは消える

ややこしいのは、**既定ポートだと`origin`からポートが消える**ことです。

| URL | `origin` |
|---|---|
| `https://atcoder.jp/settings` | `https://atcoder.jp`（443番は既定なので出ない） |
| `http://127.0.0.1/bootstrap` | `http://127.0.0.1`（80番は既定なので出ない） |
| `http://127.0.0.1:53673/bootstrap` | `http://127.0.0.1:53673` |

同じ拡張機能の中で、AtCoder側を見る`atcoder.js`は次のように書いていて、**こちらは正しく動いていました。**

```javascript
if (location.origin !== "https://atcoder.jp" || ...) return;   // 443番なので一致する
```

**同じ書き方が、片方では動き、片方では動かない。** これが見落としの温床になりました。

## 5. なぜテストで見つからなかったか

`bootstrap.js`にはテストがありました。しかし**ソースコードを文字列として検査していただけ**でした。

```javascript
// 修正前のテストの発想
assert.doesNotMatch(BOOTSTRAP, /\.click\s*\(/);   // 自動クリックしていないか
```

確かめたかったことと、実際に確かめていたことがずれていました。

| | 内容 |
|---|---|
| 確かめたかったこと | 同意画面から認証へ**進めること** |
| 実際に確かめていたこと | ソースに特定の文字列が**現れないこと** |

**後者から前者は導けません。** 結果として、一度も動いたことのないコードが、テストを全部通過したまま審査を通り、3週間以上配信されていました。

修正では、`bootstrap.js`を**実際に評価する**テストへ変えました。

```javascript
// 修正後の発想：ポート付きのURLで本当にハンドラが付くか
const { button, navigated } = runBootstrapAt("http://127.0.0.1:53673/bootstrap");
assert.notEqual(button.onClick, null);
await button.onClick();
assert.deepEqual(navigated, ["https://atcoder.jp/settings"]);
```

**新しいテストが、修正前のコードで落ちることを確認しました。** これが重要です。落ちないテストは、その不具合を検出できていません。

```console
$ node --test scripts/verification/test_atcoder_v12.mjs   # 修正前のコードに対して
✖ V-12 consent page reacts on the dynamic loopback port, not only on the default port
ℹ pass 13
ℹ fail 1
```

## 6. よくある誤解

検討の過程で出た理解と、正しい形を並べます。**どれも自然な誤解です。**

### 誤解1：helperが`origin`を設定していた

| | |
|---|---|
| 誤解 | helperが`new URL(...).origin`のような設定を持っていた |
| 実際 | helperにもそんな設定はない。`new URL(...)`は**挙動を測るために書いた確認用のコード**。実際のコードは、ブラウザが教えてくる`location.origin`をリテラル文字列と比べていた |

### 誤解2：`bootstrap.js`はサーバ側

| | |
|---|---|
| 誤解 | `bootstrap.js`がサーバで、受け付けるポートの設定に不備があった |
| 実際 | `bootstrap.js`は**ブラウザで動くクライアント側のコード**。ポートを待ち受けない。不備があったのは「自分が注入されたページが正しいか」の**自己確認** |

なお、**サーバ側のポートの検査は正しく動いていました。** helperは受け取ったリクエストの`Host`ヘッダーが自分のポートと一致するかを検査しており、固定入力テストの`host_header_must_match_the_bound_port`で確認済みです。

### 誤解3：遷移に失敗して元の画面に戻った

| | |
|---|---|
| 誤解 | 遷移しようとして失敗し、元の画面へ戻った |
| 実際 | **何も起きていない。** `return`が`addEventListener`より手前で走ったため、ボタンにハンドラが存在しなかった。遷移もリロードも発生していない |

ハンドラが付いていれば、押した瞬間にボタンの文字が「認証を準備しています…」へ変わり、その後AtCoderへ移るはずでした。文字が変わらなかったこと自体が、ハンドラの不在を示しています。

### 誤解4：これはSPAの話か

| | |
|---|---|
| 誤解 | 拡張機能が別経路で入るのは、SPAのような描画方式の違いなのではないか |
| 実際 | **軸が違う。** SSRもSPAも「**サイトが自分のUIをどう届けるか**」の選択肢で、どちらも動くJSはサイト自身が指定している。content scriptは「**サイトの外から第三者のコードが入る**」話 |

両者は独立です。組み合わせは4通りありえます。

| | content scriptあり | content scriptなし |
|---|---|---|
| SSR | ありえる | ありえる |
| SPA | ありえる | ありえる |

ちなみに今回の`consent.html`は**どちらでもありません**。helperの実行ファイルに埋め込まれた静的なHTMLを1枚配っているだけです。

### 誤解5：危険な不具合だったのでは

| | |
|---|---|
| 懸念 | 判定が壊れていたなら、Cookieが意図しない相手に渡ったのでは |
| 実際 | **安全側に壊れていた。** 通すべきページを弾いた（厳しすぎた）のであり、通すべきでないページを通したわけではない |

ただし、**審査を通って配信された成果物が本来の機能を果たせない状態で残った**という問題は残ります。

## 7. 持ち帰る教訓

1. **content scriptは、ページの外から注入される第三者のコードである。** だから自分の居場所を確認する処理が要る。そしてその確認自体がバグりうる
2. **`origin`はポートを含む。`hostname`は含まない。** 既定ポートだと`origin`からポートが消えるため、既定ポートで書いたコードは動的ポートで壊れる
3. **同じ書き方が場所によって動いたり動かなかったりする。** `atcoder.js`は443番だから動き、`bootstrap.js`は動的ポートだから動かなかった
4. **「ソースに何が書いてあるか」のテストは、「動くか」のテストではない。** 前者から後者は導けない
5. **新しいテストは、修正前のコードで落ちることを確認する。** 落ちないなら、そのテストはその不具合を検出できていない
6. **審査や検査の通過は、動作の証拠ではない。** `0.1.0`は審査に合格したが、審査担当者も同じ理由で導線を最後まで動かせなかったはずである

## 8. 手元で確かめる

```console
# URLの構成要素を見る
node -e 'const u = new URL("http://127.0.0.1:53673/bootstrap");
  console.log(u.protocol, u.hostname, u.port, u.origin, u.pathname)'

# 既定ポートだと origin からポートが消えることを見る
node -e 'console.log(new URL("http://127.0.0.1/bootstrap").origin)'

# 配られるHTMLに script タグがないことを見る
grep -c "<script" scripts/verification/atcoder_v12/helper/consent.html

# 拡張機能がどのページへ注入を申請しているかを見る
python3 -c "import json;print(json.load(open('scripts/verification/atcoder_v12/extension/manifest.template.json'))['content_scripts'])"

# 修正後のテストを走らせる
node --test scripts/verification/test_atcoder_v12.mjs
```

## 9. 関連文書

| 文書 | 内容 |
|---|---|
| [ADR-0005](../decisions/0005-verify-consent-flow-in-browser-semantics.md) | 判断の記録。前提、採らなかった案、再評価条件 |
| [V-12検証物](../../scripts/verification/atcoder_v12/README.md) | 拡張機能、helper、protocolの仕様 |
| [CWS配布準備](../verification/judge-adapter/v12-chrome-web-store-preparation.md) | 審査と配信の記録 |
| [`TD-43`](../../TODO.md#td-43-検証支援物の実行経路をbrowser相当で確認する範囲を決める) | 教訓4・5をどこまで広げるかを決める作業 |
