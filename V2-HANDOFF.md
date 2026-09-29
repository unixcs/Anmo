# V2 微信小程序 — 阶段性交接文档（切新 CLI 工具专用）

> 生成时间：2026-09-28。最后收口：2026-09-28（Codex 会话，全量验证 + 本文档更新）。本文档是 V2（微信小程序 原生 + UniApp，共用同一 Go 后端）任务的**唯一权威上下文**，供新会话/新 AI CLI 工具零损耗接手。
> 任务定义（PRD）：`.trellis/tasks/09-28-v2-miniprogram/PRD.md`。上层约束：根 `AGENTS.md`（D1–D26 冻结决策）。
> 本文档取代口头上下文：所有路径、命令、版本、状态、坑都在这里。

---

## 一、任务总览与当前进度

**目标**（用户原话：“继续执行 v2” → 接手会话原话 **“把原生小程序打磨好。不做 Uni APP。”**，上位 = /goal 88 节 V1.x 路线）：Phase C 后台准备 → Phase D 原生小程序（**Phase E UniApp 经用户指示取消**），共用同一 Go 后端；分享完整闭环；最终原则“一套后端多个前端”。

| Phase | 状态 | 说明 |
|-------|------|------|
| C：后台（微信登录 + 绑定） | ✅ **完成** | migration 013 + identity/member 能力 + 路由 + 测试全绿 |
| D：原生小程序 `apps/weapp` | ✅ **完成，验证 100%** | 11 stage IDE 端到端 **53 PASS / 0 FAIL**（含预约提交→成功→列表可见→取消、具体槽、今日单码、D21 正反两分支）+ 纯逻辑单测 19 全绿；见 §4/§7 与 `apps/weapp/tools/` |
| E：UniApp `apps/uniapp` | ⛔ **取消（用户指示）** | 不做 UniApp；§5 方案保留备查 |
| 收口（全量测试/文档/journal） | ✅ **完成** | 后端 build/vet/test 带 DSN 14 包全绿；AGENTS.md 补 D23–D26；README + journal 已写 |
| 部署（GitHub → yun1 → Tencent） | ⬜ **未做，需用户点头** | 共享状态操作：提交/推送/生产部署均需用户确认后执行（V1.x 规矩见 §8-4） |

> **收口口径**：不做 uni-app（Phase E 明确取消）；原生小程序 `apps/weapp` 为 V2 交付主体。本轮收口已复跑 `go build/vet/test ./...`（带 DSN 全绿）与 `apps/weapp/tools/unit.js`（19 全绿）、`node --check` 全过；未提交、未推送（部署链仍等用户点头）。

**Git 状态**：V1.x 已提交（HEAD `dd5fcde`）。V2 全部改动未提交：
```
M  server/internal/config/config.go
M  server/internal/e2e/e2e_test.go
M  server/internal/modules/identity/{AGENTS.md,api.go,handler.go,module.go,token.go}
M  server/internal/modules/member/AGENTS.md
?? .trellis/tasks/09-28-v2-miniprogram/   （PRD.md）
?? apps/weapp/                            （原生小程序全量）
?? scripts/allow-wsl-lan.ps1              （小程序真机联调：WSL2 LAN 防火墙放行脚本）
?? server/internal/modules/identity/{wx.go,wx_test.go}
?? server/internal/modules/member/openid.go
?? server/migrations/013_member_wx_openid.sql
```

---

## 二、环境事实（务必确认后直接用）

### 机器与进程
- WSL 发行版名：`Ubuntu-24.04`（D 盘路径引用需要：`\\wsl.localhost\Ubuntu-24.04\...`，但开发者工具**不接受**该路径，见 §6）
- Go：`export PATH=$PATH:/usr/local/go/bin`（Go 1.27）
- MySQL：`docker compose` 起，`127.0.0.1:33306`（root/anmo-root-2026，库 anmo）
- 本地后端：`/mnt/Projects/Anmo/.dev/anmo-bin -config /mnt/Projects/Anmo/server/config.example.yaml`（**含 wx 端点的二进制，务必放仓库 `.dev/`（已 gitignore），不要放 `/tmp`——本机 `/tmp` 会被系统清空，本轮已因此丢过二进制/automator/脚本**；PID 查看 `ss -ltnp | grep 8080`；日志 `.dev/anmo-bin.log`）。重启方法（本地开发栈，安全）：
  ```bash
  cd /mnt/Projects/Anmo/server && export PATH=$PATH:/usr/local/go/bin
  go build -o /mnt/Projects/Anmo/.dev/anmo-bin-new ./cmd/anmo
  kill <旧PID> && sleep 1 && mv /mnt/Projects/Anmo/.dev/anmo-bin-new /mnt/Projects/Anmo/.dev/anmo-bin
  # cwd 必须是 server/（migration 目录相对路径）
  cd /mnt/Projects/Anmo/server && setsid nohup /mnt/Projects/Anmo/.dev/anmo-bin \
    -config config.example.yaml > /mnt/Projects/Anmo/.dev/anmo-bin.log 2>&1 < /dev/null &
  ```
  无 config 文件时也可纯 env 启动（`ANMO_MYSQL_DSN` / `ANMO_AUTH_JWT_SECRET` / `ANMO_SMS_MODE=dev`）；**换 jwt_secret 重启会使旧 token 全失效**，实测请固定一种方式。
  注意：`pkill -f anmo-bin` 会匹配到自己的命令行把 shell 打死（exit 143）→ 用 `kill <PID>` 或 `setsid` 分离。
