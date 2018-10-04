# ConoHa 環境構築

## 概要

- サーバ構築手順となります。
- 最小構成は 2 台です。
    - Web アプリケーション
    - ポータル画面、ベンチマーク実行、Grafana 画面

## 構築

### Terraform の利用

- ConoHa 管理画面より API 情報を確認ください。
- `CHANGEME` を適切に変更してください。
- ドライラン

    ```
    $ terraform plan
    ```

- 適用

    ```
    $ terraform apply
    ```

### 管理画面より作成

- リージョン
    - 東京
- サービス
    - 2 GB プラン
- イメージタイプ
    - CentOS7.5(64 bit)
- オプション
    - 接続許可ポート IPv4 にて SSH(22)、Web(20/21/80/443) のみ許可

## 参照

- Terraform
    - [OpenStack Provider](https://www.terraform.io/docs/providers/openstack/index.html)
- ConoHa API
    - [ConoHa API Documantation](https://www.conoha.jp/docs/)

