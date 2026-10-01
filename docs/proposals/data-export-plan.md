# 提案：商家后台数据导出（会员 / 办卡 / 消费）

> 状态：**规划稿 v2（未实施）** — 2026-10-01 立项；同日二稿：应用户新增「手机直开 / 喂 AI 分析」场景，文本格式裁决为 **TXT（Markdown 内容）**，CSV 暂缓（见 §3.1）
> 原则：沿用项目冻结口径「最简 / 不做」；只做**只读导出**，不做导入、不做报表平台。
> 关联：AGENTS.md 模块边界与 D 系列决策；本提案若被批准，建议以 D30 落决策。

---

## 1. 要解决的问题

老板目前只能看后台页面，无法把数据带出去：

- 对账 / 报税：微信转账、现金收款流水需要能导出给记账的；
- 会员经营：想把「谁有卡、还剩几次、谁很久没来」拉一张表，微信发自己或打印；
- 备份焦虑之外的可读备份：SQLite 备份文件老板看不懂，能看懂的 Excel 才是他的"备份"；
- **AI 经营分析**（v2 新增）：把会员/消费数据直接丢给 AI 问"谁该回访了、哪张卡快耗完"——需要一个手机能直接打开、AI 能直接读懂的文本格式。

## 2. 对标调研结论

| 系统 | 可借鉴的点 | 我们不抄的部分 |
|------|-----------|---------------|
| **Shopify**（客户导出 CSV） | ①客户行上直接聚合 `Total Spent / Total Orders`——老板拿一行就知道这个客户值多少钱；②列 schema 稳定、有官方文档；③按当前筛选条件导出 | 地址簿、营销订阅字段、re-import 回导能力（无此需求） |
| **观达云 / 美业 SaaS**（会员导出） | 导出时勾选附加字段（余额、最近到店、偏好项目）——说明美业老板要的就是"会员主表 + 卡余次 + 最近到店"合一 | 自定义字段勾选器（列集合不稳定，出问题难排查；我们固定精选列，见 §5） |
| **简道云**（会员三表模型） | 会员信息表 / 充值表 / 消费登记表分开导出再互相关联——与我们 member / member_card / payment 的表模型天然同构 | 表单平台能力 |
| **excelize**（Go 生态事实标准） | 纯 Go、无 CGO，与 modernc.org/sqlite 的"零 CGO"约束一致；万行级数据用默认 API 内存构建完全够用（StreamWriter 是 10 万行级才需要的优化，见 excelize #382/#590） | — |
| **CSV 乱码实践**（Java/后端社区共识） | Windows Excel 按 GBK 猜编码 → 无 BOM 的 UTF-8 CSV 中文必乱码；文件名中文需 `filename*=UTF-8''…`（RFC 5987）；`= + - @` 开头的文本有公式注入风险 | — |
| **LLM 喂数实践**（v2 新增） | Markdown 是主流 AI 的最佳结构化输入之一（表格/标题天然可解析）；纯文本 .txt 是手机端兼容性的最大公共分母（微信内置预览稳、iOS/安卓文件管理全支持） | JSON / CSV 喂 AI（手机不可读、AI 还要额外转译） |

**结论**：双格式交付——**XLSX**（excelize，表头加粗 + 列宽，中文文件名）承担对账/电脑端；**TXT（Markdown 内容）**承担手机直开 + 喂 AI；CSV 暂缓（受众与 TXT 重叠，仅"喂其他程序"场景需要，届时 ~15 行即可补上）。金额统一由「分」换算为「元」两位小数；时间一律库内墙上时间原样输出。

## 3. 导出什么（6 个域，分期）

### P1 — 第十批（对准"已办卡、已消费"原话）

| # | 域 | 端点（POST） | 一行 = | 核心列（中文表头） |
|---|----|-------------|--------|--------------------|
| 1 | **会员总表** | `/admin/export/members` | 一个会员 | 会员编号、姓名、手机号、性别、生日、微信绑定(是/否)、标签、注册时间、**最近到店**、**累计消费(元)**、消费次数、**有效卡数**、**剩余总次数**、备注 |
| 2 | **收款流水** | `/admin/export/payments` | 一笔 payment（含 VOIDED，带状态列供老板筛选） | 收款时间、会员编号、姓名、手机号、预约号、支付方式(中文)、金额(元)、状态(有效/已作废) |
| 3 | **服务记录** | `/admin/export/service-records` | 一条 service_record | 时间、会员编号、姓名、手机号、服务名称(快照)、部位、手法、商家备注、状态(正常/已撤销)、预约号 |

