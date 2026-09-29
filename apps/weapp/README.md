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
| pages/login | 微信静默登录→短信绑定（plan §11） | `/api/auth/wx/login` `/api/auth/sms/*` `/api/auth/wx/bind` |
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

## 验证

零构建意味着没有 `tsc`/打包器兜底，所以改动后跑两层：

1. **纯逻辑单测**（无依赖，秒级，任何机器可跑）：
   ```bash
   node apps/weapp/tools/unit.js
   ```
   覆盖 display 口径（`format.statusText/yuan/aptTime`）、流水符号方向（`REDEEM` 扣次显示 `-N`）、`trimPastSlots` 过期槽裁剪，以及 `slot-picker` 门槛（闭店/时段已过/余量为 0 不可选）——后者通过 `global.Component` 桩直接驱动组件逻辑。
2. **IDE 端到端实测**（真机运行时，需微信开发者工具在 Windows 侧）：
   ```bash
   # 1) 后端在跑（:8080）+ MySQL 起来；2) 同步到 Windows 本地盘（IDE 不认 \\wsl.localhost 路径）
   rsync -a --exclude node_modules apps/weapp/ /mnt/d/anmo-weapp-build/
   # 3) 开自动化桥（先 quit 再起，否则端口不刷新）
   cmd.exe /c "D:\\Program\\soft\\wechattools\\cli.bat quit"
   cmd.exe /c "D:\\Program\\soft\\wechattools\\cli.bat auto --project D:\\anmo-weapp-build --auto-port 9420"
   # 4) 11 个 stage：登录→首页→模糊预约→我的预约→取消→具体槽→今日单→核销码→卡→我的→关于
   cd apps/weapp/tools && npm i --registry https://registry.npmmirror.com && node devtools-verify.js
   ```
   可用环境变量：`ANMO_MP_WS` / `ANMO_TEST_PHONE` / `ANMO_TEST_CODE`（dev 固定验证码）/ `ANMO_API_BASE`（默认 `http://127.0.0.1:8080`，用于"页面值=接口值"对账）/ `ANMO_SHOT_DIR`（截图落地，默认仓库 `.dev/shots/`）/ `ANMO_ATTEMPTS`（每 stage 重连次数）。失败返回非 0。
   - **该 devtools 版本（2.02.2608070）自动化桥有会话衰变**（每连约 2-3 个命令后必断），所以脚本按 stage 逐个重连；若报 `page is not on top of page stack` 属桥问题，不是应用 bug。
   - **自定义组件内部节点够不到**（`page.$('slot-picker')` 返回 null），因此对 `slot-picker` 的交互走受控回调 `page.callMethod('onPick', {detail})`，组件自身行为由 `unit.js` 覆盖。
   - 原生弹窗用 `mp.mockWxMethod('showModal', {confirm:true})`，用完 `restoreWxMethod`。

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
