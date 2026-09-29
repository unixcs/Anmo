# 验收 — 第四批（主会话浏览器 E2E + 环境说明）

## 环境准备（黑盒测试前置，独立于测试行为）

- `bash scripts/start-anmo.sh` 启动本机后端 :8080 + H5 :5173（迁移应用正常）。
- dev 库无服务数据 → 经 admin API 播种 2 分类 + 3 服务（推拿/足疗；全身推拿 ¥168/60min、肩颈推拿 ¥128/45min、足部养生 ¥138/60min）。
- 测试账号 13766778005 / 13766778006（脚本注册，密码 Passw0rd!234）；多轮调试产生的 8001/8002/8003/8004/8007 为残留数据。

## 结果：17/17 PASS，console errors = none

自动化脚本：playwright-core + 系统 Chrome（headless，390×844 移动视口），纯 GUI 黑盒（click/fill），断言 = DOM 读取 + 截图目检双重验证。证据目录 /tmp/anmo-b4-gui/（A3_sheet、A5_saved_toast、A5_back_to_booking、B3_direct_submit、sheet_settled 等 PNG + results.json）。

| # | 断言 | 结果 |
|---|------|------|
| A1 | 注册即登录，跳 /me | PASS |
| A2 | 服务列表 3 项可见可选 | PASS |
| A3 | 弹层恰两按钮：上=立即预约（class 含 primary，y=677）下=去完善资料（ghost，y=777） | PASS |
| A4 | 点去完善资料 → /me/profile | PASS |
| A5 | 保存 → toast 已保存 → 自动返回 /booking，草稿恢复（服务 .picked + 上午 .picked 各 1） | PASS |
| A6 | 资料齐全后再提交不弹层，直接预约成功（成功页文案/核销码指引齐全） | PASS |
| B1 | 第二账号注册即登录 | PASS |
| B2 | 弹层点立即预约 → 直接提交成功 | PASS |
| B4 | 第二次预约不再弹层（跳过标记持久），提交成功 | PASS |

视觉目检（sheet_settled.png）：主按钮深红凸显在上、去完善资料弱化在下、遮罩压暗正常——与用户要求一致。首拍 A3_sheet.png 为滑入动画中途帧，已用等动画 1.2s 的定妆照复核。

## 过程说明（测试基建，非产品 bug）

- 已登录用户访问 /#/login 被守卫弹回 /me（预期行为）→ 流程 B 用全新浏览器上下文模拟另一用户。
- 同 hash URL goto 不重挂载组件（SPA 预期）→ 整页回首页再进预约。
- 当天上午半日池被多轮测试订满导致提交按钮禁用（D20 容量逻辑正确工作）→ 定妆照改订明天。
- 账号注册失败自动转登录的兜底（上轮遗留号）。
