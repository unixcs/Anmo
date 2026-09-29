# Design — V2.2 第二批（密码登录体系）

## 0. 现状事实（已核实）

- admin 密码 = bcrypt（`identity/admin.go`，`golang.org/x/crypto/bcrypt`，DefaultCost）；`identity_user` 表不在本批范围。
- `member` 表（migration 002）：`phone VARCHAR(20) NOT NULL + UNIQUE(phone)`、有 `trg_member_updated_at` 触发器、`idx_member_status_created` 索引；013 已 `ALTER ADD wx_openid VARCHAR(64) NULL` + `uk_member_wx_openid`。
- 引用 member 的 FK：`member_tag_rel`（002）、`appointment`（005）——SQLite 重建表需 `PRAGMA foreign_keys=OFF` 包裹。
- `member.EnsureByPhone`（首登建号）、`FindByOpenID`、`BindOpenID`（openid.go，DupKey→WX_OPENID_BOUND）现成。
- wx 登录（wx.go）：unbound → `signBindTicket`；bind 端点要求顾客 JWT（D25）。本批废除该流程。
- SMS 面：`sms.go`（verify/send，dev 固定 123456）、`customer.go CustomerLogin`、handler `handleSMSSend/Verify`、module/api 路由、`config.SMS`、router 注册；模块测试 4 处 + e2e 4 处引用。H5 `LoginPage.vue` 短信表单；weapp `pages/login`（短信表单+bindTicket 链路，`utils/auth.js wxSilentLogin`、`app.js globalData.bindTicket/session`）。
- 顾客 profile 更新端点现仅收 `name/gender/birthday`（H5 endpoints.ts `updateProfile`、weapp me 页）。

## 1. Migration 014（member 重建：phone 可空 + password_hash）

`server/migrations/014_member_password.sql`（只增不改）：

```sql
-- 014: 密码体系（plan §二）：phone 可空（纯微信会员）+ password_hash（bcrypt，NULL=未设置）。
-- SQLite 不支持 ALTER COLUMN，重建表；UNIQUE(phone) 对 NULL 天然放行多个。
PRAGMA foreign_keys = OFF;
CREATE TABLE member_new (
  id            CHAR(26)     NOT NULL PRIMARY KEY,
  member_no     VARCHAR(20)  NOT NULL,
  name          VARCHAR(50)  NOT NULL DEFAULT '',
  phone         VARCHAR(20)  NULL,
  gender        VARCHAR(10)  NOT NULL DEFAULT '',
  birthday      DATE         NULL,
  status        VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
  remark        VARCHAR(500) NOT NULL DEFAULT '',
  last_visit_at DATETIME     NULL,
  password_hash VARCHAR(100) NULL,
  wx_openid     VARCHAR(64)  NULL,
  created_at    DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at    DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_member_phone UNIQUE (phone),
  CONSTRAINT uk_member_no UNIQUE (member_no)
);
INSERT INTO member_new (id, member_no, name, phone, gender, birthday, status, remark,
                        last_visit_at, password_hash, wx_openid, created_at, updated_at)
  SELECT id, member_no, name, phone, gender, birthday, status, remark,
         last_visit_at, NULL, wx_openid, created_at, updated_at FROM member;
DROP TABLE member;
ALTER TABLE member_new RENAME TO member;
CREATE INDEX idx_member_status_created ON member (status, created_at);
CREATE UNIQUE INDEX uk_member_wx_openid ON member (wx_openid);
CREATE TRIGGER trg_member_updated_at AFTER UPDATE ON member
  FOR EACH ROW WHEN NEW.updated_at = OLD.updated_at
  BEGIN UPDATE member SET updated_at = datetime('now','+8 hours') WHERE id = NEW.id; END;
PRAGMA foreign_keys = ON;
```

注意：列序变了（password_hash 插在 remark 后）——**全代码禁用 `SELECT *`/列序依赖**；`memberColumns` 常量按名更新（repo.go）。uk_member_wx_openid 由 013 的 ALTER 改为建表内唯一索引（迁移后旧索引随表 DROP，需重建）。

## 2. 后端 API（identity + member 模块）

