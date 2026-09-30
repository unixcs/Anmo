# PRD — V2.2 第五批：游客浏览模式 + Cloudflare Tunnel 域名接入

> 用户诉求：①未登录用户可点首页「查看项目」「预约」浏览服务与时段，只在真正提交预约时才提示登录（第一性原理：登录是手段不是目的；剃刀：删浏览门槛、身份校验后移到提交动作、不引入"游客账号"实体）。②小程序 BASE_URL 不再依赖 127.0.0.1/LAN——用 Cloudflare Tunnel 把 yun1 的 nginx 映射到已备案域名 oiob.cn 的子域名，小程序真机切 https://api.oiob.cn。
>
> 对齐结论（AskUserQuestion 未答，按推荐口径执行）：全浏览+提交才拦；ingress 做 api + admin 两条；沿用 Trellis。

## 事实基础（已核查）

- 小程序 `services.wxml:3` 未登录整页换登录卡、`services.js onShow` 仅登录才 load；`booking.wxml:51` 未登录显示登录提示卡、`onShow` 不加载时段；`booking.js submit()` 已有 `if (!loggedIn) goLogin()`（back=booking 回跳机制存在）。
- H5：/services 无 auth meta（公开），/booking meta.auth 被路由守卫拦；LoginPage 支持 `query.redirect` 回跳（LoginPage.vue:49-50）。
- 后端：router.go:50-51 在 `/api/` 前缀整体挂顾客 JWT 守卫；`GET /api/services`（service/module.go:25）、`GET /api/booking-options`（appointment/module.go:9）挂 api 组。Go 1.22 ServeMux 最长模式优先：把这两条注册到 root 组（精确路径）即可豁免守卫，其余 /api/* 不受影响。两个 handler 均为只读、不依赖 member 身份（实现时复核）。
- yun1 nginx :18090 = /api /admin /healthz 反代 anmo-server:8080 + / 静态顾客 H5；:18091 = HTTPS 自签 admin-ui。cloudflared ingress: api.oiob.cn→http://localhost:18090、admin.oiob.cn→https://localhost:18091(noTLSVerify)。
- weapp config.js 已有 storage 键 anmo.base_url 最高优先级覆盖（第四批）。

## Requirements（W1 游客浏览：后端 + 小程序 + H5）

1. 后端：`GET /api/services`、`GET /api/booking-options` 两条从 api 组移到 root 组（Mount 的 root 接收者，路径不变），handler 逻辑不动；复核两 handler 无 member 身份依赖。测试：无 token 访问两条 200；`POST /api/appointments` 无 token 仍 401；其余 /api/* 无 token 仍 401（回归）。
2. 小程序：services 页删除未登录门槛（onShow 恒 load、wxml 删登录卡）；booking 页允许游客浏览（onShow 加载店铺信息+时段、wxml 删登录提示卡），submit 的未登录→goLogin() 保持不变。
3. H5：/booking 去掉 meta.auth；BookingPage submit 开头无 token → `router.push({name:'login', query:{redirect:'/booking'}})`；myProfile 失败静默已具备；检查 http.ts 全局 401 处理不与游客浏览冲突（公开接口不会 401）。
4. 「我的」/卡包/核销码/我的预约 登录门槛不动。

## W2 小程序切生产域名

- config.js：新增 `PROD_URL = 'https://api.oiob.cn'`；resolveBaseUrl 优先级 storage 覆盖 > devtools→LOOPBACK_URL > 真机→PROD_URL（LAN_URL 保留为本地真机联调的覆盖选项）；注释同步。README 联调章节更新（真机默认 prod、本地联调方式不变、发布前配真实 wx 凭据警示）。
- 单测 config.test.js 更新：真机默认断言从 LAN_URL 改 PROD_URL，storage 覆盖用例保留。

## W3 Cloudflare Tunnel（主会话执行，需用户浏览器授权）

- yun1 安装 cloudflared → `cloudflared tunnel login` 输出授权链接给用户（选 oiob.cn zone）→ tunnel create → /etc/cloudflared/config.yml（两条 ingress + catch-all 404）→ tunnel route dns 自动建 CNAME → systemd 服务开机自启 → 验证 https://api.oiob.cn/healthz 与 https://admin.oiob.cn 200。
- 部署联动：W1 动了后端，执行完重建 yun1 镜像 + customer dist 换装（含 nginx inode 坑：换装后 restart anmo-nginx）+ 冒烟（游客无 token catalog/options 200、预约仍 401）。

## Acceptance Criteria

- [ ] 本地：go build/vet/test 全绿；公开端点测试通过；weapp 单测全绿；H5 构建零错。
- [ ] 浏览器 E2E（本地）：未登录首页→查看项目可见服务卡→进预约页可选服务/日期/时段→点提交跳登录→登录后回到预约页且预选保留→提交成功。
- [ ] yun1：游客无 token catalog/options 200；POST /api/appointments 401；H5 200。
- [ ] tunnel：api.oiob.cn/admin.oiob.cn 公网 HTTPS 可达（curl 实测），systemd enabled。
- [ ] 小程序真机默认走 https://api.oiob.cn（config.js + 单测），合法域名配置步骤交付给用户。

## Notes（不做）

- 不做游客账号/游客 token 实体；不动写接口鉴权；不做 h5 独立子域名（api 同源承载）；不改核销码/卡包门槛；不自动配置微信后台合法域名（需管理员扫码，给用户步骤）；不替用户配生产 wx 凭据（交付时给步骤）。
