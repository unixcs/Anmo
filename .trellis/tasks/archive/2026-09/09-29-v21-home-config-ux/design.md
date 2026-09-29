# Design — V2.1 首页可配置化与小程序体验修复

## 探查结论（现状事实）

| 事实 | 位置 |
|---|---|
| H5 首页硬编码 `services.value = catalog.services.slice(0, 6)` | `apps/customer/src/pages/HomePage.vue:65` |
| 小程序首页显示全部 ACTIVE 服务，无 limit | `apps/weapp/pages/home/home.js:123-131` |
| 门店电话/地址后台表单已存在（shop_phone/shop_address/shop_latitude/shop_longitude） | `apps/admin/src/pages/ContentPage.vue:88-104` |
| ShopCard 电话行两端都有，`v-if/wx:if="{{phone}}"` 空时隐藏 | `apps/customer/src/components/ShopCard.vue:44`、`apps/weapp/components/shop-card/shop-card.wxml:12` |
| PublicSettings 白名单在 content 模块，现含 shop_*/home_* | `server/internal/modules/content/repo.go:333-359` |
| validateSetting 已有 business_*/shop_* 校验模式可循 | `server/internal/modules/content/repo.go:384-413` |
| `.input` 全局类无显式 height（login 2 处、me 1 处引用） | `apps/weapp/app.wxss:352-364` |
| 小程序 tabBar 3 项、无 services 页面；H5 4 Tab | `apps/weapp/app.json:24-37`、`apps/customer/src/App.vue:6-11` |
| 服务目录默认排序 `ORDER BY sort, created_at`；admin 有 listServices | `server/internal/modules/service/repo.go:194`、`apps/admin/src/core/api/admin.ts:295` |
| weapp app-icon 已含 `sparkles` 图标（services Tab 可用） | `apps/weapp/components/app-icon/app-icon.js:41` |
| weapp 首页 loadSettings/loadHome 并行，服务筛选需两者就绪 | `apps/weapp/pages/home/home.js:61-136` |

## 方案

### P1 首页服务推荐可配置

**数据**：零 migration。两个新 settings 键（unknown key 已可自由保存，仅需补校验+下发白名单）：

- `home_service_limit`：`""`（默认 6）或 `2|4|6|8`
- `home_service_ids`：`""`（空）或 JSON 字符串数组（服务 ID 有序列表，≤50 项）

**后端**（仅 `server/internal/modules/content/repo.go`）：

1. `validateSetting` 增加两 case：
   - `home_service_limit`：空 或 ∈ {2,4,6,8}，否则 `SETTING_BAD_VALUE`
   - `home_service_ids`：空 或 `json.Unmarshal` 到 `[]string` 成功、元素非空、≤50，否则报错
2. `PublicSettings` 增加下发：
   - `home_service_limit`：存值合法则原样下发，否则空串（客户端回落 6）
   - `home_service_ids`：原样下发（客户端解析失败回落全量）
3. 测试：`status_test.go` 同目录新增/扩展 `repo` 层测试覆盖校验矩阵与下发矩阵（SQLite 内存库，现有测试基建可循）。

**筛选逻辑（两端同构，单一真源写在各自 utils）**：

```text
pickHomeServices(allServices /*已 ACTIVE*/, settings):
  limit = {2,4,6,8}.contains(int(settings.home_service_limit)) ? int(...) : 6
  ids   = parseJsonArray(settings.home_service_ids) 或 []
  if ids 非空:
    byId = services by id
    picked = ids.map(id => byId[id]).filter(存在)   // 跳过已删除/下架
  else:
    picked = allServices                            // 后端已按 sort,created_at 排序
  return picked.slice(0, limit)                     // 自适应：不足 limit 全展示
```

- H5：`apps/customer/src/core/utils/home-services.ts` 新增纯函数 + 单测（若前端有测试基建则跑，无则以实现侧小函数+构建把关）；`HomePage.vue` 用它替换 `slice(0, 6)`（settings 并行已取）。
- weapp：`apps/weapp/utils/format.js` 新增 `pickHomeServices(services, settings)`；`home.js` 调整 `loadAll`：settings 与 home/catalog 并行拉取，但在两者都就绪后统一计算 services（消除竞态：把 `loadHome` 里的服务映射移到 `finish`，或让 `loadHome` 读取 `getApp().globalData.settings` 兜底 + settings.then 重算一次）。要求：登录后首屏不闪烁错序。