### member 模块
- `CreateWithPhone(ctx, tx, phone, password)`：注册建号（phone NOT NULL 校验 + bcrypt）。
- `SetPassword(ctx, tx, memberID, password)`：bcrypt 落库；要求 `phone` 非空（MEMBER_PHONE_REQUIRED）。
- `SetPhoneOnce(ctx, tx, memberID, phone)`：当前 phone 必须为空（MEMBER_PHONE_SET），撞号 → `MEMBER_PHONE_TAKEN` 409（前端据此走 claim）。
- `HasBusinessData(ctx, tx, memberID)`：EXISTS appointment/member_card/redemption 任一 → bool（claim 保护用）。
- `DeleteShell(ctx, tx, memberID)`：物理 DELETE，仅 claim 事务内空壳使用。
- profile 更新入参扩展 `phone`（仅 SetPhoneOnce 语义，管理员端不开放改手机号）。
- admin：`PUT /admin/members/{id}/password` {new_password}（6~64，bcrypt；无 phone 也可设但登录需要 phone——允许，仅存储）；列表/详情响应加 `has_password`、`wx_bound` 布尔。

### identity 模块（customer.go 重写）
- `Register(ctx, phone, password)`：11 位手机号 + 6~64 密码。事务内查 phone：不存在 → CreateWithPhone → 签 token；存在 → 有密码 → 409 `AUTH_PHONE_EXISTS`"该手机号已注册，请直接登录"；无密码且 wx_openid 非空 → 409 `AUTH_PHONE_ON_WECHAT`"该手机号已开通微信账号，请前往小程序「我的」设置 H5 密码后登录"；无密码未绑微信 → 409 `AUTH_PHONE_EXISTS_NO_PWD`"该手机号已存在，请联系商家重置密码后登录"。
- `Login(ctx, phone, password)`：不存在 → 401 `AUTH_UNREGISTERED`"该手机号尚未注册"；password_hash 空 → 401 `AUTH_NO_PASSWORD`"该账号尚未设置密码，请联系商家重置；已使用微信小程序可在「我的」设置 H5 密码"；bcrypt 失败 → 401 `AUTH_BAD_PASSWORD`"密码错误，请重新输入"；成功签 token。
- `WxLogin` 改造：unbound 分支 → 事务内 `CreateByOpenID`（新 member：wx_openid、phone NULL、name ''、走 nextMemberNo）→ 直接签 token 返回（响应字段 `needs_bind`/`bind_ticket` 移除）。`parseBindTicket`/`signBindTicket`/`WxBind`/`handleWxBind`/bind 路由删除（token.go 相应清理）。
- `ClaimByPhone(ctx, currentMemberID, phone, password)`（`POST /api/auth/wx/claim`，顾客 JWT）：
  1. 查目标 member（by phone）：不存在 → 404 `AUTH_CLAIM_NO_ACCOUNT`（正常走 SetPhoneOnce，不该进 claim）；
  2. 目标 password_hash 空 → 401"该账号尚未设置密码，请联系商家重置"；
  3. bcrypt 校验失败 → 401"密码错误，请重新输入"；
  4. 目标 wx_openid 非空 → 409 `WX_OPENID_BOUND`"该手机号已绑定其他微信账号，请联系商家处理"；
  5. 当前 member `HasBusinessData` → 409 `AUTH_CLAIM_HAS_DATA`"当前账号已有预约/卡记录，请联系商家处理"；
  6. 同事务：目标 member.wx_openid = openid（DupKey 兜底同 409）、当前空壳 DELETE、返回目标 member 新 token。
  - openid 从当前 member 行取（登录态即 openid 持有者）。
- `SetH5Password(ctx, memberID, newPassword)`（`PUT /api/me/h5-password`）：6~64；要求 member.phone 非空（MEMBER_PHONE_REQUIRED"请先完善手机号"）；幂等覆盖（设置=重置，plan §二.5 同路径）。
- SMS：`config.SMS.Mode` 合法值增 `off`；`handleSMSSend/Verify` 在 off → 410 `SMS_DISABLED`"短信登录已下线，请升级小程序至最新版本"；dev 保持。config.example.yaml 注释更新（生产应设 off）。
- 路由：`POST /api/auth/register`、`POST /api/auth/login`、`POST /api/auth/wx/claim`（公开，claim 内部用 JWT——放顾客鉴权组）、`PUT /api/me/h5-password`（顾客鉴权）；删 `/api/auth/sms/*` 注册与否按 R6（保留但 off 拒绝；dev 可用）。wx login 响应去掉 needs_bind 字段。

