# Anmo 顾客端微信小程序（V2 原生版）

一套后端（Anmo Go `/api`），多个前端。本目录是**原生微信小程序**（零构建：源码即产物，微信开发者工具直接导入）。顾客 H5 在 [apps/customer](../customer)（Vue3），两者共用同一后端与同一会员身份。

## 页面与业务对照

| 页面 | 业务 | 后端端点 |
|------|------|----------|
| pages/home | 门店状态徽标 / 首页文案 / 服务列表 / 门店卡 | `/api/store/status` `/api/settings` `/api/services` |
| pages/booking | 服务→日期→上下午必选→可选具体时间（D20） | `/api/booking-options` `POST /api/appointments` |
| pages/appointments | 我的预约 + 取消 + 改期抽屉 | `/api/appointments*` |
| pages/cards | 会员卡 + 使用明细展开 | `/api/me/cards` `/api/me/cards/{id}/transactions` |
| pages/qrcode | 会员码 `ANMO-MEMBER:<ulid>`（D21 门槛）+ 今日预约单码 `ANMO-APT:<ulid>` | `/api/me/*` `/api/appointments*` |
| pages/login | 微信一键登录（openid 未绑定直建号发 Token，手机号撞号走 claim，D25/D27） | `/api/auth/wx/login` `/api/auth/wx/claim` |
| pages/me / pages/about | 个人中心 / 门店信息（导航+拨号） | `/api/me/profile` `/api/settings` |

分享闭环：所有业务页 `onShareAppMessage`；home/about 另有 `onShareTimeline`；`app.js onLaunch` 开启 `showShareMenu`。

## 使用

1. 微信开发者工具 → 导入本目录（`project.config.json` 就绪，appid 默认 `touristappid`，正式使用时替换为你的 AppID）。
2. 详情 → 本地设置 → 勾选"不校验合法域名"（开发期）。
3. `config.js` 会按运行环境自动选后端地址，不用手改：模拟器 → `http://127.0.0.1:8080`；真机 → `http://192.168.2.224:8080`（电脑 WLAN IP，DHCP 变了要同步改 `LAN_URL`）。
   - **真机必须走电脑局域网 IP**：手机上的 `127.0.0.1` 是手机自己，直连必报 `request:fail`。
   - WSL2 **Mirrored** 网络下，Windows 侧只有 loopback 默认能进 WSL；给手机用需放行 Hyper-V 防火墙入站（不是 Windows 防火墙）：`scripts/allow-wsl-lan.ps1`（管理员，默认只放行 TCP 8080 且限本机子网，`-Remove` 撤销）。
   - 自测不要用 Windows 本机去连自己的 LAN IP（mirrored 下会 hairpin 超时，属假阴性）；用另一个 netns 验证：`docker exec anmo-mysql curl -s -o /dev/null -w '%{http_code}\n' http://<PC_IP>:8080/healthz` → 200 才算通。
   - 真机优先用「真机调试」；「预览」在部分版本会强制 request 合法域名校验。

## 开发者工具联调

面向店主/开发者自己的联调步骤。先说常见「旧包」三症状：**tabBar 少「服务」、登录页还是手机号+验证码、手机号占位符只显示一半** —— 三者同源，都是开发者工具里跑的旧版代码：当前代码登录页已是纯微信一键登录（无手机号/验证码表单），tabBar 四项齐全（首页/服务/预约/我的）。**重新导入 + 清缓存即消失**，不需要改任何代码。

1. **重新导入项目**：项目目录 `/mnt/Projects/Anmo/apps/weapp`（Windows 侧开发者工具填 `\\wsl.localhost\<发行版>\mnt\Projects\Anmo\apps\weapp`）。AppID 选「测试号」即可跑通全流程。
2. **先清缓存**：工具栏 → 清缓存 → 全部清除。这是修复旧包三症状的唯一操作。
3. **放行请求域名**：详情 → 本地设置 → 勾选「不校验合法域名」。devtools 里 config.js 默认打 `http://127.0.0.1:8080`。
4. **起后端**：WSL 里执行 `bash scripts/start-anmo.sh`，一条命令拉起后端 :8080 + 顾客 H5 :5173 + 商家端 :5174/5175。
5. **登录为什么直接就成功（D24 dev 兜底）**：后端未配置 `wx.app_id` 时，登录走 dev 兜底，`openid = "dev:" + code`，首次登录同事务直建会员号并直发 Token（后端启动日志有警示）。**正式发布前必须在生产配置 `ANMO_WX_APPID` / `ANMO_WX_SECRET`**，dev 兜底禁止用于发布。
6. **真机预览/真机调试**：走 `LAN_URL`（`config.js` 常量，需与电脑当前局域网 IP 一致，DHCP 变了要同步改），且 Windows 侧需放行 LAN→WSL 入站：`scripts/allow-wsl-lan.ps1`（管理员执行，`-Remove` 撤销）。
7. **可选：不改代码覆盖后端地址**（含指向生产/其他机器）：开发者工具 Console 执行后重启小程序：

   ```js
   wx.setStorageSync('anmo.base_url', 'http://192.168.x.x:8080')
   ```

   storage 覆盖优先级最高（值非法自动忽略回落默认）；恢复默认用 `wx.removeStorageSync('anmo.base_url')`。注意：指向生产库的验收测试会写入**真实数据**，慎用。

## 验证

零构建意味着没有 `tsc`/打包器兜底，改动后跑单测（无依赖，秒级，任何机器可跑）：

```bash
node --test "apps/weapp/tests/*.test.js"
```

覆盖 `config.js` 的 BASE_URL 解析：storage 覆盖（`anmo.base_url`）优先、非法值（非 http(s)、含空白）忽略、devtools→LOOPBACK / 真机→LAN / 无 `wx` 环境兜底。
注意：部分 Node 版本（如 WSL 下 v24）`node --test <目录>` 的目录形式会报 `MODULE_NOT_FOUND`，请用上面的 glob 形式。

端到端实测（模拟器/真机）按上文「开发者工具联调」章节在微信开发者工具里人工操作。

## 发布前清单

- [ ] `project.config.json` 换成真实 AppID
- [ ] 后端配置 `wx.app_id` / `wx.secret`（env `ANMO_WX_APPID`/`ANMO_WX_SECRET`），否则登录走 dev 兜底（D24，**禁止用于发布**）
- [ ] 小程序后台配置 request 合法域名（必须 HTTPS + 备案域名，自签 IP 不可用）
- [ ] yun1 生产 HTTPS 反代 + 域名就绪后再提审

## 已知边界（歧义选不做）

- 微信支付 API、订阅消息通知、微信官方"服务卡片"（类目/资质）：不做
- 二维码由前端 `qrcode-generator`（vendor 于 `utils/qrcode.js`，MIT）+ canvas 2d 本地生成，不依赖后端出图
- `/api/booking-options` 不回溯裁剪"已开始"的槽（H5 同口径）：前端用 `format.trimPastSlots` 在"今天"这一天把它们隐藏，后端逻辑保持共享不动
- 首页内容块顺序（§31/D16 的 `content_page_config` blocks 编排）：与 H5 一致按 公告 → 服务 的固定顺序渲染，不做后台可配排序
- 生日等资料编辑、列表分页（超过 50 条）：不做
