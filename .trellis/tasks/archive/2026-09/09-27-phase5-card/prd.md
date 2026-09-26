# PRD — Phase 5: Card

## Goal

plan.md §114：发 10 次卡看到 10，并产生 ISSUE +10 流水。补齐卡模板、卡规则、会员卡、流水。

## Requirements

1. 卡模板 CRUD（name/type/total_count/validity_type/valid_from/valid_until/price/status，D11）
2. card_service_rule：模板级设置可核销服务列表（D4）
3. 发卡 IssueCard：单事务 member_card + card_transaction(ISSUE +N)（§127）；D10 续卡=再发新卡复用同一事务
4. 调整次数 Adjust：单事务 remaining ±N + ADJUSTMENT 流水（D8 无状态迁移）
5. 作废 Cancel：仅置 CANCELLED，不改次数不写流水（D12）
6. 流水查询、会员卡列表
7. 核销协作 API（供 Phase 7）：LockForRedeem / ApplyRedeem / ApplyReversal / UsableCards
8. 有效期：valid_until 为空=永久；D13 核销时惰性校验

## Acceptance Criteria

- [ ] 发 10 次卡后 remaining=10 且有 ISSUE +10 流水
- [ ] 余额不足/过期/不适用服务在 ApplyRedeem 校验拒绝
- [ ] 调整/作废/流水可查
- [ ] go build/vet/test 全过
