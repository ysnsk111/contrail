// Contrail Web Application & Console
// Complete Multilingual I18N Dictionary (en, zh, ja)
window.I18N = window.I18N || {
  en: {
    brand_tagline: "API Portal",
    nav_home: "Home",
    nav_docs: "Documentation",
    nav_terms: "Terms & Licensing",
    nav_console: "Console",
    btn_signin: "Sign In",
    btn_register: "Register",
    btn_signout: "Sign Out",
    btn_open_console: "Open Console",
    btn_get_key: "Get Free API Key",
    btn_view_docs: "Explore Documentation",

    hero_pill: "Academic & Non-Commercial Flight Telemetry",
    hero_title: "Global Flight Telemetry & Radar API",
    hero_desc: "High-precision ADS-B aircraft vectors, route enrichment, transponder emergency tracking (7700/7600/7500), and automated diversion prediction for researchers and developers.",

    feat_1_title: "Live ADS-B Tracking",
    feat_1_desc: "Near-real-time transponder kinematics, barometric and geometric altitudes, ground speed, track heading, and vertical rates.",
    feat_2_title: "Route & Airframe Enrichment",
    feat_2_desc: "Automatic ICAO/IATA callsign decoding, origin and destination airport pairing, distance calculations, and airframe classifications.",
    feat_3_title: "Emergency Squawk Surveillance",
    feat_3_desc: "Instant filtering for squawk 7700 (General Emergency), 7600 (Radio Failure), 7500 (Unlawful Interference) with alert metadata.",
    feat_4_title: "Diversion Prediction Engine",
    feat_4_desc: "Algorithmic heading deviation analysis and nearest suitable alternate airport identification for irregular operations.",

    // Documentation Quickstart
    docs_quickstart_title: "API Quickstart & Authentication",
    docs_quickstart_desc: "All live data endpoints require a provisioned API key passed via the X-API-Key HTTP header or ?api_key= query parameter. Rate limits apply per key: 60 RPM, 1,000 RPH, and 10,000 requests/day.",
    docs_fetch_flights_title: "Fetch Live Flights with API Key:",
    docs_attribution_title: "JSON Response Attribution Envelope:",
    docs_console_title: "Contrail API Endpoints Reference",
    docs_endpoint_flights_desc: "Query live aircraft vectors with filters (callsign, hex, squawk, origin, dest, bbox, emergency, diverting).",
    docs_endpoint_emergencies_desc: "Stream live emergency aircraft broadcasting squawk 7700, 7600, or 7500.",
    docs_endpoint_diversions_desc: "Retrieve flights algorithmically flagged as deviating from destination trajectory.",
    docs_endpoint_query_desc: "Custom JSON query engine supporting complex geospatial and altitude filtering.",

    // Compliance & Footers
    compliance_header: "Data Sources & Mandatory Legal Attribution",
    compliance_intro: "Contrail collects and redistributes aviation telemetry as-is under strict non-commercial research terms. All data remains the property of original providers:",
    src_opensky: "Live aircraft positions: The OpenSky Network (https://opensky-network.org)",
    src_adsbfi: "Live aircraft positions: adsb.lol (https://adsb.lol)",
    src_adsbdb: "Flight routes: adsbdb.com — route data by David Taylor (Edinburgh) and Jim Mason (Glasgow)",
    src_planebase: "Aircraft registrations & types: adsbdb.com, aircraft data from PlaneBase",
    src_ourairports: "Airports: OurAirports (https://ourairports.com, public domain)",
    opensky_citation_label: "OpenSky Network Mandatory Research Paper Citation:",
    commercial_disclaimer: "Strictly Non-Commercial Use Only. Not for navigation or safety-of-life use. Commercial users must apply directly to original data sources for commercial licenses.",
    footer_rights: "Contrail API • All rights reserved • Contact: team@veyino.org",

    // Modals - Login & Register
    modal_login_title: "Sign in to Contrail Console",
    modal_reg_title: "Register for Contrail API Access",
    lbl_name_org: "Name or Organization",
    lbl_email: "Email Address",
    lbl_purpose: "One-sentence Use Case",
    lbl_password: "Password",
    lbl_confirm_password: "Confirm Password",
    lbl_captcha: "Human Verification (Cap Proof of Work)",
    err_solve_cap: "Please complete the Cap human verification first.",
    err_enter_email: "Please enter your email address first.",
    msg_code_sent: "Verification code sent! Please check your inbox (and spam folder).",
    lbl_verif_code: "Email Verification Code",
    btn_send_code: "Send Code",
    btn_sending: "Sending...",
    chk_agree_terms: "I agree to the Terms of Service and strictly non-commercial use.",
    notice_no_commercial: "Commercial use is strictly prohibited by upstream data sources. Do not apply for commercial projects. Commercial users must seek authorization directly from each data provider.",

    // Dashboard Sidebar
    side_overview: "Live Radar & Stats",
    side_keys: "API Keys",
    side_docs: "API Documentation",
    side_security: "Security & Licensing",
    side_support: "Contact Support",
    side_admin: "Admin Console",

    // Overview Stats
    stat_total_tracked: "Live Aircraft Tracked",
    stat_emergencies: "Active Emergencies",
    stat_diverting: "Diverting Predictions",
    stat_today_req: "Your Today Requests",
    stat_quota: "Daily Quota",
    stat_rpm: "Current Rate (RPM)",

    // Map Radar & Filters
    map_title: "Real-Time Global Flight Radar",
    filter_all: "All Flights",
    filter_emergencies: "Emergencies Only",
    filter_diverting: "Diverting Only",

    // Key Management
    key_list_title: "Provisioned API Keys",
    key_list_desc: "Manage your personal authentication tokens. Each key is subject to non-commercial rate limits (60 RPM / 10,000 Daily).",
    btn_gen_key: "Generate New Key",
    col_name: "Key Name",
    col_prefix: "Key Prefix",
    col_created: "Created",
    col_last_used: "Last Used",
    col_today: "Today Calls",
    col_total: "Total Calls",
    col_status: "Status",
    col_actions: "Action",
    btn_revoke: "Revoke",
    msg_no_keys: "No API keys generated yet. Click \"Generate New Key\" to create one.",

    // Key Generation Modal
    lbl_key_name: "Key Name / Description",
    ph_key_name: "e.g. Trajectory Prediction Server",
    lbl_compliance_cert_title: "Mandatory Compliance Certification:",
    key_agreement_prompt: "Before creating an API key, you must certify that this key will only be used for personal, academic, or non-commercial research projects. Upstream source attribution must accompany any downstream presentation.",
    chk_key_certify: "I certify that this key will only be used for personal, academic, and non-commercial research purposes in full compliance with upstream terms.",
    btn_agree_gen_key: "Agree & Generate Key",
    modal_key_success_title: "API Key Generated Successfully",
    msg_key_created: "Your new API Key has been generated. Please copy and store it securely; it will not be shown again:",
    btn_copy_key: "Copy Key",
    msg_copied: "Copied!",
    btn_close: "Close",

    // Password Change Form
    sec_change_password: "Change Account Password",
    lbl_current_password: "Current Password",
    lbl_new_password: "New Password",
    lbl_confirm_new_password: "Confirm New Password",
    btn_update_password: "Update Password",
    err_password_mismatch: "New password and confirm password do not match.",
    msg_password_updated: "Password updated successfully!",

    // Terms of Service & Compliance in Console
    terms_title: "Contrail API Terms of Service",
    terms_item_1: "1. Strictly Non-Commercial: Learning, research, and personal hobby projects only. No reselling, commercial monetization, or advertising integration.",
    terms_item_2: "2. Attribution Retention: Must preserve upstream attribution metadata (attribution & source fields) in any public presentation.",
    terms_item_3: "3. No Bulk Mirroring: Mass scraping to construct competing commercial databases is prohibited.",
    terms_item_4: "4. No Navigation/Safety Use: Unofficial data, not for flight planning or ATC decisions.",
    terms_item_5: "5. Revocation: Keys violating terms will be throttled or permanently revoked.",
    terms_item_6: "6. Commercial Licensing: Upstream providers do not permit commercial redistribution. Inquire directly with OpenSky, adsb.lol, PlaneBase.",

    // Support Tab
    support_title: "Contrail Technical Support",
    support_desc: "For inquiries regarding upstream compliance, research collaboration, or account assistance, please contact the engineering team:",
    lbl_direct_email: "Direct Email Contact:",
    btn_send_email: "Send Email to team@veyino.org",

    // Admin Panel
    admin_panel_title: "Administrator Control Panel",
    admin_field_title: "Normal Users API Field Restrictions",
    admin_field_desc: "Configure which top-level flight fields are returned to regular API users. (Admin keys always return 100% full fields).",
    btn_save_fields: "Save Field Restrictions",
    msg_restrictions_saved: "Field restrictions saved successfully!",
    err_at_least_one_field: "Please select at least one field.",
    admin_users_title: "Registered Users Management",
    col_admin_name: "Name / Org",
    col_admin_email: "Email",
    col_admin_purpose: "Purpose",
    col_admin_role: "Role",
    col_admin_status: "Status",
    col_admin_registered: "Registered",
    col_admin_actions: "Actions",
    btn_unban: "Unban",
    btn_ban: "Ban",
    btn_delete: "Delete",
    admin_system_title: "System Rate Limit Abuse Warning Logs",
    col_alert_email: "User Email",
    col_alert_prefix: "Key Prefix",
    col_alert_reason: "Abuse Reason",
    col_alert_time: "Timestamp",
    col_alert_status: "Status",
    msg_no_alerts: "No rate limit abuse warnings dispatched recently."
  },

  zh: {
    brand_tagline: "API 控制台",
    nav_home: "首页",
    nav_docs: "开发文档",
    nav_terms: "许可与合规",
    nav_console: "控制台",
    btn_signin: "登录",
    btn_register: "注册",
    btn_signout: "退出登录",
    btn_open_console: "进入控制台",
    btn_get_key: "申请免费 API Key",
    btn_view_docs: "查阅开发文档",

    hero_pill: "学术研究与非商业航空遥测服务",
    hero_title: "全球实时航班遥测与雷达 API",
    hero_desc: "高精度 ADS-B 飞行状态矢量、智能航线与机型丰富、7700/7600/7500 特殊应答机监控以及基于矢量的备降预测引擎，面向科研与开发者全面开放。",

    feat_1_title: "实时 ADS-B 追踪",
    feat_1_desc: "亚秒级应答机运动学遥测、气压与几何高度、地速、真实航向角以及垂直爬升/下降率。",
    feat_2_title: "航线与机身数据富集",
    feat_2_desc: "自动识别 ICAO/IATA 呼号、起飞与目的机场配对、剩余距离估算及 PlaneBase 机身类别。",
    feat_3_title: "紧急应答机专线监控",
    feat_3_desc: "毫秒级筛选 7700（一般紧急）、7600（通讯失效）、7500（非法干扰），提供专用状态接口。",
    feat_4_title: "飞行备降算法预测",
    feat_4_desc: "计算航向角与目的机场几何夹角偏差，智能匹配最近适降备降机场，辅助不正常航班监测。",

    // Documentation Quickstart
    docs_quickstart_title: "API 快速入门与认证",
    docs_quickstart_desc: "所有实时数据接口均需要通过 X-API-Key 请求头或 ?api_key= 查询参数传入有效 API 密钥。单 Key 速率限制：60 RPM、1,000 RPH、每日 10,000 次。",
    docs_fetch_flights_title: "使用 API Key 获取实时飞行数据：",
    docs_attribution_title: "JSON 响应数据来源署名外壳：",
    docs_console_title: "Contrail API 接口调用参考",
    docs_endpoint_flights_desc: "查询实时飞机状态矢量，支持呼号、Hex、应答机、起降机场、地理包围盒、紧急与备降过滤。",
    docs_endpoint_emergencies_desc: "实时监控并返回发射 7700、7600 或 7500 特殊应答机代码的紧急航空器。",
    docs_endpoint_diversions_desc: "检索经算法判定偏离目的地航线并预测可能发生备降的航班。",
    docs_endpoint_query_desc: "支持复杂多维度地理范围、高度区间和多属性组合的 JSON 高级查询引擎。",

    // Compliance & Footers
    compliance_header: "数据来源声明与法定署名要求",
    compliance_intro: "Contrail 仅按原样收集并分发航空数据，严格遵守非商业科研协议。所有原始数据版权均归原提供方所有：",
    src_opensky: "实时飞行位置矢量：The OpenSky Network (https://opensky-network.org)",
    src_adsbfi: "实时飞行位置矢量：adsb.lol (https://adsb.lol)",
    src_adsbdb: "航班航线与呼号：adsbdb.com — 航线数据由 David Taylor (爱丁堡) 和 Jim Mason (格拉斯哥) 整理提供",
    src_planebase: "机型注册号与详细档案：adsbdb.com，飞机资料来源于 PlaneBase",
    src_ourairports: "机场地理数据库：OurAirports (https://ourairports.com，公有领域)",
    opensky_citation_label: "The OpenSky Network 强制科研论文引用规范：",
    commercial_disclaimer: "严格仅限非商业用途。严禁用于实际航行管制或任何生命安全相关决策。商业用途请直接向各数据源申请授权。",
    footer_rights: "Contrail API • 保留所有权利 • 支持邮箱：team@veyino.org",

    // Modals - Login & Register
    modal_login_title: "登录 Contrail 控制台",
    modal_reg_title: "注册 Contrail API 账户",
    lbl_name_org: "姓名或组织机构名",
    lbl_email: "电子邮箱",
    lbl_purpose: "一句话用途（如：毕业论文航空数据分析）",
    lbl_password: "登录密码",
    lbl_confirm_password: "确认密码",
    lbl_captcha: "人机安全验证 (Cap 工作量证明)",
    err_solve_cap: "请先完成 Cap 人机安全验证。",
    err_enter_email: "请先填写您的电子邮箱地址。",
    msg_code_sent: "验证码已成功发送，请查收您的邮件（如未收到请检查垃圾箱）。",
    lbl_verif_code: "邮箱验证码",
    btn_send_code: "获取验证码",
    btn_sending: "发送中...",
    chk_agree_terms: "我同意使用条款及仅限非商业使用。",
    notice_no_commercial: "提示：上游数据源严禁商业化分发，本项目不提供商业版/付费 Key。商业用途请直接向各数据源申请授权。",

    // Dashboard Sidebar
    side_overview: "全球雷达与概览",
    side_keys: "API 密钥管理",
    side_docs: "调用开发文档",
    side_security: "许可与账户安全",
    side_support: "联系支持",
    side_admin: "管理员控制台",

    // Overview Stats
    stat_total_tracked: "当前在线追踪飞机",
    stat_emergencies: "紧急应答航班",
    stat_diverting: "疑似备降航班",
    stat_today_req: "今日 API 调用量",
    stat_quota: "每日调用配额",
    stat_rpm: "当前调用频率 (RPM)",

    // Map Radar & Filters
    map_title: "全球航线进行时实时概览雷达",
    filter_all: "全部在飞航班",
    filter_emergencies: "仅看紧急状态",
    filter_diverting: "仅看备降航班",

    // Key Management
    key_list_title: "已生成的 API Key 列表",
    key_list_desc: "管理您的个人身份鉴权密钥。每个密钥均受严格的非商业频率限制（60 RPM / 每日 10,000 次）。",
    btn_gen_key: "生成新 API Key",
    col_name: "密钥名称",
    col_prefix: "密钥前缀",
    col_created: "创建时间",
    col_last_used: "最近调用",
    col_today: "今日调用",
    col_total: "累计调用",
    col_status: "状态",
    col_actions: "操作",
    btn_revoke: "吊销密钥",
    msg_no_keys: "尚未生成任何 API 密钥。点击右上角“生成新 API Key”立即创建。",

    // Key Generation Modal
    lbl_key_name: "密钥名称 / 用途描述",
    ph_key_name: "例如：航迹预测服务",
    lbl_compliance_cert_title: "强制合规承诺声明：",
    key_agreement_prompt: "在创建 API Key 前，您必须确认：此 Key 仅用于个人学习、学术科研等非商业项目。在任何公开展示或转发时，必须保留数据来源署名。",
    chk_key_certify: "我承诺该 Key 仅用于个人学习、学术科研及非商业用途，并严格遵守上游数据源使用协议。",
    btn_agree_gen_key: "同意并生成密钥",
    modal_key_success_title: "API Key 已成功生成",
    msg_key_created: "您的新 API Key 已成功生成，请妥善保管（仅展示一次）：",
    btn_copy_key: "复制密钥",
    msg_copied: "已复制！",
    btn_close: "关闭",

    // Password Change Form
    sec_change_password: "修改账户密码",
    lbl_current_password: "当前密码",
    lbl_new_password: "新密码",
    lbl_confirm_new_password: "确认新密码",
    btn_update_password: "更新密码",
    err_password_mismatch: "新密码与确认密码不一致。",
    msg_password_updated: "密码更新成功！",

    // Terms of Service & Compliance in Console
    terms_title: "Contrail API 使用条款与合规规范",
    terms_item_1: "1. 严格仅限非商业用途：仅供学习、学术科研及个人爱好项目。严禁转售、商业变现或植入广告。",
    terms_item_2: "2. 保留署名信息：在任何公开展现中必须保留上游数据来源署名字段（attribution 与 source）。",
    terms_item_3: "3. 禁止批量克隆：严禁通过大批量爬取构建竞争性或替代性的商业数据库与镜像服务。",
    terms_item_4: "4. 严禁航行领航：数据非官方认证，不可用于飞行航路规划、空管指挥或生命安全攸关场景。",
    terms_item_5: "5. 违规吊销：任何违反条款的 API Key 将被立即限流或永久吊销。",
    terms_item_6: "6. 商业授权指引：上游数据源不支持商业转售，商业需求请直接咨询 OpenSky、adsb.lol、PlaneBase。",

    // Support Tab
    support_title: "Contrail API 官方技术支持",
    support_desc: "如需关于上游数据合规咨询、学术合作或账户协助，请联系工程支持团队：",
    lbl_direct_email: "官方支持邮箱：",
    btn_send_email: "发送邮件至 team@veyino.org",

    // Admin Panel
    admin_panel_title: "系统管理员控制台",
    admin_field_title: "普通用户 API 返回字段限制设置",
    admin_field_desc: "在此勾选普通用户 API 允许返回的顶级字段。（注意：管理员的所有 API Key 永远返回最全字段，不受此限制）。",
    btn_save_fields: "保存字段限制规则",
    msg_restrictions_saved: "字段限制规则已成功保存并立即生效！",
    err_at_least_one_field: "请至少保留一个允许返回的字段。",
    admin_users_title: "注册用户总览与管控",
    col_admin_name: "姓名 / 机构",
    col_admin_email: "电子邮箱",
    col_admin_purpose: "使用用途",
    col_admin_role: "角色",
    col_admin_status: "状态",
    col_admin_registered: "注册时间",
    col_admin_actions: "操作",
    btn_unban: "解封",
    btn_ban: "封禁",
    btn_delete: "删除",
    admin_system_title: "全局系统监控与限流告警记录",
    col_alert_email: "用户邮箱",
    col_alert_prefix: "密钥前缀",
    col_alert_reason: "告警原因",
    col_alert_time: "触发时间",
    col_alert_status: "处理状态",
    msg_no_alerts: "近期未记录到任何超频违规调用告警。"
  },

  ja: {
    brand_tagline: "APIポータル",
    nav_home: "ホーム",
    nav_docs: "ドキュメント",
    nav_terms: "ライセンスと規約",
    nav_console: "コンソール",
    btn_signin: "ログイン",
    btn_register: "新規登録",
    btn_signout: "ログアウト",
    btn_open_console: "コンソールを開く",
    btn_get_key: "APIキーを無料取得",
    btn_view_docs: "ドキュメントを見る",

    hero_pill: "学術研究・非商用フライトテレメトリサービス",
    hero_title: "グローバルフライトテレメトリ＆レーダー API",
    hero_desc: "高精度ADS-B航空機ベクトル、飛行ルート・機体情報の強化、緊急スコーク(7700/7600/7500)監視、およびダイバージョン予測エンジンを研究者や開発者に提供します。",

    feat_1_title: "リアルタイムADS-B追跡",
    feat_1_desc: "気圧高度・幾何高度、対地速度、真方位、昇降率など、トランスポンダーの最新運動パラメータを提供。",
    feat_2_title: "ルート・機体情報の自動補完",
    feat_2_desc: "ICAO/IATA便名の自動解決、出発地・目的地空港のマッピング、PlaneBaseによる詳細な機体属性。",
    feat_3_title: "緊急スコーク即時監視",
    feat_3_desc: "7700(全般緊急)、7600(通信途絶)、7500(不法奪取)のトランスポンダーコードをミリ秒単位で抽出。",
    feat_4_title: "ダイバージョン自動予測",
    feat_4_desc: "目的地への方位偏差アルゴリズムと近隣の代替候補空港スコアリングにより、異常運航を検知。",

    // Documentation Quickstart
    docs_quickstart_title: "API クイックスタートと認証",
    docs_quickstart_desc: "すべてのライブデータエンドポイントには、X-API-Key HTTPヘッダーまたは ?api_key= クエリパラメータで渡されるAPIキーが必要です。キーごとのレート制限：60 RPM、1,000 RPH、1日あたり10,000リクエスト。",
    docs_fetch_flights_title: "APIキーを使用してライブ飛行データを取得：",
    docs_attribution_title: "JSON レスポンスの帰属エンベロープ：",
    docs_console_title: "Contrail API エンドポイントリファレンス",
    docs_endpoint_flights_desc: "フィルター（コールサイン、hex、スコーク、出発地、目的地、バウンディングボックス、緊急、ダイバージョン）を使用して航空機ベクトルを照会。",
    docs_endpoint_emergencies_desc: "スコーク 7700、7600、7500 を送信している緊急航空機をリアルタイムストリーミング。",
    docs_endpoint_diversions_desc: "目的地の飛行経路から逸脱し、ダイバージョンが予測される便をアルゴリズムで抽出。",
    docs_endpoint_query_desc: "複雑な地理範囲や高度フィルタリングをサポートするカスタム JSON クエリエンジン。",

    // Compliance & Footers
    compliance_header: "データソースの明示および法的帰属表記",
    compliance_intro: "Contrail は厳格な非商用研究規約に基づき航空データを収集・再配布しています。すべての元データの権利は提供元に帰属します：",
    src_opensky: "リアルタイム航空機位置：The OpenSky Network (https://opensky-network.org)",
    src_adsbfi: "リアルタイム航空機位置：adsb.lol (https://adsb.lol)",
    src_adsbdb: "航空路線データ：adsbdb.com — David Taylor 氏および Jim Mason 氏提供",
    src_planebase: "機体登録番号・機種：adsbdb.com、PlaneBase 資料",
    src_ourairports: "空港座標：OurAirports (https://ourairports.com、パブリックドメイン)",
    opensky_citation_label: "OpenSky Network 必須論文引用要件：",
    commercial_disclaimer: "非商用利用に限定されます。実際の航空管制や生命の安全に関わる用途には絶対に使用しないでください。商用利用を希望される場合は各データソースに直接許諾を申請してください。",
    footer_rights: "Contrail API • 無断転載を禁じます • お問い合わせ：team@veyino.org",

    // Modals - Login & Register
    modal_login_title: "Contrail コンソールにログイン",
    modal_reg_title: "Contrail API アカウント登録",
    lbl_name_org: "お名前または組織名",
    lbl_email: "メールアドレス",
    lbl_purpose: "一言用途（例：研究論文での航空データ分析）",
    lbl_password: "パスワード",
    lbl_confirm_password: "確認用パスワード",
    lbl_captcha: "人間確認 (Cap Proof of Work)",
    err_solve_cap: "先に Cap 人間認証を完了してください。",
    err_enter_email: "先にメールアドレスを入力してください。",
    msg_code_sent: "認証コードを送信しました。受信トレイ（または迷惑メールフォルダ）をご確認ください。",
    lbl_verif_code: "メール確認コード",
    btn_send_code: "コードを送信",
    btn_sending: "送信中...",
    chk_agree_terms: "利用規約および非商用限定利用に同意します。",
    notice_no_commercial: "注意：上流プロバイダの規約により商用利用は禁止されています。有料版・商用キーの申請は受け付けておりません。商用利用は各データ元に直接申請してください。",

    // Dashboard Sidebar
    side_overview: "ライブレーダー＆統計",
    side_keys: "APIキー管理",
    side_docs: "API仕様ドキュメント",
    side_security: "セキュリティ＆規約",
    side_support: "サポートへのお問い合わせ",
    side_admin: "管理者コンソール",

    // Overview Stats
    stat_total_tracked: "追跡中の航空機数",
    stat_emergencies: "緊急スコーク発生数",
    stat_diverting: "ダイバージョン予測数",
    stat_today_req: "本日のリクエスト数",
    stat_quota: "1日の利用枠",
    stat_rpm: "現在のレート (RPM)",

    // Map Radar & Filters
    map_title: "世界航路リアルタイム運航概況レーダー",
    filter_all: "すべての飛行中便",
    filter_emergencies: "緊急便のみ",
    filter_diverting: "ダイバージョン予測のみ",

    // Key Management
    key_list_title: "発行済み API キー一覧",
    key_list_desc: "個人認証トークンを管理します。各キーは非商用レート制限（60 RPM / 1日あたり10,000リクエスト）の対象となります。",
    btn_gen_key: "新しいキーを生成",
    col_name: "キー名",
    col_prefix: "プレフィックス",
    col_created: "作成日時",
    col_last_used: "最終利用",
    col_today: "本日利用数",
    col_total: "累計利用数",
    col_status: "状態",
    col_actions: "操作",
    btn_revoke: "無効化",
    msg_no_keys: "まだAPIキーが生成されていません。「新しいキーを生成」をクリックして作成してください。",

    // Key Generation Modal
    lbl_key_name: "キー名 / 用途の説明",
    ph_key_name: "例：軌道予測サーバー",
    lbl_compliance_cert_title: "必須コンプライアンス宣誓：",
    key_agreement_prompt: "APIキーを作成する前に、個人学習、学術研究などの非商用プロジェクトにのみ使用することを確認する必要があります。公開時にはデータソースの帰属を明示してください。",
    chk_key_certify: "このキーを個人学習、学術研究、非商用目的のみに使用し、上流の規約を完全に遵守することを宣誓します。",
    btn_agree_gen_key: "同意してキーを生成",
    modal_key_success_title: "APIキーが正常に生成されました",
    msg_key_created: "新しい API キーが生成されました。再表示されませんので大切に保管してください：",
    btn_copy_key: "キーをコピー",
    msg_copied: "コピー完了！",
    btn_close: "閉じる",

    // Password Change Form
    sec_change_password: "アカウントパスワードの変更",
    lbl_current_password: "現在のパスワード",
    lbl_new_password: "新しいパスワード",
    lbl_confirm_new_password: "新しいパスワード（確認）",
    btn_update_password: "パスワードを更新",
    err_password_mismatch: "新しいパスワードと確認用パスワードが一致しません。",
    msg_password_updated: "パスワードが正常に更新されました！",

    // Terms of Service & Compliance in Console
    terms_title: "Contrail API 利用規約と法的遵守要件",
    terms_item_1: "1. 非商用限定：学習、学術研究、個人趣味プロジェクトのみ。転売、商用マネタイズ、広告への組み込みは固く禁止されています。",
    terms_item_2: "2. 帰属情報の保持：公開時には上流データソースの帰属メタデータ（attributionおよびsourceフィールド）を必ず維持してください。",
    terms_item_3: "3. 大量ミラーリングの禁止：競合する商用データベースを構築するための大量スクレイピングは禁止です。",
    terms_item_4: "4. 航法・安全用途の禁止：非公式データのため、飛行計画や航空管制の判断には使用できません。",
    terms_item_5: "5. 失効措置：規約に違反したキーは制限または永久に失効されます。",
    terms_item_6: "6. 商用ライセンス：上流プロバイダは商用再配布を認めていません。直接OpenSky、adsb.lol、PlaneBaseにお問い合わせください。",

    // Support Tab
    support_title: "Contrail API 公式サポート",
    support_desc: "規約順守、研究協力、アカウントに関するお問い合わせはサポートチームまでご連絡ください：",
    lbl_direct_email: "公式サポートメール：",
    btn_send_email: "team@veyino.org へメールを送信",

    // Admin Panel
    admin_panel_title: "システム管理者コントロールパネル",
    admin_field_title: "一般ユーザー向け API レスポンス返却フィールド制限",
    admin_field_desc: "一般ユーザーの API レスポンスに含めるトップレベルフィールドを設定します。（管理者の API キーは常に制限を受けず、全フィールドを返却します）。",
    btn_save_fields: "フィールド制限を保存",
    msg_restrictions_saved: "フィールド制限が正常に保存され、直ちに適用されました。",
    err_at_least_one_field: "少なくとも1つのフィールドを選択してください。",
    admin_users_title: "登録ユーザー管理",
    col_admin_name: "氏名 / 組織",
    col_admin_email: "メールアドレス",
    col_admin_purpose: "利用目的",
    col_admin_role: "ロール",
    col_admin_status: "状態",
    col_admin_registered: "登録日時",
    col_admin_actions: "操作",
    btn_unban: "解除",
    btn_ban: "停止",
    btn_delete: "削除",
    admin_system_title: "システム統計および流量アラートログ",
    col_alert_email: "ユーザーメール",
    col_alert_prefix: "キープレフィックス",
    col_alert_reason: "アラート理由",
    col_alert_time: "発生日時",
    col_alert_status: "状態",
    msg_no_alerts: "最近ディスパッチされたレート制限乱用警告はありません。"
  }
};

