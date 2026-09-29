# Anmo — 个人到店按摩店服务管理系统 V1

> 本文件是全项目最高约束。所有开发以 `plan.md` 为准（PLAN.md），本文件是其工程化摘要。
> 开发流程由 Trellis 接管（见 `.trellis/workflow.md`）。

<!-- TRELLIS:START -->
# Trellis Instructions

These instructions are for AI assistants working in this project.

This project is managed by Trellis. The working knowledge you need lives under `.trellis/`:

- `.trellis/workflow.md` — development phases, when to create tasks, skill routing
- `.trellis/spec/` — package- and layer-scoped coding guidelines (read before writing code in a given layer)
- `.trellis/workspace/` — per-developer journals and session traces
- `.trellis/tasks/` — active and archived tasks (PRDs, research, jsonl context)

If a Trellis command is available on your platform (e.g. `/trellis:finish-work`, `/trellis:continue`), prefer it over manual steps. Not every platform exposes every command.

If you're using Codex or another agent-capable tool, additional project-scoped helpers may live in:
- `.agents/skills/` — reusable Trellis skills
- `.codex/agents/` — optional custom subagents

Managed by Trellis. Edits outside this block are preserved; edits inside may be overwritten by a future `trellis update`.
<!-- TRELLIS:END -->

## 项目定位

**个人到店按摩店服务管理系统**。不是上门按摩、不是 SaaS、不是连锁、不是多租户。

经营模型：`店主 ＝ 按摩师 ＝ 唯一服务人员`。未来家人可作为 OPERATOR 协助后台。

## 核心业务闭环（唯一主线）

```
客户 → 登录 → 查看服务 → 预约 → 老板确认 → 到店 → 开始服务
→ 服务完成 → 结算（会员卡核销 / 微信转账 / 现金）→ 记录
```

预约 ≠ 办卡。任何登录客户都可预约；会员卡只是结算方式之一。

## 角色与端

| 角色 | 端 | 说明 |
|------|----|----|
| Customer | H5（Vue3） | 手机号+密码注册/登录（V2.2，短信已下线），预约、查卡、查历史 |
| OWNER（Admin） | 后台 | 全部权限 |
| OPERATOR | 后台 | 预留，RBAC 支持 |

## 模块地图（Go Modular Monolith，单库）

```
server/
  cmd/anmo/main.go
  internal/
    config/  logger/  database/  router/  middleware/  shared/
    modules/
      identity/    登录、RBAC、Customer Token
      member/      会员、标签
      service/     服务分类、服务项目
      card/        卡模板、member_card、card_service_rule、card_transaction
      appointment/ 预约、时间冲突、状态机
      transaction/ payment、redemption、redemption_reversal（结算）
      content/     首页内容、系统设置
      ops/         操作日志、洞察、定时任务
  migrations/      00N_*.sql（只增不改）
apps/customer/     H5（Vue3 + Vite + TS）
```

模块边界：允许 `module → 另一 module/api.go`；**禁止**跨模块直接引用 repo.go / 内部 model。

## 数据库规则

- **SQLite**（2026-09-28 起，MySQL 已退役）：单文件库 + WAL，纯 Go 驱动 modernc.org/sqlite（无 CGO）
- 时区 Asia/Shanghai：DATETIME 一律存 `YYYY-MM-DD HH:MM:SS` 墙上时间字符串（驱动 `_timezone=Asia/Shanghai` 读写），DATE 存 `YYYY-MM-DD`；价格为整数分（禁 float）
- 并发模型：所有事务 `_txlock=immediate`（BEGIN IMMEDIATE）+ busy_timeout=10s；单写者天然串行化写事务（原 MySQL GET_LOCK/FOR UPDATE 已移除）
- ID 用 CHAR(26) ULID；业务编号单独生成（member_no / appointment_no，如 APT202609280001，sys_sequence 原子计数器）
- 所有 schema 变化必须走 `migrations/NNN_*.sql`，禁止改历史 migration
- 核心历史数据禁止物理删除，用状态字段
- 余额不是唯一真相：`member_card.remaining_count` 是缓存，`card_transaction` 是历史
- appointment / payment / redemption / 撤销记录 全部分离
- 服务名称/价格/时长必须 snapshot（appointment_service）
- 备份：`-backup`（VACUUM INTO 快照 + integrity_check）+ `scripts/backup-sqlite.sh` 定时执行、备份到独立挂载卷

