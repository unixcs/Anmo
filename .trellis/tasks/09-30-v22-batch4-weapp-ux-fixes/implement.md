# 实现记录 — V2.2 第四批（F1–F5，trellis-implement 子代理，2026-09-30）

> 范围：全部为前端/文档/脚本提示，未动 `server/` 与 `migrations/`。未 commit（由主会话验收后统一提交）。

## 改动清单（文件:行为）

### F1 预约资料弹层翻转 — `apps/customer/src/pages/BookingPage.vue`
- 模板资料完善半屏弹层：正文改为「先完善资料，店主更好安排；不填也可直接预约。」；按钮翻转为 **「立即预约」（primary，上）= `bookNow`** / **「去完善资料」（ghost，下）= `goProfileSheet`**。
- `skipProfileSheet` 改名 `bookNow`（函数体语义原样：置 `localStorage['anmo.profile.bookingSkipped']='1'` 永久静默 + 关弹层 + 继续 `submit()`），注释同步改为「立即预约：置静默标记继续原流程，下次不再提醒」。
- `goProfileSheet` 行为不变（sessionStorage `anmo.booking.draft` 存草稿 + 跳 `/me/profile`），段落注释「去完善：」→「去完善资料：」。
- 不变：`submit()` 主流程、`BookingDraft satisfies` 草稿结构、`takeDraft` 恢复、跳过标记键名。

### F2 资料保存后返回 — `apps/customer/src/pages/ProfilePage.vue`
- 新增 `import { useRouter } from 'vue-router'` + `const router = useRouter()`。
- `save()` 成功分支：`notify('已保存')` 之后检测 `sessionStorage.getItem('anmo.booking.draft')` 存在则 `router.push('/booking')`（**不加 return**，try 自然落回 finally，busy 正常复位）；草稿由 BookingPage `onMounted takeDraft` 恢复。保存失败停留原页报错；无草稿（普通「我的→个人资料」编辑）行为完全不变。

### F3 小程序 BASE_URL 覆盖 — `apps/weapp/config.js`
- `resolveBaseUrl()` 增加最高优先级 storage 覆盖：`wx.getStorageSync(STORAGE_KEY)` 为合法 `http(s)://…`（trim 后正则校验）才采用，非法/空值忽略；try/catch 兜底后回落平台默认（devtools→LOOPBACK_URL，否则 LAN_URL），平台探测 try/catch 语义不变。
- 新增 `STORAGE_KEY = 'anmo.base_url'` 常量并导出（供 README/测试引用）；文件头注释保留并补一行 storage 覆盖用法说明。
- 导出变为 `{ BASE_URL, LAN_URL, LOOPBACK_URL, STORAGE_KEY }`。

### F4 文档
- `apps/weapp/README.md`：新增「开发者工具联调」章节（面向店主/开发者自己），覆盖：旧包三症状=重新导入+清缓存即消失、项目目录（WSL 与 `\\wsl.localhost\<发行版>\…` 两种路径、AppID 测试号）、先清缓存、勾选「不校验合法域名」、`bash scripts/start-anmo.sh` 一键起后端、D24 dev 登录原理与发布前必须配 `ANMO_WX_APPID`/`ANMO_WX_SECRET`、真机走 `LAN_URL` + `scripts/allow-wsl-lan.ps1` 放行、Console `wx.setStorageSync('anmo.base_url', …)` 覆盖后端地址（指向生产库写真实数据的警示）。
  - 顺带修正「页面与业务对照」表 `pages/login` 行：原文写「微信静默登录→短信绑定 + /api/auth/wx/bind」（D25 已废除 bind、D27 短信下线），与本批新增章节直接矛盾；核实 weapp 实际只调 `/api/auth/wx/login` + `/api/auth/wx/claim` 后改为准确口径。
- `scripts/start-anmo.sh`：尾部提示行「顾客短信验证码(dev): 123456」→「顾客 H5 登录:     手机号 + 密码（首次注册即登录）」（D27 短信已下线）。只改这一行，未动逻辑。

