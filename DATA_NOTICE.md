# Data Notice / 数据声明

[English](#english) · [中文](#中文)

---

## English

### 1. Code vs. data

- The Contrail **source code** is licensed under the **GNU General Public License v3.0** ([LICENSE](LICENSE)).
- The GPL covers the code only. It does **not** cover flight, aircraft, route, or airport data that Contrail fetches, caches, or serves.

### 2. What Contrail does with data

Contrail does not produce any surveillance data of its own. It is an **aggregation and redistribution layer**:

1. It **collects** data from the third-party sources listed below, within each source's published rate limits.
2. It **normalizes** that data into one schema (units, field names, timestamps), **merges** records for the same aircraft, and **caches** them.
3. It **redistributes** the result as-is through its API. A derived field (for example a diversion prediction) is computed from source data and marked as an estimate.

The maintainers claim **no ownership** of any third-party data. All rights stay with the original providers. Every record carries a `source` field that names where it came from.

### 3. Sources and their terms

| Source | Data used | Key terms | Attribution |
|---|---|---|---|
| [The OpenSky Network](https://opensky-network.org) | Live state vectors (`/api/states/all`) | Licensed for non-profit research and education. Any operational use of the REST API in a live product, service or automated system needs a **written license** from OpenSky, even for non-profits. Data may not be passed on to third parties without permission. [Terms](https://opensky-network.org/about/terms-of-use) | Cite the paper below |
| [adsb.fi](https://adsb.fi) | Live global snapshot (when enabled) | Personal, non-commercial use only. No licensing, selling, renting or leasing of the data. | "Data: adsb.fi" with a link to https://adsb.fi |
| [adsbdb](https://www.adsbdb.com) | Callsign → route / airline lookups | The **flight route data** belongs to David Taylor (Edinburgh) and Jim Mason (Glasgow). It may not be copied, published or incorporated into other databases without the **explicit permission of David J Taylor**. Aircraft data comes from PlaneBase. | adsbdb.com, David Taylor and Jim Mason |
| [OurAirports](https://ourairports.com) | Airport reference data (bundled) | Public domain. | Credit appreciated |

**Required citation for OpenSky data:**

> Matthias Schäfer, Martin Strohmeier, Vincent Lenders, Ivan Martinovic and Matthias Wilhelm. "Bringing Up OpenSky: A Large-scale ADS-B Sensor Network for Research". In *Proceedings of the 13th IEEE/ACM International Symposium on Information Processing in Sensor Networks (IPSN)*, pages 83–94, April 2014.

### 4. Rules for anyone who runs or uses Contrail

- **Local, private use only.** Contrail is meant to collect data for the operator's own local, private use. Do not expose it as a public or shared service unless you have first obtained written permission from every upstream provider whose data you serve.
- **Non-commercial only.** Data served by Contrail may not be sold, licensed, or used in any paid or for-profit product or service.
- **Keep attribution.** Any public deployment, UI or downstream redistribution must show the attribution above and keep the `source` field.
- **Permissions are the operator's job.** Whoever runs a Contrail instance, must hold every permission the sources above require (for example OpenSky's written license and David J Taylor's permission for route data) before serving that source's data publicly. Turn off a source you are not licensed for (`UPSTREAM_PROVIDER`, route enrichment).
- **Respect rate limits.** Do not remove or bypass the built-in token buckets and polling intervals.
- **No warranty.** Data is unofficial, may be delayed, incomplete or wrong, and **must not be used for navigation, air traffic control or safety-of-life decisions**.

### 5. Takedown and contact

Data providers who want their data removed, attributed differently, or rate-limited further can contact **jake@veyino.org**. We will act on such requests promptly.

---

## 中文

### 1. 代码与数据分开授权

- Contrail 的**源代码**采用 **GNU 通用公共许可证第 3 版（GPL-3.0）**，见 [LICENSE](LICENSE)。
- GPL 只约束代码，**不涵盖** Contrail 抓取、缓存或分发的航班、飞机、航线和机场数据。

### 2. Contrail 对数据做了什么

Contrail 本身不产生任何监视数据，它只是一个**统一收集与再分发层**：

1. **收集**：在各数据源公布的速率限制内，从下列第三方来源获取数据。
2. **统一**：把数据转换成同一套格式（单位、字段名、时间戳），合并同一架飞机的记录，并做缓存。
3. **再分发**：通过 API 按原样提供。派生字段（如备降预测）由源数据计算得出，并标注为估算值。

维护者**不主张**对任何第三方数据的所有权，一切权利归原始提供方。每条记录都带有 `source` 字段，标明数据来源。

### 3. 数据来源与条款

| 来源 | 使用的数据 | 主要条款 | 署名 |
|---|---|---|---|
| [OpenSky Network](https://opensky-network.org) | 实时状态向量 | 限非营利研究与教育使用；在线产品、服务或自动化系统中运营性使用 REST API，须取得 OpenSky **书面许可**（非营利机构也不例外）；未经许可不得向第三方提供数据 | 引用下方论文 |
| [adsb.fi](https://adsb.fi) | 全球实时快照（启用时） | 仅限个人非商业使用；不得许可、出售、出租数据 | 标注 "Data: adsb.fi" 并附链接 |
| [adsbdb](https://www.adsbdb.com) | 呼号 → 航线 / 航司 | **航线数据**归 David Taylor 与 Jim Mason 所有，未经 **David J Taylor 明确许可**，不得复制、发布或并入其他数据库；飞机数据来自 PlaneBase | adsbdb.com、David Taylor 与 Jim Mason |
| [OurAirports](https://ourairports.com) | 机场基础数据（内置） | 公有领域 | 建议署名 |

OpenSky 数据须引用论文：Schäfer 等，《Bringing Up OpenSky: A Large-scale ADS-B Sensor Network for Research》，IPSN 2014，第 83–94 页（完整引用见英文部分）。

### 4. 部署与使用规则

- **仅限本地私用。** Contrail 用于为部署者本人收集数据并在本地私人使用。除非事先取得所提供数据的每一个上游方的书面许可，否则不得作为公开或共享服务开放。
- **仅限非商业**：Contrail 提供的数据不得出售、授权，或用于任何收费、营利的产品或服务。
- **保留署名**：任何公开部署、界面或下游再分发，都须保留上述署名和 `source` 字段。
- **许可由运营方负责**：任何 Contrail 实例的运营方（包括维护者自己的公开实例），在公开提供某一来源的数据之前，必须先取得该来源要求的全部许可（如 OpenSky 书面许可、David J Taylor 的航线数据许可）；未获许可的来源应关闭。
- **遵守速率限制**：不得移除或绕过内置的令牌桶和轮询间隔。
- **无担保**：数据为非官方数据，可能延迟、缺失或有误，**不得用于导航、空管或任何涉及人身安全的决策**。

### 5. 下架与联系

数据提供方如需下架数据、调整署名或进一步限流，请联系 **jake@veyino.org**，我们会尽快处理。
