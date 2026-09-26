# REVIEW

审查对象：`plan.md`（个人到店按摩店服务管理系统 V1｜最终 Codex Review + 执行版，§0-§140）
审查方式：只读审查，逐维度核对 §132 检查清单。

## 1. BLOCKER

无。

## 2. WARNING

- **W1 完成/结算的状态耦合**：§53 把 `appointment=COMPLETED` 放进核销事务，而 §74/§120 中"完成服务"与"核销/收款"是两个动作。执行口径（冻结）：**完成服务 = IN_SERVICE→COMPLETED 的独立状态动作；核销/收款 = 在 IN_SERVICE 或 COMPLETED 预约上的结算事务**。核销事务内的 COMPLETED 更新对已 COMPLETED 预约幂等；撤销核销后预约保持 COMPLETED（§59），可重新结算其他方式。"已完成预约不能重复完成"仅约束完成动作本身（§99 Case 10）。
- **W2 卡过期判定**：EXPIRED 状态需有产生机制。执行口径：读取/核销时惰性校验有效期为准（硬约束），ops 定时任务每日 sweep 同步状态位（软同步，仅展示）。
- **W3 短信验证码通道**：V1 无真实短信服务商。执行口径：定义 `SmsSender` 接口，dev/prod 配置注入；dev 模式验证码固定并写日志，登录逻辑与通道解耦。
- **W4 card_transaction.REFUND**（§40）：V1 无退款流程，枚举保留、不实现对应事务路径。
- **W5 appointment_no 日序列**（§68 `APT202609280001`）：每日序列需并发安全（事务内对日计数行加锁生成），实现于 Phase 6。

## 3. RECOMMENDATION

- 冲突判定与核销幂等建议在数据库层加约束兜底（唯一索引 + 事务 + 行锁），不只依赖应用判断：appointment 冲突用事务内 `SELECT ... FOR UPDATE` 串行化同一时段检查；redemption 用 `idempotency_key` 唯一索引 + `appointment_id+status=SUCCESS` 唯一索引。
- 所有金额/次数字段统一有符号整数，应用层与 `CHECK` 双重保证 `remaining_count >= 0`（MySQL 8.4 已支持 CHECK）。
- customer 端所有"我的"查询一律从 token 推导 member_id，路由参数中的 member_id 仅后台使用。

## 4. SCOPE CHECK

确认当前系统是：

- 个人到店按摩店 ✔
- 单老板/单按摩师 ✔
- 家庭成员可辅助后台（OPERATOR 预留，RBAC 支持）✔
- 无上门服务 ✔
- 无技师调度 ✔
- 无复杂排班 ✔
- 无多门店 ✔
- 无多租户 ✔
- 无线上支付 API / 派单 / 地图 / 积分 / 优惠券 / 库存 ✔（§131 全部列为禁止）

## 5. DECISION

PASS

依据：核心闭环（预约→确认→服务→核销/收款→记录）完整自洽；数据库 27 表边界清晰（§124）；状态机与不变量（remaining_count ≥ 0、一预约一有效核销、一时段一有效服务）明确可测；核销/发卡/撤销事务边界（§127）完整；并发（§98）与权限（§100）有明确要求与必测案例。W1-W5 均为执行口径澄清，不构成阻断，已写入本 REVIEW 作为后续 Phase 的冻结决策。