## 状态机（冻结）

```
appointment: PENDING_CONFIRM → CONFIRMED → IN_SERVICE → COMPLETED
             PENDING_CONFIRM|CONFIRMED → CANCELLED
             CONFIRMED → NO_SHOW
member_card: ACTIVE / USED_UP / EXPIRED / CANCELLED
```

## 核心事务（必须单事务完成）

1. 发卡：member_card + card_transaction(ISSUE)
2. 核销（预约结算）：锁卡 → 校验状态/有效期/卡服务规则/余额 → 扣次 → card_transaction(REDEEM) → redemption(记实际服务+名称快照) → payment(CARD) → appointment=COMPLETED → status log，任一步失败 ROLLBACK。预约须处于 WAITING/IN_SERVICE（COMPLETED 仅撤销后重核销时放行，MarkCompleted 幂等；CANCELLED/NO_SHOW 拒绝走散客）
3. 撤销：锁卡 → 确认未撤销 → 恢复次数 → card_transaction(REVERSAL) → redemption_reversal → **同一事务将原核销的 payment(CARD) 置 VOIDED（按 idempotency_key 定位，D19）**。payment.status ∈ {VALID, VOIDED}；收款记录只统计 VALID；一个预约最多一笔 VALID payment
4. 散客核销（V1.x，D19）：锁卡 → 校验状态/有效期/卡规则/余额 → 扣次 → card_transaction(REDEEM) → redemption(appointment NULL) → payment(CARD, 金额=服务默认价) → last_visit，无预约、无 COMPLETED 迁移
5. 现金/微信结算（V1.x，2026-09-28 §8）：预约处于 WAITING/IN_SERVICE 时，同事务写 payment(CASH/WECHAT/OTHER) 并将预约置 COMPLETED；CANCELLED/NO_SHOW 拒绝（按独立散客处理）

禁止异步事件扣卡。EventBus 仅用于通知/统计/洞察。

## 业务不变量

- `remaining_count >= 0` 恒成立
- 一个预约最多一次有效核销：redemption.status ∈ {SUCCESS, REVERSED}，撤销只置 REVERSED 不删行；生成列 active_lock（SUCCESS 时=appointment_id，否则 NULL）+ UNIQUE(active_lock) 保证不变量并支持撤销后重新核销
- idempotency_key 为请求级 UUID，UNIQUE 约束防重复提交
- 一个时间段只能有一个有效预约（WAITING/IN_SERVICE 参与冲突判定，D8 口径）
- 冲突判定：`existing.start < new.end AND existing.end > new.start`，排除自身（改期）
- 顾客只能通过 token 确定自己的 member_id，禁止信任前端传参

## 冻结决策（Phase 0 Review + 子代理审查，全部为"最简/不做"口径）

