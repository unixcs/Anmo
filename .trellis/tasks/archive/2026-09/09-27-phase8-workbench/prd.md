# PRD — Phase 8: 今日工作台

## Goal

plan.md §117：今日预约汇总（待确认/待服务/服务中/已完成）+ 卡片（客户/服务/时长/价格/状态/收款）+ 快速操作（确认/开始/完成/核销/收款/取消/爽约）。

## Requirements

1. 快速操作端点已在 Phase 6/7 交付（confirm/start/complete/redeem/payments/cancel/no-show）
2. 新增 GET /admin/workbench?date=：汇总 + 卡片 enrich（会员姓名、收款状态 VALID payment）
3. 汇总放 transaction 模块（单向依赖 appointment/member，避免环，D6/D7）
4. §75 原则：一屏直接看到"现在要做什么"

## Acceptance Criteria

- [ ] workbench 返回 summary（6 态计数）+ 卡片（姓名/服务/价格/收款）
- [ ] go build/vet/test 全过