let currentLang = localStorage.getItem('contrail_lang') || 'en';
let currentUser = null;
let currentToken = localStorage.getItem('contrail_token') || '';
let userKeys = [];
let currentCaptcha = null;
let loginCapToken = '';
let regCapToken = '';
let capConfig = null;
let flightMap = null;
let flightMarkersMap = new Map();
let mapRefreshTimer = null;
let mapDeadReckonTimer = null;
let activeFlightFilter = 'all';

document.addEventListener('DOMContentLoaded', () => {
  initI18n();
  initRouter();
  checkAuth();
  initCap();
  setupEventListeners();
});

// --- I18N Engine ---
function setLanguage(lang) {
  const dict = window.I18N || {};
  if (!dict[lang]) lang = 'en';
  currentLang = lang;
  localStorage.setItem('contrail_lang', lang);
  const langSelect = document.getElementById('lang-select');
  if (langSelect) langSelect.value = lang;
  applyI18n();
  updateCapLanguage();
}

function initI18n() {
  const langSelect = document.getElementById('lang-select');
  if (langSelect) {
    langSelect.value = currentLang;
    langSelect.addEventListener('change', (e) => setLanguage(e.target.value));
  }
  applyI18n();
}

function t(key) {
  const dict = window.I18N || {};
  if (dict[currentLang] && dict[currentLang][key]) {
    return dict[currentLang][key];
  }
  if (dict.en && dict.en[key]) {
    return dict.en[key];
  }
  return key;
}

