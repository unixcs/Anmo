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

# PHASE RESULT — V1.x 部署同步 + 验收闭环
- 验收审查（子代理）：VERDICT REVISE（无 P0，1×P1 + 4×P2）→ 全部修复：HistoryPage 同源解包 bug（P1）、RedeemWalkIn 后端拒绝今日有预约会员（D19 后端加固）、BookingRules 半天长度整除守卫、AdjustCountDialog 手输输入框、PRD 补记闭店知情口径。
- 修复后回归：go test（transaction/appointment/adversarial）-count=1 全绿、两端 vue-tsc 过、/tmp/v1x-e2e.sh 8 步全过。
- 提交链：2290918（V1.x 功能）→ e170bc4（验收修复），已推 GitHub main。
- yun1 同步：WSL 构建（admin --base=/admin-ui/、customer、linux/amd64 二进制）→ docker save|scp|load → force-recreate server；migration 011 自动应用（count=1）；公网 https://121.41.206.32:18091 实测 booking-options/closures 新端点正常；ops/health-yun.sh yun1 ALL GREEN。
- Tencent 同步：git pull --rebase 至 e170bc4，开发环境与 WSL/GitHub 三方一致。
- 用户实测入口：顾客端（5173/18090）注册→预约（上下午模糊）→我的核销码（无卡不出码）；商家端（5174/18091）扫码→散客核销、改期控件、闭店设置、营业配置。

# PHASE RESULT — V1.x 第二轮：模型修正全链收口 + 门店信息/移动端/深色模式（A1-A9）
- Changed/Why：目标 88 节 V1.x 第一阶段全部落地——状态机 WAITING 收紧前后端对齐（admin format/按钮/路由/summary、customer 文案/可取消条件清零 PENDING_CONFIRM，/confirm 端点断言 404）；统一结算页（扫码 ANMO-MEMBER/ANMO-APT 前缀分流、今日预约自动选/散客自动兜底、实际服务可改、结算时选卡、现金/微信同事务完成）；顾客端卡使用明细（GET /api/me/cards/{id}/transactions 归属校验）+ card_name 动态 + 有效期格式；capacity 双断点修复（ContentPage 挂载回填 + saveSetting 服务端校验 1~999/30|60|120/HH:MM，前端上限 999）；首页文案 home_title/home_body + 标签改名/删除/组合搜索(tag_id/card_type)+列表标签装饰；门店信息 shop_address/latitude/longitude + GET /api/store/status 动态三态（FREE/SERVING+free_at/BUSY，content.Wire 二阶段装配）+ ShopCard 三处落点（首页/预约成功页/关于我们，高德 marker 导航+tel: 拨号）；后台移动端（Records/CardTemplates/Services 手机卡片流、inline 表单单列、toolbar 换行、body overflow-x hidden、.fixed-bar safe-area）；顾客端 Design Tokens（style.css 全量重写 + 11 页硬编码色值批量 token 化 + 深色模式 prefers-color-scheme）；商家改手机号/密码 PUT /admin/auth/credentials（热生效，JWT 按 user id）。
- Files：server（identity/member/card/content/appointment/transaction + e2e/adversarial 测试迁移）+ apps/admin（10 页/组件）+ apps/customer（11 页/组件）+ AGENTS.md（D8/D9 修订、核心事务 2/5）+ appointment/AGENTS.md + PRD 增量修订。
- DB：无新 migration（012 已由前序建立并应用）。
- API：+GET /admin/appointments/{id}、+GET /api/me/cards/{id}/transactions、+GET /api/store/status、+PUT /admin/auth/credentials、+PUT/DELETE /admin/tags/{id}、/admin/members 增 tag_id/card_type 参数、redeem 增 service_id；-PUT /admin/appointments/{id}/confirm（删除）。
- Tests：全量 go test ./... -count=1 14 包全绿（含新 TestSlotCapacityConfigurable/TestStoreStatusThreeStates/TestBookingRulesCapacityFromSettings/TestUpdateCredentials/TestListFiltersByTag/TestSettleByCardWithActualServiceOverride/TestSettleByPayCompletesAndSingleValid）；两端 vue-tsc 过。
- Risk：散客现金收款无端点（明确不做）；深色模式依赖页面 var() 覆盖度（主路径已覆盖）。
- Next：B——GitHub 提交 + yun1 部署验证。

# PHASE RESULT — V1.x 第二轮 Phase B：GitHub + yun1 生产部署验证
- 提交链：dd344c9（V1.x 第二轮全量功能）→ c672f66（migration 012 就地修正：先 DROP 旧 CHECK 再 UPDATE——MySQL CHECK 对 UPDATE 生效，yun1 含历史行库首应用失败实证；012 从未成功应用过故允许就地修）。
- yun1 部署：WSL 三件套构建（admin --base=/admin-ui/、customer、linux/amd64）→ docker save|scp|load → force-recreate；migration applied count=1；历史数据迁移验证（5 条 PENDING/CONFIRMED → WAITING，状态分布 WAITING 5/COMPLETED 7/CANCELLED 2，新 CHECK 生效）。
- 生产端点实测：管理员登录 ✓、GET /admin/tags 200、credentials 空参 400（路由活）、booking-options（capacity=2 配置生效回读）、顾客端 GET /api/store/status 返回 BUSY（07:26 早于开门 09:00，计算正确）、GET /api/settings 暴露 home_*/shop_*/slot_capacity、静态两端 200。
- health-yun.sh yun1 ALL GREEN；Tencent git pull --rebase 至 c672f66，三方一致。
- 用户验收路径：顾客端 18090（首页门店状态徽标+门店卡片导航/拨号、预约成功页、核销码页含预约单码 ANMO-APT、卡使用明细、深色模式）；商家端 18091/admin-ui/（扫码结算页分流、今日预约自动选、实际服务改选、结算选卡、账号设置改手机号/密码、内容页首页文案+门店信息+营业配置、会员标签管理+组合筛选、手机端无横向溢出）。

