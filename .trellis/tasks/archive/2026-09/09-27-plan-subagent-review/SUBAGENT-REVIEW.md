# SUBAGENT REVIEW

审查对象：`plan.md`（§0-§140）+ 根 `AGENTS.md` + Phase 0 已冻结口径（`REVIEW.md` W1-W5）。
审查方式：对抗式只读审查，逐条核对 §132 检查清单。事实核对基线：以 plan.md 原文为准。

## 1. BLOCKER

- **B1 撤销核销事务未处理对应的 payment(CARD) 记录，撤销后重新结算必然造成收款双计**。
  依据：§57 撤销事务 = 锁卡→确认未撤销→余额+1→写 REVERSAL→写 reversal，**无 payment 处理**；§127/AGENTS.md 冻结的撤销事务同样无 payment 处理；而 §59 明确撤销后预约保持 COMPLETED 并"重新正确结算"，§48 规定核销同时产生 `payment.method=CARD`。推演：误核销→撤销（恢复次数）→改用现金重结 → 同一预约留下 payment(CARD,128) + payment(CASH,128) 两条有效收款，"收款记录"（§3.1/§80）收入翻倍；且 §136 验收清单只查"撤销核销能恢复次数"，不查 payment，该错误不会被任何既有验收拦住。payment.status 字段（§46）存在但取值从未定义。
  建议冻结口径：**payment.status ∈ {VALID, VOIDED}；撤销核销事务在同一事务内将原核销对应的 payment(CARD) 置为 VOIDED；收款记录只统计 VALID payment；一个预约最多一笔 VALID payment；V1 不提供独立"作废收款"功能（V2 再议）。**
  这是核心冻结事务规范（plan §57/§127 + AGENTS.md）的实质缺陷，必须先修订冻结口径才允许写 Transaction 模块代码，故 BLOCKED。

## 2. WARNING

- **W-A RBAC 三表无关联设计，OPERATOR 权限集合未定义**（§9/§63/§132-16）。§124 只有 identity_user / identity_role / identity_permission 三张互不关联的表，缺少 user↔role、role↔permission 的任何关联方式；OPERATOR 能做什么 §9 只说"预留"。
  冻结口径：**V1 在 identity_user 上加 role 枚举列（OWNER/OPERATOR）实现角色；identity_role/identity_permission 建表并预置静态种子数据，不实现动态授权管理；V1 内 OPERATOR 权限与 OWNER 相同，仅记录角色差异。**（保持 §124 表清单不变，符合"不做"原则。）

- **W-B 表数量口径错误：§124 实为 24 张表，Phase 0 REVIEW 写"27 表"**（§63/§124/§96）。逐条清点 §124：identity 3 + member 3 + service 2 + card 4 + appointment 3 + payment 1 + transaction 2 + content 4 + ops 2 = **24**。§96 的 001-008 迁移文件恰好覆盖这 24 张。"27" 无法追溯，会导致 migration 完整性检查基线错误。
  冻结口径：**以 §124 的 24 张表为唯一权威清单，修正 Phase 0 REVIEW 文本；migration 覆盖性检查按 24 张核对。**

- **W-C §61"预约不必须有会员"与 §10/§62 表面冲突，member_id 可空性未冻结**（§10/§21/§61/§62）。H5 强制手机号登录且首次登录创建/绑定 member，因此每个预约天然有 member；但 §61 字面可读成"允许无会员的预约"，直接影响 005 迁移中 appointment.member_id 是否 NOT NULL 及后台是否做代客下单。
  冻结口径：**appointment.member_id NOT NULL；所有预约均来自已登录 member；§61 仅指"不强制办卡"；后台 V1 不做代客下单/匿名预约。**

- **W-D card_service_rule 挂载点两处矛盾**（§43/§44/§65）。§44 明确规则挂在卡类型（card_template）上："card_type ↓ card_service_rule ↓ service"；§65 关系图却画成 member_card 之下。若实现成 member_card 级规则，每发一张卡都要复制规则。
  冻结口径：**card_service_rule.card_template_id → card_template.id（模板级规则）；核销时经 member_card.card_template_id 解析适用规则。**

