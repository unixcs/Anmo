# ops 模块

## 职责
ops_operation_log 查询（写入由 middleware 钩子完成，D7）、三项洞察（低余额 remaining<=2 / 沉睡 60 天 / 临期 7 天，§86）、每日定时任务（EXPIRED 状态 sweep，W2/D13；洞察快照）。

## 数据表
ops_operation_log, ops_insight_snapshot。**只读**其他模块数据——通过各模块 api.go（如 card.ListByMember、appointment.List），不直接写业务表。

## 公开 API（api.go）
- `Logs(ctx, filter, page)`（后台）
- `Insights(ctx)` — 实时三列表；`SnapshotDaily(ctx)` — 定时任务落库
- `WriteLog(ctx, entry)` — 供 middleware 操作日志钩子调用（业务模块不 import 本模块，D7）

## 事务规则
定时任务与快照为独立短事务；洞察只读。

## 测试
日志写入/查询、三个洞察的边界（=2、=60 天、=7 天）、快照幂等（UNIQUE date+kind+member）。

## 禁止
业务模块禁止 import ops（D7）；不做 AI 推荐/复杂分析（§86）。
