# PRD — Phase 7: Transaction

## Goal

plan.md §116 核心阶段：payment/redemption/redemption_reversal，数据库事务+并发。

## Requirements

1. SettleByCard 核心核销事务（§53/D6）：锁卡→校验→扣次→REDEEM 流水→redemption→payment(CARD)→appointment COMPLETED→状态日志，任一步 ROLLBACK
2. 幂等（W-E）：idempotency_key 请求级 UUID 唯一；重放同 key 返回原结果；active_lock 保证一预约一有效核销
3. SettleByPay：现金/微信/其他收款，仅写 payment(VALID)，不改预约状态（D9）；一预约最多一笔 VALID payment（D1）
4. ReverseRedemption（§57/D1）：锁卡→确认 SUCCESS→置 REVERSED→恢复次数→REVERSAL 流水→写 reversal→原 payment(CARD) 置 VOIDED，单事务；重复撤销失败
5. 撤销后预约保持 COMPLETED，可重新结算
6. 收款/核销列表查询
7. 并发必测（§98 Case1）：余额 1 两次并发核销 1 成功 1 失败

## Acceptance Criteria

- [ ] 正常核销全链路（卡余额/流水/核销记录/收款/预约完成）
- [ ] Case 1/3/4/5/6/7/8 全过（模块测试），Case 2/9/10 已在 Phase 6
- [ ] 撤销后现金重结不双计（D1）
- [ ] go build/vet/test 全过