- H5 顾客端 5173 / 商家端 5174、5175 由 start-anmo.sh 管理（未改动）
- **Windows 微信开发者工具**：`D:\Program\soft\wechattools`（WSL 路径 `/mnt/d/Program/soft/wechattools`，版本 Stable 2.02.2608070）
  - IDE HTTP 服务端口：19225（已登录状态，`cli islogin` 返回 `{"login":true}`）
  - 自动化端口：9420（`cli auto` 打开；`miniprogram-automator` 连 `ws://localhost:9420`）
  - **appid 为 touristappid（游客模式）**；游客模式下 `wx.login` 返回**固定 code**（默认 the-code）→ 后端 dev 兜底解析成 `dev:the-code`，这对联调行为有影响（见 §6.4）

### 测试 DSN（Go 集成测试用）
```bash
ANMO_TEST_MYSQL_DSN='root:anmo-root-2026@tcp(127.0.0.1:33306)/?parseTime=true&loc=Asia%2FShanghai&time_zone=%27%2B08%3A00%27&charset=utf8mb4&multiStatements=true'
```
无此变量时 `testsupport` 会 `t.Skip`（**会掩盖失败，历史教训：必须带 DSN 跑**）。跑法：
```bash
export PATH=$PATH:/usr/local/go/bin; cd /mnt/Projects/Anmo/server
ANMO_TEST_MYSQL_DSN='...' go test ./... -count=1
```

### WSL↔Windows 传物
```bash
cp -r /mnt/Projects/Anmo/apps/weapp/* /mnt/d/anmo-weapp-build/ && rm -f /mnt/d/anmo-weapp-build/README.md
```

---

## 三、Phase C 后台已完成成果（V2 微信登录链路）

### 设计（PRD/AGENTS.md 决策 D23–D26）
- **D23**：`member.wx_openid VARCHAR(64) NULL + UNIQUE`（NULL 可重复；一个 openid 只绑一个 member）。不建独立绑定表。
- **D24**：凭据 `wx.app_id`/`wx.secret`（env `ANMO_WX_APPID`/`ANMO_WX_SECRET`）；**AppID 为空时 dev 兜底 `openid = "dev:"+code`**（与 SMS dev 固定码同级开发姿态，启动日志警示）；小程序发布前必须配真凭据。
- **D25**：未绑定 → 发 **bind_ticket**（JWT `act=WXBIND`，10 分钟），不直接落库；绑定必须持顾客 Token（`POST /api/auth/wx/bind`）。
- **D26**：微信官方“服务卡片”能力不做；分享闭环 = 每页 `onShareAppMessage` + 首页/关于 `onShareTimeline` + `showShareMenu`。

### 文件与变更
| 文件 | 内容 |
|------|------|
| `server/migrations/013_member_wx_openid.sql` | member 加 wx_openid + 唯一键 |
| `server/internal/config/config.go` | `Wx{AppID,Secret,APIBase}` 段 + env + APIBase 默认 `https://api.weixin.qq.com`（APIBase 供测试 httptest 桩注入） |
| `server/internal/modules/member/openid.go`（新） | `FindByOpenID(ctx,openid)` / `BindOpenID(ctx,tx,memberID,openid)`（1062→`WX_OPENID_BOUND` 409；重复绑同对幂等） |
| `server/internal/modules/identity/wx.go`（新） | `code2session`（AppID 空→dev 兜底）/ `WxLogin(ctx,code)→(token,memberID,bindTicket,needsBind,err)` / `WxBind(ctx,memberID,bindTicket)` |
| `server/internal/modules/identity/token.go` | `actWxBind="WXBIND"` + `signBindTicket`/`parseBindTicket`（10 分钟） |
| `server/internal/modules/identity/api.go` | Provider 加 `wxHTTP *http.Client`(Timeout 10s) |
| `server/internal/modules/identity/handler.go` | `handleWxLogin`（root 公开）/ `handleWxBind`（顾客 Token 鉴权） |
| `server/internal/modules/identity/module.go` | 路由：`POST /api/auth/wx/login`（root 公开）、`POST /api/auth/wx/bind`（api mux 顾客守卫） |
| `server/internal/e2e/e2e_test.go` | +`TestWxLoginFlow`：login→needs_bind→无 Token bind 401→SMS 登录→bind→再 login 直发 token 且 member phone 一致 |