# PHASE RESULT — V2（Phase C+D 收口，Phase E 经用户指示取消）
- Completed：V2 主线（微信登录后端 + 原生顾客端小程序 `apps/weapp` 8 页 2 组件）打磨到"可交付"并补齐验证闭环。本轮改动集中在小程序：① 登录态启动链路收口（`app.js ready()` 读最新态 + 新增 `markAuth()`，login 成功/me 退出/`request` 401 三处同步全局态；login 页复用启动 `session` 不再抢跑二次 wx.login；bindTicket 经 globalData 透出）；② 核销码页 canvas 在 `wx:if` 内改为 setData 回调后再绘制（原实现整页二维码画不出）；③ 卡流水符号口径 `format.txQty`（REDEEM 存正数语义为扣次→显示 `-N`，ADJUSTMENT 保留符号，ISSUE/REVERSAL 为 `+N`），**H5 `CardsPage.vue` 同源同改**；④ `format.trimPastSlots` 前端裁掉今日已开始的槽 + slot-picker `ended/已闭店/已约满` 三态标签（后端不回溯，避免动 D20 共享口径）；⑤ qrcode/appointments 的服务名回填改按 id `findIndex`（原用闭包下标会打错位）；⑥ about 页分享落地降级（settings 缓存兜底 + 登录引导卡 + 补 `onPullDownRefresh`）；⑦ 核销码页 keepScreenOn 在 onHide/onUnload 释放；⑧ `format.homeBlocks` 按后端 `content_page_config` 编排首页内容块（D16，与 H5 同口径，banner 不渲染）。
- Tests：
  - 纯逻辑单测（新增 `apps/weapp/tools/unit.js`，零依赖）：**PASS=19 全绿**（slot-picker 门槛 6 例、trimPastSlots 4 例、txQty 3 例、homeBlocks 3 例、展示口径 3 例）。
  - IDE 端到端（新增 `apps/weapp/tools/devtools-verify.js`，11 stage 逐 stage 重连重试）：**PASS=53 FAIL=0**（`.dev/verify3.log`）；修完 screenshot 路径后复跑 **PASS=50 FAIL=0**（`.dev/verify5.log`）；持卡会员 13990497037 专跑 login+qrcode+cards **PASS=13 FAIL=0**（`.dev/verify4.log`，覆盖 D21 正分支与真实流水 REDEEM/REVERSAL/ISSUE）。
  - 实测覆盖：短信登录→token 落地、登录态首页（含 blocks 顺序=后端编排）、模糊预约 `{date,day_part}`→成功卡→我的预约可见→取消（→CANCELLED 且不可再取消）、具体槽 `{start_time}` 提交、今日预约 ANMO-APT 单码+服务名快照、D21 无卡不出会员码（DOM 断言 canvas 数）、今日已过槽不展示、关于页三字段=接口值对账、会员卡与明细展开。
  - 后端：`go build ./... && go vet ./... && go test ./... -count=1`（带 `ANMO_TEST_MYSQL_DSN`）exit=0，**14 包 ok、0 SKIP、0 FAIL**（`.dev/go-test-final.log`）。
  - H5：`vue-tsc --noEmit -p tsconfig.json` 过、`vite build` 过（`npm run build` 的 `vue-tsc -b` 报 HomePage/QRCodePage 既有类型错，与本次改动无关，未扩大范围去动）。
- Database：无新 migration（013_member_wx_openid 由 Phase C 建立并已应用）。开发库残留本轮实测预约（APT…0015 已取消、0016/0017/0019/0020 等）未清理，生产库无。
- Files Changed：`apps/weapp/`（app.js、utils/{request,api,format}.js、8 页、2 组件、新增 tools/{unit.js,devtools-verify.js,package.json}、README.md）；`apps/customer/src/pages/CardsPage.vue`（流水符号同口径）；`AGENTS.md`（决策表补 D23–D26）；`V2-HANDOFF.md`（进度表/环境/§4 打磨清单/§6 提效链路/§7 证据/§8 剩余与不做清单全面刷新）；`.gitignore`（+`.dev/`）。
- 坑（务必沿用）：① 本机 `/tmp` 会被系统清空 → 二进制、node_modules、验证脚本一律落仓库（`.dev/` 已 gitignore、`apps/weapp/tools/`）；② `cli auto` 前必须 `quit` + sleep 15s，`√ auto` 后再等 ~50s，否则 automator 首条命令必 timeout / `initialize error: read ECONNRESET`；③ 该 devtools 版本自定义组件内部节点 automator 够不到 → 受控组件走 `page.callMethod('onPick'|'pickDay')`，组件门槛交给 unit.js；④ `automator.screenshot({path})` 是**本机 Node fs 写盘**（IDE 只回传 base64），传 Windows 路径会在 cwd 生成带反斜杠的垃圾文件 → 落 `.dev/shots/`；⑤ `callWxMethod('getStorageSync')` 返回裸字符串，token key 是 `anmo_customer_token`；⑥ 取消类用例必须按单号 `findIndex` 定位按钮下标。
- Risk：门店地址/电话在开发库未配置（规范键 `shop_phone`/`shop_address`，库里只有历史 `store_phone`）→ 关于页走空态文案，实测改为"页面值=接口值"对账而非断言有值；banner 内容块小程序端不渲染（依赖外部图片域名+合法域名）；H5 的今日已过槽仍未裁剪（本轮只统一了账目符号口径）。
- Next：等用户点头后走部署链（小步 commit → 推 GitHub main → yun1 二进制/migration 013 应用验证 → Tencent pull）；小程序发布前清单见 `apps/weapp/README.md`（真实 AppID、`ANMO_WX_APPID/SECRET`、HTTPS 备案合法域名）。