| # | 决策 |
|---|------|
| D1 | payment.status ∈ {VALID, VOIDED}；撤销事务同事务置 VOIDED；收款只统计 VALID；一预约最多一笔 VALID payment；V1 无独立作废收款功能 |
| D2 | RBAC：identity_user.role 枚举（OWNER/OPERATOR）实现角色；identity_role/identity_permission 建静态种子表；V1 OPERATOR 权限与 OWNER 相同 |
| D3 | 数据库共 24 张业务表（§124 清单为权威）+ 技术表（schema_migrations、sys_sequence 原子计数器）；appointment.member_id NOT NULL，无代客下单 |
| D4 | card_service_rule 挂 card_template_id（模板级）；核销经 member_card.card_template_id 解析 |
| D5 | **(2026-09-28 修订，SQLite 迁移)** 并发预约：不再使用 NamedLock/GET_LOCK——所有写事务 `_txlock=immediate`（BEGIN IMMEDIATE）在 BEGIN 时排队获得唯一写锁，闭店/容量/冲突检查与 INSERT 同事务天然原子（busy_timeout=10s，超时返回 SQLITE_BUSY→LOCK_RETRY） |
| D6 | 跨模块单事务：transaction 模块开事务，显式 Tx 执行器传入 card/appointment 的 api.go；模块内禁止自开嵌套事务（Phase 1 落地 shared.TxRunner） |
| D7 | 操作日志由 HTTP middleware 写 ops_operation_log；业务模块不 import ops；ops 定时任务单向依赖业务模块 api.go |
| D8 | **(2026-09-28 修订)** 状态机收紧为 WAITING→IN_SERVICE→COMPLETED（异常 WAITING→CANCELLED/NO_SHOW），migration 012；创建即 WAITING，无确认环节；状态迁移一律 `UPDATE ... WHERE status=期望` 校验影响行数；改期限 WAITING、沿用 2 小时限制、同 appointment 改时间、冲突排除自身；NO_SHOW 仅从 WAITING 迁出 |
| D9 | **(2026-09-28 修订)** 仅结算事务与"完成服务"动作触发 COMPLETED；卡核销与现金/微信收款均允许 WAITING/IN_SERVICE 并同事务完成预约（§8 统一结算）；实际服务可≠预约服务（redemption 记实际服务+快照，预约快照不覆盖）；散客核销仅拦当日存在 WAITING/IN_SERVICE 预约的会员（RDM_WALKIN_BLOCKED） |
| D10 | 续卡 = 同一 member 再发一张新卡（复用 ISSUE 流水）；调整次数 = 现有卡 ADJUSTMENT ±N 流水 |
| D11 | 卡模板 type 仅 COUNT/ACTIVITY 存枚举，不产生独立逻辑，有效期由 validity_type/valid_from/valid_until 表达 |
| D12 | 作废卡：仅置 CANCELLED，不改次数、不写次数流水；核销拒绝 CANCELLED |
| D13 | 核销时 remaining_count 减至 0 同事务置 USED_UP；EXPIRED 惰性校验+每日 sweep（W2） |
| D14 | 索引补唯一约束：member.phone、member_no、appointment_no、redemption.idempotency_key |
| D15 | 营业时间边界：scheduled_end ≤ 营业结束时间，否则拒绝 |
| D16 | content_page_config 的 JSON block 引用 banner/announcement id，不复制正文 |
| D17 | 顾客多时段待确认预约无上限限制，风险知情接受（"不做"原则） |
| D18 | 撤销核销允许对 CANCELLED/EXPIRED 卡恢复次数（账目修正），但卡保持原状态不复活（对抗审查 W4/W5）；payment 表有生成列 valid_lock+UNIQUE 强制一预约一笔 VALID 收款（B1）及 idempotency_key 幂等（W2）。SQLite 版生成列：redemption.active_lock STORED（建表）、payment.valid_lock VIRTUAL（ALTER ADD），UNIQUE 语义已逐项验证 |
| D19 | 散客核销（V1.x）：redemption/payment.appointment_id 可空（NULL=无预约直接核销）；必须指定服务项（卡规则校验 + 金额=服务默认价）；payment 作废/回放定位一律按 idempotency_key（预约维度不变量只约束非空行）；有今日预约的会员不开放散客核销（防绕过 D9） |
| D20 | 预约规则（V1.x）：营业时间/时段间隔(30|60|120)/每时段容量/上下午分界存 settings（business_* 键，缺失回落 cfg→硬编码 09:00/20:00/30/1/12:00）；appointment.slot_type ∈ {SPECIFIC, HALF_DAY}，HALF_DAY 落库窗口=半天边界；逐槽并发 ≤ capacity 仅约束 SPECIFIC，半日池（< 槽数×容量）对两者一体适用；SPECIFIC 保持 ≥2h 提前量，HALF_DAY 仅要求半天未结束；校验一律 NamedLock+事务内 |
| D21 | 核销码门槛（V1.x）：顾客端仅对持有 ACTIVE member_card 的用户出示核销码；码内容协议不变（ANMO-MEMBER:<ulid>）；商家端后端校验兜底 |
| D22 | 闭店日历（V1.x）：appointment_closure 按 (date, AM\|PM) 粒度，全天=两行；创建闭店在 calendar 锁内统计 conflict_count 返回给商家知情；不自动取消/改约；改营业配置不追溯已建预约 |
| D23 | 微信身份（V2）：member.wx_openid VARCHAR(64) NULL + UNIQUE（NULL 可重复）；一个 openid 只绑一个 member，一 member 只一个 openid；不建独立绑定表 |
| D24 | code2session 凭据 `wx.app_id`/`wx.secret`（env `ANMO_WX_APPID`/`ANMO_WX_SECRET`）；缺省时 dev 兜底 `openid = "dev:"+code`（启动日志警示），正式发布前必须配真实凭据 |
| D25 | **(2026-09-29 修订，V2.2 第二批)** bind_ticket 流程废除：微信首登 openid 未绑定时同事务直建号直发 Token（uk_member_wx_openid 唯一兜底）；手机号撞号走 `POST /api/auth/wx/claim`（顾客 Token + 该手机号的 H5 密码）转绑老账号并同事务删除无业务数据的空壳（HasBusinessData=false 才可删）；`/api/auth/wx/bind` 端点不复存在 |
| D26 | 微信官方"服务卡片"能力（类目/资质/后台配置）不做；分享闭环 = 每页 onShareAppMessage + 首页/关于 onShareTimeline + showShareMenu |
| D27 | **(2026-09-29 新增，V2.2 第二批)** 顾客密码体系：bcrypt（DefaultCost，6~64 位）；H5 `POST /api/auth/register`/`login`（手机号+密码）；小程序「我的」设/重置 H5 密码 `PUT /api/me/h5-password`（微信身份即凭证，需已绑手机号 MEMBER_PHONE_REQUIRED）+ 手机号一次性设置（MEMBER_PHONE_SET，撞号 MEMBER_PHONE_TAKEN 触发 claim）；admin `PUT /admin/members/{id}/password` 重置；member.phone 可空（仅纯微信会员）；SMS 端点仅 sms.mode=dev 可用（过渡期），生产必须 off（off 时 410 SMS_DISABLED） |