### API 响应形状
- `POST /api/auth/wx/login {code}` → `{data:{token, member_id}}`（已绑定）或 `{data:{needs_bind:true, bind_ticket}}`（未绑定）
- `POST /api/auth/wx/bind {bind_ticket}`（Authorization: Bearer 顾客token）→ `{data:{bound:true}}`

### 测试结果（已跑，全绿）
```bash
cd /mnt/Projects/Anmo/server
ANMO_TEST_MYSQL_DSN='...' go test ./internal/modules/identity/ -run Wx -count=1 -v   # 3 PASS
ANMO_TEST_MYSQL_DSN='...' go test ./internal/e2e/ -run TestWxLoginFlow -count=1 -v    # PASS
```
覆盖：dev 兜底、code2session httptest 桩（appid/secret/js_code 参数校验 + errcode 上抛）、bind_ticket 伪造拒绝、同 openid 换 member 409、重复绑定幂等、完整 HTTP 链路。
**未跑**：全量 `go test ./... -count=1`（收口阶段必跑）。

---

## 四、Phase D 原生小程序 `apps/weapp` 已完成成果

### 结构（8 页 + tabBar 首页/预约/我的 + 2 组件 + 4 工具模块 + vendored QR 库）
```
apps/weapp/
  project.config.json   (appid 占位 touristappid；urlCheck:false；libVersion 3.8.10)
  app.json              (8 pages; tabBar 纯文本; lazyCodeLoading requiredComponents)
  app.js                (onLaunch: showShareMenu 含朋友圈 + ensureSession 静默登录尝试)
  app.wxss              (Design Tokens：--bg/--card/--foreground/--muted/--border/--primary 等，与 H5 同色系)
  config.js             (BASE_URL 默认 http://127.0.0.1:8080；注释含手机预览/正式发布改法)
  utils/request.js      (Bearer 注入、包络解包、401→清 token+needLogin、NETWORK 兜底)
  utils/api.js          (与 apps/customer/src/core/api/endpoints.ts 一一对应，含 wxLogin/wxBind/sendSms/verifySms)
  utils/auth.js         (wxSilentLogin / smsLogin(带 ticket 自动绑定) / ensureSession / cachedProfile / logout)
  utils/format.js       (aptTime 模糊半天、statusText WAITING→待到店、yuan 分→元、cardValidity、txText)
  utils/qrcode.js       (vendored qrcode-generator 1.4.4，UMD/CommonJS，无 DOM 依赖，2297 行)
  components/shop-card/ (地址→wx.openLocation 导航 / 无经纬度→复制地址；电话→wx.makePhoneCall)
  components/slot-picker/(上下午必选+可选具体时间；满槽禁选；triggerEvent change{part,slot})
  pages/home/           (门店状态徽标 60s 轮询 / home_title,body / 服务列表 / 门店卡 / 登录入口)
  pages/booking/        (tab页; 服务选择→30天日期条→slot-picker→模糊/具体二选一→成功页内态)
  pages/appointments/   (筛选、取消(WAITING)、改期抽屉复用 slot-picker)
  pages/cards/          (卡名/剩余/低余额高亮/有效期; 点卡头展开使用明细 ±次数着色)
  pages/qrcode/         (时钟; D21 会员码 ANMO-MEMBER:<ulid> 仅 ACTIVE 卡出示; 今日 WAITING/IN_SERVICE 预约 ANMO-APT 码; canvas 2d 绘码)
  pages/login/          (非阻塞式: SMS 表单立即可用 + 后台 wxSilentLogin 尝试; 拿到 ticket 登录后自动绑定)
  pages/me/             (资料编辑 name/gender、功能入口、退出登录)
  pages/about/          (门店信息+分享; open-type="share" 按钮)
```
**分享闭环**：所有业务页 `onShareAppMessage`；home/about `onShareTimeline`；app.js 开启 `showShareMenu`。

