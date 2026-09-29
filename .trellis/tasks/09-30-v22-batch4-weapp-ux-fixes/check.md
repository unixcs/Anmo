# Check 报告 — V2.2 第四批（trellis-check，2026-09-30）

## A. 规格符合性（design.md F1–F5 逐条）

| 项 | 结论 | 证据 |
|---|---|---|
| F1 弹层翻转 | 通过 | `apps/customer/src/pages/BookingPage.vue:383-389`：「立即预约」primary 在上 = `bookNow`（置 `localStorage['anmo.profile.bookingSkipped']='1'` + 关弹层 + `void submit()`）；「去完善资料」ghost 在下 = `goProfileSheet`（sessionStorage `anmo.booking.draft` 草稿 + `router.push('/me/profile')`）。`skipProfileSheet`→`bookNow` 仅改名，函数体语义不变；`submit()` 主流程、`BookingDraft` 结构、`takeDraft` 未动（git diff 佐证仅注释/改名/模板按钮/正文文案变化）。正文已改「先完善资料，店主更好安排；不填也可直接预约。」 |
| F2 保存后返回 | 通过 | `apps/customer/src/pages/ProfilePage.vue:38-42`：`notify('已保存')` → 有草稿才 `router.push('/booking')`；无 return，try 自然落回 finally，busy 正常复位；无草稿（普通资料编辑）行为不变；`useRouter` 已正确引入。路由 `/booking` 存在（router/index.ts:7） |
| F3 BASE_URL 覆盖 | 通过（发现并修复一处小瑕疵，见下） | `apps/weapp/config.js`：storage 覆盖最高优先级、非法值忽略回落、平台默认（devtools→LOOPBACK / 其他→LAN）不变、`STORAGE_KEY` 导出。瑕疵：校验正则 `/^https?:\/\/\S+/` 未锚定结尾，`'http://a b'` 等含空白值被误判合法且整串采用为 BASE_URL → 已修为 `/^https?:\/\/\S+$/` 并补测试 ②b |
| F4 联调文档 | 通过（修复过时章节，见 C） | 联调章节路径、键名（`anmo.base_url` 与 `STORAGE_KEY` 一致）、命令（`bash scripts/start-anmo.sh` 确拉起 :8080 + :5173 + :5174/5175）、D24 口径（`openid="dev:"+code` = `server/internal/modules/identity/wx.go:33`；env 名 = `config.go:113-114`）均与代码一致；`pages/login` 表格行已按 D25/D27 改为 `/api/auth/wx/login` + `/api/auth/wx/claim`（utils/api.js 仅此两 auth 端点，核实一致）。`scripts/start-anmo.sh` 仅提示行一处改动 |
| F5 单测 | 通过 | `apps/weapp/tests/config.test.js`：stub 隔离成立（node:test 文件内顺序执行；每 case `delete require.cache[CONFIG_PATH]`——config.js 零依赖无其他缓存；`global.wx` 按用例重设/删除）。降级路径已覆盖（④缺 `getSystemInfoSync`、⑤无 `wx`）；本审查新增 ②b 含空白降级 |

## B. 对抗式审查（逐链路）

1. **草稿闭环 — 通过**
   - `goProfileSheet` 存 `{serviceId, dayIdx, part, slotTime, note}` → ProfilePage 保存成功 `push('/booking')` → BookingPage `onMounted takeDraft`。
   - `takeDraft` 是 pop（BookingPage.vue:76-79：getItem → removeItem → parse，损坏 JSON 也一并丢弃返回 null）。
   - 恢复链条：note 立即恢复（onMounted）；serviceId 经 loadServices preset（仅目录中存在时，:106-107）；dayIdx 越界不恢复（:109-111）；part/slotTime 经 `applyPendingDraft` 且「半天仍开放 + 槽有余量」才恢复（:86-97）——不会恢复已闭店/已满的旧选择。
   - 「保存后返回」恢复成立；「不保存直接浏览器返回」：BookingPage 重新挂载，onMounted takeDraft 同样恢复，草稿被消费、不残留到下一次无关访问。
2. **跳过标记 — 通过**
   - `bookNow` 置 `'1'` 后 `submit()` 拦截条件 `member.value && pct<100 && !localStorage.getItem('anmo.profile.bookingSkipped')`：`'1'` 为真值字符串，`!'1'`=false → 放行；键不存在时 getItem 返回 null → 弹层。判断正确。
   - member 拉取失败（null）时不弹层直接提交，符合「拉不到不阻塞预约」设计。
