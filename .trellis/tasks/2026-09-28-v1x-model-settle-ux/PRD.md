# PRD — V1.x 第二阶段：业务模型修正 + 结算解耦 + 门店信息 + 移动端/UI

> 来源：用户 goal 八十八节。执行顺序遵循 §36，每 Phase 输出 Changed/Why/Files/DB/API/Tests/Risk/Next（§82）。
> 边界：不引入 Plan 未提及实体（§87/§88）；V2 小程序不在本阶段。

## N0 现状勘察结论（Review 输出）

- 状态机：创建即 PENDING_CONFIRM（lifecycle.go:161）；confirm/start/complete/no-show 端点齐备；appointment 已有 started_at/completed_at 列。
- 结算：SettleByCard 后端未校验 IN_SERVICE（仅前端禁用）；固定取预约快照第一项服务，不可指定实际服务；SettleByPay 要求 IN_SERVICE/COMPLETED 但不改预约状态；RedeemWalkIn 屏蔽当日有活跃预约会员（一刀切）。
- 卡：member_card 不带卡名；无顾客端流水端点；redemption/card_transaction 无名称快照；valid_from/until 序列化为 RFC3339。
- 标签：member_tag/member_tag_rel 已有；缺改名/删除端点；搜索仅 keyword。
- settings：KV 表 + GET/PUT /admin/settings；PublicSettings 已暴露营业规则与 shop_phone/shop_notice；无地址/经纬度键。
- 前端：admin LoginPage 固定 360px；顾客端 style.css 有未接入的 :root 变量与 #app{width:1126px} 干扰；无 design token 体系；brand-guidelines 不存在。

## N1 预约模型修正（Phase A1）

- 新状态机：`WAITING → IN_SERVICE → COMPLETED`；异常 `WAITING → CANCELLED | NO_SHOW`。PENDING_CONFIRM/CONFIRMED 不再作为业务状态。
- 创建预约直接 WAITING，删除 Confirm 动作与 `PUT /admin/appointments/{id}/confirm` 路由；Start 改为 WAITING→IN_SERVICE（记录 started_at）；NoShow 改为 WAITING→NO_SHOW；改期/取消作用于 WAITING；冲突判定/容量统计活跃态集合改为 {WAITING, IN_SERVICE}。
- Migration 012：`UPDATE appointment SET status='WAITING' WHERE status IN ('PENDING_CONFIRM','CONFIRMED')`（confirmed_at 清 NULL 不需要）；放宽 CHECK 约束；历史数据不可物理删除——状态值更新属业务迁移，允许。
- 前端：商家端去掉"确认"按钮（待到店即等客）；两端状态文案/徽标更新（WAITING=待到店）。顾客端"我的预约"不再显示"待确认"。
- AGENTS.md：D8/D9 冻结决策按新状态机改写并注明 2026-09-28 用户决策覆盖。

## N2 Settlement / Redemption 解耦（Phase A2）

- 统一语义：结算 = Settlement{Redemption | Payment}，Appointment 仅可选关联。核销与收款均允许实际服务 ≠ 预约服务。
- SettleByCard 改造：请求体加可选 `service_id`（缺省仍取预约快照第一项）；指定时校验卡服务规则并按该服务校验适用性；redemption.service_id 记实际服务；预约状态 ∈ {WAITING, IN_SERVICE} 时核销成功后 MarkCompleted；CANCELLED/NO_SHOW 的预约不允许走预约核销（前端隐藏+后端 422），改走散客核销。
- RedeemWalkIn 守卫放宽：仅当会员当日存在 WAITING/IN_SERVICE 预约时拒绝（RDM_WALKIN_BLOCKED）；当日预约已 CANCELLED/NO_SHOW 不阻止（§12 独立散客结算）。
- SettleByPay：允许 WAITING/IN_SERVICE；成功后同事务 MarkCompleted（§8 事务一致）；金额仍手动输入，服务默认价仅作提示。
- 快照：redemption 加 `service_name`、card_transaction 加 `card_name`（迁移 012），写入时取当前名称，历史不变。
- 扫码页（ScanRedeemDialog 重构为统一结算页）：显示会员姓名/手机号/有效卡/今日预约/实际服务选择/结算方式；今日预约 1 个自动选、多个可换、无预约=散客；实际服务默认预约服务可改；结算方式：有适用有效卡默认会员卡，否则现金/微信。