**admin**（`apps/admin/src/pages/ContentPage.vue` 系统设置 tab）：

- 新卡片「首页服务推荐」（位于"首页文案"与"门店信息"之间）：
  - 显示数量：`el-radio-group`（2/4/6/8 + "默认(6)"？→ razor：四个选项 2/4/6/8，未配置时回填 6 并提示"默认 6"）——采用：`el-radio-button` 2/4/6/8，空值显示为 6 的选中态，保存时若等于 6 存 ""（表示默认）。**再简化**：直接存所选值，默认 6 只在 key 缺失时生效；表单回填 `Number(s.home_service_limit) || 6`。
  - 展示服务：ACTIVE 服务 checkbox 列表（`listServices()` 过滤，按 sort 排序）；已勾选项支持上移/下移（维护一个有序 id 数组状态）；顶部 hint 说明"不勾选 = 按服务排序取前 N 张"。
  - 保存：`saveSetting('home_service_limit', ...)` + `saveSetting('home_service_ids', JSON.stringify(ids))`（顺序两次调用，V1 单店主可接受）。
  - 复用 `fillBiz`/`load` 回填。

### P2 门店电话生效链路

- 代码已闭环，改动最小：
  1. admin 门店信息卡片 hint 补一句"保存后顾客端首页门店卡与关于页立即显示电话/地址行"。
  2. weapp `pages/about`：补营业时间行（对齐 H5 AboutPage 的 `营业时间 HH:MM - HH:MM`，取 publicSettings 的 open_time/close_time）。
  3. 不改 ShopCard 条件渲染逻辑（空值隐藏是正确行为）。
- 生产生效依赖：商家在后台填 `shop_phone`（验收时用临时库演示 + 提示用户生产操作路径）。

### P3 登录占位截断

- `apps/weapp/app.wxss` `.input`：`height: 48px; padding: 0 14px;`（原视觉总高 12+22+12+2border=48px 不变，box-sizing 已是 border-box）；移除 `min-height: 0`。`.textarea` 不动。
- 微信原生 input 需要显式 height 才能保证占位/正文垂直居中；这是小程序已知渲染约定。

### P4 小程序服务 Tab

- 新增 `apps/weapp/pages/services/services.{js,wxml,wxss,json}`：
  - `api.catalog()` → 按 categories 分组（无分类的落"其他"组，对齐 H5 byCategory()）；卡片行 = 名称/描述(2行截断)/时长·价格 + 「预约」按钮。
  - 点预约：`globalData.pendingServiceId = id; wx.switchTab('/pages/booking/booking')`（与 home.goBooking 同模式）。
  - 样式复用全局 token（.card/.btn/.section-title），wxss 只写页内布局。
- `apps/weapp/app.json`：pages 注册 + tabBar 第二位插入 `{ "pagePath": "pages/services/services", "text": "服务" }`（无图标，与现有 Tab 一致）。
- weapp 首页 service_list 区块头加 `全部服务` 文本链接 → `wx.switchTab` 服务 Tab；对齐 H5 section-head 的 more。
- 未登录口径：catalog 是 JWT 资源（router.go:51 全 /api 守卫），services 页与首页同策略——未登录显示登录引导（复用 empty 模式）。

## 权衡记录

- **为何 settings KV 而非 service 表加 home_visible 列**：无需 migration、无新实体（AGENTS V1 边界），admin 已有通用 saveSetting 通道；service 表列会引入"目录语义 vs 首页语义"耦合。
- **为何客户端筛选而非后端 /api/home 聚合服务**：/api/services 语义保持纯净（预约页要全量）；改动面最小（服务端只加 2 个白名单键）；两端口径由同一伪代码约束。
- **为何 admin 保存两次 saveSetting**：无批量事务端点；KV 写无一致性风险（两键独立生效皆安全）。

## 回滚

- 全部改动可 git revert；无 schema 变更；settings 键残留无害（客户端忽略）。
