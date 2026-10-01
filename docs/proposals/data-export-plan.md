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

> 会员总表的聚合口径：`累计消费 = SUM(payment.amount_cents WHERE status='VALID')`；
> `有效卡数/剩余总次数` 只算 `status='ACTIVE'` 的卡。余额以流水/卡表为准导出快照值，与"remaining_count 是缓存"不变量一致（导出是只读视图，不改真相）。

> 三个域均以请求体 `format` 切换格式：`"xlsx"`（默认）| `"txt"`（Markdown 内容，文件扩展名 .txt）。P2 各域同规则。

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

**手机号打码（仅 TXT）**：TXT 默认输出 `138****1234`——TXT 的两大用途（手机看、喂 AI）都不需要拨号，而喂云端 AI 属于把 PII 送出系统，默认最小化；真实手机号在 XLSX 里全量保留（对账/联系场景走 Excel）。不加参数、不加 UI，按格式内置。

**其余口径同 XLSX**：金额分→元两位小数、时间原样、`text/plain; charset=utf-8` + `Content-Disposition: attachment`（中文文件名 RFC 5987 双写法）。

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
shared/export（新公共工具包，≈300 行）
  • 格式分发：format=xlsx → excelize 默认 API 内存构建（单店量级 ≤ 数万行远够；
    留 100_000 行硬顶 → 超限 400 EXPORT_TOO_LARGE）
  • format=txt → Markdown 渲染器：表格/卡片双排版（按域声明 layout）
    + 手机号打码 + UTF-8 直出（无 BOM 需求，.txt 不经 Excel）
  • 写响应：Content-Type (xlsx / text/plain;charset=utf-8)
    Content-Disposition: attachment; filename="…xlsx|txt"; filename*=UTF-8''<URL编码中文名>
  • 安全：xlsx 文本单元格过公式注入防护（^=+-@ 前缀补 `'`）；
    txt 粘贴场景无公式执行面，无需处理
  • 金额：分→元 two-decimal；空手机号/生日输出空串
  ▼
审计：复用 ops_operation_log（POST 天然被 OperationLog 捕获，
      detail 含 domain + 筛选 + 行数）——无需业务模块 import ops（守 D7）
```

前端（apps/admin）：

- `core/api` 新增 `downloadBlob(path, body)`：fetch → Authorization 头 → `blob()` → `URL.createObjectURL` → 触发保存（端点要带 token，不能用裸 `<a href>`）；
- **导出按钮放在既有页面随筛选走**：MembersPage 工具栏「导出 Excel」「导出 TXT」两枚（P1）、RecordsPage 两个 tab 各两枚；不新做"导出中心"页（不做原则）；
- 导出中按钮 loading + 完成后 toast「已导出 N 行」；失败 toast 错误码。

## 5. 明确不做（冻结口径）

- ❌ 自定义字段勾选器 — 列集合固定精选，稳定可文档化；
- ❌ 导入 / 回写（Shopify 式 re-import）— 永不；
- ❌ 异步导出任务、文件下载中心、OSS 上传 — 单店量级同步流式即可；
- ❌ 定时邮件/微信推送报表 — 老板自己点；
- ❌ Excel 复杂样式（配色/图表/多 sheet 汇总页）— 仅表头加粗 + 合理列宽；
- ❌ 顾客端任何导出入口 — 数据只从后台出，且每次导出留审计；
- ❌ **.md 与 .txt 双格式并行**（v2 裁决）— 同字节双份冗余；只出 .txt（Markdown 内容）；
- ❌ **CSV 端点暂缓**（v2 裁决）— 与 TXT 受众重叠（纯文本都可读），仅"喂其他程序/再导入"场景需要，届时 ~15 行补上；
- ❌ JSON 导出 — 手机不可读，喂 AI 不如 Markdown。

## 6. 风险与对策

| 风险 | 对策 |
|------|------|
| 导出文件含 PII（手机号）外泄 | 仅 ADMIN 角色可用；POST 全量审计（谁、何时、哪个域、多少行）；文件名含日期便于追溯 |
| 慢查询拖垮库 | 全部只读 SELECT；行数硬顶 10 万；会员总表聚合走 payment(member_id,status) 既有索引，量级异常时 `?date_from` 限窗 |
| excelize 内存 | 当前量级（千行）无虞；行数硬顶兜底；超顶提示分日期段导出 |
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
| admin downloadBlob + 三处导出按钮（每处 Excel/TXT 两枚） | ~90 行 |
| e2e：登录→导出→断言 xlsx 魔数 PK / txt Markdown 结构与打码与行数 | ~70 行 |

## 8. 验收口径

1. MembersPage 按当前筛选导出 → Excel 双击打开无乱码，手机号列文本格式不丢 0；
2. 收款流水含 VOIDED 行且状态列可筛；金额与 Dashboard 当期 VALID 合计一致；
3. 服务记录中已撤销行状态=已撤销（与 D28 联动快照语义一致）；
4. 每次 POST 导出在操作日志可见 domain+行数；
5. ~~CSV 格式用 Excel（Windows）打开中文不乱码~~（CSV 暂缓，随端点顺延）；
6. **TXT（v2）**：微信发送到手机后可直接打开阅读；会员总表为卡片块、收款流水为表格；
7. **TXT（v2）**：整文件粘给 AI 后结构可解析（标题/表格/字段名完整）；所有手机号已打码 `138****`。
