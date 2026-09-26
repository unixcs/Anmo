# identity 模块

## 职责
后台登录（JWT）、RBAC（D2：identity_user.role 枚举 OWNER/OPERATOR）、顾客手机号+验证码登录与 Customer Token（§10）、SMS Sender 接口（W3：dev 固定码）。

## 数据表
identity_user / identity_role / identity_permission（静态种子）。本模块不写业务表。

## 公开 API（api.go，其他模块唯一入口）
- `TokenVerifier() middleware.TokenVerifier` — 解析 admin/customer JWT 成 Principal
- `AdminLogin(phone, password) (token, error)` / `SendSMS(phone)` / `CustomerLogin(phone, code) (token, memberID, error)`
- `member.Provider` 由 main 注入：顾客首次登录创建/绑定 member（D3）

## 事务规则
无跨模块事务；member 创建/绑定复用 member 模块 API。

## 测试
登录成功/失败、token 解析、RBAC 401/403、验证码限频、member 自动创建。

## 禁止
不做动态授权管理（D2）；不做 token 刷新；不信任请求参数中的 member_id（§100）。