**并发/不变量**：所有建号/转绑在 `_txlock=immediate` 事务内；openid 唯一由 uk 兜底映射 409；手机号唯一由 uk 兜底。

## 3. 前端

### H5（apps/customer）
- `endpoints.ts`：删 `sendSms/verifySms`；增 `register`、`login`（POST auth/register|login）。
- `LoginPage.vue`：手机号+密码（登录）⇄ 注册（手机号+密码+确认密码）切换；错误码→文案直接用服务端 msg；"忘记密码？"→ toast"请联系商家后台重置密码"。删除倒计时/验证码逻辑。
- `session.ts` 登录态逻辑复用（signIn(token, member_id)）。

### weapp（apps/weapp）
- `utils/api.js`：endpoint 同步（sms 删、register/login/claim/h5Password 增）；`utils/auth.js`：`wxSilentLogin` 去掉 bindTicket 分支（token 直存）；`app.js`：删 `globalData.bindTicket`。
- `pages/login`：重写为一键微信登录页（logo + 微信登录按钮/自动尝试 + 失败重试），无表单。
- `pages/me`：资料编辑区增加手机号输入（仅当当前 phone 为空显示）；保存 → `SetPhoneOnce`；409 `MEMBER_PHONE_TAKEN` → 弹密码框（半屏复用第一批 sheet 模式）→ `POST /api/auth/wx/claim` → 成功后以返回 token 重置本地登录态并刷新页面；新增「设置 H5 密码」cell（无手机号时引导完善）→ 密码输入弹层 → `PUT /api/me/h5-password`。
- 首登弹窗/进度条（第一批）无缝生效（新号 pct=0）。

### admin（apps/admin）
- `MembersPage.vue`（现有会员页）：行操作加「重置密码」对话框（新密码输入，≥6）；会员详情/列表展示 `has_password`/`wx_bound` 徽标。

## 4. 测试

- identity：register/login 矩阵（5 类提示）、wx 首登幂等（同 openid 两登同 member；并发由事务+唯一索引兜底）、claim 全矩阵（无账号/无密码/密码错/已绑微信/有业务数据/成功转绑+空壳删除+token 切换）、h5-password（无 phone 拒、设置后 H5 可登）、sms off/dev 两态。
- member：SetPhoneOnce（空→成功、已有→拒、撞号 409）、SetPassword、HasBusinessData、CreateByOpenID member_no 生成。
- 迁移：现有测试库自动应用 014；补一条"旧数据无损"断言（建库后插 member+w x_openid → 迁移后仍在）。若 testsupport 建库即应用全迁移，则以"phone NULL 会员可插两条"断言替代。
- e2e 中 SMS 建会员处改 register。

## 5. 兼容与部署顺序

- 生产 `config.yaml` 必须设 `sms.mode: off`（runbook 记录）；未设时沿用 dev=危险，runbook 强调。
- 旧版小程序（线上）在服务端部署后：已绑微信用户不受影响；未绑定新用户会收到"短信登录已下线，请升级小程序"提示（send/verify 410）——可接受，审核通过后消失。
- 响应字段 `needs_bind/bind_ticket` 移除属于破坏性变更，仅旧版小程序消费且仅 unbound 场景 → 同上一条覆盖。
- 回滚：revert 代码 + 014 不回滚（新列 NULL 默认无害，旧代码不读 password_hash 可继续跑——phone NULL 旧行为 NOT NULL 约束已不存在，兼容）。

## 6. AGENTS.md 冻结决策修订（随本批提交）

- D25 修订：bind_ticket 流程废除 → 微信首登直建号（openid 唯一兜底）+ `POST /api/auth/wx/claim` 手机号+密码认领转绑（空壳无业务数据才可删）。
- 新增 D27：顾客密码体系——bcrypt、H5 register/login、小程序设/重置 H5 密码（微信身份即凭证、需已绑手机号）、admin 重置；SMS 端点仅 dev 可用、生产 off；member.phone 可空（仅微信会员）。
