# 微信开发者工具自动化运维手册（V2.2 第七批沉淀）

> 目标：任何一次会话都能在 3 条命令内重建「真实 DevTools → 生产 API」的小程序全流程自动化验收。
> 适用：本仓库 apps/weapp（V2 原生零构建小程序）。沉淀于 2026-09-30 第七批。

## 1. 环境拓扑（一次性，已就绪）

```
WSL2 (ZCode / Anmo 仓库)
  └─ cmd.exe 跨界调用 ──► Windows 微信开发者工具  D:\Program\soft\wechattools\
       ├── HTTP V2 服务端口 19225（进程级：open/close/preview/upload/islogin/quit）
       ├── 自动化 ws 端口 9420（页面级：automator 协议）
       └── 项目副本  C:\Users\fengx\anmo-weapp\（WSL 路径 /mnt/c/Users/fengx/anmo-weapp）
```

- 端口写在 `%APPDATA%/../Local/微信开发者工具/User Data/<hash>/.ide`（默认 19225）。
- WSL2 mirrored 网络下 `localhost` 双向可达，WSL 侧直接 `curl http://localhost:19225/v2/islogin`。
- **副本是测试专用**：与仓库 apps/weapp 同步，但 `config.js` 的 `LOOPBACK_URL` 已改为 `https://anmo.oiob.cn`（devtools 模拟器天然命中 LOOPBACK 分支）。仓库真码不动，见 §4。

## 2. 两层自动化（官方机制，非 hack）

### 层1：HTTP V2（进程级，不开窗口也能用）

```bash
curl http://localhost:19225/v2/islogin        # {"login":true}
curl http://localhost:19225/v2/quit           # 完全退出 IDE（下次冷启动清 storage！）
cmd.exe /c "D:\Program\soft\wechattools\cli.bat preview --project C:\Users\fengx\anmo-weapp"
```

### 层2：miniprogram-automator（页面级，须先开自动化会话）

```bash
# 启动自动化会话（会打开 IDE + 项目 + 9420 ws）。阻塞式，放后台跑。
cmd.exe /c "D:\Program\soft\wechattools\cli.bat auto --project C:\Users\fengx\anmo-weapp --auto-port 9420" &
# 端口就绪探测（426=ws 服务在）
curl -s -o /dev/null -w "%{http_code}" http://localhost:9420
```

```js
// /tmp/anmo-b7/test.cjs 剧本骨架（npm i miniprogram-automator 的目录里跑）
const automator = require('miniprogram-automator')
const mp = await automator.connect({ wsEndpoint: 'ws://localhost:9420' })
await mp.callWxMethod('clearStorageSync')          // 强制游客态起点
await mp.reLaunch('/pages/home/home')
const page = await mp.currentPage()                // 页面路径判定
const d = await page.data('services')              // 读页面 data
await page.callMethod('skipProfile')               // 调页面方法（比 tap 选择器稳）
await mp.mockWxMethod('showModal', { confirm: true, cancel: false, errMsg: 'showModal:ok' })
mp.on('console', ...)                              // 收集 console.error
await mp.disconnect()
```

## 3. 踩坑实录（每条都真踩过）