### 关键实现口径（与后端/H5 严格对齐）
- 内容接口（/api/services、/api/settings、/api/store/status、/api/home、/api/appointments 等）都是**顾客 JWT 资源**（middleware 强制）→ 小程序与 H5 同口径：**匿名只显示登录引导**（home 登录卡、booking 登录提示），登录后（onShow 检测 token）自动补拉。
- D21：无 ACTIVE 卡不出会员码；预约单码只取当日 WAITING/IN_SERVICE 且服务名逐单补拉。
- 模糊预约提交：`{date, day_part}`；具体时间：`{start_time}`（建 target 逻辑与 H5 BookingPage 一致）。
- HOME→booking 跳转：booking 是 **tab 页**，`navigateTo` 带参会失败 → 用 `getApp().globalData.pendingServiceId` + `switchTab`（已修）。

### V2 第二轮打磨修复（本轮，全部经 IDE 实测或单测复验）
- **登录态竞态**：`app.js ready(cb)` 改为读最新 globalData（原实现回灌启动时的旧快照 → 登录成功后页面仍渲染匿名态）；新增 `markAuth(loggedIn)`，login 成功 / me 退出 / request 收到 401 三处同步全局态（401 之前只清 storage，页面徽标不变）。
- **启动链路去重复请求**：login 页 onLoad 复用 `globalData.session`（原本再发一次 wx.login → 与 app 启动的静默登录抢跑）；`bindTicket` 由启动链路缓存透出，登录页直接进绑定态。
- **核销码画布时机**：canvas 在 `wx:if` 内，`setData` 回调里才 `renderAll()`（原实现在 setData 之后同步取节点 → 整页二维码画不出来）。
- **卡流水符号**：`format.txQty`——后端 REDEEM 存正数（语义是扣次）、ADJUSTMENT 自带符号、ISSUE/REVERSAL 为加次；此前一律显示 `+N`（**H5 CardsPage 同源同改，口径一致**）。
- **今日已过槽**：`format.trimPastSlots`（后端不回溯裁剪，改后端会同时影响 H5/D20，故前端收口）+ slot-picker 的 `ended` 态与“时段已过/已闭店/已约满”标签。
- **stale index 补丁**：qrcode 服务名回填、appointments 详情回填一律按 id `findIndex`（原用闭包下标，列表刷新后打错位）。
- **首页 blocks 编排**（D16，与 H5 同口径）：`format.homeBlocks` 按后端 `content_page_config` 编排排序，未知类型/`banner` 忽略，全被过滤时回落默认；实测确认“后台未编排 announcement 时公告不展示”两端一致（数据配置问题，非 bug）。
- **分享落地降级**：about 页匿名从分享进来不再空屏（settings 缓存兜底 + 登录引导卡）；补 `onPullDownRefresh`（json 开了开关却没处理函数会一直转）。
- keepScreenOn 在 onHide/onUnload 释放（原只在成功回调释放，退出页面会一直锁屏）。

### 验证状态（详见 §7）
- ✅ node --check 全部 JS、JSON parse 全部 JSON
- ✅ qrcode.js node 冒烟（29×29 矩阵）
- ✅ **纯逻辑单测** `node apps/weapp/tools/unit.js` → 19 全绿（slot-picker 门槛 / trimPastSlots / txQty 流水方向 / homeBlocks 编排 / 展示口径）
- ✅ **IDE 端到端** `node apps/weapp/tools/devtools-verify.js` → **11 stage 53 PASS / 0 FAIL**（每 stage 独立重连重试，失败退出非 0）
- ✅ 预约闭环实测：模糊预约 `{date,day_part}` → 成功卡 → 我的预约可见 → 取消（CANCELLED 且不可再取消）；具体槽 `{start_time}` 提交；今日单 ANMO-APT 单码 + 服务名快照
- ✅ D21 正反两分支实测：无 ACTIVE 卡 → 会员码卡与 canvas 均不出现；持卡会员（13990497037）→ 会员码 canvas 渲染 + 明细流水符号方向正确
- ✅ 匿名门禁 / qrcode→login 跳转 / wx 游客自动登录 / 登录后核销码页数据加载（截图证实）

---

## 五、Phase E UniApp —— ⛔ 用户指示不做（方案保留备查）

> 原则（plan §103）：uni-app 是“一套业务代码，H5+小程序双端”的长期基座；本轮**不替换线上 H5**（歧义选不做）。

