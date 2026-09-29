# Implement — V2.2 第二批

按序执行；每步验证绿了再进下一步。注意本环境 npm 用 `npm --prefix`、Go 需 `export PATH=$PATH:/usr/local/go/bin`、Bash 会话 cwd 必须保持在仓库根（.zcode hook 按 cwd 相对路径解析）。

## Step 1 — Migration 014 + member 模块
- [ ] `server/migrations/014_member_password.sql`（design §1 原样，含 PRAGMA 包裹/触发器/索引重建）
- [ ] member repo：`memberColumns` 更新；模型加 `PasswordHash`（json 不外泄）`HasPassword`/`WxBound` 派生；新增 `CreateWithPhone`/`CreateByOpenID`/`SetPassword`/`SetPhoneOnce`/`HasBusinessData`/`DeleteShell`（design §2）
- [ ] profile 更新入参扩展 `phone`（SetPhoneOnce 语义）
- [ ] admin 端 `PUT /admin/members/{id}/password` + 列表/详情 `has_password`/`wx_bound`
- [ ] 测试：repo/openid/迁移断言
- [ ] 验证：`go -C /mnt/Projects/Anmo/server test ./internal/modules/member/ -count=1`

## Step 2 — identity 模块重写
- [ ] `Register`/`Login`/`WxLogin`（直建号）/`ClaimByPhone`/`SetH5Password`；删 bind_ticket 链与 `CustomerLogin`；token.go 清理
- [ ] handler+module+api 路由：register/login/wx/claim/me/h5-password；wx login 响应去 needs_bind
- [ ] SMS：config.Mode 增 `off`（config.go 合法值校验如有）+ 两 handler off→410 `SMS_DISABLED`；config.example.yaml 注释
- [ ] 测试：identity 全矩阵（design §4）
- [ ] 验证：`go -C /mnt/Projects/Anmo/server test ./internal/modules/identity/ -count=1`

## Step 3 — e2e/其余测试改造
- [ ] grep 全仓 `sms/123456/CustomerLogin` 引用，e2e 建会员改 register
- [ ] 验证：`go -C /mnt/Projects/Anmo/server build ./... && go -C /mnt/Projects/Anmo/server vet ./... && go -C /mnt/Projects/Anmo/server test ./...`（全绿 0 跳过）

## Step 4 — H5
- [ ] endpoints.ts + LoginPage.vue 重写（登录/注册切换、错误文案、忘记密码引导）
- [ ] 验证：`npm --prefix /mnt/Projects/Anmo/apps/customer run build`

## Step 5 — weapp
- [ ] api.js/auth.js/app.js 清理（bindTicket 移除、新端点）
- [ ] login 页重写（一键微信登录+重试）；me 页手机号完善 + 撞号 claim 密码弹层 + 「设置 H5 密码」cell
- [ ] 验证：`node --check` 所有改动 .js

## Step 6 — admin
- [ ] MembersPage 重置密码对话框 + has_password/wx_bound 徽标
- [ ] 验证：`npm --prefix /mnt/Projects/Anmo/apps/admin run build`

## Step 7 — AGENTS.md 决策修订 + runbook 提示
- [ ] AGENTS.md：D25 修订、新增 D27（design §6 措辞）
- [ ] 部署 runbook 位置如存在（.trellis 归档 P4），追加"sms.mode: off"要求；否则记入任务报告由主会话在部署阶段落实

## Step 8 — 主会话执行（Murphy live 验收 + 提交），子代理勿做
