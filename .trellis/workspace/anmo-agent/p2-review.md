# P2-0 双端摸底问题清单（2026-09-28）

> 审视方式：H5 = Playwright 移动端视口（390×844）逐页截图；小程序 = miniprogram-automator 连微信开发者工具（9420）逐页截图。
> 截图：/tmp/p2-review/shots/（m1/m2/m2fix=H5，w1/w2=weapp）。数据：本地 MySQL 迁移出的 SQLite（/tmp/p2-review/anmo-dev.db）。

## A. 功能性 Bug（必须修）

1. **H5 QRCodePage 无参调用 `api.myAppointments()`** → `status=undefined` → 后端返回 `data:null` → 今日预约**永远显示「今日没有待到店的预约」**（13712340000 有 4 个今日 WAITING 也不显示）。weapp 同签名但 `status||''` 兜底为全量，故只 H5 中招。
2. **多核销码 bug（任务书核心）**：两端 QR 页为每个今日 WAITING 预约渲染独立 `ANMO-APT:<id>` 二维码 + 会员码 `ANMO-MEMBER:<id>`。实测：2 预约=3 码（m2fix-qrcode.png）、4 预约=4 码+（w2-qrcode.png）。顾客无从知道该出示哪张，业务逻辑不严谨。
3. **H5 CardsPage 无卡崩溃**：`cards.value = await api.myCards()`，无卡时 data:null → `[...null]` TypeError（pageerror 实录）。
4. **H5 QR 页服务名显示为空**：`d.services?.[0]?.service_name_snapshot` 解包层级错（详情接口包 `{data:{appointment,services}}`），显示「今日预约 · 服务 · 14:00」。
5. **H5 我的预约卡片不带服务名**（列表接口无 name，页面未补齐；weapp 补了）。

## B. E2E/UI 验证脏数据（必须清洗）

- 服务分类×5 全叫「UI验证分类」、服务项「UI验证肩颈60分钟」（描述 ui e2e）、「扫码E2E足疗…1790502972」等；卡模板「UI验证10次卡」「扫码E2E卡」「v1x散客卡」；会员「UI验证顾客179…」「扫码E2E顾客B…」；banner「UI验证Banner」（图挂，alt 裸露）；announcement 同类。
- 13712340000：12 WAITING + 7 CANCELLED 预约（E2E 灌的）。两台端全部原样渲染。
- 清洗范围：yun1 生产库迁移时过滤/清理测试数据（任务书明确要求），并给 banner/图片缺失做优雅降级。

## C. 视觉/设计体系缺失（P2-1..P2-5 主战场）

- **两端调性不一**：H5 暗沉砖红 hero + 灰白卡；weapp 米色（#faf9f7/#c85f5f）暖调。无共享 token。
- **图标全靠 emoji**：H5 tab bar 🏠💆📅👤、菜单列表 emoji、🔒 锁；weapp 同病。无图标体系、无品牌感。
- 排版：无层级节奏（标题/正文/辅助字号随意）、卡片间距不均、大面留白、无阴影/描边体系。
- 组件原始：按钮只有 disabled 变粉色、无按压缩感、无骨架屏、toast 原生。
- Login 页无品牌元素；About/History 空洞。
- weapp 信息架构反而更好：预约列表有筛选 chips、服务名+时长、「去核销码页」直达；重构时保留其 IA 思路。

## D. 冗余代码清理

- `apps/weapp/tools/`（devtools-verify.js、unit.js、node_modules、package-lock）为 V2 验收期 E2E 脚手架，混在 app 源码树里 → 移除或外移。
- H5/`apps/customer`、weapp 中 console 调试残留待扫。

## E. 复现环境备忘（迭代期间复用）

- 后端：`cd server && ANMO_DB_PATH=/tmp/p2-review/anmo-dev.db nohup /tmp/anmo-sqlite-bin -config server/config.example.yaml &`
- H5：`cd apps/customer && npm run dev`（5173）
- 小程序桥：`cmd.exe /c "D:\Program\soft\wechattools\cli.bat auto --project D:\anmo-weapp-build --auto-port 9420"`（先 `cp -r apps/weapp/{app.js,app.json,app.wxss,config.js,pages,components,utils,sitemap.json} /mnt/d/anmo-weapp-build/`）
- 截图脚本：/tmp/p2-review/tour.mjs（H5）、weapp-tour2.js（小程序）；token 用 `/api/auth/sms/send`+`/verify`（dev 123456，60s 限频）注入 `anmo.customer.token` / storage。
- 复现多码数据：member 01M3H4R7PTKEZTCFGBV1M45C49（13690503034，ACTIVE 卡）今日 14:00/16:00 两条 WAITING 已插入。
