# Contrail: Near-Real-Time Global Flight Telemetry & Diversion Prediction System

<div align="center">

[![Code: GPL v3](https://img.shields.io/badge/Code-GPLv3-blue.svg)](LICENSE) [![Data: Non-Commercial](https://img.shields.io/badge/Data-Non--Commercial-orange.svg)](DATA_NOTICE.md)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Docker Ready](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](Dockerfile)
[![Tracked Aircraft](https://img.shields.io/badge/Tracked%20Aircraft-10%2C000%2B-orange)]()
[![Translations](https://img.shields.io/badge/Languages-EN%20%7C%20%E4%B8%AD%E6%96%87%20%7C%20%E6%97%A5%E6%9C%AC%E8%AA%9E-blue)]()

**English** | [简体中文](README.zh-CN.md) | [日本語](README.ja-JP.md)

</div>

---

> [!IMPORTANT]
> **Personal, local use only.** This repository is published so the code can be reviewed and run privately. Contrail is intended to collect flight data for the operator's own local, private use. **Do not run it as a public or shared service.** Serving its data to anyone else (a public API, website, app or shared endpoint) requires prior written permission from each upstream provider (for example The OpenSky Network and David J Taylor for adsbdb route data). Obtain those permissions yourself before any public access. See [DATA_NOTICE.md](DATA_NOTICE.md).

Contrail is a **self-hosted collector and local API for near-real-time global flight data**, meant for personal, private use. One REST/JSON API returns live aircraft positions, flight numbers, routes and origin/destination airports, emergency squawk codes, and locally computed diversion predictions.

Polling public sources aircraft-by-aircraft would quickly exhaust their rate limits. Contrail instead pulls **one global snapshot per cycle, enriches it locally, caches the results, and serves them from its own API**. It automatically retains a rolling 3-month history in a local SQLite database (WAL mode) and exposes a unified JSON REST API, allowing consumers to query flight telemetry and predictions without risking upstream rate-limit bans.

---

## Table of Contents

1. [Architecture & Design Principles](#1-architecture--design-principles)
2. [Five Core Data Dimensions](#2-five-core-data-dimensions)
3. [Upstream Quotas & Safe Refresh Policies](#3-upstream-quotas--safe-refresh-policies)
4. [In-House Diversion Prediction Algorithm](#4-in-house-diversion-prediction-algorithm)
5. [Universal API Specification & JSON Formats](#5-universal-api-specification--json-formats)
6. [API Call Manual (cURL & Python)](#6-api-call-manual-curl--python)
7. [Installation & Deployment Guide (Docker & Native)](#7-installation--deployment-guide-docker--native)
8. [Environment Configuration Reference](#8-environment-configuration-reference)
9. [License & Data Compliance](#9-license--data-compliance)

---

## 1. Architecture & Design Principles

```
  +------------------+         +-----------------------+
  |  adsb.fi Feed    |         |  OpenSky Network API  |
  | (/v2/snapshot)   |         |     (/states/all)     |
  +--------+---------+         +-----------+-----------+
           |                               |
           +---------------+---------------+
                           |
          [Intelligent Failover / Fallback Manager]
                           | (35s ~ 90s Safe Buffer Snapshots)
                           v
           +-------------------------------+
           |    Poller Processing Core     |
           +---------------+---------------+
                           |
     +---------------------+---------------------+
     |                     |                     |
     v                     v                     v
[adsbdb Route Cache] [OurAirports Spatial DB] [4-Criteria Diversion]
  (Token Bucket 400/m)  (6,169 Global Airports) (Descent/Dev/Alt/Squawk)
     |                     |                     |
     +---------------------+---------------------+
                           |
                           v
            +------------------------------+
            |  Local SQLite (WAL Mode)     |
            |  - live_flights (Real-time)  |
            |  - flight_snapshots (3-Mo)   |
            |  - callsign_cache (24h TTL)  |
            |  - Automatic Retention Job   |
            +--------------+---------------+
                           | (Local Queries)            
                           v
            +------------------------------+
            |    Go High-Concurrency API   |
            |  (airline.api.veyino.org)    |
            +--------------+---------------+
                           |
                +----------+----------+
                |                     |
                v                     v
         [Standard REST]      [Custom POST JSON]
```

- **Global Snapshot Retrieval**: Pulls comprehensive spatial kinematics across the entire global airspace in a single HTTP request.
- **Asynchronous Local Enrichment**: Flight routes and airports are looked up via `adsbdb` regulated by a token bucket rate limiter ($\le 400$ req/min). Results are cached for 24 hours. Non-commercial, charter, or private callsigns are negatively cached for 24 hours to prevent repeated hits.
- **Local Persistence & Retention**: Live kinematics are stored in SQLite (WAL mode). Time-series snapshots are kept for 3 months (90 days) with an automated cleaner daemon pruning expired logs.
- **Zero-Upstream-Impact Local Access**: Your own tools query Contrail's local API, so repeated local queries never touch upstream provider quotas.

---

## 2. Five Core Data Dimensions

Every flight record emitted by Contrail provides normalized data across 5 dimensions:

| Dimension | Field Path | Type | Description |
| :--- | :--- | :--- | :--- |
| **1. Flight Identity** | `aircraft.hex` | string | 24-bit ICAO transponder hex address (e.g. `39de4a`) |
| | `flight.callsign_icao` | string | ICAO callsign (e.g. `CES5101`, `AAL1`) |
| | `flight.callsign_iata` | string | Commercial IATA flight number (e.g. `MU5101`, `AA1`) |
| | `flight.airline` | object | Operating airline (Name, ICAO, IATA, Country) |
| **2. Route & Airports** | `route.origin` | object | Origin airport (ICAO, IATA, Name, City, Coordinates, Elevation) |
| | `route.destination` | object | Destination airport (ICAO, IATA, Name, City, Coordinates, Elevation) |
| | `route.summary` | string | Summary route label (e.g. `SHA -> PEK`) |
| **3. Kinematics** | `telemetry.position` | object | Coordinates (`latitude`, `longitude`) |
| | `telemetry.altitude` | object | Barometric (`barometric_ft`, `barometric_m`) & geometric altitude |
| | `telemetry.speed` | object | Ground speed in knots (`ground_speed_kts`) and `km/h` |
| | `telemetry.attitude` | object | True track (`track_deg`) & vertical rate (`vertical_rate_fpm`) |
| | `telemetry.on_ground`| bool | Ground contact flag |
| **4. Emergency Codes** | `emergency.squawk` | string | 4-digit transponder squawk code (`7700`, `7600`, `7500`, `1200`) |
| | `emergency.is_emergency` | bool | Emergency status indicator |
| | `emergency.category` | string | Severity (`GENERAL_EMERGENCY`, `RADIO_FAILURE`, `HIJACK`, `NONE`) |
| | `emergency.description` | string | Human-readable emergency diagnosis |
| **5. Diversion Prediction** | `diversion_analysis.predicted_diversion` | bool | Diversion status flag ($\ge 2$ criteria triggered) |
| | `diversion_analysis.probability_level` | string | Probability assessment (`NONE`, `LOW`, `MODERATE`, `HIGH`, `CRITICAL`) |
| | `diversion_analysis.distance_to_dest_km`| float | Great-circle distance to scheduled destination (km) |
| | `diversion_analysis.bearing_to_dest_deg`| float | Direct bearing angle to destination (0–360°) |
| | `diversion_analysis.heading_deviation_deg`| float | Angular difference between track and destination bearing |
| | `diversion_analysis.triggered_criteria`| array | Detailed list of triggered criteria and measured telemetry |
| | `diversion_analysis.candidate_alternate`| object | Nearest candidate alternate airport aligned with descent trajectory |

---

## 3. Upstream Quotas & Safe Refresh Policies

Contrail operates close to the maximum permissible limits of upstream providers while strictly maintaining **safe buffers** to avoid IP rate-limit blocks from network jitter or clock drift:

| Provider / Endpoint | Official Maximum Limit | Contrail Production Cadence | Safe Buffer Margin | Commercial Use |
| :--- | :--- | :--- | :--- | :--- |
| **adsb.fi Feeder Dedicated**<br>`/v2/snapshot` | 1 request / 30s | **35 seconds / poll** | +5s clock margin. If unauthorized (HTTP 403), gracefully falls back to OpenSky | **Prohibited** |
| **OpenSky (Active Feeder)**<br>`/api/states/all` | 8,000 credits/day<br>$\approx$ 43.2s / call | **50 seconds / poll** | 15% credit reserve for retry bursts | Licensed / **Non-commercial** |
| **OpenSky (Registered User)**<br>`/api/states/all` | 4,000 credits/day<br>$\approx$ 86.4s / call | **95 seconds / poll** | ~3,600 credits/day consumption (10% safety margin) | Licensed / **Non-commercial** |
| **OpenSky (Anonymous)**<br>`/api/states/all` | IP bucketed | **60–90 seconds / poll** | Adaptive backoff respecting `X-Rate-Limit-Retry-After-Seconds` | Licensed / **Non-commercial** |
| **adsbdb Callsigns**<br>`/v0/callsign/{callsign}` | 512 req / min | **Token Bucket 400 req / min** | 22% headroom; 24h positive cache, 6h negative cache | **Prohibited** |
| **OurAirports Data**<br>`airports.csv` | Static dataset | **Embedded Pre-indexed**<br>(Instant startup) | 6,169 medium/large airports pre-indexed in memory | Public Domain |

---

## 4. In-House Diversion Prediction Algorithm

Real-world commercial aviation offers no free real-time API exposing "actively diverting" flags. Contrail calculates diversion status locally using spatial geometry and flight dynamics:

### Four Evaluation Criteria

1. **Descent Far From Destination (`DESCENT_FAR_FROM_DEST`)**:
   - Barometric altitude $< 10,000\text{ ft}$;
   - Negative vertical rate with sustained descent $v_{\text{vertical}} < -500\text{ ft/min}$;
   - Great-circle distance to scheduled destination remains $> 150\text{ km}$.
2. **Sustained Heading Deviation (`HEADING_DEVIATION`)**:
   - Track deviation from direct bearing to destination $\Delta\theta > 60^\circ$ while distance to destination $> 100\text{ km}$;
   - Condition sustained across $\ge 3$ consecutive snapshots ($\ge 90\text{s}$) to rule out temporary vectoring, downwind turns, or airway turns.
3. **Approaching Candidate Alternate Airport (`APPROACHING_ALTERNATE_AIRPORT`)**:
   - Evaluated only when distance to scheduled destination $> 80\text{ km}$;
   - Aircraft is actively descending ($v_{\text{vertical}} < -400\text{ ft/min}$);
   - Matched against OurAirports database within $80\text{ km}$, with track pointing within $20^\circ$ of candidate alternate airport.
4. **Emergency Transponder Squawk (`EMERGENCY_SQUAWK`)**:
   - Transponder squawking `7700` (Mayday/General Emergency), `7600` (Radio Failure), or `7500` (Hijacking).

### Anti-False-Positive Filtering

- **Normal Arrival Exemption**: If aircraft is within $40\text{ km}$ of scheduled destination, or below $3,000\text{ ft}$ and within $60\text{ km}$ of destination, it is recognized as normal terminal approach/landing and will **not** be flagged as a diversion.
- **Circular/Survey Flight Exemption**: Flights where origin and destination are identical ($\text{ICAO}_{\text{origin}} = \text{ICAO}_{\text{dest}}$) do not participate in diversion analysis.
- **Stale Flight Cleanup**: Any aircraft inactive for $> 5\text{ minutes}$ is pruned on every polling cycle.
- **Telemetry Anomaly Filter**: Discards or clamps sensor errors ($\text{altitude} > 60,000\text{ ft}$, $\text{ground speed} > 750\text{ kts}$).
- **Callsign IATA Integrity**: Commercial airline callsigns map to IATA flight numbers; unconfirmed or general aviation flights strictly return `null` instead of echoing ICAO.
- **Deterministic Pagination**: Supports `limit` up to $1,000$ with secondary `hex ASC` sort to prevent record shifting.
- **Strict RFC 3339 UTC**: All timestamps formatted in standard ISO 8601 / RFC 3339 (`YYYY-MM-DDTHH:MM:SSZ`).

### Decision Matrix

- **`predicted_diversion = true`**: Triggers $\ge 2$ criteria;
- `Squawk 7700` present: Probability marked as **`CRITICAL`**;
- $\ge 3$ criteria triggered: Probability marked as **`HIGH`**;
- $2$ criteria triggered: Probability marked as **`MODERATE`**;
- $< 2$ criteria: Probability marked as **`LOW`** or **`NONE`** (`predicted_diversion = false`).

---

## 5. Universal API Specification & JSON Formats

Universal Public Base URL: `https://airline.api.veyino.org`

### Endpoint Index

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/api/v1/health` | Health probe, upstream poller telemetry, and storage metrics |
| `GET` | `/api/v1/flights` | Global live flight vectors (filters: callsign, airport, bbox, alt, squawk) |
| `GET` | `/api/v1/flights/{identifier}` | Comprehensive flight record by hex code or callsign |
| `GET` | `/api/v1/emergencies` | Live flights declaring squawk 7500/7600/7700 or upstream emergency |
| `GET` | `/api/v1/diversions` | Live flights predicted to be diverting with candidate alternates |
| `GET` | `/api/v1/airports/{code}` | Airport metadata and active inbound/outbound traffic |
| `GET` | `/api/v1/stats` | Global aviation metrics and top origin/destination airports |
| `POST`| `/api/v1/query` | **Custom structured JSON search engine** |

---

### Custom JSON Search (`POST /api/v1/query`)

```json
{
  "filters": {
    "callsign_prefix": "UAL",
    "is_emergency": false,
    "is_diverting": true,
    "min_altitude_ft": 5000,
    "max_altitude_ft": 35000,
    "bounding_box": {
      "min_lat": 20.0,
      "min_lon": -125.0,
      "max_lat": 50.0,
      "max_lon": -65.0
    }
  },
  "sort_by": "last_seen",
  "sort_order": "desc",
  "limit": 25,
  "page": 1
}
```

---

### Standard JSON Response Envelope

```json
{
  "api_version": "v1.0",
  "base_url": "https://airline.api.veyino.org",
  "generated_at": "2026-11-21T14:49:16Z",
  "execution_ms": 0.82,
  "status": "success",
  "total_records": 12980,
  "filtered_count": 1,
  "pagination": {
    "page": 1,
    "limit": 50,
    "total_pages": 1,
    "total_items": 1
  },
  "data": [
    {
      "aircraft": { "hex": "a2ded8" },
      "flight": {
        "callsign_icao": "SWA1385",
        "callsign_iata": "WN1385",
        "airline": { "name": "Southwest Airlines", "icao": "SWA", "iata": "WN", "country": "United States" }
      },
      "route": {
        "summary": "DEN -> SBA",
        "origin": { "icao": "KDEN", "iata": "DEN", "name": "Denver International Airport" },
        "destination": { "icao": "KSBA", "iata": "SBA", "name": "Santa Barbara Municipal Airport" }
      },
      "telemetry": {
        "position": { "latitude": 30.204, "longitude": -97.6579 },
        "altitude": { "barometric_ft": 475, "barometric_m": 145 },
        "speed": { "ground_speed_kts": 139.0, "ground_speed_kmh": 257.5 },
        "attitude": { "track_deg": 178.8, "vertical_rate_fpm": -705 },
        "on_ground": false,
        "last_contact_epoch": 1795272554
      },
      "emergency": {
        "squawk": "0771",
        "is_emergency": false,
        "category": "NONE",
        "description": "Normal operations"
      },
      "diversion_analysis": {
        "predicted_diversion": true,
        "probability_level": "HIGH",
        "distance_to_destination_km": 2132.1,
        "bearing_to_destination_deg": 288.5,
        "heading_deviation_deg": 109.8,
        "triggered_criteria": [
          { "code": "DESCENT_FAR_FROM_DEST", "message": "Altitude 475ft and descent rate -705fpm while 2132.1km from destination (>150km)" },
          { "code": "HEADING_DEVIATION", "message": "Heading deviation of 109.8° sustained away from scheduled destination" },
          { "code": "APPROACHING_ALTERNATE_AIRPORT", "message": "Descending towards alternate airport Austin Bergstrom International (KAUS, 0.8km away)" }
        ],
        "candidate_alternate": {
          "icao": "KAUS",
          "iata": "AUS",
          "name": "Austin Bergstrom International Airport",
          "distance_km": 0.8,
          "bearing_deg": 208.8
        }
      },
      "metadata": {
        "upstream_source": "opensky",
        "last_updated": "2026-11-21T14:49:01Z"
      }
    }
  ]
}
```

---

## 6. API Call Manual (cURL & Python)

### cURL Examples

```bash
# 1. Fetch system health and poller status
curl -s "https://airline.api.veyino.org/api/v1/health" | jq .

# 2. Query all active transponder emergencies (7700/7600/7500)
curl -s "https://airline.api.veyino.org/api/v1/emergencies" | jq .

# 3. Query all predicted diversions
curl -s "https://airline.api.veyino.org/api/v1/diversions" | jq .

# 4. Filter by geographic bounding box (min_lat, min_lon, max_lat, max_lon)
curl -s "https://airline.api.veyino.org/api/v1/flights?bbox=25.0,-125.0,45.0,-70.0&limit=10" | jq .

# 5. Execute custom structured POST JSON search
curl -s -X POST "https://airline.api.veyino.org/api/v1/query" \
  -H "Content-Type: application/json" \
  -d '{"filters":{"is_diverting":true,"min_altitude_ft":3000},"limit":5}' | jq .
```

### Python Example

```python
import requests

BASE_URL = "https://airline.api.veyino.org"

# Check system operational health
health = requests.get(f"{BASE_URL}/api/v1/health").json()
print("System Status:", health["status"], "Airborne Flights:", health["data"]["flight_stats"]["airborne_count"])

# Perform structured query for diverting flights
payload = {
    "filters": {
        "is_diverting": True
    },
    "limit": 10
}
resp = requests.post(f"{BASE_URL}/api/v1/query", json=payload).json()
for flight in resp["data"]:
    cs = flight["flight"]["callsign_iata"] or flight["flight"]["callsign_icao"]
    dest = flight["route"]["destination"]["name"] if flight["route"].get("destination") else "Unknown"
    alt_cand = flight["diversion_analysis"].get("candidate_alternate")
    cand_name = alt_cand["name"] if alt_cand else "N/A"
    print(f"Flight {cs} diverting! Destination: {dest} -> Alternate: {cand_name}")
```

---

## 7. Installation & Deployment Guide (Docker & Native)

### Method A: Docker Compose (Recommended)

1. Clone repository:
   ```bash
   git clone https://github.com/ysnsk111/contrail.git
   cd contrail
   ```

2. Configure environment:
   ```bash
   cp .env.example .env
   ```

3. Launch container:
   ```bash
   docker compose up -d --build
   docker compose logs -f
   ```

### Method B: Native Go Service

1. Compile binary:
   ```bash
   go build -o bin/contrail ./cmd/contrail
   ```

2. Run binary:
   ```bash
   PORT=8080 ./bin/contrail
   ```

---

## 8. Environment Configuration Reference

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Internal HTTP listener port |
| `API_BASE_URL` | `https://airline.api.veyino.org` | Universal public base URL |
| `DB_PATH` | `./data/contrail.db` | Path to persistent SQLite database |
| `POLLER_ENABLED` | `true` | Toggle background snapshot & enrichment worker |
| `UPSTREAM_PROVIDER` | `auto` | Upstream mode: `auto`, `adsb_fi`, `opensky` |
| `ADSB_FI_SNAPSHOT_URL` | `https://opendata.adsb.fi/api/v2/snapshot` | Feeder snapshot endpoint |
| `ADSB_FI_POLL_INTERVAL_SEC` | `35` | Polling interval in seconds ($>30\text{s}$ limit) |
| `OPENSKY_URL` | `https://opensky-network.org/api/states/all` | OpenSky Network state vector endpoint |
| `OPENSKY_IS_FEEDER` | `false` | Enable feeder credit tier (8,000 credits/day) |
| `OPENSKY_POLL_INTERVAL_SEC`| `90` (`50` for feeder) | Polling interval in seconds |
| `ADS_B_DB_BASE_URL` | `https://api.adsbdb.com/v0/callsign/` | adsbdb route lookup endpoint |
| `ADS_B_DB_RATE_LIMIT_PER_MIN`| `400` | Token bucket rate limit ($<512$ limit) |
| `CALLSIGN_CACHE_TTL_HOURS` | `24` | Callsign cache expiration window |
| `RETENTION_DAYS` | `90` | Rolling historical snapshot retention (3 months) |

---

## 9. License & Data Compliance

- **Code**: GNU General Public License v3.0, see [LICENSE](LICENSE).
- **Data**: Contrail collects data from third-party sources and redistributes it as-is. The data is **not** covered by the GPL. It stays under each source's terms: non-commercial use only, attribution required, and some uses need written permission from the provider. See [DATA_NOTICE.md](DATA_NOTICE.md).
- **Attribution**: The OpenSky Network · adsb.fi · adsbdb.com (route data: David Taylor and Jim Mason) · OurAirports.