## N3 顾客端状态同步 + 卡历史（Phase A3）

- 顾客端：预约状态文案（待到店/服务中/已完成/已取消/未到）；卡详情页新增"使用明细"（新端点 `GET /api/me/cards/{id}/transactions`，校验卡属于 token 会员）。
- `/api/me/cards` JOIN card_template 输出 card_name；有效期格式化：DATE 列在 Go 侧转 `YYYY-MM-DD` 字符串输出（或前端统一 formatter），禁止 RFC3339；永久显示"永久"。

## N4 capacity 链路 + 登录 Bug（Phase A4）

- capacity 全链路验证：后台输入 2/67 保存→刷新仍为该值；saveSetting 对 business_* 数值键加服务端校验（正整数、上限 999；slot_minutes 限 30|60|120；时间格式 HH:MM）。容量语义已通用化（逐槽 capacity + 半日池 slots×capacity），补测试证明 30/60/120 分钟档位下均生效。
- admin LoginPage 响应式修复（max-width+100%，移动端单列）；顾客端 Login 页验证码按钮/输入框宽度修复。

## N5 首页内容 + 会员标签（Phase A5）

- 首页文案后台可编辑：settings 新键 `home_title`、`home_body`；GET /api/settings 暴露；顾客端 HomePage hero 动态渲染（缺省回落现有文案）；公告沿用 shop_notice（UI 决定展示样式）。
- 标签：新增 `PUT /admin/tags/{id}`（改名）、`DELETE /admin/tags/{id}`（无关联才允许删，有关联 409）；会员列表搜索支持 `keyword + tag_ids + card_type` 组合（repo 组合 WHERE）；MembersPage 增加标签/卡类型筛选器。

## N6 门店信息 + 门店状态（Phase A6）

- settings 新键：`shop_address`、`shop_latitude`、`shop_longitude`（shop_phone 已有）；admin ContentPage"门店信息"卡片编辑四项。
- PublicSettings 暴露四项；顾客端三处门店信息卡片（地址上、电话下）：首页、预约成功页、关于我们（新增 About 页或并入"我的"）。
- 交互（轻量原则，不做地图 SDK）：地址点击打开 `https://uri.amap.com/marker?position=lng,lat&name=地址`（移动浏览器唤起地图 App）；缺经纬度时仅展示不可点；电话 `tel:` 链接。
- 门店状态：新公开端点 `GET /api/store/status` 动态计算（不落库）：有 IN_SERVICE → 服务中（预计空闲= started_at+快照服务时长，营业结束截断）；无 IN_SERVICE 且当前营业时间内可安排一场最短服务 → 空闲中；否则忙碌中。顾客端首页展示。

## N7 后台移动端优化（Phase A7）

- Mobile First：≤768px 所有列表页用卡片布局（AppointmentsPage 已有，补 RecordsPage/ContentPage/CardTemplatesPage）；高频按钮（保存/提交/开始服务/完成并结算/确认/取消/扫码）进入首屏可视区；表单单列；关键表单底部固定操作栏 + safe-area-inset-bottom；禁止横向溢出（全局 overflow-x 检查）。

## N8 H5 UI 重构 + 深色模式（Phase A8）

- 顾客端引入 Design Tokens（style.css :root：background/foreground/card/muted/border/primary/success/warning/destructive 等），替换页面硬编码色值；`prefers-color-scheme: dark` 全局生效；删除 style.css 模板残留（#app 宽度、hero 等）。shadcn-vue 不引入依赖（不换框架、不为风格重造轮子），采用 shadcn 风格 token + 现有组件体系。

## N9 测试与验收（Phase A9）

- 后端：状态机改造后全部既有测试迁移到 WAITING 口径 + 新增（直接 WAITING、WAITING 开始服务、异常预约散客结算、SettleByPay 自动 COMPLETED、实际服务核销、快照列、capacity 2/67、门店状态三态）。
- E2E 脚本扩展：预约→WAITING→开始→完成并结算（卡）→COMPLETED；无预约扫码核销；现金结算完成。
- 前端两端 vue-tsc + build；移动端按钮可视区人工/验收代理检查。

