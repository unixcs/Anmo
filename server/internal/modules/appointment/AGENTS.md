# appointment 模块

## 职责
appointment / appointment_service / appointment_status_log：创建预约（冲突检查；串行化由 SQLite BEGIN IMMEDIATE 单写者模型承接，原 D5 GET_LOCK 已退役）、开始、完成、取消（顾客 2h 限制/后台无限制）、改期（D8）、爽约、编号生成（APT+日期+日序号，事务内原子计数器）。

## 数据表
appointment（member_id NOT NULL，D3）, appointment_service（snapshot：名称/时长/价格 §23）, appointment_status_log。

## 状态机（§24-25，D8 条件 UPDATE 守卫）
WAITING→IN_SERVICE→COMPLETED；WAITING→CANCELLED|NO_SHOW（2026-09-28 migration 012 收紧：创建即 WAITING，无确认环节，历史 PENDING_CONFIRM/CONFIRMED 已数据迁移）。改期仅 WAITING，改时间不改单号，冲突检查排除自身。

## 公开 API（api.go）
- 顾客：`Create(ctx, memberID, serviceID, start, note)`、`CancelByCustomer(ctx, memberID, id)`、`Reschedule(ctx, memberID, id, newStart)`、`ListMine(ctx, memberID, filter, page)`、`GetMine(ctx, memberID, id)`
- 后台：`Confirm/Start/Complete/Cancel/NoShow(ctx, id, operatorID)`、`List(ctx, filter, page)`、`Today(ctx)`（Phase 8 工作台）
- 协作：`Get(ctx, tx, id)`、`MarkCompleted(ctx, tx, id, operatorID)`（核销事务内调用）、`ActiveExists(ctx, tx, start, end, excludeID)`

## 事务规则
创建/改期：单个写事务（BEGIN IMMEDIATE 全库串行）→ 闭店/容量/冲突判定（§71 WAITING/IN_SERVICE 参与，排除自身）→ 插入/更新 → 提交。完成动作幂等（已 COMPLETED 不报错）。

## 测试
冲突判定（相邻/重叠/包含）、取消后时段释放（Case 9）、重复完成拒绝（Case 10）、改期冲突、状态守卫。

## 禁止
不做技师/排班/容量（§18/§20）；不做 buffer（V1=0 §29）；不信任前端时间——服务端校验营业时间/提前量（D15）。
