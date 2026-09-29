# Implement — V2.1 执行计划

## 顺序清单

### Step 1 后端：settings 校验 + 下发（P1 地基）

- [ ] `server/internal/modules/content/repo.go`：`validateSetting` 加 `home_service_limit`（空或 2/4/6/8）与 `home_service_ids`（空或合法 JSON 字符串数组、元素非空、≤50）校验；错误码 `SETTING_BAD_VALUE`
- [ ] `PublicSettings` 下发 `home_service_limit`（非法存值下发空串）与 `home_service_ids`（原样）
- [ ] 新增/扩展 content 模块测试：校验矩阵（合法/非法/边界）+ PublicSettings 下发断言
- [ ] `cd server && go build ./... && go vet ./... && go test ./...` 全绿

### Step 2 H5：首页筛选逻辑（P1）

- [ ] 新增 `apps/customer/src/core/utils/home-services.ts`：`pickHomeServices(services, settings)` 纯函数（见 design 伪代码）
- [ ] `HomePage.vue`：替换 `slice(0, 6)`；settings 拉取失败时回落（limit 6、无 ids 过滤）
- [ ] `npm run build`（apps/customer）通过

### Step 3 小程序：首页筛选 + 服务 Tab + 输入框 + 关于页（P1/P3/P4/P2）

- [ ] `apps/weapp/utils/format.js`：`pickHomeServices` 同构实现
- [ ] `pages/home/home.js`：settings 与 home/catalog 都就绪后再计算展示服务（消除并行竞态）；保留失败回落
- [ ] `pages/home/home.wxml`：service_list 区块头加"全部服务"→ switchTab services；wxss 补 head 右侧链接样式
- [ ] 新增 `pages/services/services.{js,wxml,wxss,json}`（分组列表 + 预约带 pendingServiceId；未登录登录引导）
- [ ] `app.json`：注册页面 + tabBar 插入"服务"（第 2 位）
- [ ] `app.wxss` `.input`：`height: 48px; padding: 0 14px;`，去掉 `min-height: 0`；检查 me/login 页无布局回退
- [ ] `pages/about/about.js/.wxml`：补营业时间行（open_time/close_time）

### Step 4 admin：首页服务推荐配置卡（P1 + P2 提示）

- [ ] `apps/admin/src/pages/ContentPage.vue`：系统设置 tab 新卡片「首页服务推荐」（数量 radio 2/4/6/8 默认 6 + ACTIVE 服务勾选 + 上移/下移排序 + 保存两键）
- [ ] 门店信息卡片 hint 补"保存后顾客端首页/关于页立即生效"
- [ ] `npm run build`（apps/admin）通过

## 验证命令

```bash
cd server && go build ./... && go vet ./... && go test ./...
cd apps/customer && npm run build
cd apps/admin && npm run build
```

## 审查门

- Step 1 完成后：后端测试绿才进 Step 2-4
- 全部完成后：trellis-check 全量检查（跨层一致性：两端 pickHomeServices 口径逐行对照）

## 回滚点

- 每个 Step 独立可 revert；无 migration、无数据变更