# PHASE RESULT — P1 存储 MySQL→SQLite（2026-09-28）

## 交付物
1. migrations 001-013 全量 SQLite 方言（生成列 active_lock STORED / valid_lock VIRTUAL + UNIQUE、updated_at 触发器、闭店 ENUM→CHECK、JSON→TEXT+json_valid；005/006/011/012 按"全新库=终态"折叠 MySQL 专用 DDL，文件头有说明）
2. 并发锁模型：`_txlock=immediate`（BEGIN IMMEDIATE）+ busy_timeout 10s + WAL + FK；NamedLock/GET_LOCK/FOR UPDATE 全移除（database/lock.go 删除、shared.NamedLocker 删除）
3. 驱动：modernc.org/sqlite v1.59.0 纯 Go（CGO_ENABLED=0）；DSN `_timezone=Asia/Shanghai&_time_format=datetime` —— time.Time 绑定/扫描零改动，JSON 时间戳保持 +08:00 与 MySQL 版一致（实测）
4. shared：IsDupKey（2067/1555）/IsBusy（5/6）方言谓词；sequence INSERT OR IGNORE
5. 方言清理：NOW()→datetime('now','+8 hours')、CURDATE/INTERVAL→Go 侧算截止、ON DUPLICATE→ON CONFLICT DO UPDATE、INSERT IGNORE→OR IGNORE、DATE_FORMAT→strftime、CONCAT→||、mysql 1062/1213 检测→shared 谓词
6. config：mysql.dsn → database.path（ANMO_DB_PATH）；config.example.yaml 更新
7. testsupport：SQLite 临时文件库（t.TempDir），**测试零外部依赖**：14 包全绿 0 skip
8. 对抗测试：A5 并发收款/A10 改期锁等待 按单写者模型重写（语义等价）；预约风暴 A13 / 并发核销 A6 / 并发收款 A5 全 PASS
9. 迁移工具 server/cmd/mysql2sqlite（独立 go.mod，主模块无 go-sql-driver）：本地 26 表全量搬迁，行数/FK/integrity/不变量/序列延续/应用级冒烟全过（REPORT.md）
10. 部署：server/Dockerfile（静态二进制+非 root+VOLUME data/backups）入仓；docker-compose.yml 去 MySQL 挂 ./data ./backups；scripts/backup-sqlite.sh（VACUUM INTO 快照+完整性校验+保留 14 份+恢复演练）；main.go 增 `-backup`/`-migrate`
11. 文档：AGENTS.md（数据库规则/环境/D5/D18 修订标记）、HANDOFF.md、appointment/card AGENTS.md

## 验证
- `go build ./... && go vet ./...` 干净；`go test ./... -count=1` 14 包 ok 0 skip
- 容器实测：镜像构建→启动自动 13 migration→API 冒烟→容器内 -backup 落宿主机卷
- 服务器二进制 `go version -m` 无 mysql 依赖

## 决策记录
- SQLite 版 migration 保留文件编号与历史语义（012 在 SQLite 上为空操作+说明）
- mysql2sqlite 独立嵌套 go.mod：工具一次性使用，主服务二进制零 MySQL
- yun1 生产数据迁移在 P4 部署时执行（步骤见 cmd/mysql2sqlite/REPORT.md），旧 mysql-data 卷保留 ≥2 周作回退

# PHASE RESULT — P2 双端 UI 重构（2026-09-28）

## Completed
1. 设计体系落地（docs/BRAND-GUIDELINES.md 为唯一来源）：H5 `src/style.css` 全量重写（token+类库 .btn/.card/.cell/.chip/.badge/.input/.empty/.skeleton/.seg/.stat），`apps/weapp/app.wxss` 同名同值镜像；两端 UI 图标统一为 AppIcon.vue / components/app-icon（SVG data-uri + mask，25 名同 PATHS）；H5 新增 src/components/ui/ 12 个薄组件
2. 核心业务 bug 修复（多核销码）：两端核销码页改为单一会员码 ANMO-MEMBER:<member_id>（D21，有 ACTIVE 卡才出示）；今日预约降级为文字列表（服务名取列表内嵌 services 快照）；商家端 ANMO-MEMBER 扫码流程闭环（后端本就按会员+当日预约分流）
3. 后端配套（附加式）：顾客预约列表内嵌 services 快照（AppointmentWithServices + ServicesOfMany 批量查询，两端删 N+1）；InternalNote 顾客端一律置空 + json omitempty（§107）
4. H5 全部 10 页 + weapp 全部 8 页（home/booking/appointments/cards/qrcode/me/login/about）按新体系重构，业务逻辑保留（weapp 登录链路 app.js ready/markAuth/bindTicket 未动）；weapp 删除 tools/ E2E 脚手架与全部 console.log；品牌口径统一"安摩"，页面标题统一
5. 预约模块交互强化（两端同构）：步骤编号 1-2-3-4、服务两列网格、日期横滑 chip、上午/下午大卡（余量三态：已闭店/时段已过/已约满/剩 N）、具体槽 4 列网格（满槽 badge 禁选）、提交按钮动态汇总文案、成功页（对勾+单号+门店卡+双按钮）
6. 卡片页改"选卡→明细"模型（金卡视觉+badge+流水 ±符号三色），me 页加头像+统计瓦片（有效卡/剩余次数/即将到店，实时拉取），login/about 品牌印章头
7. 禁用按钮态两端修复：微信内置 button[disabled]:not([type]) 特异性压制 → .btn[disabled] 用 !important + 高特异性选择器；H5 .btn.primary 在级联后段 → 补全 variant:disabled 选择器；统一 muted 灰底
8. E2E/UI 验证脏数据清理：/tmp/p2-review/anmo-dev.db 全业务表清空（18 member/10 service/31 appointment/7 卡/15 流水/281 操作日志），保留 identity_user OWNER 与 content_system_setting/home page_config，integrity_check ok；建干净会员 13900000001