### P2 — 视 P1 使用情况追加

| # | 域 | 端点 | 一行 = | 核心列 |
|---|----|------|--------|--------|
| 4 | 会员卡明细 | `/admin/export/cards` | 一张 member_card | 会员编号、姓名、手机号、卡名称、类型(次卡/活动卡)、总次数、剩余次数、生效日、失效日、状态、发卡时间 |
| 5 | 预约记录 | `/admin/export/appointments` | 一条预约 | 预约号、会员编号、姓名、手机号、服务(快照)、日期、起止时间、状态、备注、创建时间 |
| 6 | 卡流水 | `/admin/export/card-transactions` | 一条 card_transaction | 时间、会员编号、姓名、卡名称、类型(发卡/核销/调整/撤销)、次数变动、发生时间 |

> 会员总表的聚合口径：`累计消费 = SUM(payment.amount WHERE status='VALID')`（**列名是 `amount`，单位分**——v3 修订，原稿误写 amount_cents）；`消费次数 = COUNT(VALID payment 且 member_id 非空)`；`有效卡数/剩余总次数` 只算 `status='ACTIVE'` 的卡。余额以流水/卡表为准导出快照值，与"remaining_count 是缓存"不变量一致（导出是只读视图，不改真相）。
>
> **聚合取法（v3 修订）**：member 导出查询**直接在 member 模块内写 GROUP BY JOIN SQL**（JOIN payment / member_card，先例 `member/repo.go:170`）——不注入 card.Api/transaction.Api。原因：transaction.Api import 了 member 包，反向注入即 import cycle；模块边界禁的是 Go import 而非 SQL。标签列走批量 JOIN（模板 `repo.go:201-229` decorateTags），禁逐会员 N+1。
>
> **收款流水 LEFT JOIN 口径（v3 修订）**：payment.member_id 自 migration 015 起可空（D29 散客不录手机号）——LEFT JOIN member，空行会员列留白，老板能对账散客现金。

> 三个域均以请求体 `format` 切换格式：`"xlsx"`（默认）| `"txt"`（Markdown 内容，文件扩展名 .txt）。P2 各域同规则。
>
> **每域筛选 schema（v3 修订钉死，与列表端点参数逐字同名；DecodeJSON 开 DisallowUnknownFields，多传即 400）**：
> - `members`：`{format, keyword?, tag_id?, card_type?}`（对照 GET /admin/members）
> - `payments`：`{format, status?, date_from?, date_to?}`（status ∈ VALID/VOIDED，对照 GET /admin/payments）
> - `service-records`：`{format, date_from?, date_to?}`
> - 空 body 拒绝：前端至少发 `{"format":"xlsx"}`；日期字段 P1 后端即支持、前端控件 P2。

### 3.1 文本格式裁决与专项设计（v2 新增）

**Markdown 和 TXT 只保留一个，且不是"二选一"而是"合一"**：

- 同内容的 `.md` 与 `.txt` 是**同一份字节**，区别只在扩展名决定手机上用哪个 App 打开 → 双格式 = 双份冗余文件 + 两个按钮的困惑，否决；
- 扩展名取 **`.txt`**：微信内置文件预览、iOS/安卓系统文本查看器 100% 兼容；`.md` 在部分安卓机上没有关联应用，点开即失败——手机直开是首要场景，公共分母优先；
- 内容用 **Markdown**：`#` 标题 / 表格 / 列表语义完整，粘进任意 AI 即得结构化输入；人眼阅读时这些符号噪音极小。

**排版规则（手机窄屏优先，AI 兼容）**：

