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

## Usage (API Request)

エンドポイントは `/api/v2/traffic` 以下に存在します。

サーバー処理上の問題がある場合、リクエストの種類を問わず 500 Internal Server Error を返す可能性があります。

### GET `/station?keyword=`

駅データ一覧を取得します。
クエリパラメータ`keyword`に部分一致する駅を取得します。

- Responses
    - 200 OK
        ```json
        {
            "stations": [
                {
                    "id": 1,
                    "name": "候補駅名",
                    "name_en": "Candidate station name"
                }
            ]
        }
        ```

    - Error

        | Status code | error | 説明 |
        |-------------|-------|------|
        | 400 | Keyword must be specified. | `keyword`クエリパラメータの指定が必要ですが、指定されていません。 |

### GET `/station/:id`

駅IDをパスパラメータにとり、該当する駅情報を1件取得します。

- Responses
    - 200 OK
        ```json
        {
            "id": 1,
            "name": "駅名",
            "name_en": "Station name"
        }
        ```

    - Errors

        | Status code | error | 説明 |
        |-------------|-------|------|
        | 400 | Invalid Request. | パスに設定された駅IDは、0以上の整数である必要があります。 |
        | 404 | Station not found. | パスに設定されたIDの駅は、DBに登録されていません。 |

### POST `/search`

乗り換え検索を行います。

- Request
    ```json
    {
        "depart_station_name": "出発駅名",
        "depart_station_id": 1,
        "depart_datetime": "2024-10-01T10:30:00+09:00",
        "arrive_station_name": "到着駅名",
        "arrive_station_id": 2,
        "depart_datetime": "2024-10-10T14:30:00+09:00"
    }
    ```
    - 出発駅指定 `depart_station_name`/`depart_station_id`のどちらか片方を指定します。
    - 到着駅指定 `arrive_station_name`/`arrive_station_id`のどちらか片方を指定します。
    - 出発・到着日時指定 `depart_datetime`/`arrive_datetime`のどちらか片方をISO8601で指定します。タイムゾーンは、自動で日本標準時(JST)に変換されます。

- Responses
    - 200 OK
        ```json
        {
            "stations": [
                {
                    "id": 1,
                    "name": "経由駅名",
                    "name_en": "Via station name"
                }
            ],
            "routes": [
                {
                    "operations": [
                        {
                            "train_id": 1,
                            "order": 1,
                            "depart_station_id": 1,
                            "depart_datetime": "2024-10-01T10:30:00+09:00",
                            "arrive_station_id": 2,
                            "arrive_datetime": "2024-10-01T10:40:00+09:00"
                        }
                    ]
                }
            ]
        }
        ```

        - `stations`は、`routes`内で使用する駅のみの情報をID順に返します。
        - `routes`は、複数のルート候補で構成されます。デフォルトでは5件を上限としています。
        - `routes`の1要素(route)は、複数の時系列順にソートされたoperation(`operations`)で構成されます。
        - `train_id`, `order`は今後問い合わせ機能を実装した際に使用します。

    - Errors
        | Status code | error | 説明 |
        |-------------|-------|------|
        | 400 | Parameters are missing. | 必要なJSONパラメータが与えられていません。 |
        | 400 | Either the departure time or the arrival time must be set, but not both. | `depart_datetime`/`arrive_datetime`の両方が指定されているか、まったく指定されていません。 |
        | 400 | Eithor the departure station name or the departure station id must be set, but not both. | `depart_station_name`/`depart_station_id`の両方が指定されているか、まったく指定されていません。 |
        | 400 | Eithor the arrive station name or the arrive station id must be set, but not both. | `arrive_station_name`/`arrive_station_id`の両方が指定されているか、まったく指定されていません。 |
        | 400 | Eithor the arrive station name or the arrive station id must be set, but not both. | 一時的な問題です。到着時刻を指定した経路探索がメンテナンス中のため、利用できません。(今後対処予定です。) |
        | 400 | Error resolving departure station name. | `depart_station_name`の名前解決に失敗しました。指定された駅が存在しないか、複数候補が存在します。 |
        | 400 | Error resolving arrive station name. | `arrive_station_name`の名前解決に失敗しました。指定された駅が存在しないか、複数候補が存在します。 |
        | 400 | Departure station ID and arrival station ID must be different. | 出発駅と到着駅は異なっている必要があります。 |
        | 400 | Invalid depart station ID. | 指定された`depart_station_id`は存在しません。 |
        | 400 | Invalid arrive station ID. | 指定された`arrive_station_id`は存在しません。 |

## API Sample

[サンプルページ](https://outtech105.com/api/v2/traffic/)でリクエスト可能です。(メンテナンス中等、接続できない場合もあります)

## Author

Outtech105k

## License

[MIT License](/LICENSE) を適用します。
