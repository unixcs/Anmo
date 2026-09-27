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

---

## Session: Phase 10 — Content + Ops (2026-09-27)

# PHASE RESULT

## Completed

- content：GET /api/home（blocks+ACTIVE banner/公告）、GET /api/settings（公开项+config 兜底）；后台 page config 校验保存（D16）、banner/公告 CRUD、settings 读写
- ops：OperationLog 中间件（admin 非 GET 全量审计，D7 经 main 注入不产生 import 环）、GET /admin/logs、三项洞察（走 card/member api，§86 边界精确 <=2/60d/7d）、POST /admin/ops/daily（EXPIRED sweep + 快照落库）
- card/member api 扩展：LowBalanceCards/ExpiringCards/SweepExpired/DormantMembers

## Tests

- go build/vet/test 全过（12 包 ok）；ops 测试 3 项（洞察边界/沉睡边界/每日 sweep+快照）

## Database

- 无新 migration

## Next

Phase 11 E2E

---

## Session: Phase 11 — E2E (2026-09-27)

# PHASE RESULT

## Completed

- internal/app 装配抽取（main 与 E2E 共用一套构建路径）
- E2E（真实 HTTP 栈 + 真实 MySQL，12 包全部挂载）：老板登录→建服务/卡模板/规则→顾客登录→发卡(10)→预约→确认→开始→完成→核销→余额 9→流水 ISSUE+REDEEM→撤销恢复 10→重核销→现金收款(不改状态)→第二笔 VALID 拒→工作台→日志→洞察→首页/设置→改期/取消/跨顾客 403/未登录 401
- API 级并发抢时段（6 顾客并发仅 1 成功，Case 2）

## Tests

- E2E 2 项全过；go build/vet/test 全过（13 包 ok）

## Database

- 无新 migration

## Next

对抗式审查（独立任务）

---

## Session: 对抗式审查 (2026-09-27)

# PHASE RESULT

## Completed

- 独立子代理构造 25 个对抗用例（DB 级 18 + HTTP 级 7）真实运行：BLOCKER 1 + WARNING 8，全部复现
- 正面确认：卡余额并发不变量、日历锁风暴、认证/跨顾客隔离/JWT 伪造拦截、SQL 参数化
- 修复 B1（payment valid_lock+UNIQUE、先锁预约）与 W1-W8（幂等回放/跨会员校验/撤销语义/锁序/状态机收紧），migration 010
- REVEAL 用例翻转为 GUARD 断言后 25 用例全过；E2E 重跑全过
- 结果落盘 .trellis/tasks/对抗审查结果.md（含修复记录与复验证据）；冻结口径新增 D18

## Tests

- go build/vet/test -count=1：13 包全部 ok

## Database

- migration 010_payment_idem.sql（payment.valid_lock 生成列 + idempotency_key）

## Next

最终验收（§136 + V1 FINAL STATUS）

---

## Session: V1 FINAL STATUS (2026-09-27)

# V1 FINAL STATUS

## 交付总览

- Phase 0-11 全部走完 Trellis Plan→Execute→Finish 生命周期并归档；Phase 0 Review PASS；编码前子代理审查 B1 修复后复审 PASS；Phase 11 后独立子代理对抗审查（25 用例实跑）发现 1 BLOCKER + 8 WARNING，全部修复复验
- 后端：Go Modular Monolith（server/，13 个测试包）；前端：apps/customer（Vue3+Vite+TS）
- migration 001-010（24 业务表 + sys_sequence/schema_migrations 技术表）

## 验证命令（最终态）

- go build ./...：PASS
- go vet ./...：PASS
- go test ./... -count=1：**13 包全部 ok**（单测 60+ 用例 + DB 集成 + 对抗 25 用例 + E2E 全链路）
- 干净库启动冒烟：migrate 全成功、/healthz 200、OWNER 登录成功
- npm run build（vue-tsc + vite）：PASS

## Plan §136 验收清单（逐项核对）

### 顾客
- [x] 手机登录（短信码 dev=123456，限频/过期/消费）
- [x] 查看服务（/api/services 仅 ACTIVE）
- [x] 查看会员卡（/api/me/cards，token 推导）
- [x] 创建预约（冲突检查+日历锁+营业时间/槽位/提前量校验）
- [x] 取消预约（2h 限制，后台无限制）
- [x] 改期（仅 PENDING/CONFIRMED，冲突排除自身）
- [x] 查看历史预约（status=COMPLETED 列表）