function applyI18n() {
  document.querySelectorAll('[data-i18n]').forEach((el) => {
    const key = el.getAttribute('data-i18n');
    el.textContent = t(key);
  });
  document.querySelectorAll('[data-i18n-ph]').forEach((el) => {
    const key = el.getAttribute('data-i18n-ph');
    el.setAttribute('placeholder', t(key));
  });
}

// --- Navigation & Routing ---
function initRouter() {
  window.addEventListener('hashchange', handleRoute);
  handleRoute();
}

function handleRoute() {
  const hash = window.location.hash || '#home';
  const pageHome = document.getElementById('view-landing');
  const pageConsole = document.getElementById('view-console');

  if (hash.startsWith('#console')) {
    if (!currentToken) {
      window.location.hash = '#home';
      openLoginModal();
      return;
    }
    if (pageHome) pageHome.style.display = 'none';
    if (pageConsole) pageConsole.style.display = 'flex';
    handleConsoleTab(hash);
  } else {
    if (pageHome) pageHome.style.display = 'block';
    if (pageConsole) pageConsole.style.display = 'none';
    if (hash === '#docs') scrollToSection('docs-section');
    if (hash === '#terms') scrollToSection('compliance-section');
  }
}

function scrollToSection(id) {
  const el = document.getElementById(id);
  if (el) el.scrollIntoView({ behavior: 'smooth' });
}