## Tests
- weapp devtools automation（miniprogram-automator + cli auto :9420）：8 页 reLaunch 截图全过（w4-*，持卡会员含 QR/今日预约/待到店 badge）、交互态 2 张（选卡出明细 w4-cards-detail、选服务出日期条 w4-booking-picked）、无卡会员锁定态（w4n-qrcode）、干净数据空态 7 张（w5-*）
- H5 playwright 截图：全 10 页两轮（h5b/h5c），禁用态修复后 booking 复拍确认
- `npm run build`（H5）通过；后端无新改动（P2-2 已验：列表内嵌 services、internal_note 不外泄）
- automation 脚本留存 .dev/verify-weapp/（gitignored）

## Database
- 无新 migration；开发库（评审用）脏数据已清空如上；生产库未动

## 坑（新增）
- weapp devtools automation：connect 后首次 reLaunch 若 devtools 后台编译未完成必 timeout —— 逐页 try 3 次 + 先探 currentPage
- app 实例 session 只在 onLaunch 跑一次：automation 换 storage token 后必须 evaluate getApp().markAuth(true)，否则 ready() 命中旧缓存全页按未登录渲染
- /tmp 清空风险仍在：验证脚本已落 .dev/；cli.bat 位于 D:\Program\soft\wechattools\

## Next
- P3：10 轮强制迭代（剃刀规划 + 墨菲对抗子代理审查）+ ≥1 项自选高价值任务
- P4：GitHub 推送 + yun1 生产数据迁移（REPORT.md runbook）+ 部署 + health-yun.sh 全绿

# PHASE RESULT — P3 十轮强制迭代（2026-09-28/29）

## Completed
1. R1 weapp 预约主链路 E2E 16/16（automator :9420，真实后端建单）；墨菲对抗审查 12 项发现全部修复并回归（silent-login 挂起不阻塞、svcIdx 对齐 id、跨零点重算日期条等）
2. R2 H5 预约主链路 13/13（playwright 冷加载 + addInitScript 种 token，绕开 hash 同文档导航的 automation 幻影重定向）
3. R3 改期/取消异常路径 13/13（API 级）：他人预约 403、2h 窗口、闭店日、容量冲突、改期排除自身、HALF_DAY 半日已结束拒绝
4. R4 结算闭环对抗 65/65：核销单事务、幂等键、撤销置 payment VOIDED（valid_lock 生成列）、散客核销 RDM_WALKIN_BLOCKED、撤销后重核销、一预约一笔 VALID 不变量
5. R5 弱网/宕机/401 双端全绿：H5 http.ts 补 15s AbortController 超时 + NETWORK 归一（修复 Vite 代理下 fetch 永久挂起 → 页面永卡骨架）；weapp 宕机失败态 + 重试恢复全链路（booking svcErr 态为本次新增——加载失败不得伪装"暂未上架"）；401 双端清理 + 回跳；stale-response 竞态以响应篡改注入验证丢弃
6. R6 视觉对照 BRAND-GUIDELINES：双端 16 张截图逐张过审全 on-brand；修 3 处（qrcode 页 tab 高亮、NO_SHOW 筛选口径、头像缺字回落'客'）
7. R7 分享落地/登录回跳：weapp 8/8 页 onShareAppMessage（D26，home/about 另有 onShareTimeline）；无 token 落预约页出登录引导 → 点按 → 短信登录 → navigateBack 回预约页即登录态（8/8）；H5 守卫 redirect 参数 → 登录后回原目标页（7/7）
8. R8 展示口径对账（双端逐项 diff）：NO_SHOW '爽约'→'未到店'（H5 badge+format）；AppStatusBadge 补历史态 PENDING_CONFIRM/CONFIRMED 徽标（旧数据曾会裸显英文码）；'店家'→'店主'统一；'即将到店'口径统一为"今天(北京)及以后的活跃预约"（weapp 补日期过滤 / H5 弃设备本地时区）；改期日历与核销码页"今天"改北京时间锚定（MyAppointmentsPage.buildDays / QRCodePage.todayStr）；R2 13/13 + R3 13/13 回归过
9. R9 包体与首屏审计：H5 首屏 4 API 全并行无重复（~177ms）；产物 index 105KB(gzip 41KB)+路由懒加载 chunk（最大 QRCodePage 27KB 含 qrcode 库）；weapp 包 384K（限 2M），home 双请求并行 —— 均无需改动（剃刀）
10. R10 自选高价值任务：生产迁移 runbook 本地全彩排（REPORT.md 逐条复现）——mysql2sqlite 重建 /tmp 库，26 表行数与 MySQL 全对齐，fk/integrity/不变量校验过，应用级冒烟（既有会员短信登录、19 条预约读取含 +08:00 JSON、新建 APT202609290001 落库格式正确）；彩排中顺带验证迁移后的闭店行在运行时真实生效（APT_CLOSED 拦截 9-30 上午）