```
apps/uniapp/                （待建）
  package.json              vite 5.2.8 / vue 3.4.21 / @dcloudio/* 全链 3.0.0-5020620260917001
  vite.config.js            import { uni } from '@dcloudio/vite-plugin-uni'; plugins:[uni()]
  index.html                (H5 打包用)
  src/main.js               createSSRApp(App)
  src/App.vue
  src/manifest.json         (mp-weixin appid 占位)
  src/pages.json            (home/booking/me 为 tab；appointments/cards/qrcode/login/about 非 tab)
  src/utils/                (复用 weapp 的 request/api/auth/format 改造为 uni.* 口味：
                             uni.request / uni.login({provider:'weixin'}) 仅 #ifdef MP-WEIXIN，H5 走短信)
  src/utils/qrcode.js       (vendored 同上)
  src/components/shop-card.vue、slot-picker.vue
  src/pages/{home,booking,me,appointments,cards,qrcode,login,about}/**.vue
```
安装：`npm i --registry https://registry.npmmirror.com`（npmmirror 已确认 `@dcloudio/vite-plugin-uni@3.0.0-5020620260917001` 存在，其 peer 恰为 vite 5.2.8；vue 用 3.4.21）。
验证：`npm run build:h5` + `npm run build:mp-weixin` 双端编译过；mp-weixin 产物可再用 §6 的 devtools CLI 跑一次冒烟。

---

## 六、Windows 开发者工具提效链路（重大发现，务必沿用）

用户提供了专用 skill（官方 tencent-adm/wechatide-skill v0.3.2，安装：让 Agent 访问 https://skillhub.cn/install/skillhub.md 或 `npx skills add tencent-adm/wechatide-skill`）。**本环境下直接驱动官方 CLI 更高效**（skill 面向 CodeBuddy 生态；新 CLI 工具可按需装它）。

### 6.1 标准验证流程（已验证可行，脚本已固化进仓库）
```bash
# 0) 后端在跑：见 §2（cwd 必须 server/）；MySQL 起来
# 1) 同步代码到 Windows 本地盘（IDE 不认 \\wsl.localhost UNC 路径 → code 10）
rsync -a --delete --exclude node_modules --exclude .dev /mnt/Projects/Anmo/apps/weapp/ /mnt/d/anmo-weapp-build/
# 2) 重启 IDE 自动化（坑：hot compile 不可靠——cp 覆盖后一律 quit+auto）
cd /tmp && cmd.exe /c "D:\\Program\\soft\\wechattools\\cli.bat quit"
sleep 15   # 少了这一步立刻 auto 会 "initialize error: read ECONNRESET"
setsid nohup cmd.exe /c "D:\\Program\\soft\\wechattools\\cli.bat auto --project D:\\anmo-weapp-build --auto-port 9420" > /mnt/Projects/Anmo/.dev/cli-auto.log 2>&1 < /dev/null &
sleep 50   # 见到 "√ auto" 还要再等：IDE 内部重编译期间 automator 首条命令必 timeout
# 3) 跑实测（依赖装在仓库里，不再放 /tmp）
cd /mnt/Projects/Anmo/apps/weapp/tools && npm i --registry https://registry.npmmirror.com && node devtools-verify.js
```
- `cli islogin` 返回 `{"login":true}`（IDE 已扫码登录）。
- WSL→Windows `localhost:9420` **可直接连通**（mirrored 网络下）。
- CLI 需从非 UNC 目录运行（先 `cd /tmp`），否则报 UNC warning。
- 冒烟单条命令诊断：`connect` + `currentPage` + `page.data()` 三件套能过，说明桥健康、页面已编译（本机 `node -v` 为 v22，全局 `fetch` 可用）。
- **`mp.screenshot({path})` 是运行侧 Node fs 写盘**（IDE 只回传 base64）→ path 必须是 WSL 可写路径；传 `D:\...` 会在 cwd 生成名字里带反斜杠的垃圾文件（本轮踩过并已清）。脚本默认落 `.dev/shots/`。

### 6.2 CLI 子命令备忘
- `cli islogin` / `cli quit` / `cli auto --project <winpath> --auto-port 9420`
- `cli preview --project <winpath> --qr-format base64`：**touristappid 会失败**（“AppID 不存在”）——预发布需真实 AppID+上传密钥。
- 全屏截图（诊断 IDE 状态用）：
  `/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe -NoProfile -Command "Add-Type -AssemblyName System.Windows.Forms,System.Drawing; \$b=New-Object System.Drawing.Bitmap([System.Windows.Forms.Screen]::PrimaryScreen.Bounds.Width,[System.Windows.Forms.Screen]::PrimaryScreen.Bounds.Height); \$g=[System.Drawing.Graphics]::FromImage(\$b); \$g.CopyFromScreen(0,0,0,0,\$b.Size); \$b.Save('D:\\anmo-weapp-build\\ide-screen.png')"`