参考报告：`.trellis/tasks/archive/2026-09/09-27-plan-subagent-review/SUBAGENT-REVIEW.md`、`.../09-27-phase0-review/REVIEW.md`

## V1 边界 / 禁止事项（硬约束）

禁止引入 Plan 未提及的实体：staff、staff_schedule、member_address、service_area、tenant_id、points、coupon、mall、inventory、commission、online_payment、dispatch、map、gps、技师、排班、多门店、多租户、上门服务、微信支付 API、AI。

歧义处理原则：**选"不做"**。

## 开发规则

1. 读取根 AGENTS.md → 判断涉及模块 → 读模块 AGENTS.md → 读模块 api.go → 读相关代码 → 修改 → 测试
2. 禁止一次性读取整个项目
3. 每个 Phase：代码 + 测试 + migration + 文档一起完成
4. 每完成一个 Phase 在 Trellis journal 追加 `# PHASE RESULT`（格式见 plan.md §135）

## 环境

- Go 1.27（`export PATH=$PATH:/usr/local/go/bin`）
- 存储：SQLite 单文件（默认 `data/anmo.db`，env `ANMO_DB_PATH`）；2026-09-28 前的 MySQL 8.4 容器仅作历史数据源保留（端口 33306，数据迁移工具 `server/cmd/mysql2sqlite` 消费）
- 后端默认监听 `:8080`；H5 dev 由 Vite 提供
- 验证命令：`go build ./... && go vet ./... && go test ./...`（SQLite 后测试零外部依赖、0 跳过）