| 域宽 | 排版 | 示例 |
|------|------|------|
| ≤8 列（收款流水 8 列） | Markdown 表格 | `\| 收款时间 \| 姓名 \| 方式 \| 金额 \|` |
| >8 列（会员总表 14 列、服务记录 10 列） | 每条记录一个"卡片块" | 见下 |

```markdown
### M202609280001 张三
- 手机号：138****1234
- 累计消费：¥3,580.00（12 次）
- 有效卡数：1，剩余总次数：8
- 最近到店：2026-09-28 15:30
```

> 14 列管道表在手机上必然横向滚动成灾难；卡片块在手机上从上往下读，粘给 AI 同样逐字段可解析。各域排版在导出注册表里声明（`layout: table|cards`），一行配置。

**手机号打码（仅 TXT）**：TXT 默认输出打码号——TXT 的两大用途（手机看、喂 AI）都不需要拨号，而喂云端 AI 属于把 PII 送出系统，默认最小化；真实手机号在 XLSX 里全量保留（对账/联系场景走 Excel）。不加参数、不加 UI，按格式内置。打码边界（v3 修订）：11 位标准 `前3+****+后4`；8~10 位保头尾各 2；不足 8 位一律 `***`；空串输出空串。前端按钮 tooltip 注明「TXT 手机号已打码」，防老板误判数据坏了。

**其余口径同 XLSX**：金额分→元两位小数、时间原样、`text/plain; charset=utf-8` + `Content-Disposition: attachment`（中文文件名 RFC 5987 双写法）。

## 4. 架构方案（遵守模块边界与 D7）

```
请求: POST /admin/export/{domain}   body: {"format":"xlsx"|"txt", ...筛选}   （Admin JWT）
  │  实际中间件链（v3 修订为真实顺序）：RequestID→BodyLimit(1MB)→Logging→OperationLog
  │  →AuthRateLimit(对导出 no-op，仅登录路径)→Recover→AdminAuth
  │  OperationLog 在鉴权之外：未认证 401 也落审计（现状行为，预期内）
  ▼
模块 handler（各域数据各回各家，禁止跨模块抄 repo）
  member/handler.go      ExportMembers   ← 模块内 GROUP BY JOIN payment/member_card（先例 repo.go:170）
  transaction/handler.go ExportPayments  ← LEFT JOIN member + appointment
  service/handler.go     ExportServiceRecords
  ▼
shared/export（新公共工具包，单一出口写响应；≈300 行）
  • build-then-write 纪律（v3）：先全量内存渲染，成功才设头+单次 Write；
    失败一律发生在首字节前 → 正常 shared.Fail JSON 错误
  • format=xlsx → excelize 默认 API 内存构建；行数硬顶 50_000（v3 下调，
    10 万行×14 列默认 API 可达数百 MB）→ 超限 400 EXPORT_TOO_LARGE
  • format=txt → Markdown 渲染器：表格/卡片双排版（按域声明 layout）
    + 手机号打码 + UTF-8 直出（无 BOM 需求，.txt 不经 Excel）
  • 写响应：Content-Type (xlsx / text/plain;charset=utf-8)
    Content-Disposition 用 mime.FormatMediaType 生成（ASCII 兜底 + UTF-8 中文名）
  • 安全：xlsx 文本单元格过公式注入防护（^=+-@ 前缀补 `'`）；
    txt 粘贴场景无公式执行面，无需处理
  • 金额：分→元两位小数（数字型，Excel 可 SUM）；空手机号/生日输出空串
  ▼
审计（v3 修订落地机制）：
  • handler 在 WriteHeader 前设响应头 X-Export-Rows
  • OperationLog 回调签名扩为 func(r, status int, hdr http.Header)，
    app.go logEntry 读 hdr.Get("X-Export-Rows") 写入 detail（domain 在 path 里天然有）
  • 筛选回显 P2 再说（detail VARCHAR(2000) 需截断）；行数同时供前端 toast「已导出 N 行」
