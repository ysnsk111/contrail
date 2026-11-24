# Contrail: 准实时全球航班数据与智能备降预测系统

<div align="center">

[![Code: GPL v3](https://img.shields.io/badge/Code-GPLv3-blue.svg)](LICENSE) [![Data: Non-Commercial](https://img.shields.io/badge/Data-Non--Commercial-orange.svg)](DATA_NOTICE.md)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Docker Ready](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](Dockerfile)
[![Tracked Aircraft](https://img.shields.io/badge/Tracked%20Aircraft-10%2C000%2B-orange)]()
[![Translations](https://img.shields.io/badge/Languages-EN%20%7C%20%E4%B8%AD%E6%96%87%20%7C%20%E6%97%A5%E6%9C%AC%E8%AA%9E-blue)]()

[English](README.md) | **简体中文** | [日本語](README.ja-JP.md)

</div>

---

> [!IMPORTANT]
> **仅限个人本地私用。** 本仓库公开仅供查看和私下部署代码。Contrail 只用于为部署者本人收集航班数据并在本地私人使用，**不得作为公开或共享服务运行**。如需向他人提供数据（公开 API、网站、应用或共享接口），必须事先取得各上游数据提供方的书面许可（例如 OpenSky Network，以及 adsbdb 航线数据的 David J Taylor），请自行联系上游获取。详见 [DATA_NOTICE.md](DATA_NOTICE.md)。

Contrail 是一个**自托管的准实时全球航班数据采集器与本地 API**，仅供个人私用。通过统一的 REST/JSON 接口，可查询在空飞机的实时位置、航班号、航线与起降机场、紧急应答机代码，以及本地计算的备降预测。

针对全球同时在空约 1.8 万架飞机的现实场景，系统摒弃了“单机轮询 API”的限额自毁模式，采用**“拉取全球快照 + 本地富化推算 + 缓存削峰 + 本地 API 分发”**的核心架构。本地自动维护最近 3 个月的飞行轨迹快照，并通过基于 Go 语言的 REST/JSON 服务对外提供统一调用，调用方的请求不会消耗上游数据源的配额。

---

## 目录

1. [核心架构与设计原则](#1-核心架构与设计原则)
2. [五大核心数据与字段集](#2-五大核心数据与字段集)
3. [上游数据源配额与安全刷新政策](#3-上游数据源配额与安全刷新政策)
4. [本地智能备降预测算法](#4-本地智能备降预测算法)
5. [通用 API 规范与请求格式](#5-通用-api-规范与请求格式)
6. [API 调用手册 (cURL & Python)](#6-api-调用手册-curl--python)
7. [安装与部署教程 (Docker & 原生)](#7-安装与部署教程-docker--原生)
8. [环境变量配置参考](#8-环境变量配置参考)
8. [许可与数据合规](#8-许可与数据合规)

---

## 1. 核心架构与设计原则

```
  +------------------+         +-----------------------+
  |  adsb.fi Feed    |         |  OpenSky Network API  |
  | (/v2/snapshot)   |         |     (/states/all)     |
  +--------+---------+         +-----------+-----------+
           |                               |
           +---------------+---------------+
                           |
            [智能故障转移 Fallback Manager]
                           | (35s ~ 90s 带缓冲安全快照)
                           v
           +-------------------------------+
           |    Poller 核心数据处理引擎    |
           +---------------+---------------+
                           |
     +---------------------+---------------------+
     |                     |                     |
     v                     v                     v
[adsbdb 路由富化]    [OurAirports 地理库]  [4 维度备降预测]
  (令牌桶 400/min)     (6,169 全球中大型机场)  (下降/航偏/备降场/紧急码)
     |                     |                     |
     +---------------------+---------------------+
                           |
                           v
            +------------------------------+
            |  本地持久化存储 (SQLite WAL) |
            |  - live_flights (当前态势)   |
            |  - flight_snapshots (历史)   |
            |  - callsign_cache (24h 缓存) |
            |  - 3 个月过期自动清理 Worker |
            +--------------+---------------+
                           | (本地查询)
                           v
            +------------------------------+
            |      Go 高并发 API Server    |
            |  (airline.api.veyino.org)    |
            +--------------+---------------+
                           |
                +----------+----------+
                |                     |
                v                     v
         [标准 REST 查询]     [自定义 POST JSON 引擎]
```

- **全球快照拉取**：单次拉取包含全空域所有正在飞行的飞机位置、高度、速度与 squawk 代码。
- **本地异步富化**：航线与起降机场通过 `adsbdb` 按呼号获取，并由令牌桶严格限制在 400 次/分钟以下，结果持久化缓存 24 小时；未知呼号（包机、公务机等）标记后同样缓存 24 小时，彻底避免反复请求撞限。
- **本地数据库缓存**：所有实时态势存入本地 SQLite（WAL 模式），历史轨迹保留 3 个月（90 天），后台守护进程定期自动清理过期数据。
- **上游零侵扰对外分发**：外部所有用户和系统直接查询 Contrail 本地 API，无论并发多高均不消耗上游 API 配额。

---

## 2. 五大核心数据与字段集

本系统输出的标准统一航班对象包含以下 5 大核心维度字段：

| 维度 | 字段名称 | 类型 | 说明 |
| :--- | :--- | :--- | :--- |
| **1. 航班识别** | `aircraft.hex` | string | ICAO 24 位应答机十六进制识别码 (如 `39de4a`) |
| | `flight.callsign_icao` | string | ICAO 呼号 (如 `CES5101`, `AAL1`) |
| | `flight.callsign_iata` | string | IATA 商业航班号 (如 `MU5101`, `AA1`) |
| | `flight.airline` | object | 运营航空公司信息 (名称、ICAO、IATA、所属国家) |
| **2. 航线起降** | `route.origin` | object | 起飞机场 (ICAO、IATA、名称、城市、国家、经纬度、标高) |
| | `route.destination` | object | 目的机场 (ICAO、IATA、名称、城市、国家、经纬度、标高) |
| | `route.summary` | string | 航线摘要 (如 `SHA -> PEK`) |
| **3. 遥测状态** | `telemetry.position` | object | 经纬度坐标 (`latitude`, `longitude`) |
| | `telemetry.altitude` | object | 气压高度 (`barometric_ft`) 与几何高度 |
| | `telemetry.speed` | object | 地速 (节 `ground_speed_kts` 与 `km/h`) |
| | `telemetry.attitude` | object | 航向 (`track_deg`) 与垂直速度 (`vertical_rate_fpm`) |
| | `telemetry.on_ground`| bool | 是否位于地面 |
| **4. 紧急代码** | `emergency.squawk` | string | 4 位应答机应答编码 (如 `7700`, `7600`, `7500`, `1200`) |
| | `emergency.is_emergency` | bool | 是否处于紧急状态 |
| | `emergency.category` | string | 紧急级别 (`GENERAL_EMERGENCY`, `RADIO_FAILURE`, `HIJACK`, `NONE`) |
| | `emergency.description` | string | 紧急情况详细语义说明 |
| **5. 备降预测** | `diversion_analysis.predicted_diversion` | bool | 是否预测为正在备降 (满足 >=2 条判定准则) |
| | `diversion_analysis.probability_level` | string | 备降概率评估 (`NONE`, `LOW`, `MODERATE`, `HIGH`, `CRITICAL`) |
| | `diversion_analysis.distance_to_dest_km`| float | 当前距计划目的地大圆距离 (km) |
| | `diversion_analysis.bearing_to_dest_deg`| float | 指向计划目的地的理论航向角 (0-360°) |
| | `diversion_analysis.heading_deviation_deg`| float | 当前实际航迹与目的地航向的偏角差 (0-180°) |
| | `diversion_analysis.triggered_criteria`| array | 触发的具体备降判定依据列表及测量值 |
| | `diversion_analysis.candidate_alternate`| object | 正在对正并下降的目标备降候选机场信息 |

---

## 3. 上游数据源配额与安全刷新政策

Contrail 在设计中严格贴近上游提供商的最高限额，并留出足够的安全缓冲区（Safe Buffer），防止因网络抖动、时钟漂移或重试导致 IP 被上游暂时封禁。

| 提供商 / 端点 | 官方最大额度 | Contrail 生产刷新策略 | 安全缓冲说明 | 商用限制 |
| :--- | :--- | :--- | :--- | :--- |
| **adsb.fi 接收站专属**<br>`/v2/snapshot` | 1 次 / 30 秒 | **35 秒 / 次** | 保留 5 秒时钟抖动余量；若非 Feeder IP（返回 403）自动平滑降级至 OpenSky | **严禁商用** |
| **OpenSky Network (Feeder)**<br>`/api/states/all` | 8,000 点数/天<br>≈ 43.2 秒/次 | **50 秒 / 次** | 留出约 15% 点数余额，确保突发重试不超配额 | 需授权 / **严禁商用** |
| **OpenSky Network (注册用户)**<br>`/api/states/all` | 4,000 点数/天<br>≈ 86.4 秒/次 | **95 秒 / 次** | 全天消耗约 3,600 点数，安全裕度 10% | 需授权 / **严禁商用** |
| **OpenSky (匿名/无秘钥)**<br>`/api/states/all` | IP 动态限额桶 | **60 ~ 90 秒 / 次** | 自适应退避，捕获 429 自动按照 `Retry-After` 退避 | 需授权 / **严禁商用** |
| **adsbdb 呼号查询**<br>`/v0/callsign/{callsign}` | 512 次 / 分钟 | **令牌桶 400 次 / 分钟** | 留出 22% 安全空间；24 小时正缓存，6 小时负缓存，优先查航司格式呼号 | **严禁商用** |
| **OurAirports**<br>`airports.csv` | 静态 GitHub Pages | **系统内置预载入**<br>(启动零依赖) | 本地内置 6,169 座全球中大型机场空间网格索引，微秒级匹配 | 开放数据 (公有领域) |

---

## 4. 本地智能备降预测算法

真实航空运行中没有任何免费公用 API 直接给出“正在备降”的实时标志。Contrail 依靠多维空间几何与飞行动力学，在本地实时推算备降状态：

### 判定条件集合

1. **远距非正常下降 (`DESCENT_FAR_FROM_DEST`)**：
   - 飞机已降至 $10,000\text{ ft}$ 以下；
   - 垂直速度持续为负且降率 $v_{\text{vertical}} < -500\text{ ft/min}$；
   - 距离计划目的地大圆距离仍 $> 150\text{ km}$。
2. **大角度航向偏离 (`HEADING_DEVIATION`)**：
   - 距离目的地 $> 100\text{ km}$ 且当前航迹与目的地真实方位角的角偏差 $\Delta\theta > 60^\circ$；
   - 该偏差在连续 3 次以上快照中持续维持（持续时间 $\ge 90\text{s}$），排除近场进近、五边盘旋或航路转向。
3. **对正备降候选机场 (`APPROACHING_ALTERNATE_AIRPORT`)**：
   - 距离计划目的地 $> 80\text{ km}$；
   - 处于下降状态（降率 $< -400\text{ ft/min}$）；
   - 从 OurAirports 全球机场库中匹配到非起降场的候选机场，距离 $< 80\text{ km}$，且飞机航向正对该机场（夹角 $\le 20^\circ$）。
4. **紧急代码激活 (`EMERGENCY_SQUAWK`)**：
   - 应答机设置为 `7700`（遇险/紧急）或 `7600`（无线电失效）或 `7500`（被劫持）。

### 防误报与数据完整性保障

- **目的地近场豁免**：距离目的地 $\le 40\text{ km}$ 或高度 $< 3,000\text{ ft}$ 且距离目的地 $\le 60\text{ km}$ 时，属于正常进近着陆，**不判定为备降**。
- **本场起降/环飞过滤**：起降相同（$\text{ICAO}_{\text{origin}} = \text{ICAO}_{\text{dest}}$）的飞行任务不参与备降分析。
- **过期飞机自动清理**：超过 5 分钟未收到信号的飞机在每个轮询周期自动移出实时数据库。
- **异常值过滤**：过滤高度 $> 60,000\text{ ft}$ 与地速 $> 750\text{ kts}$ 的传感器脏数据。
- **IATA 航班号保真**：仅匹配成功航司格式时生成 IATA 航班号，非航司航班严格返回 `null`，杜绝伪造。
- **稳定分页与排序**：支持最高 `limit=1000` 并引入 `hex ASC` 辅排序键，彻底杜绝翻页丢数据或跨页重复。
- **RFC 3339 标准时间戳**：全部时间戳严格格式化为 UTC ISO 8601 / RFC 3339（`YYYY-MM-DDTHH:MM:SSZ`）。

### 判决矩阵

- **`predicted_diversion = true`**：满足上述条件中的 **2 条或以上**；
- 若激活 `7700` 紧急码，风险等级直接标记为 **`CRITICAL`**；
- 满足 3 条以上条件，风险等级标记为 **`HIGH`**；
- 满足 2 条条件，风险等级标记为 **`MODERATE`**；
- 单一条件或无异常标记为 **`LOW` / `NONE`**。

---

## 5. 通用 API 规范与请求格式

通用 API 公共基地址：`https://airline.api.veyino.org`

### 接口总览

| 方法 | 端点 | 功能说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/health` | 服务健康检查、Poller 抓取状态与数据库统计 |
| `GET` | `/api/v1/flights` | 全球实时航班列表（支持呼号、机场、高度、范围、状态过滤） |
| `GET` | `/api/v1/flights/{identifier}` | 按 ICAO 24位 hex 或呼号单架飞机全面状态精细查询 |
| `GET` | `/api/v1/emergencies` | 全球正在宣布紧急代码（7700/7600/7500）的航班专区 |
| `GET` | `/api/v1/diversions` | 全球系统预测处于疑似备降状态的航班专区 |
| `GET` | `/api/v1/airports/{code}` | 查询指定机场详情及当前进出港航班 |
| `GET` | `/api/v1/stats` | 全球在空飞行总体概览指标 |
| `POST`| `/api/v1/query` | **自定义 JSON 高级组合查询引擎** |

---

### 自定义 JSON 查询格式 (`POST /api/v1/query`)

客户端可发送结构化 JSON 请求体进行多条件组合查询：

```json
{
  "filters": {
    "callsign_prefix": "CES",
    "is_emergency": false,
    "is_diverting": true,
    "min_altitude_ft": 5000,
    "max_altitude_ft": 35000,
    "origin_icao": "ZSSS",
    "destination_icao": "ZBAA",
    "bounding_box": {
      "min_lat": 20.0,
      "min_lon": 110.0,
      "max_lat": 40.0,
      "max_lon": 125.0
    }
  },
  "sort_by": "last_seen",
  "sort_order": "desc",
  "limit": 50,
  "page": 1
}
```

---

## 6. API 调用手册 (cURL & Python)

### 1. 查询所有紧急代码航班 (Squawk 7700/7600/7500)

```bash
curl -s "https://airline.api.veyino.org/api/v1/emergencies" | jq .
```

### 2. 查询所有预测备降航班

```bash
curl -s "https://airline.api.veyino.org/api/v1/diversions" | jq .
```

### 3. 空间地理矩形范围过滤 (Bounding Box)

```bash
curl -s "https://airline.api.veyino.org/api/v1/flights?bbox=30.0,120.0,40.0,125.0&limit=20" | jq .
```

### 4. Python 客户端调用示例

```python
import requests

BASE_URL = "https://airline.api.veyino.org"

# 查询在空与备降航班
res = requests.post(f"{BASE_URL}/api/v1/query", json={"filters": {"is_diverting": True}}).json()
for flight in res["data"]:
    callsign = flight["flight"]["callsign_iata"] or flight["flight"]["callsign_icao"]
    dest = flight["route"]["destination"]["name"] if flight["route"].get("destination") else "未知"
    alt_cand = flight["diversion_analysis"].get("candidate_alternate")
    cand_name = alt_cand["name"] if alt_cand else "未知"
    print(f"航班 {callsign} 正在备降！原目的地: {dest} -> 预测备降场: {cand_name}")
```

---

## 7. 安装与部署教程 (Docker & 原生)

### 方式 A：Docker Compose 极速部署（推荐）

```bash
git clone https://github.com/ysnsk111/contrail.git
cd contrail
cp .env.example .env
docker compose up -d --build
docker compose logs -f
```

### 方式 B：本地原生 Go 编译运行

```bash
go build -o bin/contrail ./cmd/contrail
./bin/contrail
```

---

## 8. 许可与数据合规

- **代码**：GNU 通用公共许可证第 3 版（GPL-3.0），见 [LICENSE](LICENSE)。
- **数据**：Contrail 只负责统一收集并按原样再分发第三方数据。数据**不受** GPL 约束，仍须遵守各数据源条款：仅限非商业使用，必须署名，部分用途还需取得提供方书面许可。详见 [DATA_NOTICE.md](DATA_NOTICE.md)。
- **署名**：OpenSky Network · adsb.fi · adsbdb.com（航线数据：David Taylor 与 Jim Mason）· OurAirports。