### 6.3 automator 桥不稳定的成因与对策（重要教训）
- 症状：每个会话约 2 个轻命令（reLaunch/currentPage）后第三个元素查询必死，报 `timeout waiting for automator response` 或 `page is not on top of page stack`；断开重连可恢复 2 个命令，循环衰减。
- 诊断结论（带时间戳实验 + 全屏截图）：
  1. `cli auto` 后 IDE 尚在初始化，脚本过早连上 → IDE 内部重编译导致 page 对象批量失效 → **连上后先 sleep 20s 再断言**。
  2. 该 devtools 版本（2.02.2608070）的自动化桥本身有会话衰变。
  3. 曾误判为“登录页卡死”——真相是游客模式 wx.login 返回固定 the-code，已绑定会员被静默自动登录，login 页一等兵未留就 navigateBack 回 qrcode（截图证实 qrcode 渲染完全正常）。
- 已沉淀到代码的对策：login 页改成**非阻塞**（表单常显 + 后台静默登录），本身是更好的 UX。
- **本轮收口做法（已固化进 `apps/weapp/tools/devtools-verify.js`，下一轮直接跑）**：
  a) **逐 stage 重连 + 重试**（`ANMO_ATTEMPTS`，默认 3）：一个 stage 一个连接，死掉就重连重做该 stage——实测只有 `cards` 需要重试一次，其余 stage 一次通过；
  b) **自定义组件内部节点够不到**（`page.$('slot-picker')`、`page.$$('.part')`、`'slot-picker >>> .part'` 全部返回空，实测确认）→ 对受控组件一律 `page.callMethod('onPick', {detail:{part,slot}})` / `callMethod('pickDay', {currentTarget:{dataset:{idx}}})`，组件自身门槛交给 `unit.js`；
  c) 原生弹窗 `mp.mockWxMethod('showModal', {confirm:true})` + `restoreWxMethod`；截图 `mp.screenshot({path:'D:\\...'})`；
  d) `callWxMethod('getStorageSync', key)` 返回**裸字符串**（不是 `{result}`）；token 的 key 是 **`anmo_customer_token`**（不是 `anmo_token`）；
  e) 取消用例必须 `list.findIndex(a => a.no === 目标单号)` 定位按钮下标（曾误点 `btns[0]` 取消到历史单，产生假失败）；
  f) 具体槽用例不要固定日期：闭店/满槽会让某天 0 槽 → 遍历 day1..6 找 `remaining>0` 的槽；
  g) “页面值 vs 接口值”对账：脚本用 Node `fetch` 直连后端（`ANMO_API_BASE`，默认 `http://127.0.0.1:8080`）比对 `/api/settings`、`/api/home`，避免把后台数据没配当成小程序 bug。
- 依赖与脚本都在仓库里（`apps/weapp/tools/`），**不要再放 `/tmp`**（本机 `/tmp` 会被清空，本轮已因此丢过二进制、node_modules 和全部 walk 脚本）。旧 walk1–walk5 已由 `devtools-verify.js` 取代。

### 6.4 touristappid 联调特性（必读）
- `wx.login` 在游客模式返回**固定 code**（the-code）→ 后端 dev 兜底 openid=`dev:the-code`。
- 一旦某手机号经`/api/auth/wx/bind`绑过该 member，后续任何 wxSilentLogin 都会**静默自动登录**为该 member（联调便利，也解释了 walk 中“登录表单瞬间消失”）。
- 新手机号换绑：直接 SMS 登录新号即可（同一 member 换 openid 视为 UPDATE，唯一键不冲突）。

---

## 七、Phase D 验证证据清单（当前结论）