function handleConsoleTab(hash) {
  const parts = hash.split('/');
  const tab = parts[1] || 'overview';

  document.querySelectorAll('.sidebar-item').forEach((item) => {
    if (item.getAttribute('data-tab') === tab) {
      item.classList.add('active');
    } else {
      item.classList.remove('active');
    }
  });

  document.querySelectorAll('.mobile-console-pill').forEach((pill) => {
    if (pill.getAttribute('data-tab') === tab) {
      pill.classList.add('active');
    } else {
      pill.classList.remove('active');
    }
  });

  document.querySelectorAll('.console-tab-content').forEach((panel) => {
    panel.style.display = 'none';
  });

  const activePanel = document.getElementById(`tab-${tab}`);
  if (activePanel) {
    activePanel.style.display = 'block';
  }

  if (tab === 'overview') {
    initFlightRadarMap();
    loadOverviewTelemetry();
  } else if (tab === 'keys') {
    loadUserKeys();
  } else if (tab === 'admin') {
    if (currentUser && currentUser.role === 'admin') {
      loadAdminData();
    } else {
      window.location.hash = '#console/overview';
    }
  }
}

// --- Authentication & User State ---
async function checkAuth() {
  if (!currentToken) {
    updateAuthUI(null);
    return;
  }
  try {
    const res = await fetch('/api/v1/auth/me', {
      headers: { Authorization: `Bearer ${currentToken}` }
    });
    if (res.ok) {
      const data = await res.json();
      currentUser = data.data.user;
      updateAuthUI(currentUser);
    } else {
      logout();
    }
  } catch (e) {
    logout();
  }
}

function updateAuthUI(user) {
  const unauthNav = document.getElementById('unauth-nav');
  const authNav = document.getElementById('auth-nav');
  const userGreeting = document.getElementById('user-greeting');
  const adminNavItem = document.getElementById('admin-sidebar-item');
  const mobileAdminTab = document.getElementById('mobile-admin-tab');

  if (user) {
    if (unauthNav) unauthNav.style.display = 'none';
    if (authNav) authNav.style.display = 'flex';
    if (userGreeting) userGreeting.textContent = user.name || user.email;
    if (adminNavItem) {
      adminNavItem.style.display = user.role === 'admin' ? 'flex' : 'none';
    }
    if (mobileAdminTab) {
      mobileAdminTab.style.display = user.role === 'admin' ? 'inline-flex' : 'none';
    }
  } else {
    if (unauthNav) unauthNav.style.display = 'flex';
    if (authNav) authNav.style.display = 'none';
    if (adminNavItem) adminNavItem.style.display = 'none';
    if (mobileAdminTab) mobileAdminTab.style.display = 'none';
  }
}

function logout() {
  currentUser = null;
  currentToken = '';
  localStorage.removeItem('contrail_token');
  updateAuthUI(null);
  window.location.hash = '#home';
}

// --- Modals Management ---
function openLoginModal() {
  closeAllModals();
  const loginCap = document.getElementById('login-cap');
  if (loginCap && typeof loginCap.reset === 'function') {
    loginCap.reset();
  }
  loginCapToken = '';
  document.getElementById('modal-login').classList.add('active');
}

function openRegisterModal() {
  closeAllModals();
  const regCap = document.getElementById('reg-cap');
  if (regCap && typeof regCap.reset === 'function') {
    regCap.reset();
  }
  regCapToken = '';
  document.getElementById('modal-register').classList.add('active');
}

function openCreateKeyModal() {
  document.getElementById('modal-create-key').classList.add('active');
}

function closeAllModals() {
  document.querySelectorAll('.modal-overlay').forEach((m) => m.classList.remove('active'));
  const keyContainer = document.getElementById('created-key-container');
  if (keyContainer) keyContainer.innerHTML = '';
}

async function initCap() {
  try {
    const res = await fetch('/api/v1/auth/cap-config');
    const json = await res.json();
    if (json.data) {
      capConfig = json.data;
      const endpoint = capConfig.api_endpoint || '/cap/b12b141f33/';
      const loginCap = document.getElementById('login-cap');
      const regCap = document.getElementById('reg-cap');
      if (loginCap) {
        loginCap.setAttribute('data-cap-api-endpoint', endpoint);
        loginCap.addEventListener('solve', (e) => {
          loginCapToken = e.detail?.token || '';
        });
        loginCap.addEventListener('reset', () => {
          loginCapToken = '';
        });
      }
      if (regCap) {
        regCap.setAttribute('data-cap-api-endpoint', endpoint);
        regCap.addEventListener('solve', (e) => {
          regCapToken = e.detail?.token || '';
        });
        regCap.addEventListener('reset', () => {
          regCapToken = '';
        });
      }
      updateCapLanguage();
    }
  } catch (e) {
    console.error('Failed to init Cap config', e);
  }
}

function updateCapLanguage() {
  let capLang = 'en';
  if (currentLang === 'zh') capLang = 'zh-cn';
  else if (currentLang === 'ja') capLang = 'ja';
  
  const loginCap = document.getElementById('login-cap');
  const regCap = document.getElementById('reg-cap');
  if (loginCap) loginCap.setAttribute('data-cap-lang', capLang);
  if (regCap) regCap.setAttribute('data-cap-lang', capLang);
}