## Tests
- 双端 E2E/对抗累计 130+ 断言全绿；`go build ./... && go vet ./... && go test ./...` 全绿（SQLite 后 0 跳过）
- H5 `vite build` 过；脚本与 README 持久化 .dev/verify-weapp/（含运行前提、数据清理规则、wx.login mock 残留提醒）

## Database
- 无新 migration（P3 纯验证+双端展示层修复）；开发库预约数据按套件间清理规则管理

## Files Changed（P3 全部）
- apps/customer: core/api/http.ts（15s 超时+NETWORK）、components/ui/AppStatusBadge.vue、core/utils/format.ts、pages/{Booking,MyAppointments,Me,QRCode}Page.vue
- apps/weapp: pages/booking/{booking.js,booking.wxml}（svcErr+重试+分享）、pages/me/me.js（upcoming 口径）、pages/qrcode/qrcode.js、pages/login/login.js、pages/appointments/appointments.js（NO_SHOW 筛选）、utils/format.js（状态词对齐）
- 后端零改动（R4 曾修 Adjust 归零置 USED_UP + 单测，属 R4 内）

## Remaining
- 无（P3 目标 10 轮 + 自选高价值全部完成）

## Next
- P4：推送 GitHub → yun1 生产数据迁移（REPORT.md runbook，R10 已本地彩排）→ 部署 → health-yun.sh 全绿

# PHASE RESULT — P4 生产迁移与部署（2026-09-29）

## Completed
1. GitHub 推送：406e69e（server SQLite+微信地基+迁移工具）/ 0a2fa06（weapp+H5+P3）/ c350743（docs）/ 762dc3f（backup fix）
2. yun1 生产迁移（REPORT.md runbook 逐步执行）：mysqldump 预备份（backup-pre-migration-20260929.sql, 88K）→ 停写 → mysql2sqlite（linux/amd64 交叉编译）→ 26 表全量复制，fk_check=0、integrity ok、不变量=0
3. 栈切换：anmo-server:v2 镜像（Dockerfile SQLite 版，uid 10001, /data 卷）→ 新 compose（去 MySQL 化，secrets 原值承接）→ force-recreate → anmo-mysql 停用保留（回退：mysql-data 卷 + compose.mysql-bak + mysqldump，保留两周+）
4. 前端原子换装：admin/customer dist 上传 staging 后 mv 换名，admin-ui 与顾客 H5 均 200
5. 生产冒烟全绿：admin 登录+预约端点；顾客登录（18656864931）5 条预约（+08:00 JSON 格式正确）、12 次卡、profile member_no M202609270004 序号延续；/healthz 200；server 日志 0 error
6. 备份体系：backup-sqlite.sh 部署 + cron 03:30（保留 14 份）；首份快照 464K 落 /opt/anmo/backups；--restore-test 启动级恢复演练通过（修两处：临时副本演练保持快照只读、chown 10001 对齐容器 uid；镜像 tag 对齐 v2）
7. yun1 README 同步 SQLite 现状；health-yun.sh yun1 ALL GREEN

## Tests
- 生产端到端：登录/读/卡/profile/健康端点全过；容器状态 anmo-server Up、日志无 error

## Database
- 生产库 /opt/anmo/data/anmo.db（迁移自 MySQL 全量 26 表）；MySQL 容器停用保留

## 坑（新增）
- restore-test 原实现以 ro 挂载直测快照 → WAL 开库写失败；已改临时副本演练（快照本体不可变）
- dev SMS 60s 频控：冒烟脚本内 send→verify 必须同会话一次完成，跨命令复用旧 code 会"验证码错误或已过期"

## Next
- 发布小程序前必须配置真实 wx.app_id/secret（D24）与正式短信通道；生产 compose 中 ANMO_SMS_MODE=dev 记得切换


## Session 2: V2.1 首页可配置化与小程序体验修复
<!-- trellis-session: v=2 fp=6891e93eaed25c05 -->

**Date**: 2026-09-29
**Task**: V2.1 首页可配置化与小程序体验修复
**Branch**: `main`

### Summary

四项修复：①首页服务推荐商家可配置（admin 配置卡 + settings home_service_limit/ids + H5/weapp pickHomeServices 同构筛选，limit∈{2,4,6,8}默认6，ids 有序子集优先，自适应）②门店电话链路核验（代码已闭环，生产需后台填 shop_phone）+ weapp 关于页补营业时间 ③weapp .input 显式 48px 修复登录占位截断 ④weapp 新增服务 Tab（pages/services 四件套 + tabBar 4 项对齐 H5）+ 首页全部服务入口。验收：trellis-check 全绿；墨菲对抗式 17 API 检查 + 10 场景矩阵全绿（真实服务临时库）；go/vue-tsc 构建全绿。提交 e073b69/65dee67/fd296f8 + spec 两条沉淀。

### Git Commits

| Hash | Message |
|------|---------|
| `fd296f8` | docs(spec): weapp 原生 input 显式高度陷阱 + 顾客可见 settings 配置模式约定 |

### Status

[OK] **Completed**


