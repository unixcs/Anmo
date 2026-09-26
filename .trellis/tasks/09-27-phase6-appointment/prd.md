# PRD — Phase 6: Appointment

## Goal

plan.md §115：appointment/appointment_service/appointment_status_log + 时间冲突/确认/取消/改期/爽约。验收：两个客户不能预约同一服务时间。

## Requirements

1. 顾客创建预约：选服务+时间 → snapshot 写 appointment_service；D5 事务内 GET_LOCK('anmo:appointment:calendar') + §71 冲突判定；营业时间/提前量校验（D15/§30）
2. 预约号 APT+yyyymmdd+4位日序列（GET_LOCK 生成）
3. 状态机（D8 条件 UPDATE）：confirm(PENDING→CONFIRMED)、start(CONFIRMED→IN_SERVICE)、complete(IN_SERVICE→COMPLETED 幂等)、cancel(顾客 2h 限制；后台无限制)、no-show(CONFIRMED→NO_SHOW)、reschedule(仅 PENDING/CONFIRMED，改时间+日志+冲突排除自身)
4. 全部状态变化写 appointment_status_log
5. 查询：顾客 ListMine/GetMine；后台 List（日期/状态筛选）
6. 时间槽计算：duration 由 snapshot 来，end=start+duration

## Acceptance Criteria

- [ ] 重叠预约被拒（409 APPOINTMENT_CONFLICT），相邻不拒
- [ ] 取消后时段释放（Case 9）；已完成不能重复完成（Case 10）
- [ ] 并发抢同一时段只有一个成功（Case 2，E2E 复验）
- [ ] go build/vet/test 全过
