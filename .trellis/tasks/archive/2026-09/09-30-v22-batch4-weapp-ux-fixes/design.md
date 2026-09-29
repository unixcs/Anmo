# 设计 — V2.2 第四批：预约资料弹层 UX 翻转 + 小程序登录/联调修复 + 全流程深测

## 事实核查结论（对用户三问的答案）

| 用户观察 | 事实 |
|---|---|
| tabBar 缺「服务」 | **旧包**。当前 app.json tabBar = 首页/服务/预约/我的，齐全。开发者工具重新导入 `apps/weapp` + 清缓存即恢复 |
| 登录回退到手机号+验证码界面 | **旧包**。当前 login 页是纯微信一键登录（无短信表单）。短信下线是第二批已拍板决策（D27，AGENTS.md 冻结），有意为之 |
| 手机号占位符只显示一半 | **旧包**。v2.1.0 已修（`.input` 显式 height:48px，app.wxss:365） |
| 点击登录连不上 127.0.0.1 | **真问题但非 bug**：devtools 模拟器跑在 Windows，走 loopback 进 WSL；后端没在本机跑自然连不上。本批给一键启动 + README 联调章节 |
| H5 资料保存后不返回预约页 | **真 bug**（ProfilePage.save() 无导航）。本批修复 |
| 弹层按钮顺序（去完善在上且凸显） | **真问题**（BookingPage.vue:386-387）。本批翻转 |

## 对齐默认（用户未及回答，按推荐口径执行）

1. **登录方式**：保持纯微信一键登录（D27 既定）。恢复短信仅当用户明确要求。
2. **devtools 联调**：默认 devtools→`http://127.0.0.1:8080`（现状），另支持 wx storage 键 `anmo.base_url` 覆盖成任意合法 http(s) 地址（含局域网/生产，README 写明风险）。真机仍走 LAN_URL。

## 修复规格

### F1 H5 弹层翻转（BookingPage.vue）

- 模板 383-389：**立即预约**（`variant="primary"`，block）在上 → 行为 = 原 `skipProfileSheet`（置 `anmo.profile.bookingSkipped` 永久静默 + 继续 submit）；**去完善资料**（`variant="ghost"`，block）在下 → 行为 = 原 `goProfileSheet`（存 sessionStorage 草稿 + `router.push('/me/profile')`）。
- 函数改名对齐语义：`skipProfileSheet` → `bookNow`，`goProfileSheet` 保留；sheet 文案微调（正文一行说明「先完善资料，店主更好安排；也可直接预约」）。
- 不变：草稿结构、takeDraft 恢复、跳过标记键名、submit 主流程。

### F2 资料保存后返回（ProfilePage.vue）

- `save()` 成功分支：`sessionStorage.getItem('anmo.booking.draft')` 存在 → `router.push('/booking')`（BookingPage onMounted 会 takeDraft 恢复草稿并回填）；无草稿 → 留在原地（普通资料编辑不受影响）。
- 通知顺序：先 `notify('已保存')` 再跳，或跳后由 BookingPage 提示恢复——任选其一，保证有且只有一次明确反馈。

### F3 小程序 BASE_URL 覆盖（apps/weapp/config.js）

- `resolveBaseUrl()` 增加最高优先级：`wx.getStorageSync('anmo.base_url')` 为合法 `http(s)://…` 字符串时直接采用（trim）；非法/空值忽略，回落平台默认（devtools→LOOPBACK_URL，否则 LAN_URL）。
- 导出 `STORAGE_KEY` 常量供 README/测试引用；平台探测 try/catch 语义不变。

### F4 联调文档 + 提示修正

- `apps/weapp/README.md` 新增「开发者工具联调」章节：项目路径重导入（Windows 侧 `\\wsl.localhost\…\apps\weapp` 或 WSL 内路径）、AppID 用测试号、详情→本地设置→勾选「不校验合法域名」、**清除缓存（修复旧包三症状的唯一操作）**、后端一键启动 `bash scripts/start-anmo.sh`、D24 dev 登录原理（`wx.app_id` 未配置时 openid=`"dev:"+code` 一键建号，发布前必须配真实凭据）、真机预览需 LAN_URL 与电脑 IP 一致 + `scripts/allow-wsl-lan.ps1`、覆盖 BASE_URL 的 console 命令示例。
- `scripts/start-anmo.sh` 尾部提示行：「顾客短信验证码(dev): 123456」过时 → 改为 H5 手机号+密码口径（V2.2 D27）。

### F5 测试与深测

- `apps/weapp/tests/config.test.js`（node:test，require 前 stub `global.wx`）：覆盖 storage 覆盖生效/非法忽略/devtools→LOOPBACK/真机→LAN/无 wx 兜底。`node --test apps/weapp/tests/` 全绿。utils 里若有纯函数（format.js）可顺带补，不强制。
- trellis-check 对抗式审查：H5 弹层/跳过标记/草稿闭环 + 小程序 auth 链路（login/claim/401 重试）、booking/cancel/reschedule、QR 门槛。
- 我做浏览器 E2E：H5 真实注册→预约→弹层顺序与配色→去完善→保存→自动返回且草稿恢复→提交成功；跳过标记第二次不再弹。

## 不做

- 不恢复短信/验证码登录（D27）；不把生产地址设为默认；不改 submit 主流程与状态机；不动后端。
