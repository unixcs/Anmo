# Implement — V2.2 第五批：W1 游客浏览 + W2 小程序切生产域名

> 执行：trellis-implement 子代理，2026-09-30。W3 Cloudflare Tunnel 由主会话执行，未触碰。
> 未 commit（按纪律）。

## 文件改动

### W1 后端（server/）

- `server/internal/modules/service/module.go` — `GET /api/services` 从 `api.HandleFunc` 移到 `root.HandleFunc`（Mount 首参 root，路径不变），附游客开放注释。
- `server/internal/modules/appointment/module.go` — `GET /api/booking-options` 同样移到 `root.HandleFunc`。
  - 原理：router.go:51 `root.Handle("/api/", NewAuth(verify,false)(api))` 整体挂守卫；Go 1.22 ServeMux 最长模式优先，root 上精确注册的两条优先命中，匿名可达；其余 `/api/*` 照旧 401。
- `server/internal/e2e/adversarial_test.go`
  - H1 的 api 401 矩阵移除 `{"GET", "/api/services"}`（该路径现公开，且 ADMIN token 反向探测也返回 200，留在矩阵必假红），加注释指向 H8。
  - 新增 `TestGUARD_H8_GuestBrowsePublicEndpoints`：无 token `GET /api/services` → 200 且有 data 包络；无 token `GET /api/booking-options?date=<+2天>` → 200；回归三连：无 token `POST /api/appointments` / `GET /api/me/profile` / `GET /api/appointments` → 401（豁免没有扩大化）。
  - 该文件新增 import `anmo/server/internal/shared`（构造 booking-options 日期用 `shared.NowShanghai()`）。

### W1 小程序（apps/weapp/）

- `pages/services/services.js` — onShow 恒 `this.load()`（catalog 已公开，不再等 ready/loggedIn）；data 删 `loggedIn`；删 `goLogin()`（wxml 删卡后无引用）；onPullDownRefresh 去掉 loggedIn 短路。
- `pages/services/services.wxml` — 删未登录登录卡，解开原 `<block wx:else>` 包裹（骨架/空态/分组列表成为唯一分支）。
- `pages/booking/booking.js` — onShow 仅删 `if (!loggedIn) return` 一行（改为注释说明游客可浏览服务目录+时段）；`loggedIn` 保留（submit 的 goLogin 分支与 onPullDownRefresh 仍用）。
- `pages/booking/booking.wxml` — 删未登录登录提示卡（选最小改动：不留弱文案，submit 未登录按钮点击即 goLogin）。
- `pages/booking/booking.wxss` — 删登录提示条的死样式 `.login-tip` / `.login-go`。
- `utils/request.js`（核查未改）— token 拼接对空值安全（`getToken() ? {Authorization...} : {}`），401 分支只在真实 401 响应时触发；公开接口 200 不会进入。

### W1 H5（apps/customer/）

- `src/router/index.ts` — `/booking` 删 `meta: { auth: true }`（/services 本就公开）。
- `src/pages/BookingPage.vue`
  - import `currentToken`（`../platform/auth/session`，与 router 同源，真实导出名已核实）。
  - `submit()` 最开头：无 token → `notify('登录后即可提交预约')` + `saveDraft()` + `router.push({name:'login', query:{redirect:'/booking'}})` + return。
  - 草稿逻辑从 `goProfileSheet` 抽出为 `saveDraft()` 复用：登录跳转会重建页面实例，不存草稿则"登录后回到预约页预选保留"这条验收必挂；onMounted 的 `takeDraft()`→`loadServices` preset→`applyPendingDraft()` 链路原样负责恢复。
  - `onMounted` 的 `myProfile()` 加 `currentToken()` 门（见下"与派发单的偏差"）。
- `src/pages/HomePage.vue` — onMounted 增加游客分支：无 token 只拉公开 `api.catalog()` 渲染服务推荐 + DEFAULT_BLOCKS，hero/营业时间走模板默认回落；`loadStatus` 与 60s 轮询仅登录时启动（见下"偏差"）。
- `src/core/api/http.ts`（核查未改）— 无"启动时主动 401 跳登录"残留；tokenProvider 空值安全（`if (token)`）；全局 401 handler 仅在真实 401 响应触发（main.ts:15 清会话+跳登录）。冲突点不在 http.ts 本身，而在游客态调用登录态接口（已修 HomePage/BookingPage）。

### W2 小程序切生产域名（apps/weapp/）