### 老板
- [x] 今日工作台（/admin/workbench：6 态汇总+卡片+收款状态）
- [x] 管会员（列表/新建/详情/编辑/标签/备注）
- [x] 管服务（分类+项目 CRUD/上下架/排序）
- [x] 发卡（member_card + ISSUE 流水单事务）
- [x] 续卡（=再发新卡，D10）
- [x] 调整次数（ADJUSTMENT ±N，USED_UP 复活）
- [x] 看预约（列表筛选+今日视图）
- [x] 确认预约 / 开始服务 / 完成服务（状态机条件 UPDATE 守卫；重复完成 409，§99 Case 10）
- [x] 核销（§53 全链路单事务 + 幂等键）
- [x] 收款（现金/微信/其他，一预约一笔 VALID）
- [x] 撤销核销（恢复次数+payment VOIDED，支持作废/过期卡上的账目修正）
- [x] 看交易流水（payments/redemptions/卡流水）
- [x] 看操作日志（middleware 全量审计 + /admin/logs）

### 核心正确性（均有自动化测试证据）
- [x] 同一时间不能重复预约（并发 8/6/24/30 协程风暴均恰 1 赢家，Case 2）
- [x] 卡余额不能小于 0（CHECK + FOR UPDATE + 16 协程混战账实一致）
- [x] 卡扣次必须有流水（REDEEM 同事务，账实链校验）
- [x] 重复核销不能重复扣次（Case 6：同键回放 + active_lock）
- [x] 撤销核销能恢复次数（Case 7）
- [x] 重复撤销失败（Case 8：6 协程并发恰 1 成功）
- [x] 过期卡不能使用（Case 4：惰性校验为准 + sweep 同步状态）
- [x] 不适用服务的卡不能使用（Case 5：card_service_rule 模板级）
- [x] 已取消预约不能完成（Case 9/10：状态守卫 + 时段释放）
- [x] 已完成预约不能重复完成（Case 10：409）
- [x] 客户不能读取其他客户数据（跨顾客 403/列表隔离，H3）
- [x] 所有重要后台操作有日志（middleware 审计 + 断言非空）

## 范围合规

- §131 禁止实体零引入（无 staff/tenant/积分/优惠券/库存/线上支付/派单/地图等）
- 后台管理 Web UI 不在 Plan Phase 清单内，按"歧义不做"以 admin API 交付（Phase 8 journal 记录）
- Phase 12 上线项（HTTPS/备份/监控）需真实服务器，本环境不执行，生产部署建议见 config.example.yaml 注释

## 结论

**V1 达成 Plan §140 定义的全部核心目标，验收通过。**

## 2026-09-27 POST-V1 FIX: 根路径 404
- 现象：打开 http://localhost:8080/ 显示 "404 page not found"。
- 排查：/healthz 200、进程为最终验收二进制 → 服务本身健康；根因是 router.go 从未注册 `/` 路由（仅 /healthz、/admin/、/api/），Go ServeMux 对未注册路径返回默认 404。
- 修复：router 注册 `GET /{$}`（精确匹配根路径，未知路径仍保持 404 语义），返回 JSON 服务索引（service/version/endpoints）。补 router 测试（根路径 200 + 未知路径 404）。
- 验证：go build / go vet / go test ./... -count=1 全绿（13 包，含 adversarial）；重启后 curl 实测 / 200、/healthz 200、/nope 404、/admin 无 token 401。commit bf635bb。
- 备注：顾客端 H5 是独立应用（Vite dev server 端口 5173，代理到本后端）；后端根路径不做 H5 静态托管（Plan 未定义，歧义选不做）。

