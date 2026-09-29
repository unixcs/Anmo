# Implement — V2.2 第一批

按序执行；每步末尾的验证命令必须绿了再进下一步。步骤 1-2 是后端面（最小），3-7 是前端面，8 全量验证。

## Step 1 — server：settings 三键
- [ ] `server/internal/modules/content/repo.go`：
  - `validateSetting` 新增 `amap_js_key` / `amap_js_code`（"" 或 trim 后 ≤64 字符，否则 SETTING_BAD_VALUE）与 `profile_first_login_prompt`（""/"0"/"1"，否则 SETTING_BAD_VALUE）
  - `PublicSettings` 下行 `profile_first_login_prompt`（新增 `profilePromptDownlink()`：仅 "1" 透出，其余 ""）
- [ ] `settings_test.go` 补 3 组：两 key 超长拒绝、prompt 非法值拒绝、下行 sanitizer（"1"→"1"，"0"/""/脏值→""）
- [ ] 验证：`cd server && go build ./... && go vet ./... && go test ./internal/modules/content/ -run Settings -v`

## Step 2 — admin：地图选点 + 经纬度校验
- [ ] 新组件 `apps/admin/src/components/AmapPicker.vue`（props/emit、script 加载器、PlaceSearch、拖拽 marker、确认/取消）——见 design §4
- [ ] `ContentPage.vue`：门店信息卡「地图选点」按钮（未配 Key 先提示）+ 「高德地图配置」块（amap_js_key/amap_js_code 入 saveKeys/fillBiz）+ 手动输入范围校验（73~135 / 3~53）与 `toFixed(6)` 归一
- [ ] 验证：`npm --prefix /mnt/Projects/Anmo/apps/admin run build`

## Step 3 — weapp：预约页固定提交条（R1）
- [ ] `pages/booking/booking.wxml|wxss|js`：submit-bar 固定化、三态置灰文案、摘要行、内容底留白、safe-area——见 design §2；success 分支不动
- [ ] 自查：`node --check` 改动的 .js（或等价语法检查）

## Step 4 — H5：预约页固定提交条（R1）
- [ ] App.vue/全局样式抽 `--tabbar-h`；`pages/BookingPage.vue` 固定条 + 底部留白 + 摘要行/置灰文案与 weapp 同语义
- [ ] 验证：`npm --prefix /mnt/Projects/Anmo/apps/customer run build`

## Step 5 — 资料完善纯函数 + 进度条（R4）
- [ ] H5 `core/utils/profile.ts`：`profileProgress(member)`；weapp `utils/format.js` 镜像
- [ ] H5 `MePage.vue` 进度条（pct<100 渲染，去完善 → `/me/profile`）；weapp `pages/me` 进度条（去完善 → 展开本页姓名编辑）
- [ ] 验证：customer build + weapp js 语法检查

## Step 6 — 预约半屏提示 + 首登弹窗（R4）
- [ ] 两端 booking 提交入口：pct<100 且未跳过 → 半屏（去完善/先跳过）；跳过置本地标记后放行提交；去完善往返后已选保留（H5 必要时 sessionStorage 暂存已选）
- [ ] 两端 home：settings `profile_first_login_prompt==="1"` && 登录 && pct<100 && 未提示过 → 提示一次并置本地标记
- [ ] 本地标记键名按 design §3
- [ ] 验证：两端 build

## Step 7 — H5 ShopCard 兜底（R3）
- [ ] `core/utils/amap.ts`：`isValidLngLat` / `amapUri`
- [ ] `ShopCard.vue`：openMap 走 amapUri；pagehide/visibilitychange 取消 pending；3s 未离开显示兜底条（重试链接 target=_blank、复制地址含 execCommand 回落、高德搜索链接）——见 design §5
- [ ] 验证：customer build

## Step 8 — 全量验证（Murphy 前置）
- [ ] `cd server && go build ./... && go vet ./... && go test ./...`（SQLite 零外部依赖，0 跳过）
- [ ] `npm --prefix /mnt/Projects/Anmo/apps/admin run build && npm --prefix /mnt/Projects/Anmo/apps/customer run build`
- [ ] 起 live server（临时库），curl 验证：PUT /admin/settings 三键合法/非法值矩阵；GET /api/settings 仅含 profile_first_login_prompt 且 "0"→""、"1"→"1"；GET /admin/settings 含 amap 两键
- [ ] weapp 人工验收清单（交付说明列出）：4 tab 正常、预约页固定条真机滚动/置灰/摘要行、半屏提示、我的页进度条、iPhone 安全区

## 回滚点
- 任一步骤失败可独立 revert 该步文件集；后端（Step 1）与前端（Step 2-7）无迁移耦合。