- **W-E §55/§56 的幂等与唯一性方案在"撤销→再核销"循环下自相矛盾，且 MySQL 无部分索引**（§55/§56/§57/§59/§98 Case6-8）。两个具体缺陷：(1) §56 若用复合唯一 (appointment_id, status)，"核销→撤销→再核销→再撤销"会产生两行 (appointment_id, REVERSED)，第二次撤销违反唯一约束；(2) §55 固定键 `redeem:{appointment_id}` 加唯一索引后，撤销后的第二次合法核销（§59 流程）会撞唯一键无法插入。Phase 0 REVIEW 推荐的"appointment_id+status=SUCCESS 唯一索引"同样有缺陷 (1)。
  冻结口径：**redemption.status ∈ {SUCCESS, REVERSED}，撤销只置 REVERSED 不删行；用生成列 active_lock（status=SUCCESS 时 = appointment_id，否则 NULL）+ UNIQUE(active_lock) 保证"一预约最多一笔有效核销"并支持撤销后重新核销；idempotency_key 改为请求级 UUID（客户端每次生成），仅防重复提交，不再使用 redeem:{appointment_id} 格式。**

- **W-F 并发预约的锁机制未落地，Case 2 需要确定性结果**（§27/§71/§72/§98 Case2）。"事务+冲突检查+适当锁"过于含糊：空档时段无既有行可 FOR UPDATE，RR 下靠 gap lock + 死错重试不可控。同时冲突检查需排除自身（改期场景）。
  冻结口径：**预约创建/改期/确认事务内先 GET_LOCK('anmo:appointment:calendar', 超时N秒)（单店单师，日历级串行完全够用；或等价地锁一个固定锚行）再执行 §71 冲突判定，判定条件排除自身 appointment_id；锁在 COMMIT/ROLLBACK 后释放。**

- **W-G 跨模块单事务的传播机制未定义**（§53/§90/§91/§93/§127）。核销单事务要同时写 card/appointment/transaction 三个模块的表，§93 只允许"调用另一模块 api.go"，未定义事务如何跨模块传递；若各模块自开事务则违反 §53。
  冻结口径：**由 transaction 模块开启数据库事务，通过显式 Tx/Runner 参数传入 card/appointment 的 api.go（api 方法签名统一接受事务执行器）；模块内部禁止自开事务再嵌套；该约定在 Phase 1 骨架（shared 层 TxRunner）落地。**

- **W-H ops 模块存在 Go 导入环风险**（§85/§87/§89/§93/§132-17）。业务模块要写 ops_operation_log（→ops），ops 洞察/定时任务又要读 member/card/appointment（ops→业务模块），双向 api.go 调用在 Go 里就是循环导入。
  冻结口径：**后台操作日志统一由 HTTP middleware 写入（或经 EventBus，§54/§87 明确允许"非关键日志"走 EventBus），业务模块不 import ops；ops 定时任务只单向依赖各模块 api.go。**

- **W-I 状态机守卫与改期规则有未覆盖项**（§24/§25/§31/§32/§99 Case10）。未定义：改期允许的来源状态、改期是否受 2 小时限制、并发状态下迁移如何仲裁（如顾客取消 vs 老板开始服务同时发生）。
  冻结口径：**所有状态迁移一律"UPDATE ... WHERE status=期望值"并校验受影响行数；改期仅允许 PENDING_CONFIRM/CONFIRMED、沿用提前 2 小时限制、在同一 appointment 上改时间并写状态日志、冲突检查排除自身；NO_SHOW 仅限从 CONFIRMED 迁出（§25 已如此规定，落实为守卫）。**

