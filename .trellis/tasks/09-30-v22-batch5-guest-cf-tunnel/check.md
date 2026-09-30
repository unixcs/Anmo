# Check — V2.2 第五批：游客浏览 + 小程序切生产域名

> 执行：trellis-check 子代理，2026-09-30。规格符合性 + 对抗式审查 + 全量验证，发现 1 处批内自相矛盾已自修，全量验证复跑全绿。未 commit。

## A. 规格符合性 — 全部符合

1. **后端公开面**：`service/module.go:27`、`appointment/module.go:11` 两条以 `root.HandleFunc("GET /api/...")` 注册（精确路径不变）；两 handler 零改动——`handleCustomerCatalog` 只调 `ListCategories/ListItems(ctx, true)`，`handleCustomerBookingOptions` 只读 date query 调 `BookingOptions`，均无 `PrincipalFrom`/memberID 读取。其余 `/api/*` 逐条核对仍全在 api 组。
2. **weapp**：services `onShow` 恒 `load()`、wxml 无登录卡；booking `onShow` 经 `app.ready` 游客同样加载目录+时段、wxml 无登录提示卡；`submit` 未登录 → `goLogin()`（`?back=booking`）保留。
3. **H5**：`/booking` 无 meta.auth；`BookingPage.submit` 开头无 token → notify + `saveDraft()` + `router.push({name:'login', query:{redirect:'/booking'}})`；HomePage 游客只拉公开 catalog；`/me`、`/me/cards`、`/me/qrcode`、`/me/appointments`、`/me/history`、`/me/profile` 六条 meta.auth 全保留。
4. **config.js**：优先级 storage 覆盖（合法 http(s) 无空白）> devtools→LOOPBACK > 真机→`https://api.oiob.cn`（try/catch 兜底同）；`PROD_URL`/`LAN_URL` 均导出；README 使用/联调/验证/发布前清单四处同步。

## B. 对抗式审查

1. **公开面扩大化**：全仓 grep `root.Handle(HandleFunc)` —— 匿名可达仅 router.go 既有 `GET /healthz`、`GET /{$}`，identity 既有 6 条 auth 端点（POST admin/auth/login、POST api/auth/sms/send|verify、POST api/auth/register、POST api/auth/login、POST api/auth/wx/login），加本批 2 条 GET。`GET /api/appointments` 仍在 api 组（appointment/module.go:8）。Go 1.22 最长模式优先成立；HEAD 隐式跟随 GET（同一只读 handler，无新增面）；POST /api/services 落 /api/ 守卫。**豁免没有扩大化**（H8 回归三连 401 实证）。
2. **options 泄露面**：`BookingOptions` 仅 `{date, open, am/pm:{closed,total,remaining,slots:[{time,remaining}]}}`（booking.go:436-518），来源 bizRules 派生值 + 闭店行计数 + 当日有效预约数；无 member 数据、无 settings KV 原文、无内部字段；与登录顾客/admin 看到的同函数同数据，无提权。预期相符。
3. **H5 草稿闭环**：`saveDraft` 写 `anmo.booking.draft` = `{serviceId,dayIdx,part,slotTime,note}`（`satisfies BookingDraft`）；`takeDraft` 同键读+removeItem；onMounted→pendingDraft→loadServices preset→refreshOptions→`applyPendingDraft`（part 半天仍开放才恢复、slotTime 仍有余量才恢复）逐字段匹配。登录成功 `router.replace(redirect ?? '/me')`（LoginPage.vue:49-50）→ /booking 无 auth meta 不再被拦；H5 无 keep-alive，返回即重挂必走 takeDraft。**闭环成立**。
4. **weapp 闭环**：submit !loggedIn → `navigateTo login?back=booking`；login.js:31-37 back==='booking' → navigateBack（fail 兜底 switchTab）→ booking 是 tab 页实例未销毁，onShow→`app.ready`（markAuth 后同步 cb(true)），选区全在 page data 保留。pendingServiceId 链路：游客路径即时消费预选（旧代码游客根本不消费），登录回跳时 services.length>0 不重拉、不丢选中。**闭环成立**。
5. **HomePage 游客分支**：成功→`pickHomeServices(services,{})`（默认 limit 6）+ DEFAULT_BLOCKS；失败→services=[] + DEFAULT_BLOCKS；banner/announcement 块模板侧 `length>0` 才渲染，游客空数组无空壳；状态灯 v-if statusText 隐藏；hero/营业时间/ShopCard 走模板默认回落。statusTimer 仅登录分支创建，游客分支提前 return → **无轮询泄漏**；登录后进首页因无 keep-alive 重挂恢复 authed 全量拉取。
6. **401 全局跳转**：unauthorizedHandler 仅由 http.ts:58 真实 401 响应触发；游客路径请求只有 catalog + booking-options（均 200），myProfile 被 `currentToken()` 门禁，无人为 401 残留。游客直访 /me 走 router.beforeEach meta.auth 拦截（带 redirect），与 401 handler 互不干扰。
7. **request.js 空 token**：`getToken() ? {Authorization...} : {}`（request.js:23-26）——不发送空 Authorization、不抛错；401 分支仅真实 401 触发。安全。
8. **services onShow 恒 load**：每次切 tab 一次幂等 GET；人类操作频率、响应同源 last-write-wins 无状态污染，双请求最坏重复一次。无防抖但不构成请求风暴，**可接受**。

## 发现并自修（1 处）

- `apps/weapp/pages/booking/booking.js` onPullDownRefresh 原保留 `if (!loggedIn) return` 短路——与本批"游客可浏览时段"自相矛盾（游客下拉零反馈）。已移除短路（目录/时段均公开）；`loggedIn` 仍被 submit 使用，无死代码。

## 观察未修（非本批回归，报告留决策）

- weapp services 与 H5 ServicesPage 目录**加载失败**均静默显示"暂无服务项目"空态（无错误+重试）。booking 两端有 svcErr 模式而 services 两端皆无——既有跨端一致行为；修复需同时动 H5 ServicesPage.vue（不在本批清单），留给主会话。
- weapp 首页未登录仍整页登录卡（home.wxml:25）：PRD W1 明确只动 services/booking 两页（/api/home 等仍 401），游客可经底 tab 直达服务/预约页，属 PRD 口径内。

## C. 全量验证（自修后复跑，原始输出）

1. `go build ./... && go vet ./... && go test ./...`（server/）— build/vet 零输出，14 包 ok / 4 包无测试文件、0 跳过；Go 代码与实现时一致走缓存，另以 `-count=1 -v` 新鲜复跑两条关键用例：
   ```
   --- PASS: TestGUARD_H1_AdminRouteMatrixAndAnonymous (0.37s)
   --- PASS: TestGUARD_H8_GuestBrowsePublicEndpoints (0.09s)
   PASS
   ok  	anmo/server/internal/e2e	0.460s
   ```
2. `node --test "apps/weapp/tests/*.test.js"` — `tests 7 / pass 7 / fail 0 / skipped 0`。
3. `npm --prefix apps/customer run build` — vue-tsc 零类型错误，`✓ built in 399ms`。

## 结论

实现与 PRD/冻结决策逐条相符，公开面收口严格（仅两条只读 GET 豁免且有测试实证回滚线），游客→登录→回跳双端闭环经代码级逐行核对成立。1 处批内不一致已自修，全量验证全绿。**可交付主会话执行 W3 部署联动**（交接提示见 implement.md）。