## 不做（V1.x 阶段）

微信小程序（V2）、地图 SDK/拖拽地图、微信支付 API、确认预约流程、独立 Settlement 表（redemption+payment+appointment 关联即结算事实，不新增实体——避免违反"禁止 Plan 未提及实体"；§4 的 Settlement 是概念聚合不是新表）。

## 审查修订记录（2026-09-28，VERDICT: PASS 有条件）

- P0-1：MarkCompleted（appointment/queries.go:51-55）显式改为允许 WAITING|IN_SERVICE→COMPLETED，SettleByPay 守卫 settle.go:360 同步放宽；W8 注释改写。
- P0-2：dashboard summary 的 pending_confirm/confirmed 计数字段（queries.go:163-246 + DashboardPage.vue）迁移为 waiting/in_service 口径。
- P1-3：指定实际服务时 CARD payment 金额=该服务当前售价（对齐 RedeemWalkIn 口径），快照仅作缺省。
- P1-4：migration 012 具名 `DROP CHECK ck_appointment_status` + 新 CHECK（WAITING...）+ `MODIFY status DEFAULT 'WAITING'`。
- P1-5：GET /api/store/status 挂 content 模块，经 appointment api.go 新增只读查询（依赖方向 content→appointment 合法，D6/D7）。
- P1-6：顾客端卡流水复用 card 模块 Transactions 查询，handler 层做归属校验。
- P2-7：status log 历史值（PENDING_CONFIRM/CONFIRMED）两端 format 保留旧值映射显示。
- P2-8：confirmed_at 列保留只读历史，不再写入。
- P2-9：门店信息卡片落点定死：顾客端首页 + 预约成功页 + 独立 About 路由（"我的"加入口）。

## Phase 顺序

A1 状态机 → A2 结算解耦 → A3 顾客端同步/卡历史 → A4 capacity/登录 → A5 内容/标签 → A6 门店信息 → A7 后台移动端 → A8 UI/深色 → A9 测试碰撞 → B 部署。

## 增量修订（2026-09-28 第二轮，用户补充 + 新需求）

- 补充②扫码分流：顾客端核销码页新增**预约单码 ANMO-APT:{appointment_id}**（今日 WAITING/IN_SERVICE 预约逐单出码）；商家扫码按前缀分流（ANMO-MEMBER→会员结算，ANMO-APT→单查 `GET /admin/appointments/{id}` 预填），统一结算页落地（今日预约 1 个自动选/多个可选/无预约=散客；实际服务默认预约快照可改；结算时才选卡；现金/微信默认卡不可用时展示）。散客现金收款无端点，明确提示不做（歧义选不做）。
- A5.5 商家后台修改登录手机号/密码（热生效）：`PUT /admin/auth/credentials`（校验当前密码；手机号 11 位+唯一；密码≥6 位 bcrypt）；JWT 以 user id 为主题，当前会话保持、下次登录即用新凭据；AdminLayout 头部"账号设置"弹窗。
- capacity 链路两个实锤断点修复：ContentPage 挂载即回填（原刷新后停留在其他 tab 显示硬编码默认值）；`saveSetting` 服务端校验（slot_capacity 1~999、slot_minutes 30|60|120、HH:MM）；前端容量上限 20→999。
- e2e/adversarial 套件迁移 WAITING 口径并补断言（/confirm 404；结算即完成；重完成 409）；SettleByCard 门禁放行 COMPLETED（撤销后重核销 D18，MarkCompleted 幂等）；redemption 预约路径补 service_name 快照写入。
- 新增测试：TestSlotCapacityConfigurable（capacity=2 逐槽+半日池 12）、TestStoreStatusThreeStates、TestBookingRulesCapacityFromSettings（67 往返+非法值拦截）、TestUpdateCredentials、TestListFiltersByTag、TestSettleByCardWithActualServiceOverride、TestSettleByPayCompletesAndSingleValid（含 WAITING 直收+CANCELLED 拒绝）。
