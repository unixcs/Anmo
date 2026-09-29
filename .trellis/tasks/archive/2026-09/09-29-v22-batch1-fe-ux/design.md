# Design — V2.2 第一批

## 0. 影响面总览

| 端 | 文件 | 变更 |
|----|------|------|
| server | `internal/modules/content/repo.go` | validateSetting 新增 3 键；PublicSettings 下行 1 键 |
| server | `internal/modules/content/settings_test.go` | 新键测试 |
| weapp | `pages/booking/*` | 底部固定提交条（R1）+ 半屏提示（R4） |
| weapp | `pages/me/*` | 资料完善进度条（R4） |
| weapp | `pages/home/*` | 首登弹窗触发点（R4） |
| weapp | `utils/format.js` | `profileProgress` 跨端镜像函数 |
| H5 | `pages/BookingPage.vue` | 底部固定提交条（R1）+ 半屏提示（R4） |
| H5 | `pages/MePage.vue` / `HomePage.vue` | 进度条 / 首登弹窗（R4） |
| H5 | `core/utils/profile.ts`、`core/utils/amap.ts`（新） | 跨端逻辑纯函数化 |
| H5 | `components/ShopCard.vue` | URI 校验 + 兜底条（R3） |
| admin | `pages/ContentPage.vue`、`components/AmapPicker.vue`（新） | 地图选点（R2）+ 经纬度校验 + Key 配置 |

不改：migration、登录/身份/交易/预约后端、核销链路。

## 1. Settings KV 设计（R2/R4 后端面）

新增三键（`content_system_setting`，零迁移）：

| key | 取值 | 校验（validateSetting） | 下行 |
|-----|------|------------------------|------|
| `amap_js_key` | "" 或 ≤64 字符串（trim） | 超长 → SETTING_BAD_VALUE | **不下行**（仅 admin 经 GET /admin/settings 全量读取；JS API Key 本就是浏览器公开凭据，但顾客端无需知悉） |
| `amap_js_code` | "" 或 ≤64 字符串（trim） | 同上 | 不下行 |
| `profile_first_login_prompt` | "" / "0" / "1" | 其他 → SETTING_BAD_VALUE | PublicSettings 下行，`profilePromptDownlink()`：仅 "1" 原样透出，其余返回 ""（缺省=关，脏数据安全） |

沿用既有模式（spec/backend/quality-guidelines.md「Customer-Visible Merchant Config Pattern」）：validateSetting → PublicSettings sanitizer → 客户端兜底默认。测试照 `settings_test.go` 现有三例结构补 3 组。

## 2. R1 预约页底部固定提交条

**weapp（pages/booking）**：
- wxml：提交按钮 + 模糊提示从内容流移入 `<view class="submit-bar">`（仍在 `wx:else` 表单分支内，success 分支不受影响）。
- 结构：`.submit-bar`（fixed，left/right 0，bottom 0）→ 内部：`.submit-info`（已选摘要行，选完才显示）+ `button.btn.primary.lg.block` + `.submit-hint`（模糊预约说明）+ `padding-bottom: calc(16rpx + env(safe-area-inset-bottom))`。
- wxss：`.page-body { padding-bottom: 240rpx; }`（表单态预留，避免备注被盖）；`.submit-bar` 加白色背景 + 顶部 hairline + `box-shadow`，防内容透底。
- js/置灰逻辑：disabled = `busy || svcIdx < 0 || dayIdx < 0 || !part`；按钮文案三态提示：未选服务→"请先选择服务"，未选日期→"请先选择日期"，未选上下午→"请先选择上午或下午"；选完→"立即预约"。摘要行：`服务名 · 日期 上/下午 (具体时间|店家安排时间)`。
- 提交函数 submit() 逻辑不变（仅入口换位置）。

**H5（pages/BookingPage.vue）**：
- 同样结构：`.submit-bar` fixed；**bottom 必须让开 App.vue 的 `.tabbar`**（预约页是 tab 页）。读取 App.vue 实际 tabbar 高度，抽成全局 CSS 变量 `--tabbar-h`（App.vue 或全局 style 定义），BookingPage 用 `bottom: calc(var(--tabbar-h) + env(safe-area-inset-bottom))`，禁止两处硬编码漂移。
- 摘要行/置灰文案与 weapp 同语义；成功态路由跳走，无残留问题。
- 内容容器 `padding-bottom` 同步预留。

## 3. R4 资料完善三件套

**共享逻辑纯函数**（两端行为一致）：
- H5 `core/utils/profile.ts`：`profileProgress(member: {name, phone}): { pct: 0|50|100; missing: ('name'|'phone')[] }`。
- weapp `utils/format.js`：`profileProgress(member)` 逐行镜像（同 v2.1 `pickHomeServices` 的跨端镜像惯例）。

**进度条（我的页）**：
- 位置：头像区下方；`pct<100` 显示：`资料完善度 {pct}%｜{missing 提示文案} [去完善]`；100% 不渲染。
- 文案映射：缺 name→"完善称呼"；缺 phone→"完善手机号，方便预约联系"；全缺→"完善称呼与手机号"。
- 去完善跳转：H5 → `/me/profile`（ProfilePage 已有姓名编辑）；weapp → 本页姓名编辑区（me 页已有内联 input，聚焦/展开即可；weapp 手机号由登录产生，不提供改手机号入口）。