### F5 小程序单测 — `apps/weapp/tests/config.test.js`（新建，node:test + assert，零依赖）
- 每个 case 前 `delete require.cache` 并重设 `global.wx` stub 再 `require('../config.js')`（config.js 在模块加载时同步求值 BASE_URL）。
- 五个用例：① storage 合法覆盖优先生效；② 覆盖值非 http(s)（'abc'）忽略回落默认；③ 无覆盖 + devtools → LOOPBACK_URL；④ 无覆盖 + ios → LAN_URL（stub 仅给 getStorageSync/getDeviceInfo，缺省 getSystemInfoSync 测降级路径）；⑤ global.wx 未定义 → LAN_URL（try/catch 兜底）。

## 验证输出

1. **weapp 单测**（注意：本环境 Node v24.14.0 的 `node --test <目录>` 目录形式普遍失效，见下「遇到的问题」；等价用 glob 形式，语义相同）：
   ```
   $ node --test 'apps/weapp/tests/*.test.js'
   ✔ ① storage 覆盖合法 http(s) 值优先生效 (1.394167ms)
   ✔ ② 覆盖值非 http(s) 被忽略，回落平台默认 (0.327879ms)
   ✔ ③ 无覆盖 + platform=devtools → LOOPBACK_URL (0.26654ms)
   ✔ ④ 无覆盖 + platform=ios → LAN_URL（仅 getDeviceInfo，缺省 getSystemInfoSync） (0.257612ms)
   ✔ ⑤ global.wx 未定义 → LAN_URL（try/catch 兜底） (0.406477ms)
   ℹ tests 5  ℹ pass 5  ℹ fail 0  ℹ skipped 0
   ```
2. **H5 构建（含 vue-tsc 类型检查）**：
   ```
   $ npm --prefix apps/customer run build
   dist/assets/BookingPage-BQJ5dpSk.js   9.09 kB │ gzip: 3.92 kB
   dist/assets/ProfilePage-P83HIMUA.js   2.20 kB │ gzip: 1.11 kB
   ✓ built in 551ms   ← 零错误、零类型告警
   ```
3. **git diff --stat**：
   ```
   apps/customer/src/pages/BookingPage.vue | 12 ++++++------
   apps/customer/src/pages/ProfilePage.vue |  7 +++++++
   apps/weapp/README.md                    | 20 +++++++++++++++++++-
   apps/weapp/config.js                    | 10 +++++++++-
   scripts/start-anmo.sh                   |  2 +-
   5 files changed, 42 insertions(+), 9 deletions(-)
   ?? apps/weapp/tests/   （新增，未跟踪）
   ```

## 遇到的问题

1. **本环境 `node --test <目录>` 目录形式失效**（非测试代码问题）：`node --test apps/weapp/tests/` 与 `/tmp` 最小复现均报 `MODULE_NOT_FOUND: Cannot find module '…/tests'`（Node v24.14.0，WSL2）；glob 形式 `node --test 'apps/weapp/tests/*.test.js'` 与文件形式均正常。验证用 glob 形式等价执行，结论全绿。后续若有人复现同问题，属 Node 构建怪癖，不是用例写错。
2. **README「验证」章节引用的 `apps/weapp/tools/unit.js` 在仓库中不存在**（目录整个缺失，历史遗留）。本批未动该章节；既有单测无从回归，但 config.test.js 独立覆盖 config.js。建议 check 阶段决定是否清理该过时引用（超出本批指令范围，未擅自改）。
3. **F4 README 表格行自相矛盾**：新增章节声明"当前代码已是纯微信一键登录"，而同文件页面对照表仍写短信绑定/`wx/bind`。已按 D25/D27 冻结决策并核实 weapp 代码（`utils/api.js` 仅含 `wxLogin`/`wxClaim`）后修正该行，属于让文档自洽的最小改动。

## 后续（不在本批范围）

- H5 真浏览器 E2E（弹层顺序/配色、去完善→保存→自动返回且草稿恢复、跳过标记第二次不再弹）与 trellis-check 对抗式审查由主会话按 design.md 继续。