| 项 | 证据 | 状态 |
|----|------|------|
| JS/JSON 语法 | `find apps/weapp -name '*.js' \| node --check` 全过 + JSON.parse 全过 | ✅ |
| QR 编码 | node 冒烟 29×29 | ✅ |
| 端点对照 | utils/api.js 路由与后端 module.go 逐条核对 | ✅ |
| 纯逻辑单测 | `node apps/weapp/tools/unit.js` → **PASS=19 全绿** | ✅ |
| 真实编译+运行 | `cli auto` + IDE 2.02.2608070 编译，8 页可达 | ✅ |
| 登录+绑定+会员 | 早期 walk1 12 PASS（SMS 123456→token→bind→me 渲染）；现 `devtools-verify.js login/me` stage 复验 | ✅ |
| 匿名→引导登录→自动登录 | 早期 walk3 10 PASS；现 login stage 复验 | ✅ |
| 核销码页数据渲染 | IDE 截图（时钟/空态卡/就绪数据） | ✅ |
| **预约提交→成功** | `book-fuzzy`（HALF_DAY：单号 APT202609280015）+ `book-slot`（SPECIFIC 10:00：APT202609280016）PASS | ✅ |
| **我的预约列表 + 取消** | `apt-list`（新单可见/状态“待到店”/服务名快照/可取消/DOM 与数据条数一致）+ `apt-cancel`（→CANCELLED、不可再取消）PASS | ✅ |
| **今日预约单码 ANMO-APT** | `book-today`（APT202609280017）+ `qrcode`（今日单出现在单码列表、服务名补齐）PASS | ✅ |
| **D21 会员码门槛（正反）** | 无卡会员：`.qr-card`=0、canvas 数=应出示数；持卡会员 13990497037：`hasActiveCard=true` + 会员码渲染 PASS | ✅ |
| **卡流水符号** | `cards` stage 展开明细断言 REDEEM→`-N`、ISSUE/REVERSAL→`+N`、ADJUSTMENT 带符号（真实数据 ISSUE:10/REDEEM:1×2/REVERSAL:1） | ✅ |
| **今日已过槽不展示** | `book-today` stage 断言（trimPastSlots 生效） | ✅ |
| **首页 blocks 编排** | `home` stage：页面顺序 == `homeBlocks(后端 blocks)` | ✅ |
| **关于页字段=接口值** | `about` stage：address/phone/home_body 与 `GET /api/settings` 逐字段对账 | ✅ |
| 后端不变量 | `go build/vet/test ./... -count=1`（带 `ANMO_TEST_MYSQL_DSN`）14 包全绿 | ✅ |

最近三次实测日志（本地 `.dev/`，已 gitignore 不入库）：`verify3.log`（全 11 stage，PASS=53 FAIL=0）、`verify4.log`（持卡会员 13990497037 跑 login+qrcode+cards，PASS=13 FAIL=0，覆盖 D21 正分支与真实流水）、`verify5.log`（修完截图路径后复跑，PASS=50 FAIL=0）。截图证据落 `.dev/shots/*.png`。

**后端业务不变量**由 e2e/identity 测试硬覆盖，与小程序桥状态无关。

### 7.1 已知受阻/非 bug 事项（如实记录）
- 门店地址/电话在开发库为空（`content_system_setting` 只有 `store_phone`/`close_time`，规范键是 `shop_phone`/`shop_address`）→ 关于页走空态文案；实测已改为“页面值=接口值”对账，不再把数据缺失当 bug。
- 后台 `blocks` 只编排了 `banner + service_list` → 公告不展示，**与 H5 行为一致**（banner 依赖外部图片域名，小程序端不渲染）。
- 自动化桥仍在 `cards` stage 出现过一次 `timeout waiting for automator response`，按 stage 重连第 2 次即通过（脚本内置 `ANMO_ATTEMPTS=3`）。

---

## 八、剩余工作清单（按依赖顺序）

1. ~~补 Phase D 验证~~ ✅ **完成**：`tools/devtools-verify.js` 11 stage 全跑通（PASS=53 FAIL=0），预约提交/成功卡/我的预约/取消/具体槽/今日单码/D21 正反分支/D20 已过槽全部有实测证据（§7）。
2. ~~Phase E UniApp~~ ⛔ **用户指示不做**（§5 方案保留）。
3. **收口** ✅ 完成：
   - 后端：`go build ./... && go vet ./... && go test ./... -count=1`（带 `ANMO_TEST_MYSQL_DSN`）14 包全绿，无 SKIP。
   - 文档：`apps/weapp/README.md`（页面↔端点对照、验证两层跑法、发布前清单、已知边界）；本 HANDOFF 进度/证据/环境已更新；`.trellis/workspace/anmo-agent/journal-1.md` 追加 `# PHASE RESULT — V2`。
   - `AGENTS.md`：D23–D26 已补进主冻结决策表（本轮完成）。
   - `.gitignore`：新增 `.dev/`（本地二进制与实测日志）。