## Session 3: V2.2 第一批：预约页固定提交条/地图选点/H5地址兜底/资料完善
<!-- trellis-session: v=2 fp=f822468ce1c88a02 -->

**Date**: 2026-09-29
**Task**: V2.2 第一批：预约页固定提交条/地图选点/H5地址兜底/资料完善
**Branch**: `main`

### Summary

审查 docs/2026929plan.md 并与用户确认三批范围（技师账号不做、H5切密码、散客快速结算）；第一批经 trellis-implement/check 落地：weapp+H5 预约页底部固定提交条（safe-area、三态置灰、摘要行）、admin 高德地图选点+经纬度校验、H5 地址标准 URI+兜底条、资料完善三件套；Murphy live-server 矩阵 12/12，修复服务端 amap 键 trim 落库缺口；spec 增补落库归一化与 pagehide 约定

### Git Commits

| Hash | Message |
|------|---------|
| `6b10091` | docs: V2.2 计划文档收编 + CHANGELOG 未发布段（第一批） |

### Status

[OK] **Completed**


## Session 4: V2.2 第二批：H5 密码登录体系
<!-- trellis-session: v=2 fp=fc5a0242f61d8d5d -->

**Date**: 2026-09-29
**Task**: V2.2 第二批：H5 密码登录体系
**Branch**: `main`

### Summary

migration 014 重建 member（phone 可空+password_hash）；H5 注册/登录；小程序纯微信登录（首登直建号，废除 D25 bind_ticket）；撞号密码认领 claim（空壳校验+转绑+删除，单事务）；设/重置 H5 密码；admin 重置会员密码+徽标；SMS 下线（dev 过渡/off 410，生产必须 off 关闭 dev 任意登录洞）；AGENTS.md D25 修订+D27 新增。check 修复 2 处（迁移 FK 断言缺口、validatePhone 口径）；Murphy live 23/23 实质全绿（3 个脚本期望口径问题已甄别）

### Git Commits

| Hash | Message |
|------|---------|
| `b434c5f` | feat(auth): V2.2 第二批——H5 手机号+密码登录体系（member 014 重建 phone 可空+password_hash、微信首登直建号、撞号密码认领 claim、设/重置 H5 密码、admin 重置、SMS 下线 off 模式） |

### Status

[OK] **Completed**


## Session 5: V2.2 第三批：服务记录与散客快速结算
<!-- trellis-session: v=2 fp=2d05f16798eabfb4 -->

**Date**: 2026-09-30
**Task**: V2.2 第三批：服务记录与散客快速结算
**Branch**: `main`

### Summary

migration 015（service_tag/service_record 建表 + payment 重建 member_id→NULL，006/010 约束索引触发器全保留，升级路径测试）；服务标签管理（两组/去重/停用/排序/仅未使用可删，停用仅历史筛选可见）；service_record 四结算入口同事务落记录+强制沟通确认 TX_NEED_CONFIRM；核销撤销联动置 REVERSED；散客快速结算 POST /admin/walkin/settle（手机号可选，不录仅记账 record_skipped；CARD 拒绝；幂等回放；EnsureByPhoneTx 建档）；会员详情服务追踪（倒序/筛选/关键词/汇总）+ 商家备注 + 散客记录独立撤销路由；service_note_presets 快捷短语 KV；admin 新页/弹窗/追踪区。AGENTS.md D28/D29。check 修复 4 处（缩进回退、既有 00:00-01:00 时钟 flake 守卫、GetItemTx 错误吞没、散客手机号前端正则）；主会话补散客撤销路由+UI 缺口；Murphy live 矩阵 44/44 PASS（前期 FAIL 均为脚本种子/断言伪影，逐一甄别）

### Git Commits

(No commits - planning session)

### Status

[OK] **Completed**


## Session 6: V2.2.0 版本收口与 yun1 生产部署
<!-- trellis-session: v=2 fp=1f1d0dbd733c9d51 -->

**Date**: 2026-09-30
**Task**: V2.2.0 版本收口与 yun1 生产部署
**Branch**: `main`

### Summary

CHANGELOG v2.2.0 三批收口（tag v2.2.0 已推 GitHub main）。yun1 部署：docker build anmo-server:v2（镜像内 CGO=0 自编译）→ save|scp|load；admin --base=/admin-ui/ + customer dist staging 原子换装；部署前 backup-sqlite.sh 快照（anmo-20260930-004615.db）；docker-compose ANMO_SMS_MODE dev→off（关闭 P4 遗留的 dev 任意手机号+123456 登录洞，SMS 端点实测 410 SMS_DISABLED）；migration 014+015 自动应用（count=2，生产库原在 013——member 重建 phone 可空+password_hash、payment 重建 member_id NULL）。踩坑新增：server 容器 force-recreate 后容器 IP 变化，nginx 缓存旧上游 IP → 502，docker restart anmo-nginx 恢复（已写入 yun1 README 更新流程第 5 步）。冒烟全绿：healthz 200、admin 登录、/admin/service-tags 200（空表符合预期，商家自建标签）、/admin/walkin/settle 无 token 401（路由活）、H5 18090 200、admin-ui 18091 200 + 新 chunk 加载、外网可达、docker logs 0 error、-backup 快照（含 integrity_check）通过。生产写入类新功能（标签/记录/散客结算）未在产线造数——以临时库 44/44 矩阵为准。Tencent 机下次使用前 git pull --rebase 即可

### Git Commits

(No commits - planning session)

### Status

[OK] **Completed**