## 2026-09-27 POST-V1 ADD: 商家端 Web 管理界面（apps/admin）
- 背景：V1 交付时商家能力全部为 /admin/ REST API（46 端点），无 Web 界面；用户反馈需要商家界面（控制用户端显示内容 + 管理预约等），按需补齐。
- 新增 apps/admin（Vue3+Vite+TS+Element Plus，:5174，/api+/admin 代理到 :8080）：登录、今日工作台（概览+今日预约状态机操作+结算工作台）、预约管理（筛选/确认/开始/完成/取消/未到店/改期+核销收款弹窗）、会员管理（搜索/新建/详情抽屉：资料/标签/会员卡/开卡/调整/作废/流水）、会员卡模板（新建/上下架/可核销服务规则）、服务管理（分类+项目 CRUD、上下架=控制用户端目录）、内容管理（轮播图/公告/首页布局块/系统设置=控制用户端首页）、收款与核销记录（撤销）、运营洞察+日志+日常运维。幂等键 crypto.randomUUID；金额分↔元换算仅展示层。
- 后端配套小改：card 模板列表回显 service_ids（D4 规则读取回显）；重名模板创建由 500 修正为 409 CARD_TEMPLATE_NAME_EXISTS（uk_card_template_name 1062 映射，UI 实测发现）。
- 顾客端配套：HomePage 接入 GET /api/home（按商家配置的内容块顺序渲染 banner/公告/服务列表/活动/富文本）+ GET /api/settings（shop_phone、open/close_time）。
- 验证：vue-tsc 两端全过、admin vite build 过；go build/vet/test ./... 13 包全绿（-count=1）；经 5174 代理的 14 步 UI 调用序列 E2E 全过（登录→建分类/服务→上下架→模板+规则→重名409→会员+开卡→顾客SMS预约→确认/开始→核销(自动完成)→VALID CARD 收款→重复完成409→撤销(次数退回+收款VOIDED)→内容四件套→洞察/运维/日志）。
- 备注：本环境无浏览器后端（__no_browser_backend__），UI 视觉层以 API 序列 E2E + vue-tsc/build 代替浏览器验收；生产部署时改 config 的 auth.admin_password_seed。
- commit: bf635bb(router根路径索引) / cfc6349(card service_ids+409) / e24427d(admin 界面+顾客端首页内容)

## 2026-09-27 POST-V1 ADD: 扫码核销 + 小白教程
- 用户确认核销交互缺口：原流程为商家后台手动定位预约/按手机号搜会员再核销，无扫码。按用户口径补齐：
- 顾客端：新增 /me/qrcode 核销码页（qrcode 库生成，内容 ANMO-MEMBER:<member_id>，固定前缀+ULID）；"我的"页加入口。核销仍由商家确认后生效（资金类操作不开放给顾客端自扫）。
- 商家端：新增 ScanRedeemDialog（jsqr 摄像头逐帧扫码 + 手动手机号输入兜底 + 多会员/多预约消歧 + CONFIRMED 可一键"开始服务" + 核销结果显示新余额）；入口=全局头部按钮+今日工作台按钮。复用既有 POST /admin/appointments/{id}/redeem 幂等链路，后端零改动。
- 验证：qrcode↔jsqr node 双库互验闭环（生成/解码内容一致、前缀与 ID 解析正确，带静区与 8x 放大）；两端 vue-tsc + vite build 全过；后端未改动无需重跑 go test。
- 已知约束：浏览器 getUserMedia 需 localhost/HTTPS，局域网 http 访问时用手动输入兜底（TUTORIAL.md 已注明）。
- 文档：新增 TUTORIAL.md（系统组成/开店准备/完整业务流七步/每日巡检/报错人话对照表/上线须知/收银台速查页）。
- commit: 9945b36

## 2026-09-27 POST-V1 FIX: 商家端手机适配 + HTTPS（扫码摄像头）
- 用户反馈：手机 http://IP:5174 打开商家端 ① UI 未适配 ② 摄像头被禁。
- 根因（第一性）：扫码核销的核心设备就是店员的手机，而浏览器 getUserMedia 仅在安全上下文（HTTPS/localhost）开放——手机经局域网 http 访问必然被禁；UI 则按桌面假设布局。
- 手机适配：≤768px 断点（useMedia）。布局侧边栏→抽屉+汉堡，头部紧凑化（隐藏用户名、保留醒目"扫码核销"）；今日工作台/预约管理/会员管理手机端改卡片流（不横滑，操作按钮直出），桌面保持表格；两处预约操作按钮抽成 AptActionButtons 复用；全部弹窗 width=min(Xpx,94vw)；全局手机 CSS（表格紧凑、message-box 92vw、分页换行）。会员详情抽屉手机端全屏。
- HTTPS：scripts/gen-dev-cert.sh 生成自签证书（SAN 含本机全部网卡 IP，10 年期，30 天内过期自动重签）；vite.config 检测 scripts/certs 存在即以 https 启动；start-anmo.sh 集成；私钥不入库（certs/.gitignore）。
- 验证：vue-tsc（修掉 1 处未用导入）+ vite build 过；https://localhost:5174 200，经 https 代理登录/接口正常，证书 SAN 覆盖全部 IP；桌面布局不变。手机端首访需点一次"继续前往"信任证书。
- commit: 3526ad7