- `config.js` — 新增 `PROD_URL = 'https://api.oiob.cn'`；resolveBaseUrl 优先级：storage 覆盖 > devtools→LOOPBACK_URL > 真机→PROD_URL（try/catch 兜底也改 PROD_URL）；module.exports 增加导出 `PROD_URL`；头部注释更新（真机默认生产域名、本地真机联调用 storage 覆盖 LAN_URL）。
- `tests/config.test.js` — ④ 真机默认断言 LAN_URL→PROD_URL；新增 ④b「真机 + storage 覆盖 LAN_URL 生效」；⑤ 无 wx 兜底断言 LAN_URL→PROD_URL；其余用例保留。
- `README.md` — 「使用」第 3 条真机默认 `https://api.oiob.cn`，本地真机联调方式移到 storage 覆盖；「开发者工具联调」第 6 条同步；「验证」章节覆盖描述同步；「发布前清单」合法域名明确 `https://api.oiob.cn`（mp 后台管理员扫码路径写明）。

## handler 身份依赖复核结论（派发单第 4 点）

- `service/handler.go:104 handleCustomerCatalog`：只调 `p.ListCategories(ctx, true)` / `p.ListItems(ctx, true)`，无 `middleware.PrincipalFrom`，无 memberID 读取。
- `appointment/handler.go:71 handleCustomerBookingOptions`：只读 `date` query 调 `p.BookingOptions(ctx, date)`（booking.go:436），内部只查 bizRules/closure/active bookings，无 principal 依赖。
- **结论：两 handler 零改动即可匿名暴露**，"如确有依赖，最小调整并报告"分支未触发。

## 与派发单的偏差（两处，均为派发单假设与代码事实不符）

1. **派发单 H5 第 2 点说"onMounted 的 myProfile().catch(()=>{}) 已静默，不动"**：`.catch` 只能吞掉 promise rejection，吞不掉 `http.ts` 全局 `unauthorizedHandler` 的 `signOut + router.push(login)`（main.ts:15-19，仅由真实 401 响应触发）。游客打开 /booking 必 401 → 被弹去登录，游客浏览整条链路断裂。最小修复：`if (currentToken())` 才拉 myProfile（与 HomePage.maybeFirstLoginPrompt 既有模式一致）。
2. **HomePage 未在派发单文件清单里，但验收 E2E 第一步是"未登录首页→查看项目"**：HomePage onMounted 无条件调 `/api/home`、`/api/settings`、`/api/store/status`（三者均留在 api 组 401，PRD 回归口径明确不动它们），游客进首页即被全局 401 handler 踢去登录。最小修复：游客分支只拉公开 catalog（服务推荐照常渲染），home/settings/status 与状态轮询仅登录后拉。weapp 首页本就有同语义的 loggedIn 门（"内容接口都是顾客 JWT 资源"），H5 补齐的是同一口径。

## 验证输出摘要（全部实跑）

1. `go build ./... && go vet ./... && go test ./...`（/mnt/Projects/Anmo/server）— build/vet 零输出，test 全 ok 零跳过：
   ```
   ok  anmo/server/internal/e2e          3.148s
   ok  anmo/server/internal/adversarial  (cached)
   ok  anmo/server/internal/middleware   (cached)
   ok  anmo/server/internal/modules/*    (cached) …（14 包 ok，4 包无测试文件）
   ```
   指定跑新用例：`--- PASS: TestGUARD_H1_AdminRouteMatrixAndAnonymous (0.37s)`、`--- PASS: TestGUARD_H8_GuestBrowsePublicEndpoints (0.08s)`。
2. `node --test "apps/weapp/tests/*.test.js"` — 7 pass / 0 fail / 0 skipped（新增 ④b，④⑤ 断言切 PROD_URL）。
3. `npm --prefix apps/customer run build`（vue-tsc -b && vite build）— 零类型错误，`✓ built in 377ms`。
4. `git diff --stat` — 14 files changed, 136 insertions(+), 95 deletions(-)（清单与上文一致，无越界文件）。

## 遇到的问题

- adversarial_test.go H1 的 401 矩阵包含 `/api/services`，路由移动后该矩阵（含 ADMIN token 反向探测段）必假红——按"豁免收口"语义移出并加注释，公开语义改由 H8 专测。
- booking-options 的 H8 用例需要合法 date 参数，用 `shared.NowShanghai().AddDate(0,0,2).Format("2006-01-02")` 构造（handler 对非法日期返回 400，不构造好日期会测成 400）。
- 其余见"与派发单的偏差"两处 401 链路冲突，均已按最小改动修复。

## 交接提示（主会话 W3 部署联动时用）

- 后端有改动：yun1 重建镜像后冒烟两条游客端点 + 一条 401 回归（对应 H8 的 curl 版）。
- customer dist 需重新换装（HomePage/BookingPage/router 都变了；nginx 换装后 restart anmo-nginx 的 inode 坑照旧）。
- 小程序合法域名 `https://api.oiob.cn` 需管理员在 mp 后台扫码添加；发布前配真实 `ANMO_WX_APPID`/`ANMO_WX_SECRET`（README 已写步骤）。