## Session 7: V2.2 第四批：H5 预约弹层翻转 + 资料保存返回 + 小程序联调修复
<!-- trellis-session: v=2 fp=1de57cf6876a36e9 -->

**Date**: 2026-09-30
**Task**: V2.2 第四批：H5 预约弹层翻转 + 资料保存返回 + 小程序联调修复
**Branch**: `main`

### Summary

用户三问实为开发者工具旧包：tabBar 服务 tab 齐全/登录页已纯微信(D27)/占位符已修——重导入 apps/weapp+清缓存即愈，README 联调章节固化。真修复：①BookingPage 资料弹层翻转（立即预约 primary 在上=提交+置静默标记 bookNow；去完善资料 ghost 在下=存草稿跳资料页）②ProfilePage 保存后检测 sessionStorage anmo.booking.draft 自动返回 /booking，草稿由 takeDraft 恢复（B3 弃单误伤窗口评估为可接受不修）③weapp config.js storage 键 anmo.base_url 覆盖 BASE_URL（合法 http(s) 锚定校验，check 阶段补 \\+\\$ 锚定+用例）④README 联调章节+清缓存指引+D24 dev 登录原理；start-anmo.sh 提示行改密码登录口径（删过时短信 123456 提示）。check 链路审查 7/7 过+自修复 2 处（README 引用不存在的 tools/unit.js 重写为 tests/config.test.js；覆盖值正则未锚尾）。GUI E2E 环境注记：dev 库播种 2 分类 3 服务；上午半日池被多轮测试订满属 D20 容量逻辑正确工作

### Git Commits

| Hash | Message |
|------|---------|
| `b6c5f89` | fix(apps): V2.2 第四批——H5 预约弹层翻转（立即预约凸显在上）+ 资料保存自动返回预约页 + 小程序 BASE_URL 覆盖与联调文档 + config 单测 |

### Testing

- [OK] weapp config.test.js 6/6；go build/vet/test 全绿零跳过；H5 vue-tsc+vite build 零错；GUI E2E 17/17 PASS + console 0 error（playwright-core+系统 Chrome 黑盒，证据 /tmp/anmo-b4-gui）

### Status

[OK] **Completed**

### Next Steps

- 小程序真机 checklist 由用户按新 README 联调章节自测；正式发布前配真实 ANMO_WX_APPID/ANMO_WX_SECRET；生产店名电话等由商家在 admin 维护


## Session 8: V2.2 第五批收尾：CF Tunnel anmo.oiob.cn 接入（生产入口切换）
<!-- trellis-session: v=2 fp=0a2ea468180b3143 -->

**Date**: 2026-09-30
**Task**: V2.2 第五批收尾：CF Tunnel anmo.oiob.cn 接入（生产入口切换）
**Branch**: `main`

### Summary

根因：yun1 出网到 login.cloudflareaccess.org 被墙（Failed to fetch resource），浏览器兜底下载的证书也未见落地——授权本身每次都成功。解法：本地 WSL ~/.cloudflared 已有 2026-05 的 origin 证书（同一账号、zone 授权 oiob.cn），直接 scp 上服务器复用，零新增点击。隧道 anmo(08368ffb) systemd 常驻：anmo.oiob.cn→18090（API+H5 同源，微信合法域名）、admin.oiob.cn→18091(noTLSVerify)。已记录直连 18090/18091 不通属安全组预期。剩余用户动作：发布前配 ANMO_WX_APPID/ANMO_WX_SECRET

### Git Commits

| Hash | Message |
|------|---------|
| `e06ad0b` | fix(weapp): 生产域名切换 api.oiob.cn → anmo.oiob.cn（对齐微信后台已绑定的 request 合法域名），单测 7/7 |

### Testing

- [OK] 公网验证：https://anmo.oiob.cn/healthz ok；匿名 GET /api/services 返回目录 JSON（游客模式线上生效）；未登录 POST /api/appointments 401；https://admin.oiob.cn/admin-ui/ 200；systemd cloudflared active（quic sjc 注册）

### Status

[OK] **Completed**


## Session 9: V2.2 第七批：真实 DevTools 全流程自动化（生产链路）+ 游客态两处真 bug 修复 + 官方 wechatide skill 评估
<!-- trellis-session: v=2 fp=c9dd9f43307227d5 -->

**Date**: 2026-09-30
**Task**: V2.2 第七批：真实 DevTools 全流程自动化（生产链路）+ 游客态两处真 bug 修复 + 官方 wechatide skill 评估
**Branch**: `main`

### Summary

用 Windows 微信开发者工具（cli.bat auto 9420 + miniprogram-automator）对生产 https://anmo.oiob.cn 跑通 16 断言全流程：游客浏览→微信 code2session 真登录→预选保留→提交真实预约号→列表→mock 弹窗取消→遗留清零→核销码门槛，console 零错误。关键工程结论：storage 不跨 cli auto 冷重启→改用副本 config.js LOOPBACK_URL 指生产；clearStorageSync 不清 app 内存登录态→正好当凭证过期模拟。对抗审查修复两处真 bug：游客首页服务列表为空（home 漏改第五批口径）、我的预约 401 冒充网络错误。官方 wechatide skill（v0.3.11）已随构建内置并验证门禁调用链，授权弹窗待用户点击。沉淀 docs/guides/wechat-devtools-automation.md。

### Git Commits

| Hash | Message |
|------|---------|
| `9260d5b` | docs(guides): 微信开发者工具自动化运维手册——两层自动化、踩坑实录、官方 wechatide skill 评估；第七批任务档案 |

### Testing

