# 提案：商家后台数据导出（会员 / 办卡 / 消费）

> 状态：**规划稿（未实施）** — 2026-10-01，第九批收口后立项
> 原则：沿用项目冻结口径「最简 / 不做」；只做**只读导出**，不做导入、不做报表平台。
> 关联：AGENTS.md 模块边界与 D 系列决策；本提案若被批准，建议以 D30 落决策。

---

## 1. 要解决的问题

老板目前只能看后台页面，无法把数据带出去：

- 对账 / 报税：微信转账、现金收款流水需要能导出给记账的；
- 会员经营：想把「谁有卡、还剩几次、谁很久没来」拉一张表，微信发自己或打印；
- 备份焦虑之外的可读备份：SQLite 备份文件老板看不懂，能看懂的 Excel 才是他的"备份"。

## 2. 对标调研结论

| 系统 | 可借鉴的点 | 我们不抄的部分 |
|------|-----------|---------------|
| **Shopify**（客户导出 CSV） | ①客户行上直接聚合 `Total Spent / Total Orders`——老板拿一行就知道这个客户值多少钱；②列 schema 稳定、有官方文档；③按当前筛选条件导出 | 地址簿、营销订阅字段、re-import 回导能力（无此需求） |
| **观达云 / 美业 SaaS**（会员导出） | 导出时勾选附加字段（余额、最近到店、偏好项目）——说明美业老板要的就是"会员主表 + 卡余次 + 最近到店"合一 | 自定义字段勾选器（列集合不稳定，出问题难排查；我们固定精选列，见 §5） |
| **简道云**（会员三表模型） | 会员信息表 / 充值表 / 消费登记表分开导出再互相关联——与我们 member / member_card / payment 的表模型天然同构 | 表单平台能力 |
| **excelize**（Go 生态事实标准） | 纯 Go、无 CGO，与 modernc.org/sqlite 的"零 CGO"约束一致；万行级数据用默认 API 内存构建完全够用（StreamWriter 是 10 万行级才需要的优化，见 excelize #382/#590） | — |
| **CSV 乱码实践**（Java/后端社区共识） | Windows Excel 按 GBK 猜编码 → 无 BOM 的 UTF-8 CSV 中文必乱码；文件名中文需 `filename*=UTF-8''…`（RFC 5987）；`= + - @` 开头的文本有公式注入风险 | — |

**结论**：主交付 **XLSX**（excelize，表头加粗 + 列宽，中文文件名），同一引擎顺带产出 **CSV（UTF-8 BOM）** 作为备选格式。金额统一由「分」换算为「元」两位小数；时间一律库内墙上时间原样输出。

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

> 会员总表的聚合口径：`累计消费 = SUM(payment.amount_cents WHERE status='VALID')`；
> `有效卡数/剩余总次数` 只算 `status='ACTIVE'` 的卡。余额以流水/卡表为准导出快照值，与"remaining_count 是缓存"不变量一致（导出是只读视图，不改真相）。

## 4. 架构方案（遵守模块边界与 D7）

```
请求: POST /admin/export/{domain}   body: {"filters...}   （Admin JWT）
  │  中间件既有链自然生效：AuthRateLimit→OperationLog(非 GET 落审计)→BodyLimit→RequestID
  ▼
模块 handler（各域数据各回各家，禁止跨模块抄 repo）
  member/handler.go      ExportMembers   ← member repo + card.Api + transaction.Api（module→api.go 合法）
  transaction/handler.go ExportPayments
  service/handler.go     ExportServiceRecords
  card / appointment / transaction（P2 同理）
  ▼
shared/export（新公共工具包，≈150 行）
  • Workbook 组装：excelize 默认 API 内存构建（单店量级 ≤ 数万行远够；
    留 100_000 行硬顶 → 超限 400 EXPORT_TOO_LARGE）
  • 写响应：Content-Type (xlsx / text/csv;charset=utf-8)
    Content-Disposition: attachment; filename="…xlsx"; filename*=UTF-8''<URL编码中文名>
  • CSV 分支：先写 0xEF 0xBB 0xBF BOM，再 encoding/csv 写行
  • 安全：所有文本单元格过公式注入防护（^=+-@ 前缀补 `'`）
  • 金额：分→元 two-decimal；空手机号/生日输出空串
  ▼
审计：复用 ops_operation_log（POST 天然被 OperationLog 捕获，
      detail 含 domain + 筛选 + 行数）——无需业务模块 import ops（守 D7）
```

前端（apps/admin）：

- `core/api` 新增 `downloadBlob(path, body)`：fetch → Authorization 头 → `blob()` → `URL.createObjectURL` → 触发保存（端点要带 token，不能用裸 `<a href>`）；
- **导出按钮放在既有页面随筛选走**：MembersPage 工具栏「导出 Excel」（P1）、RecordsPage 两个 tab 各一枚；不新做"导出中心"页（不做原则）；
- 导出中按钮 loading + 完成后 toast「已导出 N 行」；失败 toast 错误码。

## 5. 明确不做（冻结口径）

- ❌ 自定义字段勾选器 — 列集合固定精选，稳定可文档化；
- ❌ 导入 / 回写（Shopify 式 re-import）— 永不；
- ❌ 异步导出任务、文件下载中心、OSS 上传 — 单店量级同步流式即可；
- ❌ 定时邮件/微信推送报表 — 老板自己点；
- ❌ Excel 复杂样式（配色/图表/多 sheet 汇总页）— 仅表头加粗 + 合理列宽；
- ❌ 顾客端任何导出入口 — 数据只从后台出，且每次导出留审计。

## 6. 风险与对策

| 风险 | 对策 |
|------|------|
| 导出文件含 PII（手机号）外泄 | 仅 ADMIN 角色可用；POST 全量审计（谁、何时、哪个域、多少行）；文件名含日期便于追溯 |
| 慢查询拖垮库 | 全部只读 SELECT；行数硬顶 10 万；会员总表聚合走 payment(member_id,status) 既有索引，量级异常时 `?date_from` 限窗 |
| excelize 内存 | 当前量级（千行）无虞；行数硬顶兜底；超顶提示分日期段导出 |
| 中文文件名各浏览器兼容 | 同时给 `filename`（ASCII 兜底）与 `filename*=UTF-8''`（RFC 5987） |
| 公式注入（备注/姓名以 = - 开头） | shared/export 统一前置 `'` |

## 7. 工作量预估（第十批 = P1）

| 项 | 估量 |
|----|------|
| shared/export 工具包 + 单测（BOM/注入/金额/文件名） | ~150 行 + 测试 |
| member 会员总表（聚合查询 + handler + api 注入 card/transaction） | ~120 行 |
| transaction 收款流水、service 服务记录（各查询 + handler） | 各 ~60 行 |
| admin downloadBlob + 三处导出按钮 | ~80 行 |
| e2e：登录→导出→断言文件魔数 PK / BOM 与行数 | ~60 行 |

## 8. 验收口径

1. MembersPage 按当前筛选导出 → Excel 双击打开无乱码，手机号列文本格式不丢 0；
2. 收款流水含 VOIDED 行且状态列可筛；金额与 Dashboard 当期 VALID 合计一致；
3. 服务记录中已撤销行状态=已撤销（与 D28 联动快照语义一致）；
4. 每次 POST 导出在操作日志可见 domain+行数；
5. CSV 格式用 Excel（Windows）打开中文不乱码；`=cmd` 类内容不被执行。