- **W-J W1 冻结口径需补一句：非卡结算不改变预约状态**。W1 说"核销/收款 = 在 IN_SERVICE 或 COMPLETED 预约上的结算事务"，但 §53 的 COMPLETED 联动只在核销事务里；现金/微信收款（§46/§49/§50）是否改状态未写明，否则会出现"收了款但预约永远停在 IN_SERVICE"。
  冻结口径：**仅核销事务与"完成服务"动作触发 COMPLETED；现金/微信收款只写 payment，不改变预约状态。**（W1 其余部分自洽：结算状态白名单、COMPLETED 幂等、撤销后保持 COMPLETED 均与 §51/§53/§59/§99 兼容；W1 唯一的实质漏洞是 B1 的 payment 处理。）

- **W-K "续卡"语义完全未定义**（§78/§114/§136）。验收要求"续卡"，但全文未定义续卡是延长有效期、原卡加次数还是新发一张卡；直接影响 Phase 5 事务与流水类型。
  冻结口径：**V1 续卡 = 对同一 member 再发一张新卡，完全复用发卡事务与 ISSUE 流水；"调整次数" = 对现有 member_card 的 ADJUSTMENT ±N 流水；不做原卡叠加/顺期逻辑。**

## 3. RECOMMENDATION

- **R1** §35 卡模板 type=COUNT/ACTIVITY：ACTIVITY 与 COUNT 的行为差异从未定义（§36/§37 的区别其实只是有效期取值）。冻结建议：V1 一律按次卡处理，type 不产生独立业务逻辑，有效期由 validity_type/valid_from/valid_until 表达。
- **R2** 作废卡（§78/§39 CANCELLED）：仅置状态、不改 remaining_count、不写次数流水（保持"所有次数变化有流水"——状态不是次数变化）；核销校验拒绝 CANCELLED。
- **R3** §70 索引清单补唯一约束：member.phone UNIQUE、member_no UNIQUE、appointment_no UNIQUE、redemption.idempotency_key UNIQUE（§12/§55/§68 已有要求但 §70 清单遗漏）。
- **R4** 营业时间边界校验（§19/§84）：时间槽生成与提交校验必须保证 scheduled_end ≤ 营业结束时间（如 20:30+60min=21:30 越界应拒绝）。
- **R5** content_page_config 与 content_banner/content_announcement 有内容双份维护风险（§82/§83）：page_config 的 JSON block 引用素材表 id，不在 JSON 里复制正文。
- **R6** USED_UP 状态的产生机制未写明（§39）：核销事务内 remaining_count 减至 0 时同事务置 USED_UP（与 W2 的 EXPIRED 惰性+清扫机制并列）。
- **R7** 已知风险记录：顾客可同时提交多个 PENDING_CONFIRM 占住多个时段（§71 让 PENDING_CONFIRM 参与冲突），无单客待确认上限。按"不做"原则 V1 不加限制，靠老板及时确认/取消，风险已知情接受。

## 4. SCOPE CHECK

确认当前系统是：

- 个人到店按摩店 ✔（§0/§140，全文无上门/地址/距离逻辑）
- 单老板/单按摩师 ✔（§18/§20，无技师容量/派单/排班）
- 家庭成员可辅助后台（OPERATOR 预留，RBAC 支持）✔（§9，未建 TECHNICIAN/DISPATCHER/SCHEDULER）
- 无上门服务 ✔
- 无技师调度 ✔
- 无复杂排班 ✔（§20/§64 已删除 booking_slot_inventory/staff_schedule 等）
- 无多门店 ✔
- 无多租户 ✔
- 无线上支付 API / 派单 / 地图 / 积分 / 优惠券 / 库存 ✔（§131 全部列为禁止，plan 正文与 W1-W5 均未引入）

附加核对结论：

1. Plan 正文未发现 §131 禁止实体的自我矛盾引入；§45 明确不提前抽象 Entitlement，§87 EventBus 已降级为辅助机制（与 §54 一致）。
2. 会员卡与预约解耦成立：§2/§62"预约≠办卡"，普通（未办卡）登录客户可预约，核销只是结算方式之一（§48）。
3. W1-W5 与 Plan 正文无硬冲突（W4 保留枚举不实现路径、W2 惰性过期均合规）；需修正两点：表数量 27→24（W-B），W1 需补"非卡结算不改状态"（W-J）；W1 的撤销事务缺口即 B1。
4. 模块依赖除 ops 外均为单向（transaction→card/appointment、appointment→service/member），无其他循环依赖。

