# V2 微信小程序（原生 + UniApp，共用同一 Go 后端）

> 上位文档：根 AGENTS.md、plan.md §11（微信登录→绑定手机号→同一个 member_id）、§103（顾客端统一 uni-app，不要维护两套业务代码）、§122（V2 候选）。目标文档 §75：Phase C 环境准备 → Phase D 原生小程序 → Phase E UniApp。

## 核心原则

**一套后端，多个前端。** 小程序与 H5 调用同一批 `/api/*` 端点、同一套业务不变量；后端是唯一事实源。

## Phase C：环境准备（后端）

### 微信登录链路（plan §11）

```
wx.login(code)
  → POST /api/auth/wx/login {code}
    → code2session（AppID/Secret 已配置时）
    → member.wx_openid 命中 → 直接发 Customer Token（与 H5 同一 member）
    → 未命中 → {needs_bind: true, bind_ticket}（10 分钟 JWT，act=WXBIND）
      → 顾客走既有短信登录（POST /api/auth/sms/verify，零改动）
      → POST /api/auth/wx/bind {bind_ticket}（顾客 Token）→ 绑定 openid
```

### 决策

| # | 决策 |
|---|------|
| D23 | member 新增 `wx_openid`（NULL 可重复 + UNIQUE）；一个 openid 只绑一个 member，一 member 只一个 openid。不建独立绑定表（Plan 未提及实体）。 |
| D24 | code2session 凭据 `wx.app_id` / `wx.secret`（env `ANMO_WX_APPID`/`ANMO_WX_SECRET`）。**缺省时 dev 兜底 openid=`dev:<code>`**（与 SMS.Mode=dev 固定码 123456 同级的开发姿态，启动日志警示）；小程序正式发布前必须配置真实凭据。 |
| D25 | 未绑定时发 **bind_ticket**（JWT act=WXBIND、10 分钟），不直接落库 openid；绑定动作必须持顾客 Token 完成（防 openid 换 member 探测）。 |
| D26 | 微信官方"服务卡片"能力（类目/资质/微信后台配置）**不做**；分享闭环 = 每页 onShareAppMessage + 首页/关于 onShareTimeline + showShareMenu。 |

### 交付物

- migration `013_member_wx_openid.sql`（member 加列 + UNIQUE）
- member 模块：`FindByOpenID` / `BindOpenID`（api.go 公开）
- identity 模块：`WxLogin` / `WxBind` + handler + 路由（login 公开、bind 顾客鉴权）
- config：Wx 段 + env
- 测试：code2session httptest 桩（APIBase 可注入）、dev 兜底、绑定冲突、bind_ticket 过期

## Phase D：原生微信小程序 `apps/weapp`

- 页面：home（门店状态/首页文案/服务/门店卡片）/ booking（上下午必选+可选具体时间，V1.x 口径）/ me（个人页）/ appointments（我的预约+取消+改期）/ cards（卡+明细）/ qrcode（ANMO-MEMBER 卡码 + ANMO-APT 预约单码）/ login（wx.login + 短信绑定）/ about（门店信息）
- tabBar：首页/预约/我的（纯文本，不引入图片资产）
- QR：vendor `qrcode-generator`（纯 JS、无依赖）+ canvas 2d 绘制
- 分享：onShareAppMessage 全业务页 + onShareTimeline（home/about）+ showShareMenu
- 网络：utils/request.js 统一 base URL + Token；错误码人话提示
- 验证（本机无微信开发者工具）：JS `node --check`、JSON parse、路由/协议与后端端点一致性核对表

## Phase E：UniApp 统一顾客端 `apps/uniapp`

- 同一批业务页面（Vue3 + setup），`uni.` API + 条件编译（MP-WEIXIN 走 wx 登录，H5 走短信登录）
- 双端编译验证：`npm run build:h5` + `npm run build:mp-weixin`（本机可真实编译）
- 定位：**统一业务代码基座**（plan §103"不要维护两套业务代码"）；本轮不替换线上 H5（歧义选不做，Next 记录）

## 不做（硬边界）

微信支付 API、微信通知（订阅消息）、服务卡片官方能力、微信客服、小程序直播、云开发、独立后端/独立库、staff/多租户等 V1 禁止清单全部继承。

## 验收清单

- [ ] go build/vet/test 全绿（含 wx 新测试）
- [ ] `POST /api/auth/wx/login`：绑定会员直发 token；未绑定返回 needs_bind+bind_ticket；bind 成功后二次 login 直发 token
- [ ] `apps/weapp` 页面/路由与后端端点对照表全绿；JS/JSON 语法校验过
- [ ] `apps/uniapp` build:h5 + build:mp-weixin 双产物成功
- [ ] GitHub 提交 + yun1 后端同步（migration 013 + wx 端点）+ Tencent 同步
