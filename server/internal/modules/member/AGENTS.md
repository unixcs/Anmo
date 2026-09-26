# member 模块

## 职责
member / member_tag / member_tag_rel 的维护：后台会员管理、标签、备注；顾客端个人资料；`EnsureByPhone`（登录时创建/绑定）；member_no 生成（MBR+日期+序号）。

## 数据表
member, member_tag, member_tag_rel。member.phone 唯一；核心数据禁物理删除（status 软删）。

## 公开 API（api.go）
- `Get(ctx, id) / List(ctx, filter, page)`
- `EnsureByPhone(ctx, tx shared.Tx, phone, name string) (id string, created bool, err error)`
- `UpdateProfile(ctx, id, ...)` / `SetTags / ListTags / UpdateRemark`
- `TouchLastVisit(ctx, tx, memberID, at)` — 供 transaction 模块在核销后调用

## 事务规则
EnsureByPhone 接受外部 Tx（顾客登录事务内创建 member）；本模块不自行开启跨模块事务。

## 测试
创建/查询、member_no 唯一、phone 唯一冲突、标签绑定去重。

## 禁止
不做会员等级/积分/权益（§45）；不 import 其他模块 repo。
