# PRD — Phase 4: Service

## Goal

plan.md §113：后台创建"肩颈按摩 60分钟 128元"并可上下架排序；顾客端能看到。

## Requirements

1. 分类 CRUD（name/sort/status）
2. 服务项目 CRUD（category_id/name/description/duration_minutes/default_price(整数分)/cover_image/status/sort），停用启用
3. 顾客端 ListActive：仅 ACTIVE 服务 + ACTIVE 分类；按 sort 排序
4. 校验：duration>0、price>=0

## Acceptance Criteria

- [ ] 后台创建 肩颈按摩/60分钟/12800 分 成功
- [ ] 停用后顾客端不可见
- [ ] go build/vet/test 全过，service 模块测试覆盖