```

前端（apps/admin）：

- `core/api/http.ts` 内新增 `downloadBlob(path, body)`（v3 修订落位：tokenProvider/onUnauthorized 是 http.ts 模块私有，放别处拿不到）——复用既有 Authorization 头 → 先判 `res.ok`（错误仍是 JSON 包络，解析后抛 ApiError，401 走全局登出）→ `blob()` → `URL.createObjectURL` → `a.download` 触发保存；
- **文件名前端自造**（v3 修订）：blob 下载浏览器不读 Content-Disposition，按域自造 `anmo-会员-YYYYMMDD.xlsx/.txt` 等固定名；后端 RFC 5987 头保留作直连/同源兜底；
- **导出按钮落位（v3 修订）**：MembersPage 工具栏「导出 Excel」「导出 TXT」（会员总表，随 keyword/tag/card_type 筛选走）；RecordsPage 页头一个「导出」下拉（Element Plus el-dropdown）收四项——收款流水 Excel/TXT（随 payStatus 走）+ 服务记录 Excel/TXT。**tab2 是核销记录(redemption)不是服务记录**（v3 审查实锤），按域不能钉在错 tab 上；不新做"导出中心"页；
- 每域两枚按钮共享一个 `exporting` ref（导出中全禁点）；响应头 `X-Export-Rows` → 成功 toast「已导出 N 行」，0 行 toast「导出结果为空」；错误 toast 走既有 ElMessage.error(e.message)。

## 5. 明确不做（冻结口径）

- ❌ 自定义字段勾选器 — 列集合固定精选，稳定可文档化；
- ❌ 导入 / 回写（Shopify 式 re-import）— 永不；
- ❌ 异步导出任务、文件下载中心、OSS 上传 — 单店量级同步流式即可；
- ❌ 定时邮件/微信推送报表 — 老板自己点；
- ❌ Excel 复杂样式（配色/图表/多 sheet 汇总页）— 仅表头加粗 + 合理列宽；
- ❌ 顾客端任何导出入口 — 数据只从后台出，且每次导出留审计；
- ❌ **.md 与 .txt 双格式并行**（v2 裁决）— 同字节双份冗余；只出 .txt（Markdown 内容）；
- ❌ **CSV 端点暂缓**（v2 裁决）— 与 TXT 受众重叠（纯文本都可读），仅"喂其他程序/再导入"场景需要，届时 ~15 行补上；
- ❌ JSON 导出 — 手机不可读，喂 AI 不如 Markdown；
- ❌ **核销记录（redemption）导出**（v3 修订）— 服务消费事实已由 service_record 域覆盖，次数流水属 P2 卡流水域，单独再出一个核销域是第三份重复账；
- ❌ 导出筛选的日期选择器 UI（P1）— 后端 body 预留 `date_from/date_to` 字段（payments/service-records），P1 前端不提供日期控件，量级到顶再补。

## 6. 风险与对策

| 风险 | 对策 |
|------|------|
| 导出文件含 PII（手机号）外泄 | 仅 ADMIN 角色可用；POST 全量审计（谁、何时、哪个域、多少行）；文件名含日期便于追溯 |
| 慢查询拖垮库 | 全部只读 SELECT；行数硬顶 5 万；会员总表聚合用**两条 GROUP BY**（payment 按 member_id、member_card 按 member_id）+ member 主表一条，不做逐会员子查询（v3 修订：payment 无 member_id 索引，全表 GROUP BY 在单店量级是毫秒级，勿为此加 migration）；量级异常时走 body 预留的 date_from 限窗 |
| excelize 内存 | 当前量级（千行）无虞；硬顶 5 万兜底；超顶提示分日期段导出（P2 补日期控件） |
| 中文文件名各浏览器兼容 | 同时给 `filename`（ASCII 兜底）与 `filename*=UTF-8''`（RFC 5987） |
| 公式注入（备注/姓名以 = - 开头） | xlsx 统一前置 `'`；TXT 无公式执行面，不需处理 |
| **TXT 喂云端 AI 携带 PII**（v2 新增） | TXT 默认手机号打码（§3.1）；xlsx 留全量本地用；导出行为全量审计可追溯 |

## 7. 工作量预估（第十批 = P1）

