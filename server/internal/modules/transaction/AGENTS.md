# transaction 模块

## 职责
服务完成后的结算：payment（CARD/WECHAT_TRANSFER/CASH/OTHER）、redemption（核销核心事务 §53）、redemption_reversal（撤销 §57）。收款列表。

## 数据表
payment（status VALID/VOIDED，D1）、redemption（SUCCESS/REVERSED + active_lock 生成列唯一，W-E）、redemption_reversal（UNIQUE redemption_id）。

## 公开 API（api.go）
- `SettleByCard(ctx, appointmentID, cardID, operatorID, idemKey)` — 核心单事务：锁卡→校验（状态/有效期/服务规则/余额）→扣次→REDEEM 流水→redemption→payment(CARD)→appointment COMPLETED→状态日志→任一步 ROLLBACK
- `SettleByPay(ctx, appointmentID, method, amount, refNo, remark, operatorID, idemKey)` — 单事务：写 payment(VALID)（不改预约状态，D9）；appointment 需在 IN_SERVICE/COMPLETED
- `ReverseRedemption(ctx, redemptionID, reason, operatorID)` — 单事务：锁卡→确认 redemption SUCCESS→置 REVERSED→恢复次数→REVERSAL 流水→写 reversal→原 payment(CARD) 置 VOIDED（D1）
- `ListPayments(ctx, filter, page)` / `ListRedemptions(ctx, filter, page)`

## 事务规则
本模块是唯一开启跨模块事务的地方（D6）：shared.RunInTx 开事务，显式 Tx 传入 card/appointment 的 api.go；幂等：idempotency_key 请求级 UUID 唯一约束，重放返回原结果（W-E）。

## 测试
正常核销全链路、余额并发（1 次余额两并发核销 1 成功 1 失败，Case 1）、零余额（Case 3）、过期（Case 4）、不适用服务（Case 5）、重复提交幂等（Case 6）、撤销恢复（Case 7）、重复撤销失败（Case 8）、撤销后收款不双计（D1）。

## 禁止
禁止 EventBus 扣卡（§54）；禁止 DELETE 任何核销/流水记录（§58）；禁止写其他模块的表（核销内 appointment 更新必须走 appointment.api.go）。