3. **ProfilePage 返回时机的误伤窗口 — 评估后不修（规格即此口径）**
   - 窗口：弹层 → 去完善 → **不保存** → 手动导航去别的页（非 /booking）→ 草稿残留 sessionStorage → 之后任何一次正常「个人资料保存」都会被带到 /booking 并回填旧已选。
   - 影响评估：一次性多余跳转 + 旧选择回填，无数据破坏（提交仍需显式点「立即预约」，摘要行可见当前选择）；用户下次进 /booking 草稿即被 takeDraft pop 自愈；sessionStorage 生命周期随 tab 关闭终止。触发需同时满足「弹层后弃保存 + 弃保存后不再进预约页 + 之后恰好编辑资料并保存」，概率低。
   - 不修依据：design.md F2 明确以「sessionStorage 存在草稿」为回跳信号，实现与规格一致；若改为独立标记，需在 ProfilePage 离开时清理，会破坏 B1 要求的「不保存直接返回仍可恢复」属性。按改动最小原则接受，写明于此。
4. **小程序 auth 链路 — 通过**
   - `wx.login → code → POST /api/auth/wx/login`（module.go:17 公开路由；D25：openid 未绑定同事务直建号直发 Token；dev 兜底 `openid="dev:"+code` = wx.go:33）。
   - 401：request.js:34-44 `setToken('')` + `getApp().markAuth(false)`（全局登录态同步失效，避免其他页仍显示已登录）+ `err.needLogin`；页面侧 catch needLogin → goLogin/needLogin。恢复路径闭合：login 页 trying（防重入 + disabled loading 按钮）/ failed（重试按钮 `bindtap="tryWx"`）两态，可反复重试。
   - claim：`MEMBER_PHONE_TAKEN`（me.js:145）弹认领层 → `POST /api/auth/wx/claim {phone, password}`（顾客 JWT，module.go:18）→ 返回新 token → `adoptToken` → `ensureSession` 验活 → `markAuth(true)`。与后端 `ClaimByPhone` 契约一致（identity/customer.go:113-160：密码错 401 / 撞绑 WX_OPENID_BOUND / 空壳有数据 AUTH_CLAIM_HAS_DATA，文案均可读）。客户端密码 ≥6 校验与 D27（6~64 位）一致。
   - 观察（不改）：`markAuth(false)` 会置 `manualAuth=true`，被动 401 也算「手动过」——语义稍宽，但 401 后本就应保持登出态直至重新登录，无实际影响。
5. **QR 门槛（D21）— 通过**
   - qrcode.js:91 `hasActiveCard = cards.some(c => c.status === 'ACTIVE')`；`renderMemberQR`（:108-111）无 ACTIVE 卡或无 memberId 不画码；码内容 `ANMO-MEMBER:<member_id>` 协议不变；无卡时仅展示今日预约文字列表。商家端后端校验兜底属后端职责（D21 口径）。
6. **预约/取消/改期 — 通过**
   - 参数契约：选槽 → `{start_time: "YYYY-MM-DD HH:MM"}`、仅半天 → `{date, day_part}`（weapp booking.js:221 / appointments.js:175，与 H5 BookingPage:217-220 同口径，与后端 BookingReq 一致）。
   - 取消/改期入口均限 `WAITING`（D8：改期限 WAITING、状态机收紧后无 CONFIRMED）。
   - 错误码处理：改期 `APT_SLOT_FULL/APT_HALFDAY_FULL/APT_CLOSED/APT_TOO_SOON` 重拉时段、其余保留已选；后端文案用户可读（booking.go:192「预约需至少提前 2 小时」、:230 店铺休息、:275/:296 约满）。H5 submit 对 SLOT_FULL/HALFDAY_FULL/CLOSED 有专门 toast + refreshOptions。
   - 跨零点防护：H5 submit 与 weapp `ensureFreshDays` 均重算日期条并拒绝提交过期日期。
7. **config.test.js — 通过**
   - stub 隔离成立：每 case 前 delete require.cache + `global.wx` 重设/删除；config.js 在 require 时同步求值 BASE_URL，模式匹配正确。
   - 降级路径实测：④ 仅 `getDeviceInfo`（缺省 `getSystemInfoSync`）→ 走 getDeviceInfo 分支；⑤ `delete global.wx` → 两层 try/catch 兜底 LAN_URL；② ②b 非法值（'abc' / 含空白）忽略回落。

## 发现并修复的问题