// --- Event Listeners ---
function setupEventListeners() {
  document.getElementById('btn-open-login')?.addEventListener('click', openLoginModal);
  document.getElementById('btn-open-register')?.addEventListener('click', openRegisterModal);
  document.getElementById('hero-btn-register')?.addEventListener('click', openRegisterModal);
  document.getElementById('hero-btn-console')?.addEventListener('click', () => {
    if (currentToken) {
      window.location.hash = '#console/overview';
    } else {
      openLoginModal();
    }
  });

  document.getElementById('btn-logout')?.addEventListener('click', logout);

  document.querySelectorAll('.modal-close').forEach((btn) => {
    btn.addEventListener('click', closeAllModals);
  });

  // Send verification code
  document.getElementById('btn-send-code')?.addEventListener('click', async () => {
    const email = document.getElementById('reg-email').value.trim();
    const btn = document.getElementById('btn-send-code');

    if (!email) {
      alert(t('err_enter_email') || 'Please enter your email address first.');
      return;
    }

    if (!regCapToken) {
      alert(t('err_solve_cap') || 'Please complete the human verification (Cap challenge) first.');
      return;
    }

    btn.disabled = true;
    btn.textContent = t('btn_sending');

    try {
      const res = await fetch('/api/v1/auth/send-code', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email,
          cap_token: regCapToken
        })
      });
      const data = await res.json();
      if (res.ok) {
        alert(t('msg_code_sent') || 'Verification code sent! Please check your inbox (and spam folder).');
        let countdown = 60;
        const timer = setInterval(() => {
          countdown--;
          btn.textContent = `${countdown}s`;
          if (countdown <= 0) {
            clearInterval(timer);
            btn.disabled = false;
            btn.textContent = t('btn_send_code');
          }
        }, 1000);
      } else {
        alert(data.error_message || 'Failed to send code. Please try again.');
        btn.disabled = false;
        btn.textContent = t('btn_send_code');
        document.getElementById('reg-cap')?.reset?.();
        regCapToken = '';
      }
    } catch (e) {
      alert('Network error while requesting code.');
      btn.disabled = false;
      btn.textContent = t('btn_send_code');
      document.getElementById('reg-cap')?.reset?.();
      regCapToken = '';
    }
  });

  // Submit Register Form
  document.getElementById('form-register')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = document.getElementById('reg-name').value.trim();
    const email = document.getElementById('reg-email').value.trim();
    const purpose = document.getElementById('reg-purpose').value.trim();
    const password = document.getElementById('reg-password').value;
    const confirmPassword = document.getElementById('reg-confirm-password').value;
    const code = document.getElementById('reg-code').value.trim();
    const agree = document.getElementById('reg-agree').checked;

    if (password !== confirmPassword) {
      alert('Passwords do not match.');
      return;
    }
    if (!agree) {
      alert('You must accept the non-commercial terms of service.');
      return;
    }

    try {
      const res = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name,
          email,
          password,
          purpose,
          verification_code: code,
          agree_terms: agree,
          cap_token: regCapToken
        })
      });
      const data = await res.json();
      if (res.ok) {
        if (data.data && data.data.token) {
          currentToken = data.data.token;
          currentUser = data.data.user;
          localStorage.setItem('contrail_token', currentToken);
          updateAuthUI(currentUser);
          closeAllModals();
          window.location.hash = '#console/overview';
        } else {
          closeAllModals();
          openLoginModal();
        }
      } else {
        alert(data.error_message || 'Registration failed.');
      }
    } catch (e) {
      alert('Error during registration.');
    }
  });

  // Submit Login Form
  document.getElementById('form-login')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const email = document.getElementById('login-email').value.trim();
    const password = document.getElementById('login-password').value;

    if (!loginCapToken) {
      alert(t('err_solve_cap') || 'Please complete the human verification (Cap challenge) first.');
      return;
    }

    try {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password, cap_token: loginCapToken })
      });
      const data = await res.json();
      if (res.ok) {
        currentToken = data.data.token;
        currentUser = data.data.user;
        localStorage.setItem('contrail_token', currentToken);
        updateAuthUI(currentUser);
        closeAllModals();
        window.location.hash = '#console/overview';
      } else {
        alert(data.error_message || 'Invalid email or password.');
        document.getElementById('login-cap')?.reset?.();
        loginCapToken = '';
      }
    } catch (e) {
      alert('Login error.');
      document.getElementById('login-cap')?.reset?.();
      loginCapToken = '';
    }
  });

  // Mobile Hamburger Toggle
  const mobileToggle = document.getElementById('mobile-toggle-btn');
  const rightPanel = document.getElementById('topbar-right-panel');
  if (mobileToggle && rightPanel) {
    mobileToggle.addEventListener('click', () => {
      rightPanel.classList.toggle('mobile-active');
    });
    // Auto-close when clicking links
    rightPanel.querySelectorAll('.nav-link, .btn').forEach((el) => {
      el.addEventListener('click', () => {
        rightPanel.classList.remove('mobile-active');
      });
    });
  }

  // Sidebar navigation clicks
  document.querySelectorAll('.sidebar-item').forEach((item) => {
    item.addEventListener('click', () => {
      const tab = item.getAttribute('data-tab');
      window.location.hash = `#console/${tab}`;
    });
  });

  // Mobile console pills clicks
  document.querySelectorAll('.mobile-console-pill').forEach((pill) => {
    pill.addEventListener('click', () => {
      const tab = pill.getAttribute('data-tab');
      window.location.hash = `#console/${tab}`;
    });
  });

  // Create Key Button
  document.getElementById('btn-open-create-key')?.addEventListener('click', openCreateKeyModal);

  // Submit Key Creation
  document.getElementById('form-create-key')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const keyName = document.getElementById('create-key-name').value.trim();
    const agree = document.getElementById('create-key-agree').checked;

    if (!agree) {
      alert('You must certify non-commercial usage.');
      return;
    }

    try {
      const res = await fetch('/api/v1/portal/keys', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${currentToken}`
        },
        body: JSON.stringify({ key_name: keyName, agree_terms: agree })
      });
      const data = await res.json();
      if (res.ok) {
        closeAllModals();
        document.getElementById('create-key-name').value = '';
        document.getElementById('create-key-agree').checked = false;
        showCreatedKeyModal(data.data.raw_key);
        loadUserKeys();
      } else {
        alert(data.error_message || 'Failed to create key.');
      }
    } catch (e) {
      alert('Error creating API key.');
    }
  });

  // Map Filter Buttons
  document.querySelectorAll('.map-filter-btn').forEach((btn) => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.map-filter-btn').forEach((b) => b.classList.remove('btn-amber'));
      document.querySelectorAll('.map-filter-btn').forEach((b) => b.classList.add('btn-outline'));
      btn.classList.remove('btn-outline');
      btn.classList.add('btn-amber');
      activeFlightFilter = btn.getAttribute('data-filter');
      if (flightMap) {
        flightMarkersMap.forEach((item) => flightMap.removeLayer(item.marker));
        flightMarkersMap.clear();
      }
      fetchLiveFlightsForMap();
    });
  });

  // Toggle User Registration Channel (Admin)
  document.getElementById('btn-toggle-registration')?.addEventListener('click', async () => {
    const target = !currentRegistrationEnabled;
    try {
      const res = await fetch('/api/v1/admin/registration-status', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${currentToken}`
        },
        body: JSON.stringify({ registration_enabled: target })
      });
      const data = await res.json();
      if (res.ok) {
        currentRegistrationEnabled = target;
        updateRegistrationStatusUI();
      } else {
        alert(data.error_message || 'Failed to update registration status.');
      }
    } catch (e) {
      alert('Network error.');
    }
  });

  // Save Field Restrictions (Admin)
  document.getElementById('btn-save-field-restrictions')?.addEventListener('click', async () => {
    const checkboxes = document.querySelectorAll('.admin-field-chk');
    const selected = [];
    checkboxes.forEach((cb) => {
      if (cb.checked) selected.push(cb.value);
    });

    if (selected.length === 0) {
      alert(t('err_at_least_one_field') || 'Please select at least one field.');
      return;
    }

    try {
      const res = await fetch('/api/v1/admin/field-restrictions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${currentToken}`
        },
        body: JSON.stringify({ allowed_fields: selected })
      });
      const data = await res.json();
      if (res.ok) {
        alert(t('msg_restrictions_saved') || 'Field restrictions saved successfully!');
        loadAdminFieldRestrictions();
      } else {
        alert(data.error_message || 'Failed to save restrictions.');
      }
    } catch (e) {
      alert('Network error.');
    }
  });

  // Historical flight track map search
  document.getElementById('btn-map-search-track')?.addEventListener('click', searchHistoricalTrack);
  document.getElementById('input-map-search-track')?.addEventListener('keyup', (e) => {
    if (e.key === 'Enter') searchHistoricalTrack();
  });
  document.getElementById('btn-clear-track')?.addEventListener('click', clearActiveRouteAndTrack);

  // Password Change
  document.getElementById('form-change-password')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const oldPassword = document.getElementById('pwd-old').value;
    const newPassword = document.getElementById('pwd-new').value;
    const confirmPassword = document.getElementById('pwd-new-confirm')?.value;

    if (confirmPassword !== undefined && newPassword !== confirmPassword) {
      alert(t('err_password_mismatch'));
      return;
    }

    try {
      const res = await fetch('/api/v1/auth/password', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${currentToken}`
        },
        body: JSON.stringify({ old_password: oldPassword, new_password: newPassword })
      });
      const data = await res.json();
      if (res.ok) {
        alert(t('msg_password_updated'));
        document.getElementById('pwd-old').value = '';
        document.getElementById('pwd-new').value = '';
        const confirmEl = document.getElementById('pwd-new-confirm');
        if (confirmEl) confirmEl.value = '';
      } else {
        alert(data.error_message || 'Failed to update password.');
      }
    } catch (e) {
      alert('Error updating password.');
    }
  });
}

// --- Key Reveal Modal ---
function showCreatedKeyModal(rawKey) {
  const container = document.getElementById('created-key-container');
  if (container) {
    container.innerHTML = `
      <div style="background:#090a0d; padding:12px; border-radius:6px; font-family:monospace; font-size:14px; color:#10b981; word-break:break-all; margin:14px 0; border:1px solid #1f2937;">
        ${rawKey}
      </div>
      <button id="btn-copy-raw-key" class="btn btn-primary" style="width:100%">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="vertical-align:middle; margin-right:6px;"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
        <span id="btn-copy-raw-text">${t('btn_copy_key')}</span>
      </button>
    `;
    document.getElementById('btn-copy-raw-key')?.addEventListener('click', () => {
      navigator.clipboard.writeText(rawKey).then(() => {
        const textEl = document.getElementById('btn-copy-raw-text');
        if (textEl) textEl.textContent = t('msg_copied');
        setTimeout(() => {
          if (textEl) textEl.textContent = t('btn_copy_key');
        }, 2000);
      });
    });
    document.getElementById('modal-show-key').classList.add('active');
  }
}

// --- Overview & Live Radar Map ---
let activeRouteLayers = [];

function clearActiveRouteAndTrack() {
  if (flightMap && activeRouteLayers.length > 0) {
    activeRouteLayers.forEach((layer) => flightMap.removeLayer(layer));
    activeRouteLayers = [];
  }
  const clearBtn = document.getElementById('btn-clear-track');
  if (clearBtn) clearBtn.style.display = 'none';
}

function createAirplaneIcon(track, color) {
  const iconSvg = `
    <div style="transform: rotate(${track}deg); width:24px; height:24px; display:flex; align-items:center; justify-content:center; pointer-events:none;">
      <svg viewBox="0 0 24 24" width="22" height="22" fill="${color}" stroke="#000000" stroke-width="0.5">
        <path d="M12 2c-.55 0-1 .45-1 1v7.5L3.5 14v2l7.5-2.5V19l-2 1.5V22l3-1 3 1v-1.5l-2-1.5v-5.5l7.5 2.5v-2L13 10.5V3c0-.55-.45-1-1-1z"/>
      </svg>
    </div>
  `;
  return L.divIcon({
    className: 'airplane-div-icon',
    html: iconSvg,
    iconSize: [24, 24],
    iconAnchor: [12, 12]
  });
}

function initFlightRadarMap() {
  if (flightMap) {
    flightMap.invalidateSize();
    fetchLiveFlightsForMap();
    return;
  }

  const mapEl = document.getElementById('flight-radar-map');
  if (!mapEl) return;

  flightMap = L.map('flight-radar-map', {
    center: [20.0, 0.0],
    zoom: 2,
    minZoom: 2,
    maxZoom: 18
  });

  // Google Earth Hybrid Tiles (Satellite + Road / Labels overlay, NO API KEY REQUIRED)
  L.tileLayer('https://mt{s}.google.com/vt/lyrs=y&x={x}&y={y}&z={z}', {
    subdomains: ['0', '1', '2', '3'],
    maxZoom: 20,
    attribution: '&copy; Google Earth &copy; OpenStreetMap'
  }).addTo(flightMap);

  fetchLiveFlightsForMap();

  let mapMoveTimeout = null;
  flightMap.on('moveend', () => {
    clearTimeout(mapMoveTimeout);
    mapMoveTimeout = setTimeout(fetchLiveFlightsForMap, 400);
  });

  if (mapRefreshTimer) clearInterval(mapRefreshTimer);
  mapRefreshTimer = setInterval(fetchLiveFlightsForMap, 10000); // refresh and calibrate from API every 10s

  if (mapDeadReckonTimer) clearInterval(mapDeadReckonTimer);
  mapDeadReckonTimer = setInterval(runMapKinematicsTick, 1000); // dead-reckoning animation every 1s
}

// Continuously advances aircraft markers along their heading and speed in real-time
function runMapKinematicsTick() {
  if (!flightMap || flightMarkersMap.size === 0) return;

  const dtSec = 1.0;
  const earthRadiusKm = 6371.0;

  for (const item of flightMarkersMap.values()) {
    if (item.onGround || item.speedKts < 25) continue;

    const distKm = (item.speedKts * 1.852) * (dtSec / 3600.0);
    const angularDist = distKm / earthRadiusKm;

    const lat0Rad = item.lat * Math.PI / 180.0;
    const lon0Rad = item.lon * Math.PI / 180.0;
    const headingRad = item.track * Math.PI / 180.0;

    const sinLat0 = Math.sin(lat0Rad);
    const cosLat0 = Math.cos(lat0Rad);
    const sinDist = Math.sin(angularDist);
    const cosDist = Math.cos(angularDist);

    const lat1Rad = Math.asin(sinLat0 * cosDist + cosLat0 * sinDist * Math.cos(headingRad));
    const lon1Rad = lon0Rad + Math.atan2(Math.sin(headingRad) * sinDist * cosLat0, cosDist - sinLat0 * Math.sin(lat1Rad));

    let lat1 = lat1Rad * 180.0 / Math.PI;
    let lon1 = lon1Rad * 180.0 / Math.PI;

    while (lon1 > 180.0) lon1 -= 360.0;
    while (lon1 < -180.0) lon1 += 360.0;

    item.lat = lat1;
    item.lon = lon1;
    item.marker.setLatLng([lat1, lon1]);
  }
}

async function showFlightRouteAndTrail(f) {
  if (!flightMap) return;
  clearActiveRouteAndTrack();

  const clearBtn = document.getElementById('btn-clear-track');
  if (clearBtn) clearBtn.style.display = 'inline-block';

  const hex = f.aircraft?.hex;
  const pos = f.telemetry?.position;
  const callsign = f.flight?.callsign_icao || f.aircraft?.hex || '';

  // 1. Draw planned route connecting origin -> aircraft -> destination
  const orig = f.route?.origin;
  const dest = f.route?.destination;
  if (orig && dest && orig.latitude && orig.longitude && dest.latitude && dest.longitude) {
    const routePoints = [
      [orig.latitude, orig.longitude],
      pos ? [pos.latitude, pos.longitude] : null,
      [dest.latitude, dest.longitude]
    ].filter(Boolean);

    const routePoly = L.polyline(routePoints, {
      color: '#3b82f6',
      weight: 2,
      opacity: 0.7,
      dashArray: '6, 8'
    }).addTo(flightMap);
    activeRouteLayers.push(routePoly);

    const origMarker = L.circleMarker([orig.latitude, orig.longitude], {
      radius: 6,
      fillColor: '#10b981',
      color: '#ffffff',
      weight: 1.5,
      fillOpacity: 0.95
    }).bindTooltip(`Departure: ${orig.icao} (${orig.name || ''})`).addTo(flightMap);
    activeRouteLayers.push(origMarker);

    const destMarker = L.circleMarker([dest.latitude, dest.longitude], {
      radius: 6,
      fillColor: '#ef4444',
      color: '#ffffff',
      weight: 1.5,
      fillOpacity: 0.95
    }).bindTooltip(`Destination: ${dest.icao} (${dest.name || ''})`).addTo(flightMap);
    activeRouteLayers.push(destMarker);
  }

  // 2. Fetch and draw historical track points
  const ident = hex || callsign;
  if (ident) {
    try {
      const res = await fetch(`/api/v1/flights/${encodeURIComponent(ident)}/history?limit=250`, {
        headers: currentToken ? { Authorization: `Bearer ${currentToken}` } : {}
      });
      if (res.ok) {
        const json = await res.json();
        const pts = json.data?.track_points || [];
        if (pts.length > 1) {
          const latlngs = pts.map((p) => [p.latitude, p.longitude]);
          const trackPoly = L.polyline(latlngs, {
            color: '#06b6d4',
            weight: 3.5,
            opacity: 0.95
          }).addTo(flightMap);
          activeRouteLayers.push(trackPoly);
        }
      }
    } catch (e) {
      console.warn('Failed to load historical track', e);
    }
  }
}

async function searchHistoricalTrack() {
  const query = document.getElementById('input-map-search-track')?.value.trim();
  if (!query) return;

  try {
    const res = await fetch(`/api/v1/history?callsign=${encodeURIComponent(query)}&limit=300`, {
      headers: currentToken ? { Authorization: `Bearer ${currentToken}` } : {}
    });
    const json = await res.json();
    if (!res.ok || !json.data || !json.data.track_points || json.data.track_points.length === 0) {
      alert(`No historical records found for "${query}" within retention period.`);
      return;
    }

    clearActiveRouteAndTrack();
    const data = json.data;
    const pts = data.track_points;
    const latlngs = pts.map((p) => [p.latitude, p.longitude]);

    const trackPoly = L.polyline(latlngs, {
      color: '#06b6d4',
      weight: 3.5,
      opacity: 0.95
    }).addTo(flightMap);
    activeRouteLayers.push(trackPoly);

    if (data.route?.origin && data.route?.destination) {
      const orig = data.route.origin;
      const dest = data.route.destination;
      if (orig.latitude && orig.longitude) {
        const om = L.circleMarker([orig.latitude, orig.longitude], {
          radius: 6,
          fillColor: '#10b981',
          color: '#ffffff',
          weight: 2,
          fillOpacity: 1
        }).bindTooltip(`Departure: ${orig.icao} (${orig.name || ''})`).addTo(flightMap);
        activeRouteLayers.push(om);
      }
      if (dest.latitude && dest.longitude) {
        const dm = L.circleMarker([dest.latitude, dest.longitude], {
          radius: 6,
          fillColor: '#ef4444',
          color: '#ffffff',
          weight: 2,
          fillOpacity: 1
        }).bindTooltip(`Destination: ${dest.icao} (${dest.name || ''})`).addTo(flightMap);
        activeRouteLayers.push(dm);
      }
    }

    flightMap.fitBounds(trackPoly.getBounds(), { padding: [40, 40] });
    const clearBtn = document.getElementById('btn-clear-track');
    if (clearBtn) clearBtn.style.display = 'inline-block';
  } catch (e) {
    alert('Error searching flight track.');
  }
}

async function fetchLiveFlightsForMap() {
  if (!flightMap) return;

  try {
    let bboxParam = '';
    const zoom = flightMap.getZoom();
    if (zoom >= 3) {
      const bounds = flightMap.getBounds();
      const south = Math.max(-85, bounds.getSouth());
      const north = Math.min(85, bounds.getNorth());
      let west = bounds.getWest();
      let east = bounds.getEast();
      if (east - west < 360) {
        west = Math.max(-180, Math.min(180, west));
        east = Math.max(-180, Math.min(180, east));
        if (west < east) {
          bboxParam = `&bbox=${south.toFixed(4)},${west.toFixed(4)},${north.toFixed(4)},${east.toFixed(4)}`;
        }
      }
    }

    let url = `/api/v1/flights?limit=1500${bboxParam}`;
    if (activeFlightFilter === 'emergencies') {
      url = `/api/v1/emergencies?limit=300`;
    } else if (activeFlightFilter === 'diverting') {
      url = `/api/v1/diversions?limit=300`;
    }

    // Pass auth header or portal key
    const res = await fetch(url, {
      headers: currentToken ? { Authorization: `Bearer ${currentToken}` } : {}
    });

    if (!res.ok) return;
    const json = await res.json();
    const flights = json.data || [];
    const now = Date.now();
    const seenHexes = new Set();

    flights.forEach((f) => {
      const hex = f.aircraft?.hex;
      const pos = f.telemetry?.position;
      if (!hex || !pos || !pos.latitude || !pos.longitude) return;

      seenHexes.add(hex);
      const track = f.telemetry?.attitude?.track_deg || 0;
      const isEmerg = f.emergency?.is_emergency;
      const isDiv = f.diversion_analysis?.predicted_diversion;
      const speedKts = f.telemetry?.speed?.ground_speed_kts || 0;
      const onGround = f.telemetry?.on_ground || false;
      const alt = Math.round(f.telemetry?.altitude?.barometric_ft || 0);

      const posSource = f.position_source || f.telemetry?.position_source || 'observed';
      const isStale = posSource === 'stale';
      const isEstimated = posSource === 'estimated';
      const estimatedAge = f.estimated_age_s || f.telemetry?.estimated_age_s || 0;

      let color = '#3b82f6';
      let markerOpacity = 1.0;
      if (isStale) {
        color = '#9ca3af';
        markerOpacity = 0.4;
      } else if (isEmerg) {
        color = '#ef4444';
      } else if (isDiv) {
        color = '#f59e0b';
      } else if (isEstimated) {
        markerOpacity = 0.55;
      }

      const callsign = f.flight?.callsign_icao || f.aircraft?.hex || 'UNKNOWN';
      const origin = f.route?.origin?.icao || 'N/A';
      const dest = f.route?.destination?.icao || 'N/A';
      const squawk = f.emergency?.squawk || '----';

      let sourceBadge = '<span style="color:#10b981;">Observed</span>';
      if (isStale) {
        sourceBadge = `<span style="color:#9ca3af;">Stale (+${estimatedAge}s)</span>`;
      } else if (isEstimated) {
        sourceBadge = `<span style="color:#f59e0b;">Estimated (+${estimatedAge}s)</span>`;
      }

      let divDetails = '';
      if (isDiv && f.diversion_analysis) {
        const da = f.diversion_analysis;
        const altInfo = da.candidate_alternate ? `${da.candidate_alternate.icao || ''} - ${da.candidate_alternate.name || ''} (${Math.round(da.candidate_alternate.distance_km || 0)} km)` : '';
        const reasons = (da.trigger_reasons || []).join(', ');
        divDetails = `
          <div class="flight-popup-row" style="color:#f59e0b;"><strong>DIVERSION ALERT (${da.probability_level || 'HIGH'})</strong></div>
          ${altInfo ? `<div class="flight-popup-row"><span>Pred. Alternate:</span> <span style="font-weight:600;color:#f59e0b;">${altInfo}</span></div>` : ''}
          ${reasons ? `<div class="flight-popup-row" style="font-size:11px;color:#9ca3af;"><span>Reasons:</span> <span>${reasons}</span></div>` : ''}
        `;
      }

      const popupHtml = `
        <div class="flight-popup-title">
          <span>${callsign}</span>
          <span style="font-size:11px; color:#9ca3af;">${hex}</span>
        </div>
        <div class="flight-popup-row"><span>Route:</span> <strong>${origin} &rarr; ${dest}</strong></div>
        <div class="flight-popup-row"><span>Altitude:</span> <span>${alt.toLocaleString()} ft</span></div>
        <div class="flight-popup-row"><span>Ground Speed:</span> <span>${Math.round(speedKts)} kts</span></div>
        <div class="flight-popup-row"><span>Heading / Squawk:</span> <span>${Math.round(track)}° / ${squawk}</span></div>
        <div class="flight-popup-row"><span>Source:</span> <span>${sourceBadge}</span></div>
        ${isEmerg ? `<div class="flight-popup-row" style="color:#ef4444;"><strong>EMERGENCY: ${f.emergency?.category || 'General'}</strong></div>` : ''}
        ${divDetails}
      `;

      if (flightMarkersMap.has(hex)) {
        // Recalibrate existing marker position with upstream ground truth
        const item = flightMarkersMap.get(hex);
        item.lastSeen = now;
        item.track = track;
        item.speedKts = speedKts;
        item.onGround = onGround;
        item.altFt = alt;

        const dLat = pos.latitude - item.lat;
        const dLon = pos.longitude - item.lon;
        if (Math.abs(dLat) > 2.0 || Math.abs(dLon) > 2.0) {
          item.lat = pos.latitude;
          item.lon = pos.longitude;
        } else {
          // Smooth calibration nudge towards freshly calibrated coordinates
          item.lat = item.lat * 0.3 + pos.latitude * 0.7;
          item.lon = item.lon * 0.3 + pos.longitude * 0.7;
        }

        item.marker.setLatLng([item.lat, item.lon]);
        item.marker.setOpacity(markerOpacity);
        item.marker.setPopupContent(popupHtml);

        if (Math.abs((item.lastTrack || 0) - track) > 4 || item.color !== color) {
          item.marker.setIcon(createAirplaneIcon(track, color));
          item.lastTrack = track;
          item.color = color;
        }
      } else {
        // Instantiate new marker
        const icon = createAirplaneIcon(track, color);
        const marker = L.marker([pos.latitude, pos.longitude], { icon, opacity: markerOpacity })
          .bindPopup(popupHtml)
          .addTo(flightMap);

        marker.on('click', () => {
          showFlightRouteAndTrail(f);
        });

        flightMarkersMap.set(hex, {
          marker,
          lat: pos.latitude,
          lon: pos.longitude,
          track,
          speedKts,
          altFt: alt,
          onGround,
          color,
          lastTrack: track,
          lastSeen: now
        });
      }
    });

    // Remove stale aircraft markers
    for (const [hex, item] of flightMarkersMap.entries()) {
      if (!seenHexes.has(hex) && now - item.lastSeen > 35000) {
        flightMap.removeLayer(item.marker);
        flightMarkersMap.delete(hex);
      }
    }
  } catch (e) {
    console.error('Map update error', e);
  }
}

async function loadOverviewTelemetry() {
  try {
    const res = await fetch('/api/v1/stats', {
      headers: currentToken ? { Authorization: `Bearer ${currentToken}` } : {}
    });
    if (res.ok) {
      const json = await res.json();
      const stats = json.data || {};
      document.getElementById('stat-val-tracked').textContent = (stats.total_tracked || 0).toLocaleString();
      const obsEl = document.getElementById('stat-val-observed');
      if (obsEl) obsEl.textContent = (stats.tracked_observed_count || 0).toLocaleString();
      const estEl = document.getElementById('stat-val-estimated');
      if (estEl) estEl.textContent = (stats.tracked_estimated_count || 0).toLocaleString();
      const staleEl = document.getElementById('stat-val-stale');
      if (staleEl) staleEl.textContent = (stats.tracked_stale_count || 0).toLocaleString();

      document.getElementById('stat-val-emergencies').textContent = (stats.emergencies || 0).toLocaleString();
      document.getElementById('stat-val-diverting').textContent = (stats.diverting || 0).toLocaleString();

      // Check upstream paused banner
      const banner = document.getElementById('upstream-paused-banner');
      const bannerText = document.getElementById('upstream-paused-text');
      if (stats.upstream_pause && stats.upstream_pause.is_paused) {
        if (banner) banner.style.display = 'flex';
        if (bannerText) {
          bannerText.textContent = `实时数据暂停，预计 ${stats.upstream_pause.resume_time_str || '--:--'} 恢复（${stats.upstream_pause.reason || '上游限额冷却中'}）`;
        }
      } else {
        if (banner) banner.style.display = 'none';
      }
    }

    // Load user stats
    const meRes = await fetch('/api/v1/portal/stats', {
      headers: { Authorization: `Bearer ${currentToken}` }
    });
    if (meRes.ok) {
      const meJson = await meRes.json();
      const s = meJson.data || {};
      document.getElementById('stat-val-today-req').textContent = (s.today_requests || 0).toLocaleString();
      document.getElementById('stat-val-quota').textContent = `${(s.today_requests || 0).toLocaleString()} / 10,000`;
      document.getElementById('stat-val-rpm').textContent = `${s.current_rpm || 0} / 60`;
    }
  } catch (e) {
    console.error('Failed to load overview telemetry', e);
  }
}

// --- Key Management ---
async function loadUserKeys() {
  const tbody = document.getElementById('keys-table-body');
  if (!tbody) return;

  try {
    const res = await fetch('/api/v1/portal/keys', {
      headers: { Authorization: `Bearer ${currentToken}` }
    });
    if (!res.ok) return;
    const json = await res.json();
    userKeys = json.data || [];

    const validKeys = userKeys.filter((k) => k.status !== 'revoked');

    tbody.innerHTML = '';
    if (validKeys.length === 0) {
      tbody.innerHTML = `<tr><td colspan="8" style="text-align:center; color:#6b7280; padding:24px;">${t('msg_no_keys')}</td></tr>`;
      return;
    }

    validKeys.forEach((k) => {
      const tr = document.createElement('tr');
      const statusBadge = `<span class="badge-tag badge-active">Active</span>`;

      tr.innerHTML = `
        <td><strong>${k.key_name}</strong></td>
        <td><code>${k.key_prefix}</code></td>
        <td>${new Date(k.created_at).toLocaleDateString()}</td>
        <td>${k.last_used_at ? new Date(k.last_used_at).toLocaleTimeString() : 'Never'}</td>
        <td>${k.today_requests.toLocaleString()}</td>
        <td>${k.total_requests.toLocaleString()}</td>
        <td>${statusBadge}</td>
        <td>
          <button class="btn btn-sm btn-danger" onclick="revokeKey(${k.id})">${t('btn_delete') || 'Delete'}</button>
        </td>
      `;
      tbody.appendChild(tr);
    });
  } catch (e) {
    console.error('Failed to load keys', e);
  }
}

window.revokeKey = async function (id) {
  if (!confirm('Are you sure you want to permanently delete this API key? This action cannot be undone.')) {
    return;
  }
  try {
    const res = await fetch(`/api/v1/portal/keys/${id}/revoke`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${currentToken}` }
    });
    if (res.ok) {
      loadUserKeys();
      loadPortalStats();
    } else {
      alert('Failed to delete key.');
    }
  } catch (e) {
    alert('Error deleting key.');
  }
};