| # | 坑 | 结论 |
|---|----|------|
| 1 | **storage 不跨 `cli auto` 冷重启**：prep 写入的 `anmo.base_url` 重启后为空 | 想改请求基址不要走 storage，走 §4 副本 config.js |
| 2 | **`clearStorageSync` 不清 app 内存登录态**：app.js `markAuth()` 置的 `globalData.loggedIn` 仍在，`ready()` 同步回调 true | 模拟「token 过期」正要靠这一点：页面级 401 catch 必须落游客引导（本批 appointments 修复） |
| 3 | **tabBar 页不能 `navigateTo`**：home/services/booking 是 tab | 一律 `mp.switchTab` / `mp.reLaunch` |
| 4 | **config.js 的 BASE_URL 是模块加载快照**（request.js 顶层 require） | 改基址必须重启编译，热重载无效 |
| 5 | **模拟器点击被遮挡/多元素同名**：`.sheet-ops button` 之类要先缩窄作用域 | 优先 `page.callMethod('handler', {...})` 直接驱动，选择器只用于展示断言 |
| 6 | **新会员首提预约会先弹 R4 资料半屏**（V2.2 第五批设计） | 剧本里 tap 提交后查 `profileSheet`，true 则 `callMethod('skipProfile')`（自带续提交） |
| 7 | **测试单遗留生产**：E2E 创建的 WAITING 单不会自己消失 | 剧本收尾 T8b：mock showModal 后遍历 `list` 中 `canCancel` 全部取消 |
| 8 | cmd.exe 从 WSL 路径启动报 UNC 不支持 | 正常现象，自动落到 Windows 目录，不影响 `cli.bat` 绝对路径调用 |
| 9 | node 侧 fetch 生产偶发 `fetch failed` | 一切外呼包一层 3 次重试（剧本 `fetchRetry`） |
| 10 | 改仓库代码后副本未同步 | 只 `cp` 变更文件到副本；**别整目录 rsync**，会覆盖副本专属 config.js。**批 8 实锤后果**：`rsync --delete` 覆盖后 LOOPBACK 回落 `127.0.0.1:8080`，15/16 断言全绿但打在本地 dev 库（假绿）——识别指纹：`prod` 与 `wx` 服务名完全不同、新单号序号比历史倒退（APT…0009 < 批 7 的 0011）。批量同步必须 `rsync --exclude config.js`（§4 命令已改） |
| 11 | **`.page-body` 入场动画 `fill-mode: both` 让 fixed 弹层整体错锚**：动画终态 transform 残留 → page-body 持续充当 fixed 子元素包含块，`.mask` 量得 856px（>视口 671），弹层底部（保存按钮）沉到屏幕外 | 动画去 `both`（无 delay 无需保终态）即可，keyframes `to` 保持 `transform: none`。同类隐患：booking submit-bar / R4 弹层 / 改期抽屉，一改全修 |

## 4. 重建/重跑（3 条命令）

```bash
# 1) 同步仓库 → 副本（排除 config.js——副本专属指向生产，见坑 #10）
rsync -a --exclude config.js /mnt/Projects/Anmo/apps/weapp/ /mnt/c/Users/fengx/anmo-weapp/

# 2) 起/重启自动化会话（若 IDE 已开着同一项目，先 /v2/quit 再起）
cmd.exe /c "D:\Program\soft\wechattools\cli.bat auto --project C:\Users\fengx\anmo-weapp --auto-port 9420" &

# 3) 跑全流程 16 断言（T0 版本 → T9 核销码门槛，T8b 自动清理遗留单）
cd /tmp/anmo-b7 && node test.cjs
```

剧本 16 断言（第 16 项见 §5 修复回归 T10，独立脚本 `t10-guest-apts.cjs`）：
T0 about 版本=V2.2.6 → T1/T1b 游客首页数据与生产 catalog 实时一致 → T2 服务分组 → T3 选服务/日期/下午 → T4 未登录提交跳登录 → T4b 微信 code2session 真登录 → T5b 预选保留 → T6(R4 半屏→)提交成功拿到预约号 → T7 列表可见 → T8 mock 弹窗取消 → T8b 遗留清零 → T9 核销码门槛。

## 5. 第七批对抗审查修复（已验证）

