# PRD — Phase 3: Member

## Goal

plan.md §112：能新增会员、能查询会员。补齐 member 模块的后台管理与顾客端个人资料（EnsureByPhone 已于 Phase 2 落地）。

## Requirements

1. 后台：会员列表（姓名/手机号筛选 + 分页）、新增（手机号唯一，member_no 生成）、详情、编辑资料（name/gender/birthday/remark）、标签（列表/设置/新建）
2. 顾客端：查看/编辑自己的资料（GET/PUT /api/me/profile，member_id 取自 token，§100）
3. 软删除不做（V1 member 状态仅 ACTIVE；无删除功能，符合"不做"）
4. birthday 用 DATE；gender 仅 男/女/空

## Acceptance Criteria

- [ ] 后台可创建/查询/编辑会员与标签，手机号重复返回 409
- [ ] 顾客只能读写自己的资料（token member_id）
- [ ] go build/vet/test 全过，member 模块单测覆盖