// --- Admin Console Functions ---
let currentRegistrationEnabled = true;

async function loadAdminData() {
  loadAdminRegistrationStatus();
  loadAdminUsers();
  loadAdminFieldRestrictions();
  loadAdminAlertLogs();
  loadAdminUpstreamStatus();
  loadAdminUpstreamLogs();
}

async function loadAdminRegistrationStatus() {
  try {
    const res = await fetch('/api/v1/admin/registration-status', {
      headers: { Authorization: `Bearer ${currentToken}` }
    });
    if (!res.ok) return;
    const json = await res.json();
    currentRegistrationEnabled = json.data?.registration_enabled !== false;
    updateRegistrationStatusUI();
  } catch (e) {
    console.error('Failed to load registration status', e);
  }
}

function updateRegistrationStatusUI() {
  const badge = document.getElementById('admin-reg-status-badge');
  const btn = document.getElementById('btn-toggle-registration');
  if (!badge || !btn) return;

  if (currentRegistrationEnabled) {
    badge.textContent = t('status_channel_enabled') || 'Channel Enabled';
    badge.style.background = 'rgba(16, 185, 129, 0.15)';
    badge.style.color = '#10b981';
    badge.style.border = '1px solid rgba(16, 185, 129, 0.3)';
    btn.textContent = t('btn_disable_channel') || 'Disable Channel';
    btn.className = 'btn btn-outline';
    btn.style.color = '#ef4444';
    btn.style.borderColor = 'rgba(239, 68, 68, 0.4)';
  } else {
    badge.textContent = t('status_channel_disabled') || 'Channel Disabled';
    badge.style.background = 'rgba(239, 68, 68, 0.15)';
    badge.style.color = '#ef4444';
    badge.style.border = '1px solid rgba(239, 68, 68, 0.3)';
    btn.textContent = t('btn_enable_channel') || 'Enable Channel';
    btn.className = 'btn btn-amber';
    btn.style.color = '';
    btn.style.borderColor = '';
  }
}

