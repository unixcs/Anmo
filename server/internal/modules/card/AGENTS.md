# card 模块

## 职责
card_template / card_service_rule / member_card / card_transaction：模板 CRUD、发卡（D10 续卡=再发新卡）、调整次数（ADJUSTMENT）、作废（D12 仅状态）、流水查询、卡余额/有效期/适用服务校验。

## 数据表
card_template, card_service_rule（D4 挂模板级）, member_card, card_transaction。remaining_count 是缓存，card_transaction 是真相（§125）。

## 公开 API（api.go）
- 模板：TemplateCRUD、SetServiceRules
- 顾客卡：`IssueCard(ctx, tx, memberID, templateID, operatorID)`、`Adjust(ctx, tx, cardID, delta, remark, operatorID)`、`Cancel(ctx, tx, cardID, operatorID)`
- 核销协作：`LockForRedeem(ctx, tx, cardID)`（SELECT ... FOR UPDATE）、`ApplyRedeem(ctx, tx, cardID, quantity, ref)`（扣次+REDEEM 流水，返回 before/after）、`ApplyReversal(ctx, tx, cardID, quantity, ref)`（恢复+REVERSAL 流水）
- 查询：`ListByMember / Transactions(cardID) / UsableCards(ctx, memberID, serviceID)`

## 事务规则
IssueCard/Adjust/Cancel/核销扣次/撤销恢复 全部在调用方事务内执行（D6）；任何次数变化必须同事务写 card_transaction；核销时校验 status=ACTIVE（D13 USED_UP/EXPIRED 拒绝）、valid_until、card_service_rule（D4）、remaining_count >= quantity。

## 测试
发卡 ISSUE 流水、余额不足拒绝、过期拒绝、服务不适用拒绝、调整/作废、并发扣次（Phase 7 补齐）。

## 禁止
禁止异步扣卡（§54）；不做积分/优惠券/折扣（§45）；不做原卡叠加续期（D10）。