**预约半屏提示（booking 页）**：
- 触发：点提交且 `pct<100` 且 本地标记未置位 → 拦截首次点击，弹半屏；「先跳过」→ 置本地标记并继续原提交流程；「去完善」→ 跳完善页（已选状态天然保留：weapp data 不销毁 / H5 keep state on route leave，返回后恢复——两端都是页面实例保活场景，无需额外存储；若 H5 路由离开会丢组件状态，则把已选 key 存 sessionStorage 恢复）。
- 半屏实现：两端各自最简 bottom-sheet（遮罩 + 底部面板 + 两按钮），不引组件库（weapp 无构建）。
- 本地标记：weapp `wx.setStorageSync('anmo_profile_booking_skipped', 1)`；H5 `localStorage['anmo.profile.bookingSkipped']`。只记"跳过"，用户点「去完善」未完成资料前再次提交仍可再提示（更符合"提醒"本意）——即仅跳过才永久静默。

**首登弹窗（home 页，两端）**：
- 条件：PublicSettings `profile_first_login_prompt === "1"` && 登录态 && `pct<100` && 本地未记录"已提示" → 弹一次（weapp `wx.showModal`"去完善/暂不"；H5 同语义对话框）。关闭/拒绝即置本地标记（`anmo_profile_first_prompted` / `anmo.profile.firstPrompted`）。
- 触发点放在 home 页数据加载完成后（settings 与 profile 均已在手），不在 app.js（避免与静默登录竞态）。

## 4. R2 admin 地图选点

**AmapPicker.vue（新组件）**：
- props：`{ key: string; jsCode: string; initLng?: string; initLat?: string }`；emit：`confirm({lng, lat})`、`cancel`。
- 加载器（组件内函数）：先 `window._AMapSecurityConfig = { securityJsCode: jsCode }`（有才设），再插 `<script src="https://webapi.amap.com/maps?v=2.0&key=...">`，promise 缓存防重复加载；onerror reject → el-message「高德地图加载失败，请检查 Key 配置或手动输入经纬度」。
- 地图交互：`AMap.Map` + 中心点（有 init 坐标用之，否则默认杭州 120.153576,30.287459，zoom 12）；点击地图/拖拽 marker 移动标记；`AMap.PlaceSearch().search(keyword)` 结果列表（名称+地址+district），点选 → `map.setCenter(poi.location)` + marker 落点；无结果 → 提示"未找到匹配地址，请拖拽地图手动选点"。
- 「确认选点」读 marker 经纬度 → `toFixed(6)` → emit confirm；「取消」不回填。
- 未配 Key：ContentPage 点「地图选点」时校验 `amap_js_key` 非空，空则 el-message 提示"未配置高德 Key，请在下方「高德地图配置」填写"，不弹窗。

**ContentPage 集成**：
- 门店信息卡新增：经纬度输入框旁「地图选点」按钮；新折叠块「高德地图配置」（amap_js_key / amap_js_code 两个输入框 + hint："在高德开放平台申请 Web端(JS API) Key 并配置域名白名单，仅用于本页选点"）。两键并入现有 saveKeys 保存链与 fillBiz 回填链。
- 手动输入校验（保存前 + blur）：非空时数值化，经度 ∉ [73,135] 或纬度 ∉ [3,53] → 校验失败提示"请输入正确的经纬度"并阻断保存；保存时 `Number(v).toFixed(6)` 四舍五入回写（空值不处理）。

## 5. R3 H5 ShopCard 地址跳转

**`core/utils/amap.ts`（新）**：
```ts
export function isValidLngLat(lng?: string, lat?: string): boolean
  // 数值化，经度∈[73,135] 且 纬度∈[3,53]
export function amapUri(address: string, lng?: string, lat?: string): string
  // 合法 → https://uri.amap.com/marker?position=lng,lat&name=enc(address)&src=anmo&callnative=1
  // 否则 → https://uri.amap.com/search?keyword=enc(address)&src=anmo&callnative=1
```

**ShopCard.vue 状态机**：
- `openMap()`：计算 uri；置 `navPending=true`；注册一次性 `pagehide`/`visibilitychange(hidden)` 监听（跳转成功页面离开即取消 pending）；`location.href = uri`；3s 定时器到期若仍 pending → `showFallback=true`（组件内兜底条，不阻塞后续操作）。
- 兜底条内容：① `<a :href="uri" target="_blank">重试打开地图</a>` ② 「复制地址」按钮：`navigator.clipboard.writeText` 失败回落 `textarea+execCommand('copy')`，成功显示"已复制" 2s ③ `<a :href="searchUri" target="_blank">在高德搜索</a>`。
- **不做 iframe 内嵌**（决策）：uri.amap.com 大概率 `X-Frame-Options` 拒嵌且无法可靠探测，计划本身允许该层降级；纯文字+复制+搜索链接已满足"用户至少拿到地址"。
- 不做 UA 分支（微信内置/普通浏览器统一 URI）；不做后端失败上报（无通道，P1 再议）。

## 6. 兼容与回滚

- 全部改动向后兼容：settings 新键缺省即旧行为（进度条默认渲染、半屏默认开、首登默认关、无 Key 时地图按钮提示但手动可用）。
- 回滚点：R1/R3/R4 为纯前端 revert 即回旧行为；R2 仅 content 模块校验+下行扩展，revert 无数据残留（KV 值残留无害）。
- 风险：① H5 tabbar 高度变量化若与现有样式冲突 → fallback 直接量取 App.vue 实际高度写死注释同步点；② PlaceSearch 插件加载失败 → 已有 onerror 提示 + 手动输入兜底；③ weapp 半屏/固定条真机表现 → 列入人工验收清单。