## 2026-09-27 POST-V1 FIX: 商家端双端口（HTTP + HTTPS 并存）
- 用户反馈：http://localhost:5174 打不开（ZCode 内置 Electron 浏览器报 ERR_EMPTY_RESPONSE）。根因：上一轮把 5174 整体切到 HTTPS，HTTP 请求打到 TLS 端口被断开；且内置浏览器无法点过自签证书告警。
- 修正（第一性：桌面日常入口不能断，手机摄像头必须 TLS）：vite 按 --mode 拆分——默认 HTTP :5174（电脑/内置浏览器），dev:https HTTPS :5175（手机扫码专用）；gen-dev-cert.sh 不变；start-anmo.sh 启动并巡检两个实例；扫码弹窗错误提示、TUTORIAL.md 端口表/手机节/报错表同步更新。
- 验证：5174 HTTP 200、5175 HTTPS 200，两端口登录+today 接口经代理均通；vue-tsc 过。手机扫码路径 = https://<电脑IP>:5175（首访信任证书→允许摄像头）。
- commit: 3578afe

## 2026-09-27 POST-V1 FIX: 扫码核销不可用 → 新增「拍照识别」路径（HTTP 全场景可用）
- 用户反馈：Windows 电脑与安卓手机均以 HTTP 打开商家端，点"扫码核销"仍提示摄像头不可用。
- 根因（第一性）：getUserMedia 仅安全上下文（HTTPS/localhost）可用，是浏览器硬策略而非配置问题——手机走局域网 HTTP 永远调不了摄像头；原设计把"实时扫码"当唯一扫码路径，等于把核心操作绑死在 HTTPS 上。
- 修复：ScanRedeemDialog 新增「📷 拍照识别二维码」——`<input type=file accept=image/* capture=environment>`（HTTP 页面也能调起手机原生相机/电脑选图）→ createImageBitmap → canvas 缩放(≤1600px) → jsQR(attemptBoth) 解码，与摄像头路径共用 handleCode；摄像头按钮仅在 `navigator.mediaDevices.getUserMedia` 存在（安全上下文）时出现，提示文案改为人话（HTTP 直接教用户点拍照按钮）。顾客核销码生成参数不变（ANMO-MEMBER:<ulid>）。
- 验证：① node 侧 jsQR 解码真实生成的核销码 PNG，内容/前缀/ID 全对；② 弹窗 resolveMember 同款四接口序列（getMember/listMemberCards/listCardTemplates/admin today）对真实会员实测——会员命中、ACTIVE 卡 10/10、今日 CONFIRMED 预约 APT202609270005 自动定位；③ vue-tsc + vite build 过。核销动作链路由既有 14 步 E2E 覆盖，后端零改动。测试数据保留：会员 13690503034（10 次卡 + 今日 20:30 CONFIRMED 预约），留给用户 UI 实测"拍照识别→开始服务→核销"。
- 测试踩坑备忘：顾客预约受 config.business 约束（提前 2h、30 分钟槽对齐、close 21:00 读 config.yaml 而非设置表；设置表 close_time 仅顾客端展示用）；SMS 验证码一次性 + 60s/手机号限流。
- 文档：TUTORIAL.md 扫码相关三处改写（拍照识别为主路径、修 5174/5175 端口笔误、报错表更新）。
- 备注：内置浏览器后端仍不可用（__no_browser_backend__），拍照解码链路以 node jsQR + API 序列验证代替浏览器实机验收。

## 2026-09-27 POST-V1: 部署 yun1（公网）+ 迁移 Tencent 开发环境 + GitHub 单一事实源
- 用户新指令：① 修复扫码核销（已完成，见上条）② 部署 yun1 外网可访问 ③ 项目迁至 Tencent /mnt/Projects/Anmo 作为第二开发入口（Codex/OpenCode + Trellis 接力），GitHub unixcs/Anmo 作为中枢。
- **GitHub**：仓库初始化收口（根 .gitignore、master→main），推送 unixcs/Anmo@main（公开仓）。352 文件含 .trellis 全量（journal/spec/task 随仓走，天然跨机交接，无需额外快照）。
- **yun1 部署**（2C/1.6G，红线：禁本机 build、禁 Node）：
  - 架构：anmo-mysql(8.4, buffer pool 64M/perf-schema off/mem_limit 480m) + anmo-server(GOOS=linux CGO=0 12M 二进制，alpine+tzdata 镜像 20M，WSL build→save|gzip|scp|load) + anmo-nginx（静态 + /api /admin 反代；商家 SPA 挂 /admin-ui/，顾客端挂 /）。
  - 入口：http://121.41.206.32:18090 / https://…:18091（自签证书 SAN=公网 IP——信任一次后手机可全程摄像头实时扫码）；后端 ANMO_* 环境变量注入（JWT/管理员密码随机化，凭据在 /opt/anmo/credentials.txt 600）。
  - 验收：外网 healthz/两端页面/登录/全业务闭环（发卡5次→预约→确认→开始→核销5→4→撤销回5）全过；ops/health-yun.sh yun1 ALL GREEN（AI-chat 未受影响）；Anmo 栈实占 ~299MB，宿主可用 705MB。踩坑：alpine 缺 tzdata → 镜像补装。
  - 运维文档：yun1:/opt/anmo/README.md（更新五步流程）。
