# Journal - anmo-agent (Part 1)

> AI development session journal
> Started: 2026-09-27

---



## Session 1: Phase 0: Plan Review PASS
<!-- trellis-session: v=2 fp=39c217cfa7d39716 -->

**Date**: 2026-09-27
**Task**: Phase 0: Plan Review PASS
**Branch**: `master`

### Summary

对 plan.md 完成只读审查，输出 REVIEW.md（5 段格式完整，DECISION=PASS）。冻结 5 项执行口径：W1 完成/结算状态耦合、W2 卡过期惰性+sweep、W3 短信通道接口化、W4 REFUND 保留不实现、W5 appointment_no 日序列并发安全。

### Git Commits

| Hash | Message |
|------|---------|
| `fd51383` | chore: trellis init, plan.md, root AGENTS.md, Phase 0 REVIEW (PASS) |

### Status

[OK] **Completed**

---

## Session: Phase 0 — Plan Review (2026-09-27)

# PHASE RESULT

## Completed

- Trellis init（--zcode），父任务 09-27-anmo-v1 建立
- 根 AGENTS.md 编写（项目定位/闭环/模块地图/数据库规则/状态机/事务/不变量/禁止事项）
- MySQL 8.4 容器（anmo-mysql, 33306, db=anmo）
- plan.md 只读审查，`.trellis/tasks/archive/2026-09/09-27-phase0-review/REVIEW.md`，DECISION=PASS

## Tests

- 纯审查任务，无测试；git commit fd51383

## Database

- docker-compose.yml（mysql:8.4, utf8mb4, Asia/Shanghai, 端口 33306）

## Files Changed

