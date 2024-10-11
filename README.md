# Transit Server

## Overview

- 乗り換え案内を提供するREST APIサーバです。
- DBに任意の駅情報・時刻情報を保存することで、架空鉄道の乗り換え案内も設定できます。

## Get started

1. Docker実行環境にクローン
1. `cp db_sec.env.sample db_sec.env` で、設定ファイルをコピーし、パスワードを設定
(WEBサーバからはユーザ`transit_serv`としてアクセスします)
1. [compose.yaml](/compose.yaml) の接続ポートを必要に応じて変更
1. `docker compose up -d`でサーバ実行

## Author

Outtech105k

## License

MIT License