- [OK] 16/16 PASS（生产真实链路）+ T10 游客引导回归 2/2 + console errors: none + node --check 全过

### Status

[OK] **Completed**

### Next Steps

- 用户在 DevTools 点击 zcode 授权弹窗后，补官方 skill 实操段到手册 §6；发布前 SMS 切 off；owner 清理生产测试服务与 E2E 测试会员

## Session 10: V2.2 第八批：用户实测 UX 反馈修复（适老化 + 弹窗化 + 服务切换）+ 全局弹层包含块大坑
<!-- trellis-session: v=2 fp=c9dd9f43307227d5 -->

**Date**: 2026-09-30
**Task**: 09-30-09-30-v22-batch8-profile-ux-booking-fixes
**Branch**: `main`

### Summary

用户真机试用反馈三问题，逐一修复并实证：① 我的页编辑卡观感弱/性别不居中/字重过细 → 编辑改半屏弹窗（与 claim/pwd 同构）、seg 居中、--font-body 400→500（H5 镜像同值）、field-label 提级；② 备注框只显示一半 → .textarea min-height 122px + 连带根因 submit-bar 未钉底；③ 服务切换"被束缚" → automator 实证切换机制本可用（svcIdx 0→1 针灸），补 ✓ 角标视觉反馈。排查弹窗底部沉屏时抓到全局大坑：.page-body 入场动画 fill-mode both 终态 transform 残留 → page-body 持续充当 fixed 子元素包含块（.mask 实测 856px > 视口 671px），me 三弹层/booking R4/改期抽屉/submit-bar 全部错锚，动画去 both 一改全修。批 8 还踩实了坑 #10：rsync --delete 覆盖副本 config.js 致 15/16 假绿打在本地 dev 库（识别指纹：wx 与 prod 服务名完全不同 + 单号序号倒退 0009<0011），§4 同步命令改 --exclude config.js。

### Git Commits

| Hash | Message |
|------|---------|
| `d09e1f4` | fix(weapp): 第八批 UX 适老化与弹层体系修复——真机可感三问题 + 全局包含块大坑 |

### Testing

- [OK] 16/16 PASS（rsync 修复后真生产：APT…0013 接续批 7 序列、T1b prod==wx 实时一致）
- [OK] 弹窗遮罩 390x671 精确视口、submit-bar bottom=671 钉底、五菜单项 tap 巡检 + console errors: none
- [OK] H5 vite build ✓；「我的预约」「设置H5密码」cell 慢节奏复测 PASS（首轮异常为脚本导航竞态，非产品 bug）

### Status

[OK] **Completed**

### Next Steps

- 发布前 SMS 切 off（SMS_DISABLED）；owner 清理生产测试服务（打人 ¥0 / 云端冒烟推拿60分钟）
- 弹窗几何验证基于 DevTools WebView；真机微信客户端如仍有视觉偏差，下批用预览版复核

# PHASE RESULT

**Phase**: V2.2 第八批（用户实测反馈修复）
**Result**: 完成
**验证**: automator 16/16 真生产全绿 + 几何复核（mask=视口、submit-bar 钉底）+ 巡检零 console error + H5 build
**提交**: d09e1f4
**遗留**: 无代码遗留；发布运维项同 Next Steps

## Session 11: V2.2 十轮子代理对抗式审查（墨菲定律验收）——只审不改
<!-- trellis-session: v=2 fp=c9dd9f43307227d5 -->

**Date**: 2026-10-01
**Task**: 10-01-adversarial-review-10rounds
**Branch**: `main`

### Summary

十轮独立子代理对抗审查（6 域并行：架构性能/并发完整性/安全越权/H5契约/小程序UI/业务规则对抗 → 2 轮墨菲终审：端到端灾难剧本+双端一致性 → 2 轮对抗证伪）。26 条高优先级发现逐条复核：0 条 REFUTED，严重度系统性校准（8 条 P1 仅 2 条维持），3 条加重（备份 cron 零日志、BUSY 主路径裸奔、UI 撤销守卫同缺）。核心结论：架构骨架健康（并发模型/不变量/事务纪律站住），乱麻感来自五个可批量修复的系统性模式——配置开发友好兜底无生产守卫、失败伪装空态（双端同病）、部署运维层裸奔、适老化硬指标未达标、双端漂移。F9/F10/F12/D17/D21 确认为冻结决策知情接受。产出 .trellis/tasks/10-01-adversarial-review-10rounds/REVIEW.md，含第九批修复批次建议（六组约 290 行）。

### Testing

- [OK] 证伪轮逐条打开 file:line 复核，含反证检索（compose 实际值、UI 侧守卫、login 路由行为、官方保留字文档）
- [OK] R2 附带 adversarial/transaction/appointment 测试全绿佐证；R4 npm run build、R1/R3/R6/R7 go build 通过

### Status

[OK] **Completed**

### Next Steps

- 按 REVIEW.md §7 六组批次推进第九批（安全闸最优先：compose ANMO_SMS_MODE 一行即堵住任意会员接管）
- 运维侧一次核验：systemctl is-enabled docker nginx cloudflared；yun1 backups 独立盘核实

# PHASE RESULT

**Phase**: V2.2 十轮对抗式审查（只审不改）
**Result**: 完成
**验证**: 26 条高优发现全部经证伪轮复核（0 REFUTED / 16 CONFIRMED / 10 PARTIAL）；REVIEW.md 含墨菲十剧本判定与修复批次
**提交**: 本 session（REVIEW.md + 归档）
**遗留**: 修复待第九批执行
