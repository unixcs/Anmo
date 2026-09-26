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
| Customer | H5（Vue3） | 手机号+短信验证码登录，预约、查卡、查历史 |
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

- MySQL 8，utf8mb4，时区 Asia/Shanghai，价格为整数分（禁 float）
- ID 用 CHAR(26) ULID；业务编号单独生成（member_no / appointment_no，如 APT202609280001）
- 所有 schema 变化必须走 `migrations/NNN_*.sql`，禁止改历史 migration
- 核心历史数据禁止物理删除，用状态字段
- 余额不是唯一真相：`member_card.remaining_count` 是缓存，`card_transaction` 是历史
- appointment / payment / redemption / 撤销记录 全部分离
- 服务名称/价格/时长必须 snapshot（appointment_service）

## 状态机（冻结）

```
appointment: PENDING_CONFIRM → CONFIRMED → IN_SERVICE → COMPLETED
             PENDING_CONFIRM|CONFIRMED → CANCELLED
             CONFIRMED → NO_SHOW
member_card: ACTIVE / USED_UP / EXPIRED / CANCELLED
```

## 核心事务（必须单事务完成）

1. 发卡：member_card + card_transaction(ISSUE)
2. 核销：锁卡 → 校验状态/有效期/卡服务规则/余额 → 扣次 → card_transaction(REDEEM) → redemption → payment(CARD) → appointment=COMPLETED → status log，任一步失败 ROLLBACK
3. 撤销：锁卡 → 确认未撤销 → 恢复次数 → card_transaction(REVERSAL) → redemption_reversal

禁止异步事件扣卡。EventBus 仅用于通知/统计/洞察。

## 业务不变量

- `remaining_count >= 0` 恒成立
- 一个预约最多一次有效核销（appointment_id + SUCCESS 唯一），幂等键 `redeem:{appointment_id}`
- 一个时间段只能有一个有效预约（PENDING_CONFIRM/CONFIRMED/IN_SERVICE 参与冲突判定）
- 冲突判定：`existing.start < new.end AND existing.end > new.start`
- 顾客只能通过 token 确定自己的 member_id，禁止信任前端传参

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
- MySQL 8.4：docker compose（本目录），端口 33306，库 `anmo`，用户 `anmo` / `anmo-dev-2026`，root / `anmo-root-2026`
- 后端默认监听 `:8080`；H5 dev 由 Vite 提供
- 验证命令：`go build ./... && go vet ./... && go test ./...`