| bug | 根因 | 修复 |
|-----|------|------|
| 游客首页服务列表为空 | home.js 游客分支直接清空 `services`，还是「内容接口全是 JWT 资源」的旧口径；目录 API 早已对游客开放（第五批），services 页/H5 都开放，唯首页漏改 | home.js 抽 `applyCatalog()` 共用，游客分支 `loadGuestCatalog()`；wxml 骨架+服务区常驻，shop-card/快捷入口包进 `loggedIn` |
| 游客开「我的预约」显示「网络不可用」 | 401 走进 catch 的网络错误分支；凭证过期（storage 清了/服务端失效）同样中招 | appointments.js catch 判 `e.needLogin` → 落 `guest:true` 登录引导卡；wxml 新增引导卡 + 空态排除游客 |

## 6. 官方 Skill 评估（2026-09-30，文章：juejin 7659331189153333298）

结论先行：**可行，且本机构建已内置**（`resources/app.asar.unpacked/wechatide-skill/`，v0.3.11），入口 `D:\Program\soft\wechattools\wechatide.cmd`。不必再从 skillhub 下载（那里要 API key）。

- 定位：官方给 AI Agent 的「看得见模拟器」工作流——编译报错、console/network 取证、`simulator_screenshot` 截图、页面自动化、预览/上传、云开发，9 个 scene（installer/initializer/project-manager/project-config/compiler/previewer/automator/debugger/cloudbase-operator）。
- 与本仓库现状的关系：官方 automator/debugger scene 的底层与 §2 层2 同源（automator 协议），**不是替代是封装升级**；它把我在剧本里手写的「截图取证、console 收集、编译验证、异步任务轮询」标准化成 `wechatide -c <client> <tool>` 一条命令。
- 效率对比：
  - 交互式开发/排障（看编译错、截模拟器、读 console）：**官方 Skill 更高效**，免写脚本。
  - 可重复的 16 断言回归套件：**剧本脚本仍更合适**（确定性、可进 CI/批处理），Skill 的 automator scene 也能做但无套件概念。
  - 推荐：日常排障/改动验证用官方 Skill；批次验收跑剧本；两者共用同一 IDE 实例。
- 已验证的调用链（2026-09-30 实测全通）：
  1. 门禁：`wechatide.cmd -c zcode check_wechatide_status --skill-version 0.3.11` → 首次触发 IDE 授权弹窗（返回 `taskId` + pending，用 `polling_task_result` 轮询；用户在 IDE 点一次允许后 `versionRelation: "equal"`、`loginExpired: false`、CLI 模式 `tokenRequired: false`）
  2. 开窗：`wechatide.cmd -c zcode open_project_window --project "D:\anmo-weapp-build"` → `{winId: "s1"}`
  3. 截图：`wechatide.cmd -c zcode simulator_screenshot --project ... --path ...` → 289×625 JPEG 落盘，导航/tab/内容区完整可见
- **路径坑**：skill 工具只认「IDE 项目注册表里已登记的路径」（`project_list` 可查）。`C:\Users\fengx\anmo-weapp` 磁盘存在但未登记 → 一律 `PROJECT_PATH_NOT_FOUND`（连 project_import 也拒）；登记过的 `D:\anmo-weapp-build` 直接可用。新目录先经 IDE/cli 正常打开一次（进入注册表）再用 skill 工具。
- `mcpTokenRequired: true` 仅 MCP 模式需要（设置→安全复制 MCP 配置）；`-c` CLI 模式免 token。
- 效率结论（实测后确认）：交互式排障/取证**一条命令替代一个脚本**（开窗→截图→读 console→轮询异步任务，全部标准化），比手写 automator 脚本快；回归套件仍用 §4 剧本。

## 7. 相关

- skillhub 上的 tencent-adm/wechatide-skill 页面是同一 skill 的分发渠道，需 skillhub API key；本机内置版优先。
- 生产 API：https://anmo.oiob.cn（CF Tunnel，见 `/mnt/Projects/CloudFlare/Cloudflare-CLI-Runbook.md`，三设备同步）。
- 微信凭据只进 yun1 的 `/opt/anmo/docker-compose.yml`（ANMO_WX_APPID/ANMO_WX_SECRET），严禁入仓。
