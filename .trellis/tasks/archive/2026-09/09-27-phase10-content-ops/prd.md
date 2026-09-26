# PRD — Phase 10: Content + Ops

## Goal

plan.md §119：首页配置/Banner/公告/操作日志/三项洞察。

## Requirements

1. content：GET /api/home（blocks+ACTIVE banner/公告，D16 引用 id）、GET /api/settings（营业时间等公开项）；后台 page config 读写、banner/公告 CRUD（最简）、settings 读写
2. ops：操作日志由 middleware 写入（D7，admin 非 GET 请求记录 action/path/状态），GET /admin/logs 查询
3. 洞察（§86）：低余额 remaining<=2、沉睡 60 天、临期 7 天；实现走 card/member api（D7 单向）
4. 定时任务入口：card.SweepExpired（EXPIRED sweep，W2/D13）+ insights 快照落库；POST /admin/ops/daily 触发（生产由 cron 调）
5. 洞察边界测试（=2、=60、=7）

## Acceptance Criteria

- [ ] /api/home 与 /api/settings 可用；后台可配置
- [ ] admin 写操作产生 ops_operation_log 且可查
- [ ] 三项洞察边界正确；sweep 过期卡生效
- [ ] go build/vet/test 全过
