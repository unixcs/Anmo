# PRD — Phase 2: Identity

## Goal

按 plan.md §111：老板可以进入后台（登录），顾客可以登录（手机号+验证码），为全系统提供两种 JWT 令牌与 RBAC 中间件。

## Requirements

1. 后台登录：POST /admin/auth/login {phone, password} → OWNER/OPERATOR JWT（含 user_id, role）。首次启动种子 OWNER（config 注入手机号/密码，bcrypt）
2. 顾客登录：POST /api/auth/sms/send {phone}（dev 模式固定码 123456，写日志，限频 60s）→ POST /api/auth/sms/verify {phone, code} → 成功则查找或创建 member（D3：绑定已有 member），签发 Customer JWT（含 member_id）
3. RBAC 中间件：admin 路由要求合法 admin JWT；/api 路由要求 customer JWT；OPERATOR 与 OWNER 同权（D2）；无权限 401/403 标准错误体
4. 刷新不做（V1），token 有效期：admin 12h，customer 7d
5. identity 种子 migration（001 已建表，本 Phase 补数据写入逻辑于运行时 seed，非 migration 硬编码密码）
6. 顾客身份解析：middleware 从 token 取 member_id 注入 context，禁止信任请求参数（§100）

## Acceptance Criteria

- [ ] 种子 OWNER 可登录后台并拿到含 role=OWNER 的 JWT
- [ ] 顾客 dev 码登录后 member 自动创建（member_no 生成），二次登录绑定同一 member
- [ ] 无 token 访问受保护路由 401；customer token 访问 admin 路由 403
- [ ] go build/vet/test 全过；identity 模块单测（登录/权限）+ 简单集成测试