1. `apps/weapp/README.md`「验证」章节（原 47-69 行）引用不存在的 `apps/weapp/tools/unit.js` 与 `devtools-verify.js`（tools/ 目录整个不在仓库，历史遗留）→ 整节重写：指向 `apps/weapp/tests/config.test.js` + `node --test "apps/weapp/tests/*.test.js"`（含 Node v24 `node --test <目录>` 目录形式报 MODULE_NOT_FOUND、须用 glob 形式的注意事项），E2E 改为指向上文「开发者工具联调」人工步骤，README 恢复自洽。
2. `apps/weapp/config.js:16` 覆盖值校验正则 `/^https?:\/\/\S+/` 未锚定结尾：含空白值（如 `'http://a b'`）被误判合法并整串采用为 BASE_URL，导致后续 wx.request 必败且难排查 → 修为 `/^https?:\/\/\S+$/`（trim 后含任何空白即忽略回落默认）；`apps/weapp/tests/config.test.js` 补用例 ②b 覆盖该降级路径。

## 未修（已知边界 / 超范围观察）

- B3 草稿误伤窗口：见上文第 3 条，规格即此口径，接受不修。
- README「使用」章节 `docker exec anmo-mysql curl …` 的 netns 连通性验证写法沿自 MySQL 时代（2026-09-28 起 SQLite，该容器仅作历史数据源保留）。命令作纯网络探针仍可用，属本批 diff 之外的历史内容，未动；后续可改用任意常驻容器或直接 curl 验证。
- request.js 401 时 `markAuth(false)` 置 `manualAuth=true` 语义稍宽（见 B4 观察），无实际影响，不改。

## 全量验证输出

1. 后端（前端改动未波及，全绿零跳过）：
   ```
   $ go build ./... && go vet ./... && go test ./...
   ok  	anmo/server/internal/adversarial	1.611s
   ok  	anmo/server/internal/database	0.257s
   ok  	anmo/server/internal/e2e	3.540s
   ok  	anmo/server/internal/middleware	(cached)
   ok  	anmo/server/internal/modules/appointment	0.955s
   ok  	anmo/server/internal/modules/card	0.495s
   ok  	anmo/server/internal/modules/content	0.712s
   ok  	anmo/server/internal/modules/identity	1.958s
   ok  	anmo/server/internal/modules/member	1.168s
   ok  	anmo/server/internal/modules/ops	0.263s
   ok  	anmo/server/internal/modules/service	0.621s
   ok  	anmo/server/internal/modules/transaction	1.100s
   ok  	anmo/server/internal/router	(cached)
   ok  	anmo/server/internal/shared	(cached)
   ```
2. 小程序单测（glob 形式）：
   ```
   $ node --test "apps/weapp/tests/*.test.js"
   ✔ ① storage 覆盖合法 http(s) 值优先生效 (1.683522ms)
   ✔ ② 覆盖值非 http(s) 被忽略，回落平台默认 (0.335517ms)
   ✔ ②b 覆盖值含空白（非完整合法 URL）被忽略，回落平台默认 (0.275975ms)
   ✔ ③ 无覆盖 + platform=devtools → LOOPBACK_URL (0.283212ms)
   ✔ ④ 无覆盖 + platform=ios → LAN_URL（仅 getDeviceInfo，缺省 getSystemInfoSync） (0.404127ms)
   ✔ ⑤ global.wx 未定义 → LAN_URL（try/catch 兜底） (0.282941ms)
   ℹ tests 6  ℹ pass 6  ℹ fail 0  ℹ skipped 0
   ```
3. H5 构建（`build = vue-tsc -b && vite build`，类型检查通过）：
   ```
   $ npm --prefix apps/customer run build
   dist/assets/BookingPage-BQJ5dpSk.js   9.09 kB │ gzip:  3.92 kB
   dist/assets/ProfilePage-P83HIMUA.js   2.20 kB │ gzip:  1.11 kB
   ✓ built in 411ms
   ```

## 总结

检查 5 个改动文件 + 新增测试，另核对 weapp auth/booking/appointments/qrcode/me 链路与后端契约（wx.go / customer.go / booking.go / module.go）。规格 F1–F5 全部符合；对抗审查 7 条链路全部通过；发现 2 处问题（README 过时 tools 引用、config.js 正则未锚定）均已修复并复验；B3 误伤窗口按规格接受并写明。后端零改动全绿，weapp 单测 6/6，H5 构建含类型检查通过。未 commit。