4. **部署链** ⬜ **未执行——需用户明确点头**（提交/推送/生产部署属共享状态操作）。V1.x 同规矩：
   - GitHub 小步 commit 建议：`feat(v2): 微信登录后端（migration 013 + identity wx 端点 + member openid）` → `feat(v2): 原生顾客端小程序 apps/weapp（含 tools 实测脚本）` → `fix(customer): 卡流水符号方向与小程序同口径` → `docs(v2): HANDOFF/README/journal 收口`；推 main
   - yun1：WSL 构建 linux/amd64 二进制→Dockerfile stage→docker save|scp|load→force-recreate；**migration 013 自动应用验证（count=1）**；wx 端点生产 smoke（needs_bind）；`health-yun.sh` ALL GREEN；小程序端**不需要**部署（IDE 上传，需真实 AppID + 合法域名）
   - Tencent：`git pull --rebase origin main`
5. **用户验收提示**（交付时给）：小程序导入 `apps/weapp`（touristappid）+ `config.js` BASE_URL 指后端；发布前按 README 清单配 AppID / request 合法域名（HTTPS+备案）/ `ANMO_WX_APPID`+`ANMO_WX_SECRET`（否则走 D24 dev 兜底，禁止发布）。

### 8.1 本轮明确不做/留待决策（歧义选不做）
- **不做 uni-app（用户指示）**：Phase E 取消；原生小程序是 V2 交付主体，UniApp 方案仅保留备查（§5）。
- **后端不回溯裁剪“已开始”的槽**：改 `booking.go` 会同时改变 H5 与小程序口径（D20 邻域），现由小程序 `format.trimPastSlots` 前端收口；若希望两端根治，需产品决策后单开任务（同时修 H5）。
- **H5 同源缺陷**：`trimPastSlots`（今日已过槽仍显示、点击吃 409）与 blocks 之外的小程序改进未反哺 H5；本轮只顺手修了 H5 **卡流水符号**（与小程序必须同口径，且是顾客可见账目错误）。
- **banner 内容块**：依赖外部图片域名与合法域名配置，小程序端不渲染（发布后如需，加 `image` 组件 + 域名白名单即可）。
- **资料编辑（生日/性别）、列表分页超过 50 条**：不做。
- **商家端 MembersPage “±次数”列**直接显示 `card_transaction.quantity` 原值（REDEEM 显示为正）：属后台口径问题，非本次范围，留待用户决定。
- **开发库残留测试数据**：本轮实测新建的预约（APT202609280015 已取消、0016/0017 及更早 session 的测试单）留在本地 dev 库未清理；生产库无此数据。

---

## 九、硬约束速查（防新会话跑偏）

- 模块边界：跨模块只准调 api.go；member/identity 约束见各自 AGENTS.md（本轮已更新公开 API 清单）。
- 顾客身份只从 Token 取（D25 bind 需顾客 Token；禁信前端 member_id）。
- 顾客 API 全部需 JWT（现有 V1 口径）；未登录→登录引导（H5/小程序同口径）。
- D24 dev 兜底仅无凭据开发环境；小程序发布前必须 `ANMO_WX_APPID/SECRET`；**凭据/证书私钥禁止入库**。
- 禁止 Plan 外实体：staff/多租户/积分/优惠券/微信支付 API/微信订阅消息/微信服务卡片官方能力/云开发（CloudBase MCP/Skill 本项目**不适用**——用户给的云开发文档指向 CloudBase，本项目不用云数据库/云函数，已确认跳过）。
- yun1 红线：1.6G 内存禁本机构建、禁 Node；凭据在 yun1:/opt/anmo/credentials.txt(600)。
- 商店状态三态不落库；金额整数分；ID CHAR(26) ULID。
- SMS：dev 固定 123456，60s/手机号限流、一次性。
- 状态机（V1.x）：WAITING→IN_SERVICE→COMPLETED；WAITING→CANCELLED|NO_SHOW；cancel/reschedule 仅 WAITING。
- 测试必须带 `ANMO_TEST_MYSQL_DSN`（否则 t.Skip 掩盖失败）。

---

## 十、相关链接

- 用户提供：微信开发者工具 skill https://skillhub.cn/skills/tencent-adm/wechatide-skill ；安装引导 https://skillhub.cn/install/skillhub.md
- CloudBase（不适用，仅记录）https://developers.weixin.qq.com/miniprogram/dev/wxcloudservice/wxcloud/guide/development-assistant.html
- npm 镜像：https://registry.npmmirror.com（uni-app 依赖版本线已验证，见 §5）
- Trellis PRD：.trellis/tasks/09-28-v2-miniprogram/PRD.md（Phase C/D/E 范围 + 不做清单 + 验收清单，本文与之冲突时以 PRD/AGENTS.md 为准）
