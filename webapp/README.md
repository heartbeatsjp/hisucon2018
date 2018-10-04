# Web アプリケーション

## 概要

HISUCON2018 の Web アプリケーション環境です。

## 構築

- ansible playbook 内の `CHANGEME` を適切に変更ください。
- 実行

    ```
    $ cd ansible
    $ ansible-playbook site.yml -i inventory/prd 
    ```

## 環境

nginx + Python + Flask + MySQL


## 動作確認

### Web アプリケーション画面

http://xx.xx.xx.xx/ にて動作確認

- ログインアカウント
    - 管理者
        - user
            - suzuki
        - pass
            - suzuki201808
    - 一般ユーザ
        - user
            - sato
        - pass
            - sato201808
    - 他のユーザも同様にユーザ名に 201808 を付与したものがパスワードになります。

### Web アプリケーション

- Web アプリケーションは /home/hisucon/webapp に配置してます。
- 注意事項
    - 以下ディレクトリ配下は初期化用です。絶対に編集しないでください。
        - /home/hisucon/webapp/app/edu_daily_org/
        - /home/hisucon/webapp/app/static_org/
        - /home/hisucon/webapp/app/sql/
- 起動/停止コマンド
    - 起動

        ```
        $ sudo systemctl start hisucon2018-webapp
        ```
    - 停止

        ```
        $ sudo systemctl stop hisucon2018-webapp
        ```

### MySQL

3306 番ポートで MySQL が起動しています。初期状態では以下のユーザが設定されています。

- root ユーザ
    - ユーザ名
        - root
    - パスワード
        - DQCjL6Hl9HOY1Jnf#
- hisucon ユーザ
    - ユーザ名
        - hisucon
    - パスワード
        - KCgC6LtWKp5tpKkW#

- ログイン

    ```
    $ mysql -u root -p
    ```