## 5. DECISION

BLOCKED

依据：仅 1 项阻断（B1：撤销核销事务必须补 payment 置 VOIDED 的冻结口径，否则 Phase 7 按现行冻结规范实现会产生错误的收款账目，且 §136 验收无法拦截）。按 §109 只修改阻断项：将 B1 一句话口径补入 AGENTS.md 核心事务第 3 条（撤销）后即可放行进入 Phase 1；W-A~W-K 为各 Phase 开工前需按建议口径澄清的执行决策，不要求重写 Plan。

## 复审（post-fix）

复审对象：修订后的 `AGENTS.md`（核心事务第 3 条、业务不变量节、新增冻结决策 D1-D17）。

### B1 修复验证：已完全解决

闭环推演（§48/§53/§57/§59 + 修订后 AGENTS.md）：

1. 误核销：redemption(SUCCESS, active_lock=appointment_id) + payment(CARD, VALID)。
2. 撤销事务（同一事务）：锁卡 → 确认未撤销 → 恢复次数 → card_transaction(REVERSAL) → redemption_reversal → redemption 置 REVERSED（active_lock 置 NULL）→ **payment 置 VOIDED**。
3. 现金重结：新 payment(CASH, VALID)。收款记录只统计 VALID → 仅 CASH 一笔，双计消除。
4. 撤销后改用正确卡重核销：active_lock 唯一性已随 REVERSED 释放，新 redemption(SUCCESS) 可插入；旧 CARD payment 已 VOIDED → "一预约最多一笔 VALID payment" 恒成立。

payment.status 为 §46 既有字段，未引入新实体，未改动 §124 表清单。

### D1-D17 冲突比对：无冲突

D1=B1、D2=W-A、D3=W-B+W-C、D4=W-D、D5=W-F、D6=W-G、D7=W-H、D8=W-I、D9=W-J、D10=W-K、D11=R1、D12=R2、D13=R7+W2、D14=R3、D15=R4、D16=R5、D17=R6，全部与本报告及 plan.md 正文一致或为其合法执行细化：

- §55 的 `redeem:{appointment_id}` 键格式原文为"推荐"而非硬性，被请求级 UUID 取代属 review 冻结权限内的修订。
- "一预约最多一笔 VALID payment"是对 §51/§46 的收紧（防双计的必要不变量），非冲突。
- D4 以 §44 为准解释 §65 图示歧义；D8 改期 2 小时限制是 §31 的合理延伸；D5/D6/D7 为机制口径，无 schema 变化。

### 备注（不构成 finding）

1. 核心事务第 3 条流程清单未显式列"redemption → REVERSED"步骤，但业务不变量节已明确"撤销只置 REVERSED 不删行"，两处合并阅读完整可执行。
2. D6"模块内禁止自开嵌套事务"按上下文指跨模块 Tx 传入场景禁止再嵌套；单模块事务（如发卡）仍由本模块开启，实现时按此理解。
3. AGENTS.md 末尾引用本报告路径写作 `archive/2026-09/09-27-plan-subagent-review/`，文件当前实际位于 `.trellis/tasks/09-27-plan-subagent-review/`（尚未归档）；归档后自然一致，无需处理。

### 新 BLOCKER

无。

### 复审结论

DECISION: PASS

依据：B1 已通过"撤销事务同事务置 payment=VOIDED + 收款只统计 VALID + active_lock/请求级 UUID 幂等方案"完全修复且推演自洽；D1-D17 与 plan.md 正文及本报告口径无冲突，均为"最简/不做"口径的合法冻结；无新增阻断项。允许进入 Phase 1（工程骨架，含 shared.TxRunner / D5-D7 机制落地）。