async function loadAdminUsers() {
  const tbody = document.getElementById('admin-users-table-body');
  if (!tbody) return;

  try {
    const res = await fetch('/api/v1/admin/users', {
      headers: { Authorization: `Bearer ${currentToken}` }
    });
    if (!res.ok) return;
    const json = await res.json();
    const users = json.data || [];

    tbody.innerHTML = '';
    users.forEach((u) => {
      const tr = document.createElement('tr');
      const isBanned = u.status === 'banned';
      const roleBadge = u.role === 'admin' ? `<span class="badge-tag badge-admin">Admin</span>` : `<span class="badge-tag">User</span>`;
      const statusBadge = isBanned
        ? `<span class="badge-tag badge-banned">Banned</span>`
        : `<span class="badge-tag badge-active">Active</span>`;

      tr.innerHTML = `
        <td><strong>${u.name}</strong></td>
        <td>${u.email}</td>
        <td style="max-width:200px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">${u.purpose}</td>
        <td>${roleBadge}</td>
        <td>${statusBadge}</td>
        <td>${new Date(u.created_at).toLocaleDateString()}</td>
        <td>
          ${
            u.role !== 'admin'
              ? `
            <button class="btn btn-sm ${isBanned ? 'btn-outline' : 'btn-danger'}" onclick="toggleBanUser(${u.id}, '${isBanned ? 'active' : 'banned'}')">
              ${isBanned ? t('btn_unban') : t('btn_ban')}
            </button>
            <button class="btn btn-sm btn-outline" style="color:#ef4444;" onclick="deleteUser(${u.id})">${t('btn_delete')}</button>
          `
              : 'Master'
          }
        </td>
      `;
      tbody.appendChild(tr);
    });
  } catch (e) {
    console.error('Admin users load error', e);
  }
}