- **Tencent 迁移**：clone 到 /mnt/Projects/Anmo（GitHub SSH key 复用现网 unixcs key）；Go 1.27.1 独立装 /usr/local/go1.27（系统 1.23 不动）+ /etc/profile.d/dev-tools.sh（PATH+GOPROXY=goproxy.cn）；MySQL 8.4 经 daocloud 镜像 pull 后 compose up；npm 走 npmmirror；trellis 0.6.17 全局安装（服务器上已有 codex/pi）。验收：go build/vet 过、13 测试包全绿、两端 vue-tsc+build 过、后端 healthz+admin 登录通。
- **HANDOFF.md**：新增仓库根交接文档（机器拓扑/多端工作流约定/各机环境备忘/必读顺序/已知约束），任何 Agent（Codex/OpenCode/ZCode）接手从它开始。
- 工作流约定：pull --rebase 起手 → 小步提交 push → .trellis journal 即交接日志；两机严禁未推送并行改同一区域。
- commit: a34e3c2(扫码拍照识别) / 41782fb(.gitignore) / d2ca257(router BASE_URL) / 本次(部署+迁移+HANDOFF)

# PHASE RESULT — V1.x 预约体系升级 + 散客核销 + 核销码门槛 + 商家端控件打磨
- 任务：`.trellis/tasks/2026-09-27-v1x-booking-redeem-ux/PRD.md`（含子代理对抗审查 10 条问题修订，2×P0 全落实）。
- 决策增补：AGENTS.md D19（散客核销 NULL 预约 + 幂等键定位收款）/ D20（settings 驱动预约规则 + 模糊预约 + 名额池）/ D21（核销码门槛=持 ACTIVE 卡）/ D22（闭店日历知情闭店）。
- migration 011：payment/redemption.appointment_id 可空、appointment.slot_type、appointment_closure 表。
- 后端：content.BookingRules（settings>cfg>默认，20:00 新默认）+ PublicSettings 联动；appointment 域重构（planWindow/checkClosures/checkCapacity/BookingOptions/Closures CRUD）；transaction.RedeemWalkIn + Reverse/paymentForRedemption 改按 idempotency_key 定位（P0 修复）；POST /admin/cards/{id}/redeem；GET /api|admin/booking-options；appointment JSON 增 slot_type/day_part。
- 顾客端：BookingPage 重做（日期→上下午必选→可选具体时间、满槽禁用、模糊预约提示"具体时间由店主安排"）；MyAppointments 修复 bug#5（http 已解包，myAppointments 直接返回数组——原 `page.data ?? []` 恒空）+ 模糊预约显示"X月X日 上午" + 改期底部抽屉（日期/上下午/时间槽）；QRCodePage 核销码门槛（无 ACTIVE 卡显示引导不出码）+ 实时秒级时钟 + 手机号。
- 商家端：RescheduleDialog（日历控件+上下午+时间槽，替换手输）；AdjustCountDialog（±步进器+快捷 chips，替换手输正负数）；ClosureDialog（闭店设置/恢复营业）；ScanRedeemDialog 散客模式（无预约自动开启，选卡→选服务→核销）；Appointments/Dashboard 模糊徽标"上午到店/下午到店"；ContentPage 营业与预约配置卡片；RecordsPage 散客标记。
- 验证：go build/vet + 全部测试包 -count=1 全绿（新增 TestHalfDayWindowAndPool/TestSpecificCoexistsWithFuzzy/TestFuzzyToSpecificReschedule/TestClosureBlocksAndReportsConflicts/TestBookingOptionsShape/TestWalkInRedeemAndReverse/TestWalkInRuleEnforcement；对抗包边界用例重校准至 20:00 打烊）；两端 vue-tsc + vite build 过；联调 E2E（/tmp/v1x-e2e.sh）8 步全过（booking-options 形状/模糊预约/确认+模糊改具体/闭店拒约/散客核销 5→4+收款 12800/撤销回 5+收款作废/记录 appointment null）。
- 修的坑：BookingOptions SUM(NULL) 扫描失败 → COALESCE；alpine 时区/老 E2E 的 APPOINTMENT_CONFLICT → APT_SLOT_FULL 语义升级（容量模型）。
