# member 模块

## 职责
member / member_tag / member_tag_rel 的维护：后台会员管理、标签、备注；顾客端个人资料；账号面（密码体系 V2.2、微信 openid、空壳认领）；member_no 生成（M+日期+序号）。

## 数据表
member, member_tag, member_tag_rel。014 起 member.phone 可空（未设即 NULL，纯微信会员）、password_hash 可空（bcrypt，json 永不外泄）、wx_openid 可空唯一；核心业务数据禁物理删除——唯一例外：claim 事务内无业务数据的自动建号空壳（DeleteShell）。

## 公开 API（api.go）
- `Get(ctx, id) / GetByPhone(ctx, q shared.Querier, phone) / List(ctx, filter, page)`
- `EnsureByPhone(ctx, tx shared.Tx, phone, name string) (id string, created bool, err error)` — 过渡期 dev 短信登录/测试建号用
- `CreateWithPhone(ctx, tx, phone, password) (id, error)` — H5 注册建号（bcrypt；"已注册"三分支文案由 identity 在同事务先判）
- `CreateByOpenID(ctx, tx, openid) (id, error)` — 微信首登直建号（phone NULL、name ''；D25 修订）
- `SetPassword(ctx, tx, memberID, password)` — 顾客自设/重置 H5 密码（需已绑手机号 MEMBER_PHONE_REQUIRED；6~64 位）
- `AdminSetPassword(ctx, memberID, newPassword)` — admin 重置（无 phone 也可设）；路由 `PUT /admin/members/{id}/password`
- `SetPhoneOnce(ctx, tx, memberID, phone)` — 手机号一次性设置（已有 → MEMBER_PHONE_SET；撞号 → MEMBER_PHONE_TAKEN 409）
- `HasBusinessData(ctx, tx, memberID) (bool, error)` — appointment/member_card/redemption 任一存在
- `DeleteShell(ctx, tx, memberID)` — 物理删空壳（自带 HasBusinessData 守卫；先清 member_tag_rel）
- `UpdateProfile(ctx, id, ...)` / `SetTags / ListTags / UpdateRemark` — ProfileUpdate 含 `phone`（仅 SetPhoneOnce 语义）
- `TouchLastVisit(ctx, tx, memberID, at)` — 供 transaction 模块在核销后调用
- `FindByOpenID(ctx, q shared.Querier, openid) (id, ok, error)` / `BindOpenID(ctx, tx, memberID, openid) error` / `ClearOpenID(ctx, tx, memberID) error` — 微信绑定，uk_member_wx_openid 唯一兜底（claim 转绑=先 ClearOpenID 空壳再 BindOpenID 目标）

## 模型约定
`Member.PasswordHash` json:"-"（永不进响应/日志）；`HasPassword`/`WxBound` 为派生布尔。014 重建后列序变化——全代码禁 SELECT *，列清单唯一来源 `memberColumns` 常量。

## 事务规则
账号面方法（CreateWithPhone/CreateByOpenID/SetPassword/SetPhoneOnce/HasBusinessData/DeleteShell/BindOpenID/ClearOpenID/EnsureByPhone）一律接受调用方 Tx（`_txlock=immediate`，D5/D6）；AdminSetPassword 自管单语句写。

## 测试
创建/查询、member_no 唯一、phone 唯一冲突、NULL phone 并存、标签绑定去重、密码设置矩阵、SetPhoneOnce 矩阵、HasBusinessData/DeleteShell。

## 禁止
不做会员等级/积分/权益（§45）；不 import 其他模块 repo；password_hash 不得进任何 JSON 响应或日志。
