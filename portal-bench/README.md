# ポータル画面、ベンチマーク実行、Grafana 画面

## 概要

- HISUCON2018 用のポータル画面、ベンチマーク実行、Grafana 画面を提供します。

## 構築

- ansible playbook 内の `CHANGEME` を適切に変更ください。
- 実行

    ```
    $ cd ansible
    $ ansible-playbook site.yml -i inventory/prd
    ```

## 環境

### ポータル画面

- 機能
    - ベンチマーク実行結果を表示する画面
    - ベンチマーク実行をキューに登録

- 実行画面
    - http://xx.xx.xx.xx/top/[team]/[ip-address]
        - [team] 
            - チーム名を入力
        - [ip-address]
            - ローカル IP アドレスの入力
            - ※ プライベートネットワークを使用しない場合には、グローバル IP アドレスを指定してください。また `/srv/webapp/main.go` の L37 にてプライベート IP 制限をかけてますので、こちらの修正もあわせてお願いします。

- プロセス操作
    - 起動

        ```
        $ sudo systemctl start hisucon2018-portal
        ```
    - 停止

        ```
        $ sudo systemctl stop hisucon2018-portal
        ```
 
- 注意点
    - ベンチマーク実行ボタン押下後はキューに入ります。想定では 1 分 30 秒程度で結果反映されますが、同時実行数が多い場合には結果が反映されるまでに時間がかかります。
    - 結果反映が分かるように、ポータル画面は 10 秒ごとに画面を reload してます
    - 初期スコアは平均 2500 程度となってます


### ベンチマーク実行

- 機能
    - キューを取得してベンチマークを実行

- プロセス操作
    - 起動

        ```
        $ sudo systemctl start hisucon2018-bench
        ```
    - 停止

        ```
        $ sudo systemctl stop hisucon2018-bench
        ```
 
- 実装は下記を利用し、HISUCON2018 用に修正しました。
    - [ISUCON7 予選問題の参照実装とベンチマーカー](https://github.com/isucon/isucon7-qualify)
    - ビルド方法等は上記をご確認ください。


### Grafana 画面

- 機能
    - スコア遷移を表示
- データソースは MySQL を指定し、Query は下記を登録してください。

    ```
    SELECT
      UNIX_TIMESTAMP(created_at) as time_sec,
      cast(result->"$.score[0]" as SIGNED) as value,
      team as metric
    FROM bench
    WHERE $__timeFilter(created_at)
    ORDER BY time_sec ASC
    ```