| 项 | 估量 |
|----|------|
| shared/export 工具包 + 单测（BOM/注入/金额/文件名） | ~160 行 + 测试 |
| shared/export TXT(Markdown) 渲染器（表格/卡片双排版 + 手机号打码）+ 单测 | ~140 行 + 测试 |
| member 会员总表（聚合查询 + handler + api 注入 card/transaction） | ~120 行 |
| transaction 收款流水、service 服务记录（各查询 + handler） | 各 ~60 行 |
| admin downloadBlob + 导出按钮（MembersPage 两枚 + RecordsPage 下拉四项） | ~110 行 |
| OperationLog 扩 hdr 参数 + app.go logEntry 读 X-Export-Rows | ~10 行 |
| e2e：登录→导出→断言 xlsx 魔数 PK / txt Markdown 结构与打码与行数 | ~80 行 |
| 交付：admin dist 重建（--base=/admin-ui/）+ nginx 原子换装 + docs/CHANGELOG.md 条目 | 部署项 |

## 8. 验收口径

1. MembersPage 按当前筛选导出 → Excel 双击打开无乱码，手机号列文本格式不丢 0；
2. 收款流水含 VOIDED 行且状态列可筛；金额与 Dashboard 当期 VALID 合计一致；
3. 服务记录中已撤销行状态=已撤销（与 D28 联动快照语义一致）；
4. 每次 POST 导出在操作日志可见 domain+行数；
5. ~~CSV 格式用 Excel（Windows）打开中文不乱码~~（CSV 暂缓，随端点顺延）；
6. **TXT（v2）**：微信发送到手机后可直接打开阅读；会员总表为卡片块、收款流水为 5 列精简表格（v3：时间/姓名/方式/金额/状态，防 375px 横滚）；
7. **TXT（v2）**：整文件粘给 AI 后结构可解析（标题/表格/字段名完整）；所有手机号已按边界规则打码（§3.1）；
8. **v3**：导出成功 toast「已导出 N 行」、0 行 toast「导出结果为空」，N 来自响应头 X-Export-Rows；
9. **v3**：payment.member_id 为 NULL 的散客收款行出现在收款流水中，会员列留白。

## 9. 审查修订记录（v3，2026-10-01）

双子代理对抗审查（后端架构线 + 前端横切线）结论均为 **GO**，以下实锤已采纳并写回正文：

| # | 审查发现 | 修订 |
|---|---------|------|
| 1 | 列名实为 `payment.amount`（分），无 amount_cents（006:14） | §3 聚合口径更正 |
| 2 | member→transaction.Api 注入即 import cycle（transaction/api.go:10 import member） | 改为 member 模块内 GROUP BY JOIN SQL（先例 repo.go:170），card 方向同理一并简化 |
| 3 | OperationLog 只记 method/status/query，筛选与行数不可得 | X-Export-Rows 响应头 + 回调扩 hdr 参数（§4），筛选回显降 P2 |
| 4 | blob 下载浏览器不读 Content-Disposition，文件名会变 blob:UUID | 前端自造固定文件名（§4），后端头保留兜底 |
| 5 | RecordsPage tab2 是核销记录(redemption)非服务记录，按钮钉错表 | 改页头 el-dropdown 收四项；redemption 导出列入不做 |
| 6 | downloadBlob 必须落 http.ts 内（tokenProvider 模块私有）+ 错误包络/401 路径 | §4 明确 |
| 7 | payment.member_id 015 起可空，导出须 LEFT JOIN 留白 | §3 LEFT JOIN 口径 + 验收 9 |
| 8 | 打码边界（非 11 位）未定义，len<8 可能泄漏 | §3.1 边界规则 |
| 9 | 10 万行硬顶对 excelize 默认 API 偏大（数百 MB） | 下调 50_000 |
| 10 | "payment(member_id,status) 既有索引"断言不实（仅 appointment 与 status,recorded_at） | §6 改 GROUP BY 口径，不加 migration |
| 11 | 中间件链顺序写反、AuthRateLimit 对导出是 no-op | §4 更正为真实顺序 |
| 12 | TXT 收款 8 列在 375px 仍横滚 | TXT 收款降 5 列（验收 6） |
| 13 | 交付缺 admin dist 重建/换装/CHANGELOG | §7 补交付行 |
| 14 | 消费次数口径未定义 | §3 钉死 = COUNT(VALID payment 且 member_id 非空) |