- AGENTS.md, docker-compose.yml, plan.md(入库), .trellis/*, .zcode/*

## Remaining

- W1-W5 执行口径已冻结（见 REVIEW.md），进入 Phase 1

## Next

Phase 1 工程骨架

---

## Session: 子代理 Plan 独立审查（编码前门禁）(2026-09-27)

- 独立子代理对 plan.md + AGENTS.md + Phase 0 REVIEW 做对抗式审查 → 首轮 DECISION: BLOCKED
- BLOCKER B1：撤销核销事务未处理 payment(CARD)，撤销后重结导致收款双计
- 修复：AGENTS.md 撤销事务补 payment 置 VOIDED；采纳 W-A~W-K/R1-R7 为冻结决策 D1-D17
- 复审 DECISION: PASS（B1 闭环推演成立，D1-D17 无冲突，无新 BLOCKER）
- 报告：.trellis/tasks/archive/2026-09/09-27-plan-subagent-review/SUBAGENT-REVIEW.md
- 门禁结论：允许进入 Phase 1 编码

---

## Session: Phase 1 — 工程骨架 (2026-09-27)

# PHASE RESULT

## Completed

- migrations 001-008（24 张表全部落库，含 active_lock 生成列、CHECK、FK、唯一约束、种子数据）——数据库冻结
- Go Modular Monolith 骨架：config(YAML+env)/logger(slog)/database(Pool适配+迁移runner)/shared(errors/Tx/ULID/分页/响应)/middleware(requestid/logging/recover/auth骨架)/router
- 8 个业务模块挂载完成，各模块 AGENTS.md（职责/表/API/事务/测试/禁止）就位
- 跨模块事务协议落地：shared.Tx + database.Pool（D6），Migrate 可复用 shared.DB

## Tests

- go build: PASS / go vet: PASS / go test: PASS（shared/router 单测 + database 迁移集成测试：幂等 + 24 表校验）
- migration 实跑 anmo 库 8 个文件成功；/healthz 冒烟 200

## Database

- 001_identity … 008_ops（应用 8 个，schema_migrations 记录，重复执行幂等）

## Files Changed

- server/**（go.mod、cmd/anmo、migrations/8、internal/**、config.example.yaml）、AGENTS.md 补充

## Remaining

- auth 中间件等待 Phase 2 的 TokenVerifier 实装

## Next

Phase 2 Identity

---

## Session: Phase 2 — Identity (2026-09-27)

# PHASE RESULT

## Completed

- 后台登录 POST /admin/auth/login（bcrypt + JWT, role=OWNER/OPERATOR），首次启动种子 OWNER（config 注入）
- RBAC：router 三层 mux（root 公开 / admin 校验 / api 校验顾客 token），middleware.NewAuth 注入 identity.TokenVerifier；顾客 token 访问后台 403、无 token 401
- 顾客登录：POST /api/auth/sms/send（dev 固定码 123456 + 60s 限频 + 日志）→ /api/auth/sms/verify → member 自动创建/绑定（EnsureByPhone，GET_LOCK 日序列 member_no=M+yyyymmdd+4位）
- JWT: HS256, act=ADMIN/CUSTOMER, admin 12h / customer 7d

## Tests

- go build/vet/test 全过；新增 token 往返/篡改/过期、SMS 消费/限频/过期、auth 中间件 401/403 单测
- 冒烟：OWNER 登录取 token、顾客登录建 member、越权 403/401 实测通过

## Database

- 无新 migration（001 已含 identity 表与种子；OWNER 账号运行时种子化，密码不入 SQL）

## Files Changed

- identity/{api,admin,customer,sms,token,handler,module}.go、member/{ensure,clock}.go、router/router.go、middleware/auth.go、main.go

## Remaining

- 会员管理 CRUD 属 Phase 3

## Next

Phase 3 Member

---

## Session: Phase 3 — Member (2026-09-27)

# PHASE RESULT

## Completed

- 后台会员管理：列表(关键词+分页)/新增(member_no 日序列)/详情/编辑(name/gender/birthday/remark 校验)
- 标签：列表/新建(重名 409)/设置(全量替换，INSERT IGNORE 幂等)
- 顾客端：GET/PUT /api/me/profile（member_id 取自 token）
- 修复：DATE 列 parseTime=true 下按 NullTime 扫描并格式化 YYYY-MM-DD

## Tests

- go build/vet/test 全过（6 包 ok）；member 模块 DB 集成测试 4 项（创建/手机号冲突/Ensure 绑定一致/标签替换/资料校验）

## Database

- 无新 migration（002 已建表）

## Files Changed

- member/{repo,tags,handler,module,testmain,repo_test}.go

## Remaining

- 会员详情聚合卡/预约信息待 Phase 5-6 后回补

## Next

Phase 4 Service

---

## Session: Phase 4 — Service (2026-09-27)

# PHASE RESULT

## Completed

- 分类 CRUD + 上下架；服务项目 CRUD + 上下架 + 排序；价格整数分校验（duration>0, price>=0）
- 顾客端 GET /api/services 仅返回 ACTIVE 分类+项目

## Tests

- go build/vet/test 全过（7 包 ok）；service 模块 DB 测试 2 项（创建+可见性切换、参数校验）

## Database

- 无新 migration（003 已建表）

## Files Changed

- service/{repo,handler,module,repo_test,testmain}.go

## Next

Phase 5 Card

---

## Session: Phase 5 — Card (2026-09-27)

# PHASE RESULT

## Completed

- 卡模板 CRUD（type/validity 校验，D11）+ 模板级服务规则 SetServiceRules（D4）
- 发卡 IssueCard：单事务 member_card + ISSUE 流水（§127）；续卡=再发新卡（D10）
- 核销协作 API：LockForRedeem(FOR UPDATE)/ValidateForRedeem(状态+有效期+规则+余额)/ApplyRedeem(扣次+REDEEM 流水+USED_UP 联动)/ApplyReversal(恢复+REVERSAL 流水)
- Adjust（ADJUSTMENT ±N，USED_UP 复活）/Cancel（仅状态，D12）/流水查询/UsableCards
- 后台路由：模板/规则/发卡/调整/作废/流水；会员卡列表

## Tests

- go build/vet/test 全过（8 包 ok）；card 模块 DB 测试 5 项：发卡流水、核销+撤销、四类拒绝（不适用/余额不足/过期/作废）、USED_UP 联动+调整复活、可用卡过滤

## Database

- 无新 migration（004 已建表）

## Files Changed

- card/{template,redeem,handler,operator,module,repo_test}.go

## Remaining

- 并发核销（Case 1）在 Phase 7/E2E 覆盖

## Next

Phase 6 Appointment

---

## Session: Phase 6 — Appointment (2026-09-27)

# PHASE RESULT

## Completed

- 预约创建：snapshot(appointment_service)、营业时间/提前量/时间槽对齐校验（D15）、APPT 编号 APT+日序列
- 冲突判定重构：GET_LOCK 会在提交前释放导致双订窗口 → 改为 database.Pool.NamedLock（专用连接持锁至 COMMIT 后释放，D5 冻结口径按此实现）
- 业务编号竞态修复：MAX+1 弃用，新增技术表 sys_sequence（migration 009）原子计数
- 状态机全量：confirm/start/complete(幂等)/cancel(顾客 2h/后台无限制)/no-show(仅 CONFIRMED)/reschedule(排除自身+改期日志)，全部条件 UPDATE 守卫（D8）
- 后台/顾客端路由 + 今日工作台查询（Today/汇总/卡片，Phase 8 复用）

## Tests

- go build/vet/test 全过（10 包 ok）
- appointment 测试 6 项：重叠/相邻冲突、时间窗校验、状态守卫+取消释放时段（Case 9）、完成幂等+爽约守卫（Case 10）、改期排除自身、**并发抢时段 8 goroutine 仅 1 成功（Case 2）**

## Database

- migration 009_sequence.sql（sys_sequence 技术表；D3 更新为"24 业务表+技术表"）

## Files Changed

- appointment/{lifecycle,queries,handler,module,repo_test}.go、member/ensure.go、shared/{clock,sequence}.go、database/lock.go、migrations/009、AGENTS.md

## Remaining

- 核销事务联动 appointment 在 Phase 7

## Next

Phase 7 Transaction（核心）

---

## Session: Phase 7 — Transaction (2026-09-27)

# PHASE RESULT

## Completed

- SettleByCard 核心事务（§53 完整链路单事务：锁卡→校验→扣次→REDEEM 流水→redemption→payment(CARD)→appointment COMPLETED→status log→last_visit）
- 幂等：请求级 UUID idempotency_key 重放返回原结果；active_lock 生成列唯一保证一预约一有效核销
- SettleByPay（现金/微信/其他）：仅写 VALID payment，不改预约状态（D9）；一预约一 VALID payment 守卫（D1）
- ReverseRedemption：置 REVERSED→恢复次数→REVERSAL 流水→reversal→原 CARD payment 置 VOIDED，预约保持 COMPLETED（§59），撤销后可重结
- 后台路由：redeem/payments/reverse/payments 列表/redemptions 列表

## Tests

- go build/vet/test 全过（11 包 ok）
- transaction 测试 6 项：全链路、幂等重放（Case 6）、异键防重（active_lock）、撤销恢复+重复撤销失败（Case 7/8）+收款不双计（D1）、现金不改状态（D9）、**并发核销余额 1 → 1 成功 1 失败（Case 1）**

## Database

- 无新 migration（006 已建表）

## Files Changed

- transaction/{settle,queries,handler,module,api,settle_test}.go、appointment/queries.go(ServicesOfTx)、main.go

## Next

Phase 8 今日工作台

---

## Session: Phase 8 — 今日工作台 (2026-09-27)

# PHASE RESULT

## Completed

- GET /admin/workbench?date=：6 态汇总 + 卡片（会员姓名/服务 snapshot/价格/VALID 收款信息）
- 快速操作（确认/开始/完成/核销/收款/取消/爽约）端点齐备（Phase 6/7 交付）
- 决策记录：后台管理 Web UI 不在 Plan §108-121 Phase 清单内，按"歧义不做"原则以 admin API 交付能力；§136 老板项以 API 验收

## Tests

- go build/vet/test 全过（11 包 ok）；workbench 聚合测试（汇总计数/姓名/快照/收款状态）

## Database

- 无新 migration

## Next

Phase 9 Customer H5

---

## Session: Phase 9 — Customer H5 (2026-09-27)

# PHASE RESULT

## Completed

- apps/customer（Vue3+Vite+TS+vue-router）：首页/服务/预约(选服务→日期→时间槽→提交)/我的(会员卡/我的预约/历史/资料)/登录
- 目录符合 §101：core(api/models/logic/booking 时间槽生成/store/utils) + platform(auth/storage/notify，唯一 DOM/localStorage 边界) + pages/components
- 路由守卫（未登录跳登录）、冲突文案 §106、内部备注不下发 §107、预约入口带 service 预选
- 后端补 GET /api/me/cards（token 推导 member_id）
- npm run build（vue-tsc）通过；go 全测 10 包 ok

## Tests

- go build/vet/test 全过；vue-tsc + vite build PASS

## Database

- 无新 migration

## Next

Phase 10 Content + Ops
