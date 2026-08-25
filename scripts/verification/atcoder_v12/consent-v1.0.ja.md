# AlgoLoom V-12 AtCoder認証の同意事項（版1.0）

この同意は、製品実装ではなく`V-12`技術検証のためのものです。

- 専用の通常Google Chromeで、利用者自身がAtCoderへのログインと必要なTurnstileを操作します。AlgoLoomはパスワード、Turnstile、ログイン操作を取得・自動化しません。
- 限定公開の検証用拡張機能は、`https://atcoder.jp/settings`で本人アカウントを確認した後、`REVEL_SESSION` Cookieを厳密に1件だけ取得し、同じ端末の`127.0.0.1`で待つ検証用helperへ一度だけ渡します。
- helperは認証状態と本人一致を確認するため、`https://atcoder.jp/settings`へGETを2回まで行います。POST、提出ページの送信、提出、再試行は行いません。
- 確認できたCookieだけを、検証campaign専用のmacOS Keychain項目へ一時保存します。平文ファイル、通常ログ、Git、匿名化済み成果物には保存しません。
- 同意しない場合または途中で中止する場合は、同意ボタンを押さずにChromeを閉じます。中止した実行用profile、待受処理、未確認sessionは回収します。
- campaign終了時に、基準template、store用一時情報、検証用Keychain項目を削除します。ローカルでCookieを削除しても、AtCoder側のsessionを失効させたことにはなりません。

同意画面の「同意してAtCoderへ進む」は、この版の内容に対する明示操作です。拡張機能の追加時にChromeが表示する権限確認や、AtCoder自身のログイン操作に対する同意を代替しません。