window.toggleBanUser = async function (id, targetStatus) {
  try {
    const res = await fetch(`/api/v1/admin/users/${id}/status`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${currentToken}`
      },
      body: JSON.stringify({ status: targetStatus })
    });
    if (res.ok) {
      loadAdminUsers();
    } else {
      alert('Failed to update user status.');
    }
  } catch (e) {
    alert('Network error.');
  }
};

window.deleteUser = async function (id) {
  if (!confirm('Are you sure you want to permanently delete this user and all their API keys?')) {
    return;
  }
  try {
    const res = await fetch(`/api/v1/admin/users/${id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${currentToken}` }
    });
    if (res.ok) {
      loadAdminUsers();
    } else {
      alert('Failed to delete user.');
    }
  } catch (e) {
    alert('Network error.');
  }
};

async function loadAdminFieldRestrictions() {
  try {
    const res = await fetch('/api/v1/admin/field-restrictions', {
      headers: { Authorization: `Bearer ${currentToken}` }
    });
    if (!res.ok) return;
    const json = await res.json();
    const allowed = json.data?.allowed_fields || [];
    const allowedMap = {};
    allowed.forEach((f) => (allowedMap[f.toLowerCase()] = true));

    document.querySelectorAll('.admin-field-chk').forEach((cb) => {
      cb.checked = !!allowedMap[cb.value.toLowerCase()];
    });
  } catch (e) {
    console.error('Failed to load field restrictions', e);
  }
}

async function loadAdminAlertLogs() {
  const tbody = document.getElementById('admin-alerts-table-body');
  if (!tbody) return;

  try {
    const res = await fetch('/api/v1/admin/alerts', {
      headers: { Authorization: `Bearer ${currentToken}` }
    });
    if (!res.ok) return;
    const json = await res.json();
    const alerts = json.data || [];

    tbody.innerHTML = '';
    if (alerts.length === 0) {
      tbody.innerHTML = `<tr><td colspan="5" style="text-align:center; color:#6b7280; padding:18px;">${t('msg_no_alerts')}</td></tr>`;
      return;
    }
    alerts.forEach((a) => {
      const tr = document.createElement('tr');
      tr.innerHTML = `
        <td>${a.email}</td>
        <td><code>${a.key_prefix}</code></td>
        <td style="color:#f59e0b;">${a.reason}</td>
        <td>${new Date(a.sent_at * 1000).toLocaleString()}</td>
        <td><span class="badge-tag badge-revoked">Dispatched</span></td>
      `;
      tbody.appendChild(tr);
    });
  } catch (e) {
    console.error('Alert logs load error', e);
  }
}

async function loadAdminUpstreamStatus() {
  const container = document.getElementById('upstream-status-cards');
  if (!container) return;

  try {
    const res = await fetch('/api/v1/admin/upstream-status', {
      headers: { Authorization: `Bearer ${currentToken}` }
    });
    if (!res.ok) return;
    const json = await res.json();
    const upstreams = json.data?.upstreams || {};
    const acc = json.data?.accuracy_stats_24h;

    let html = '';
    for (const [key, info] of Object.entries(upstreams)) {
      const isOS = key === 'opensky';
      const name = isOS ? 'OpenSky Network (Base Map)' : 'adsb.lol (Targeted Calibration)';
      let statusBadge = '<span class="status-badge" style="background:rgba(16,185,129,0.15); color:#10b981; border:1px solid rgba(16,185,129,0.3); padding:2px 8px; border-radius:4px; font-size:11px; font-weight:600;">OK</span>';
      if (info.status === 'cooldown') {
        statusBadge = '<span class="status-badge" style="background:rgba(245,158,11,0.15); color:#f59e0b; border:1px solid rgba(245,158,11,0.3); padding:2px 8px; border-radius:4px; font-size:11px; font-weight:600;">COOLDOWN</span>';
      } else if (info.status === 'error') {
        statusBadge = '<span class="status-badge" style="background:rgba(239,68,68,0.15); color:#ef4444; border:1px solid rgba(239,68,68,0.3); padding:2px 8px; border-radius:4px; font-size:11px; font-weight:600;">ERROR</span>';
      }

      const lastFetched = (info.last_fetched_count || 0).toLocaleString();
      const lastSuccess = info.last_success_at ? new Date(info.last_success_at).toLocaleTimeString() : '-';
      const nextReq = info.next_request_at ? new Date(info.next_request_at).toLocaleTimeString() : '-';
      const cooldownUntil = info.cooldown_until ? new Date(info.cooldown_until).toLocaleTimeString() : 'None';

      let budgetHtml = '';
      if (isOS && info.global_budget && info.regional_budget) {
        budgetHtml = `
          <div style="background:rgba(0,0,0,0.15); padding:8px 10px; border-radius:6px; margin-bottom:8px; font-size:11px;">
            <div style="font-weight:600; color:var(--accent-primary); margin-bottom:4px;">Global Base Map (65% share)</div>
            <div style="display:flex; justify-content:space-between; margin-bottom:2px;">
              <span style="color:var(--text-secondary);">24h Used / Rem:</span>
              <span><strong>${info.global_budget.used_past_24h}</strong> / <strong>${info.global_budget.remaining}</strong> (limit ${info.global_budget.daily_limit})</span>
            </div>
            <div style="display:flex; justify-content:space-between;">
              <span style="color:var(--text-secondary);">Pacing Interval:</span>
              <span><strong>${info.global_budget.current_interval_sec}s</strong> uniform</span>
            </div>
          </div>
          <div style="background:rgba(0,0,0,0.15); padding:8px 10px; border-radius:6px; margin-bottom:8px; font-size:11px;">
            <div style="font-weight:600; color:#38bdf8; margin-bottom:4px;">Regional BBox Calibration (35% share)</div>
            <div style="display:flex; justify-content:space-between;">
              <span style="color:var(--text-secondary);">24h Used / Rem:</span>
              <span><strong>${info.regional_budget.used_past_24h}</strong> / <strong>${info.regional_budget.remaining}</strong> (limit ${info.regional_budget.daily_limit})</span>
            </div>
          </div>
        `;
      } else if (!isOS) {
        budgetHtml = `
          <div style="background:rgba(0,0,0,0.15); padding:8px 10px; border-radius:6px; margin-bottom:8px; font-size:11px;">
            <div style="font-weight:600; color:#10b981; margin-bottom:4px;">Daily Request Quota (450 Limit)</div>
            <div style="display:flex; justify-content:space-between; margin-bottom:2px;">
              <span style="color:var(--text-secondary);">Today Used / Rem:</span>
              <span><strong>${info.used_today || 0}</strong> / <strong>${info.remaining_today || 0}</strong></span>
            </div>
            <div style="display:flex; justify-content:space-between; margin-bottom:2px;">
              <span style="color:var(--text-secondary);">Window Reset:</span>
              <span>${info.window_reset_at ? new Date(info.window_reset_at).toLocaleTimeString() : 'Midnight UTC'}</span>
            </div>
            <div style="display:flex; justify-content:space-between;">
              <span style="color:var(--text-secondary);">Calibration Queue:</span>
              <span><strong>${info.calibration_queue_len || 0}</strong> queued (${info.calibrations_completed_24h || 0} done 24h)</span>
            </div>
          </div>
        `;
      }

      html += `
        <div style="background:var(--bg-card); border:1px solid var(--border-subtle); border-radius:var(--radius-md); padding:16px;">
          <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:12px;">
            <strong style="font-size:14px;">${name}</strong>
            ${statusBadge}
          </div>
          ${budgetHtml}
          <div style="display:flex; justify-content:space-between; margin-bottom:6px; font-size:12px;">
            <span style="color:var(--text-secondary);">Last Fetched:</span>
            <span><strong>${lastFetched}</strong> aircraft (${lastSuccess})</span>
          </div>
          <div style="display:flex; justify-content:space-between; margin-bottom:6px; font-size:12px;">
            <span style="color:var(--text-secondary);">Next Request:</span>
            <span>${nextReq}</span>
          </div>
          <div style="display:flex; justify-content:space-between; font-size:12px;">
            <span style="color:var(--text-secondary);">Cooldown Until:</span>
            <span style="${info.cooldown_until ? 'color:#f59e0b;font-weight:600;' : ''}">${cooldownUntil}</span>
          </div>
        </div>
      `;
    }

    if (acc) {
      html += `
        <div style="background:var(--bg-card); border:1px solid var(--border-subtle); border-radius:var(--radius-md); padding:16px;">
          <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:12px;">
            <strong style="font-size:14px;">Kinematics Extrapolation Accuracy</strong>
            <span class="status-badge" style="background:rgba(59,130,246,0.15); color:#60a5fa; border:1px solid rgba(59,130,246,0.3); padding:2px 8px; border-radius:4px; font-size:11px; font-weight:600;">24h METRICS</span>
          </div>
          <div style="background:rgba(0,0,0,0.15); padding:8px 10px; border-radius:6px; margin-bottom:8px; font-size:11px;">
            <div style="display:flex; justify-content:space-between; margin-bottom:4px;">
              <span style="color:var(--text-secondary);">Mean Position Error:</span>
              <span><strong>${acc.mean_displacement_error_km ?? 0} km</strong></span>
            </div>
            <div style="display:flex; justify-content:space-between; margin-bottom:4px;">
              <span style="color:var(--text-secondary);">P95 Position Error:</span>
              <span><strong>${acc.p95_displacement_error_km ?? 0} km</strong></span>
            </div>
            <div style="display:flex; justify-content:space-between;">
              <span style="color:var(--text-secondary);">Total Calibrated Samples:</span>
              <span><strong>${(acc.total_calibrated_samples || 0).toLocaleString()}</strong> points</span>
            </div>
          </div>
          <div style="font-size:11px; color:var(--text-secondary); line-height:1.4;">
            Accuracy logs compare physics-extrapolated dead reckoning coordinates against newly arrived real sensor observations.
          </div>
        </div>
      `;
    }

    container.innerHTML = html || '<div style="color:var(--text-secondary);font-size:13px;">No upstream providers configured.</div>';
  } catch (err) {
    console.error('Failed to load upstream status:', err);
  }
}

async function loadAdminUpstreamLogs() {
  const tbody = document.getElementById('admin-upstream-table-body');
  if (!tbody) return;

  const providerFilter = document.getElementById('select-upstream-provider')?.value || '';
  const statusFilter = document.getElementById('select-upstream-status')?.value || '';
  const hubFilter = document.getElementById('input-upstream-hub')?.value?.trim() || '';

  let url = `/api/v1/admin/upstream-logs?limit=50`;
  if (providerFilter) url += `&provider=${encodeURIComponent(providerFilter)}`;
  if (statusFilter) url += `&status=${encodeURIComponent(statusFilter)}`;
  if (hubFilter) url += `&hub=${encodeURIComponent(hubFilter)}`;

  try {
    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${currentToken}` }
    });
    if (!res.ok) return;
    const json = await res.json();
    const logs = json.data || [];

    tbody.innerHTML = '';
    if (logs.length === 0) {
      tbody.innerHTML = `<tr><td colspan="8" style="text-align:center; color:#6b7280; padding:18px;">${t('msg_no_upstream_logs')}</td></tr>`;
      return;
    }

    logs.forEach((l) => {
      const tr = document.createElement('tr');
      let statusBadge = `<span class="badge-tag badge-active">${l.status}</span>`;
      if (l.status === 'rate_limited') {
        const rep = (l.repeat_count && l.repeat_count > 1) ? ` <span style="font-size:11px; font-weight:700; color:#f59e0b;">(x${l.repeat_count})</span>` : '';
        statusBadge = `<span class="badge-tag" style="background:rgba(245,158,11,0.15); color:#f59e0b; border:1px solid rgba(245,158,11,0.3);">Rate Limited${rep}</span>`;
      } else if (l.status === 'error') {
        statusBadge = `<span class="badge-tag badge-banned">Error</span>`;
      }

      const codeBadge = l.status_code === 200
        ? `<code style="color:#10b981;">200 OK</code>`
        : l.status_code === 429
        ? `<code style="color:#f59e0b;">429 Rate Limit</code>`
        : l.status_code > 0
        ? `<code style="color:#ef4444;">${l.status_code}</code>`
        : `<code style="color:#6b7280;">-</code>`;

      const timeStr = l.timestamp ? new Date(l.timestamp * 1000).toLocaleTimeString() : '-';
      const durationStr = l.duration_ms > 0 ? `${l.duration_ms} ms` : '-';
      const countStr = l.aircraft_count ? l.aircraft_count.toLocaleString() : '0';
      const hubStr = l.hub ? `<code>${l.hub}</code>` : `<span style="color:#6b7280;">global</span>`;

      tr.innerHTML = `
        <td>${timeStr}</td>
        <td><code>${l.provider}</code></td>
        <td>${hubStr}</td>
        <td>${statusBadge}</td>
        <td>${codeBadge}</td>
        <td><strong>${countStr}</strong></td>
        <td>${durationStr}</td>
        <td style="max-width:280px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; color:${l.error_message ? '#f87171' : '#9ca3af'};" title="${l.error_message || ''}">
          ${l.error_message || 'Normal telemetry ingestion'}
        </td>
      `;
      tbody.appendChild(tr);
    });
  } catch (e) {
    console.error('Upstream logs load error', e);
  }
}

document.getElementById('btn-refresh-upstream-status')?.addEventListener('click', loadAdminUpstreamStatus);
document.getElementById('btn-refresh-upstream-logs')?.addEventListener('click', loadAdminUpstreamLogs);
document.getElementById('select-upstream-provider')?.addEventListener('change', loadAdminUpstreamLogs);
document.getElementById('select-upstream-status')?.addEventListener('change', loadAdminUpstreamLogs);
document.getElementById('input-upstream-hub')?.addEventListener('keydown', (e) => {
  if (e.key === 'Enter') loadAdminUpstreamLogs();
});

document.getElementById('btn-send-test-alert')?.addEventListener('click', async () => {
  const input = document.getElementById('input-test-alert-email');
  const msg = document.getElementById('test-alert-status-msg');
  const email = input?.value?.trim() || '';

  if (msg) {
    msg.style.display = 'block';
    msg.style.background = '#374151';
    msg.style.color = '#f3f4f6';
    msg.textContent = `Dispatching test alert to ${email}...`;
  }

  try {
    const res = await fetch('/api/v1/admin/send-test-alert', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${currentToken}`
      },
      body: JSON.stringify({ recipient: email })
    });
    const data = await res.json();
    if (res.ok) {
      if (msg) {
        msg.style.background = 'rgba(16,185,129,0.15)';
        msg.style.color = '#10b981';
        msg.textContent = `Test alert email successfully dispatched to ${email}!`;
      }
    } else {
      if (msg) {
        msg.style.background = 'rgba(239,68,68,0.15)';
        msg.style.color = '#ef4444';
        msg.textContent = `Failed: ${data.error || 'Server error'}`;
      }
    }
  } catch (e) {
    if (msg) {
      msg.style.background = 'rgba(239,68,68,0.15)';
      msg.style.color = '#ef4444';
      msg.textContent = `Network error: ${e.message}`;
    }
  }
});

