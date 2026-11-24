# Contrail: 準リアルタイム航空機動態・ダイバート（代替着陸）予測システム

<div align="center">

[![Code: GPL v3](https://img.shields.io/badge/Code-GPLv3-blue.svg)](LICENSE) [![Data: Non-Commercial](https://img.shields.io/badge/Data-Non--Commercial-orange.svg)](DATA_NOTICE.md)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Docker Ready](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](Dockerfile)
[![Tracked Aircraft](https://img.shields.io/badge/Tracked%20Aircraft-10%2C000%2B-orange)]()
[![Translations](https://img.shields.io/badge/Languages-EN%20%7C%20%E4%B8%AD%E6%96%87%20%7C%20%E6%97%A5%E6%9C%AC%E8%AA%9E-blue)]()

[English](README.md) | [简体中文](README.zh-CN.md) | **日本語**

</div>

---

> [!IMPORTANT]
> **個人のローカル利用に限ります。** 本リポジトリはコードの閲覧と私的な運用のために公開しています。Contrail は運用者本人がローカルかつ私的に利用するために航空機データを収集するものであり、**公開・共有サービスとして運用しないでください。** 第三者へのデータ提供（公開 API、ウェブサイト、アプリ、共有エンドポイント）には、各上流データ提供者（例：The OpenSky Network、adsbdb のルートデータについては David J Taylor）の事前の書面許可が必要です。公開前にご自身で許可を取得してください。詳細は [DATA_NOTICE.md](DATA_NOTICE.md) を参照。

Contrail は、**準リアルタイムの世界航空データを提供する公開 API サービス**です。統一された REST/JSON API で、飛行中の航空機の位置、便名、路線と出発・到着空港、緊急スコーク、およびローカルで算出したダイバート（目的地外着陸）予測を取得できます。

世界中で同時に飛行している約 18,000 機の航空機に対して「1 機ごとに外部 API を都度呼び出す」方式では即座にアクセス制限（レートリミット）に抵触します。Contrail は**「グローバル一括スナップショット取得 + ローカル高度演算・補完 + 安全バッファ + ローカル API 配信」**アーキテクチャを採用しています。直近 3 ヶ月間の飛行ログをローカル SQLite（WAL モード）に自動蓄積し、統一された JSON REST API を提供することで、上流サービスの利用上限を消費することなく、高速・最新の航空データを提供します。

---

## 目次

1. [アーキテクチャと設計思想](#1-アーキテクチャと設計思想)
2. [5 つのコアデータ次元](#2-5-つのコアデータ次元)
3. [上流プロバイダのクォータと安全更新ポリシー](#3-上流プロバイダのクォータと安全更新ポリシー)
4. [自前実装ダイバート予測アルゴリズム](#4-自前実装ダイバート予測アルゴリズム)
5. [汎用 API 仕様およびリクエスト形式](#5-汎用-api-仕様およびリクエスト形式)
6. [API 利用マニュアル (cURL & Python)](#6-api-利用マニュアル-curl--python)
7. [インストールとデプロイ手順 (Docker & ネイティブ)](#7-インストールとデプロイ手順-docker--ネイティブ)
8. [ライセンスとデータの扱い](#8-ライセンスとデータの扱い)

---

## 1. アーキテクチャと設計思想

```
  +------------------+         +-----------------------+
  |  adsb.fi Feed    |         |  OpenSky Network API  |
  | (/v2/snapshot)   |         |     (/states/all)     |
  +--------+---------+         +-----------+-----------+
           |                               |
           +---------------+---------------+
                           |
             [自動フェイルオーバー Fallback Manager]
                           | (35秒〜90秒 安全マージン付きスナップショット)
                           v
           +-------------------------------+
           |    Poller コア処理エンジン    |
           +---------------+---------------+
                           |
     +---------------------+---------------------+
     |                     |                     |
     v                     v                     v
[adsbdb 航路補完]    [OurAirports 地理DB]   [4次元ダイバート予測]
 (トークンバケット 400/分) (世界6,169主要空港)  (降下/偏向/代替場/緊急コード)
     |                     |                     |
     +---------------------+---------------------+
                           |
                           v
            +------------------------------+
            |  ローカル永続化ストレージ    |
            |  - live_flights (現況動態)   |
            |  - flight_snapshots (3ヶ月)  |
            |  - callsign_cache (24h キャッシュ)
            |  - 3ヶ月自動パージデーモン   |
            +--------------+---------------+
                           | (ローカル照会)
                           v
            +------------------------------+
            |    Go 高並列 API Server      |
            |  (airline.api.veyino.org)    |
            +--------------+---------------+
                           |
                +----------+----------+
                |                     |
                v                     v
         [標準 REST クエリ]    [カスタム POST JSON 検索]
```

- **全地球スナップショット取得**: 単一のリクエストで全空域を飛行するすべての航空機の位置、高度、対地速度、トランスポンダコードを取得します。
- **ローカル非同期航路補完**: コールサインに基づく発着空港・便名は `adsbdb` から取得し、トークンバケット（$\le 400$ 回/分）で厳格に流量制限します。結果は 24 時間ローカルキャッシュされ、チャーター機等の未登録コールサインもネガティブキャッシュすることで再試行スパムを完全防止します。
- **データローテーション**: 過去の全飛行スナップショットはローカルに 3 ヶ月間（90 日間）保持され、バックグラウンドデーモンによって自動削除されます。
- **外部 API へのゼロ負荷**: すべてのエンドユーザー・クライアントは Contrail のローカルサーバーと通信するため、どれだけ高並列にクエリを発行しても上流ノードに負荷をかけません。

---

## 2. 5 つのコアデータ次元

Contrail が出力する統一フライトオブジェクトは、以下の 5 大次元を標準網羅しています：

| 次元 | フィールド名 | 型 | 説明 |
| :--- | :--- | :--- | :--- |
| **1. 航空機・便名識別** | `aircraft.hex` | string | ICAO 24ビット 16進数トランスポンダID（例: `39de4a`） |
| | `flight.callsign_icao` | string | ICAO コールサイン（例: `CES5101`, `AAL1`） |
| | `flight.callsign_iata` | string | 商業便名 IATA コード（例: `MU5101`, `AA1`） |
| | `flight.airline` | object | 運航航空会社情報（名称、ICAO、IATA、国籍） |
| **2. 航路・発着空港** | `route.origin` | object | 出発地空港（ICAO、IATA、空港名、都市、国、経緯度、標高） |
| | `route.destination` | object | 目的地空港（ICAO、IATA、空港名、都市、国、経緯度、標高） |
| | `route.summary` | string | 航路概要ラベル（例: `SHA -> PEK`） |
| **3. テレメトリ（運動情報）** | `telemetry.position` | object | 経緯度座標 (`latitude`, `longitude`) |
| | `telemetry.altitude` | object | 気圧高度 (`barometric_ft`) および 幾何高度 |
| | `telemetry.speed` | object | 対地速度（ノット `ground_speed_kts` および `km/h`） |
| | `telemetry.attitude` | object | 実航跡方位角 (`track_deg`) および 昇降率 (`vertical_rate_fpm`) |
| | `telemetry.on_ground`| bool | 地上設置（着陸）フラグ |
| **4. 緊急トランスポンダコード** | `emergency.squawk` | string | 4桁のトランスポンダスコークコード (`7700`, `7600`, `7500`, `1200`) |
| | `emergency.is_emergency` | bool | 緊急事態フラグ |
| | `emergency.category` | string | 緊急区分 (`GENERAL_EMERGENCY`, `RADIO_FAILURE`, `HIJACK`, `NONE`) |
| | `emergency.description` | string | 人間が可読可能な緊急事態診断メッセージ |
| **5. ダイバート予測** | `diversion_analysis.predicted_diversion` | bool | 代替着陸予測フラグ（2つ以上の条件充足で true） |
| | `diversion_analysis.probability_level` | string | 確率判定 (`NONE`, `LOW`, `MODERATE`, `HIGH`, `CRITICAL`) |
| | `diversion_analysis.distance_to_dest_km`| float | 予定目的地までの大圏距離 (km) |
| | `diversion_analysis.bearing_to_dest_deg`| float | 予定目的地への理論方位角 (0–360°) |
| | `diversion_analysis.heading_deviation_deg`| float | 進行方向と目的地方位の角度差 (0–180°) |
| | `diversion_analysis.triggered_criteria`| array | 該当した判定条件および観測数値リスト |
| | `diversion_analysis.candidate_alternate`| object | 進入針路上の候補代替着陸空港（空港名、距離、角度差） |

---

## 3. 上流プロバイダのクォータと安全更新ポリシー

| プロバイダ / エンドポイント | 公式上限値 | Contrail 本番ポーリング周期 | 安全バッファ設計 | 商用利用 |
| :--- | :--- | :--- | :--- | :--- |
| **adsb.fi フィーダー専用**<br>`/v2/snapshot` | 30 秒に 1 回 | **35 秒 / 回** | +5秒のジッター・ドリフト余白。未承認(403)時は即座に OpenSky へ自動降格 | **厳格禁止** |
| **OpenSky (アクティブフィーダー)**<br>`/api/states/all` | 8,000 クレジット/日<br>$\approx$ 43.2 秒に 1 回 | **50 秒 / 回** | 突発的なリトライに備え 15% のクォータ余白を確保 | 認可制 / **非商用** |
| **OpenSky (登録ユーザー)**<br>`/api/states/all` | 4,000 クレジット/日<br>$\approx$ 86.4 秒に 1 回 | **95 秒 / 回** | 1日約 3,600 クレジット消費（安全マージン 10%） | 認可制 / **非商用** |
| **OpenSky (匿名アクセス)**<br>`/api/states/all` | IP アドレス単位制限 | **60 〜 90 秒 / 回** | `429` 検出時に `Retry-After` に基づく動的指数バックオフ | 認可制 / **非商用** |
| **adsbdb 便名照会**<br>`/v0/callsign/{callsign}` | 512 回 / 分 | **トークンバケット 400 回 / 分** | 22% の安全マージン。24 時間正キャッシュ、6 時間負キャッシュ、航空会社フォーマット優先 | **厳格禁止** |
| **OurAirports**<br>`airports.csv` | 静的オープンデータ | **バイナリ内包・事前索引**<br>(起動時即座に利用可能) | 世界 6,169 主要空港の 2度単位グリッド空間索引 | パブリックドメイン |

---

## 4. 自前実装ダイバート予測アルゴリズム

航空管制上の「ダイバート中」を直接返す無料 API は存在しません。Contrail は空間幾何学と飛行運動力学に基づきローカルでリアルタイム推算します：

1. **目的地から遠方での異常降下 (`DESCENT_FAR_FROM_DEST`)**:
   - 高度が $10,000\text{ ft}$ 未満；
   - 降下率が継続して $v_{\text{vertical}} < -500\text{ ft/min}$；
   - 予定目的地までの大圏距離がなお $> 150\text{ km}$ 残存。
2. **持続的な針路偏向 (`HEADING_DEVIATION`)**:
   - 目的地から $> 100\text{ km}$ 離れた状態で、目的地方位角との偏向角が $\Delta\theta > 60^\circ$；
   - 連続 3 回以上のスナップショット（$\ge 90\text{s}$）で継続維持（アプローチ誘導や通常のウェイポイント旋回を除外）。
3. **代替着陸候補空港への降下進入 (`APPROACHING_ALTERNATE_AIRPORT`)**:
   - 目的地から $> 80\text{ km}$ 離れた状態で評価；
   - 降下中（降下率 $< -400\text{ ft/min}$）；
   - OurAirports データベース上で $80\text{ km}$ 以内に位置し、かつ機体の進行方位と空港方位の角度差が $\le 20^\circ$ 以内。
4. **緊急スコークコードの設定 (`EMERGENCY_SQUAWK`)**:
   - トランスポンダが `7700`（メーデー/緊急）、`7600`（無線機故障）、`7500`（ハイジャック）を発信。

### 誤判定防止およびデータ健全性担保

- **目的地進入除外**: 目的地から $40\text{ km}$ 以内、または高度 $3,000\text{ ft}$ 未満かつ目的地から $60\text{ km}$ 以内の場合は通常着陸進入と判定し、**ダイバートとは判定しません**。
- **同一空港発着除外**: 出発地と目的地が同一（$\text{ICAO}_{\text{origin}} = \text{ICAO}_{\text{dest}}$）の飛行はダイバート解析から除外されます。
- **失効機体の自動除外**: 5 分以上信号が途絶えた機体はポーリング周期ごとにリアルタイム DB から自動消去されます。
- **異常値フィルタリング**: 高度 $> 60,000\text{ ft}$、対地速度 $> 750\text{ kts}$ のセンサーノイズを自動除外。
- **IATA便名の真正性**: 航空会社プレフィックスが一致した場合のみ IATA 便名を導出し、未確認機は厳格に `null` を返却。
- **決定的ページネーション**: 最大 `limit=1000` をサポートし、`hex ASC` 二次ソートによりページ跨ぎの重複を防止。
- **厳格な RFC 3339 UTC**: 全てのタイムスタンプを ISO 8601 / RFC 3339（`YYYY-MM-DDTHH:MM:SSZ`）形式で統一。

**判定ルール**: 上記 4 条件のうち **2 つ以上を満たした場合** に `predicted_diversion = true` と判定します。`7700` を含む場合は `CRITICAL`、3 条件以上は `HIGH`、2 条件は `MODERATE` と評価します。

---

## 5. 汎用 API 仕様およびリクエスト形式

公共共通 Base URL: `https://airline.api.veyino.org`

- `GET /api/v1/health` - サーバー稼働状態、Poller 受信ステータス、監視機体数
- `GET /api/v1/flights` - リアルタイム航空機動態一覧（コールサイン、空港、高度、BBox 絞り込み）
- `GET /api/v1/flights/{identifier}` - 単一機の詳細運動情報および航路解析
- `GET /api/v1/emergencies` - 緊急スコーク（7700/7600/7500）発信中の機体一覧
- `GET /api/v1/diversions` - ダイバート予測判定中の機体および候補代替空港一覧
- `GET /api/v1/airports/{code}` - 空港情報および発着機体状況
- `GET /api/v1/stats` - 全地球の飛行概況指標
- `POST /api/v1/query` - **高度なカスタム複合 JSON 検索エンジン**

---

### カスタム POST JSON 検索の例 (`POST /api/v1/query`)

```json
{
  "filters": {
    "is_diverting": true,
    "min_altitude_ft": 3000,
    "max_altitude_ft": 40000,
    "bounding_box": {
      "min_lat": 20.0,
      "min_lon": 120.0,
      "max_lat": 50.0,
      "max_lon": 150.0
    }
  },
  "sort_by": "last_seen",
  "sort_order": "desc",
  "limit": 10,
  "page": 1
}
```

---

## 6. API 利用マニュアル (cURL & Python)

```bash
# 1. ダイバート中の機体を照会
curl -s "https://airline.api.veyino.org/api/v1/diversions" | jq .

# 2. 緊急事態機（7700/7600/7500）を照会
curl -s "https://airline.api.veyino.org/api/v1/emergencies" | jq .

# 3. 複合 JSON 検索を実行
curl -s -X POST "https://airline.api.veyino.org/api/v1/query" \
  -H "Content-Type: application/json" \
  -d '{"filters":{"callsign_prefix":"ANA","is_diverting":false},"limit":5}' | jq .
```

---

## 7. インストールとデプロイ手順 (Docker & ネイティブ)

### Docker Compose によるデプロイ（推奨）

```bash
git clone https://github.com/ysnsk111/contrail.git
cd contrail
cp .env.example .env
docker compose up -d --build
docker compose logs -f
```

---

## 8. ライセンスとデータの扱い

- **コード**：GNU General Public License v3.0（[LICENSE](LICENSE)）。
- **データ**：Contrail は第三者のデータを収集し、そのまま再配布します。データは GPL の対象**外**で、各提供元の規約に従います（非商用のみ、クレジット表示必須、一部の用途は提供元の書面許可が必要）。詳しくは [DATA_NOTICE.md](DATA_NOTICE.md) を参照してください。
- **クレジット**：The OpenSky Network · adsb.fi · adsbdb.com（路線データ：David Taylor、Jim Mason）· OurAirports。
