# PRD：V1.x 预约体系升级 + 散客核销 + 核销码门槛 + 商家端控件打磨

> 任务目录：`.trellis/tasks/2026-09-27-v1x-booking-redeem-ux/`
> 状态机/资金不变量继续受 AGENTS.md D1–D18 约束；本 PRD 增补 D19–D22（文末）。
> 原则：歧义选"不做"；金额整数分；价格/服务 snapshot；核心表禁物理删除；所有 schema 变更走新增 migration。

## 0. 需求清单（用户原话归纳）

| # | 需求 | 类型 |
|---|------|------|
| N1 | 核销码页增加显示：实时时间（年月日+时分秒）与手机号 | 显示优化 |
| N2 | 无预约顾客：只要卡有剩余次数，商家端也能核销（散客到店） | 业务规则 |
| N3 | 核销码门槛：无有效付费服务（无 ACTIVE 卡）不出核销码，不管有无预约 | 业务规则 |
| N4 | 商家端改期用日期/时间控件替代手动输入；卡次数调整改 +/- 步进器 | 交互 |
| N5 | 顾客端"我的预约"不显示任何记录（已定位：前端解包 bug） | Bug 修复 |
| N6 | 预约时间选择重做：日期→必选上午/下午→选填具体时间；后台可配置营业时间/间隔/每时段人数；半天名额池；模糊预约显示"上午/下午"；闭店日历；配置不追溯 | 功能 |

## 1. 现状关键事实（已核实）

- 顾客端 `core/api/http.ts` 的 `request()` 返回 `payload.data`（已解包包络）。分页端点 `/api/appointments` 顶层即 `{data:[...],total,...}`，因此 `http.get<Page<...>>` 实际返回**数组**。`MyAppointmentsPage.vue:13-14` 取 `page.data ?? []` 恒为空 → **N5 根因**。修法：`api.myAppointments` 返回类型改 `Appointment[]`，页面直接使用。
- `redemption.appointment_id` / `payment.appointment_id` 均 NOT NULL 且有 FK；`active_lock`/`valid_lock` 生成列 `CASE WHEN status='SUCCESS' THEN appointment_id`——appointment_id 为 NULL 时生成列也是 NULL，MySQL 唯一索引允许多个 NULL → 散客核销无需改唯一约束语义。
- `card.ValidateForRedeem`（card/redeem.go:31）：校验 ACTIVE/有效期/余额/card_service_rule 含该服务。散客核销必须选服务（决定规则校验与收款金额）。
- `SettleByCard`（transaction/settle.go）：锁卡→校验→扣次→REDEEM 流水→redemption→payment(CARD)→MarkCompleted→TouchLastVisit，幂等键防重放。散客核销复制此链路但**不建预约、不 MarkCompleted**。
- 预约创建 `Create`（appointment/lifecycle.go:157）：validateWindow（读 `p.cfg.Business`，静态）→ acquireCalendar(D5 NamedLock) → 事务内 activeExists 重叠冲突。容量模型需替换 activeExists。
- settings KV 在 content 模块（`repo.go Settings()/SaveSetting()`，已有 GET/PUT /admin/settings）。appointment 模块现在不依赖 content。
- 顾客预约 UI（BookingPage.vue）目前直接平铺全天时间槽，且用本地常量 DEFAULT_HOURS（与后端配置脱节）。
- 商家端改期在 AptActionButtons → AppointmentsPage（手输时间）；卡调整在 MembersPage `doAdjust`（ElMessageBox.prompt 输入 ±数字）。
- 顾客端核销码页 QRCodePage 无门槛、无时间/手机号显示。

## 2. 数据库迁移（011_booking_v1x.sql，只增不改）

```sql
-- 散客核销：预约可空
ALTER TABLE payment    MODIFY appointment_id CHAR(26) NULL;
ALTER TABLE redemption MODIFY appointment_id CHAR(26) NULL;

-- 模糊预约类型（存量行默认 SPECIFIC）
ALTER TABLE appointment
  ADD COLUMN slot_type VARCHAR(12) NOT NULL DEFAULT 'SPECIFIC',
  ADD CONSTRAINT ck_appointment_slot_type CHECK (slot_type IN ('SPECIFIC','HALF_DAY'));

-- 闭店日历（全天 = AM+PM 两行）
CREATE TABLE appointment_closure (
  id           CHAR(26)    NOT NULL PRIMARY KEY,
  closure_date DATE        NOT NULL,
  day_part     ENUM('AM','PM') NOT NULL,
  remark       VARCHAR(200) NOT NULL DEFAULT '',
  created_at   DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_closure_date_part (closure_date, day_part)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

settings 新键（经现有 PUT /admin/settings 保存）。**读取层（content 模块新增 `BusinessSettings()` 类型化访问器）在键缺失时的硬编码默认**：open 09:00、close **20:00**（用户指定）、slot 30、capacity 1、noon 12:00；每次直读表（表极小，不做缓存）。`cfg.Business` 停用（避免双事实源）。顾客端首页 PublicSettings 的 `open_time/close_time` 输出改从 business_* 键取值（键名对前端不变），保证"后台改营业时间 → 顾客端首页联动"。

## 3. 预约域设计（N6）

### 3.1 半天边界
- AM = [open_time, noon_split)，PM = [noon_split, close_time)。半天边界由服务端从配置计算，不信任前端。
- HALF_DAY 预约落库：scheduled_start=半天开始、scheduled_end=半天结束、slot_type='HALF_DAY'。服务 snapshot 照存（结算价格用）。

### 3.2 创建 API
`POST /api/appointments` body：`{service_id, start_time?, day_part?, note}`，二选一：
- `start_time`（YYYY-MM-DD HH:MM）→ SPECIFIC（现行为不变）；
- `day_part`（"AM"|"PM"）→ HALF_DAY。
同时给两个 → 400。都缺 → 400。

### 3.3 校验（创建与改期共用 `validateBooking`，NamedLock 内执行）
1. 服务 ACTIVE；窗口检查（SPECIFIC：start≥open、end≤close、对齐 slot_minutes；HALF_DAY：窗口必须精确等于半天边界）。
2. 闭店检查：预约覆盖的每个半天（SPECIFIC 可能跨中午？不会——SPECIFIC end≤close 但可能跨 noon_split；跨中午的预约按其覆盖的两个半天都必须未闭店；容量也计入两个半天池）任一 closure 存在 → `APT_CLOSED 该时段店铺休息`。
3. 提前量：SPECIFIC 保持 start ≥ now+2h；**HALF_DAY 只要求该半天结束时间 > now**（模糊预约由店主安排，提前量无意义；当天下午仍可约下午）。
4. 容量（在 NamedLock + 事务内，改期排除自身）：
   - **逐槽并发检查（仅 SPECIFIC）**：目标窗口覆盖的每个 slot，与其他 **SPECIFIC** active 预约的重叠计数 < capacity；否则 `APT_SLOT_FULL`。保证钉死时间的预约物理上不撞车。
   - **半日名额池（SPECIFIC 与 HALF_DAY 都查）**：半日 active 总数（SPECIFIC 与该半天重叠者 + HALF_DAY 该半天者，各计 1）< 半日 slot 数 × capacity；否则 `APT_HALFDAY_FULL`。
   - 设计依据：用户明确定义"半天总名额 = 该半天时段数 × 每时段人数，模糊与具体都从总名额里扣"。若让 HALF_DAY 参与逐槽并发，一个模糊预约会占满整个半天的并发额度（上午只能接 1 单），直接违背该公式；模糊预约的物理排队由名额池 + 店主顺序安排 + 提交时"可能需要等待"提示共同兜底（用户已知情接受）。
   - active = status IN ('PENDING_CONFIRM','CONFIRMED','IN_SERVICE')（与现口径一致）。
   - SQL 形态：重叠计数 `SELECT COUNT(*) FROM appointment WHERE status IN (...) AND scheduled_start < :win_end AND scheduled_end > :win_start AND id <> :self`；逐槽用槽窗口，半日池用半天边界窗口（`AND slot_type='SPECIFIC'` 限定逐槽检查范围）。
5. 改期 `PUT /api|admin/appointments/{id}/reschedule` body 同样支持 `{start_time}` 或 `{day_part}`（模糊→具体、具体→模糊、模糊→另一半天均允许）；状态限制不变（D8：PENDING_CONFIRM/CONFIRMED，同 appointment 改期，排除自身）。

### 3.4 booking-options（顾客/商家共用内部函数）
`BookingOptions(ctx, date)` 返回：
```json
{ "date":"2026-09-28", "open":true,
  "am": {"closed":false,"total":6,"remaining":5,
         "slots":[{"time":"09:00","remaining":1}, ...]},
  "pm": {"closed":false,"total":16,"remaining":16,"slots":[...]} }
```
- slots 依据当日配置生成；`remaining = capacity − 该槽 active 重叠数`；半日 remaining = 半日 total − 半日 active 数（HALF_DAY 也计入）。
- closed = 当日该半天存在 closure。
- open = (am 未闭店且池 remaining>0) OR (pm 未闭店且池 remaining>0)（某半天闭店但另一半天可约时整日仍可选）。
- 逐槽 remaining = capacity − 该槽 SPECIFIC active 重叠数；半日 remaining = 半日总额 − 半日 active 总数（含 HALF_DAY）。
- 暴露：`GET /api/booking-options?date=`（顾客 JWT）与 `GET /admin/booking-options?date=`（商家改期对话框用）。

### 3.5 闭店管理（商家）
- `GET /admin/closures?from=`（默认今天起）列表；`POST /admin/closures {date, day_part: AM|PM|FULL, remark}`（FULL=写两行）；`DELETE /admin/closures/{id}`。
- POST 的 conflict_count 统计与写入在 D5 同一把 calendar 锁内完成（消除与顾客下单的竞态窗口）；返回 `{conflict_count}` = 该半天 active 预约数；前端 >0 时弹确认："该半天已有 N 个预约，闭店后需手动联系顾客改期"（允许继续创建——知情操作，符合"配置不追溯"）。
- 顾客端创建/改期命中 closure → APT_CLOSED。

### 3.6 展示
- appointment JSON 增加 `slot_type`、`day_part`（后端按 start 与 noon_split 计算，避免前端重复逻辑）。
- 顾客端"我的预约"：HALF_DAY → "9月28日 上午"；SPECIFIC → 现状。
- 商家端 today/预约列表：HALF_DAY → 徽标"上午到店/下午到店"，时间列显示半天窗口。

## 4. 散客核销（N2）

- 新事务 `transaction.RedeemWalkIn(ctx, memberID, cardID, serviceID, operatorID, idemKey)`：
  锁卡（复用 LockForRedeem）→ 校验属主（c.MemberID==memberID）→ ValidateForRedeem(serviceID,1) → ApplyRedeem → redemption(appointment_id=NULL, service_id) → payment(CARD, amount=服务默认价, appointment_id=NULL, remark="散客核销") → TouchLastVisit。幂等键复用现有机制；isDupKey 冲突时按 idempotency_key 回放。
- 金额来源：handler 经 service 模块 api（transaction.New 增加 services 依赖，合法方向）取 `default_price`。
- 路由：`POST /admin/cards/{id}/redeem` body `{service_id, idempotency_key}`。
- **撤销与回放定位改按幂等键（P0 修复）**：现有 `ReverseRedemption` 作废收款按 `WHERE appointment_id=?` 定位 CARD payment，散客行 appointment_id 为 NULL 永不匹配 → 必须改为 `WHERE idempotency_key=? AND method='CARD' AND status='VALID'`（redemption 与 payment 共用同一 idempotency_key，对预约流同样成立）；`paymentForRedemption` 回放定位同步改为按 redemption.idempotency_key 查 payment。此改动对预约流行为不变（键一一对应）。
- **NULL 化波及清单（漏改即 500）**：`scanRedemption`/`scanPayment` 的 appointment_id 改 `*string`/sql.NullString；`SettleByCard` 结构体字面量不变（预约流恒非空）；`admin.ts` Payment/Redemption 类型 `appointment_id: string|null`；RecordsPage 渲染 appointment_id 为 null 时显示"散客"；核销记录列表显示散客标记。
- 撤销后可再核销（active_lock NULL 语义）；撤销复用现有 Reverse 入口（仅内部定位 SQL 变更）。
- Redemption/Payment 结构体 `AppointmentID` 改 `*string`（JSON null），前端列表显示"散客"。
- ScanRedeemDialog UI：resolved 步骤增加"散客直接核销（今日无预约）"入口：
  - **仅当该会员今日无任何预约（myApts.length===0）时自动开启散客模式**；有今日预约的会员不出现散客按钮（防止绕过 D9 的 IN_SERVICE 门槛重复核销）；
  - 散客模式选卡后，按所选卡的模板规则（listCardTemplates 回显 service_ids）列出可核销服务供点选 → 核销；
  - 有预约的会员仍走原路径（预约需 IN_SERVICE，D9 不变）。

## 5. 核销码门槛与显示（N1/N3，顾客端 only）

- QRCodePage onMounted：并行取 myProfile + myCards；存在 status='ACTIVE' 卡 → 渲染二维码 + 实时时钟（每秒 setInterval，格式 YYYY-MM-DD HH:mm:ss）+ 手机号；无 ACTIVE 卡 → 引导态："暂无有效会员卡，核销码不可用。可到店办理或浏览服务项目"（不渲染二维码）。不看预约状态（按用户口径）。
- 后端零改动（myCards/myProfile 已有）。
- 商家侧兜底不变：扫到无卡会员时弹窗显示"该会员没有可用会员卡"。

## 6. 商家端控件（N4）

- **RescheduleDialog**（新组件，AppointmentsPage 复用，AptActionButtons 触发）：el-date-picker(date) + 后端 booking-options 拉当天选项 → "上午/下午"两个 chip + 该半天具体时间槽 chips（满槽禁用，显示剩余）→ 确定。模糊预约可改成具体时间，反之亦然。
- **AdjustCountDialog**（新组件，MembersPage）：当前剩余大字展示 + `el-input-number`（步长 1，可负，手输仍允许）+ 快捷 chips（-3/-1/+1/+3）+ 备注输入 → adjustCard。
- **ClosureDialog**（新组件，AppointmentsPage 工具栏"闭店设置"）：日期 + 全天/上午/下午 + 备注；下方列表可删（§3.5）。
- **设置页**（ContentPage 系统设置卡片追加"营业配置"）：open/close（el-time-select）、slot_minutes（radio 30/60/120）、slot_capacity（el-input-number 1-20）、noon_split（el-time-select）→ 逐键 PUT /admin/settings。
- ScanRedeemDialog 散客模式（§4）。

## 7. 顾客端（Phase 3）

- **BookingPage 重做**：选服务 → 选日期（candidateDays 保留）→ 拉当天 booking-options → "上午/下午"必选 chip（closed 的半天隐藏；全天闭店显示"该日休息"）→ 该半天时间槽 chips（选填、满槽禁用）→ 提交：未选具体时间则 `day_part` 模糊预约，弹 toast"具体时间由店主安排，可能需要等待"；选了具体时间走 start_time。错误码文案：APT_CLOSED→"该时段店铺休息"，APT_SLOT_FULL/APT_HALFDAY_FULL→"这个时间刚被约满，换个时间试试"，并刷新 booking-options。
- **MyAppointmentsPage**：修 N5（直接用数组）；时间列按 slot_type 渲染"X月X日 上午/下午"；改期从 window.prompt 换成轻量选择层（日期 + 上/下午 + 可选时间槽，同 booking-options）。
- **QRCodePage**（§5）。

## 8. 不做（歧义选不做口径）

- 不做：预约时段人数的人员分配/排队叫号、闭店自动取消或自动改期、短信通知、模糊预约的优先级、多服务组合预约、顾客选指定商家、CORE 状态机新增状态。
- OPERATOR 账号管理、真实短信通道（另行立项）。
- closure 不做"每周固定休息日"模板（手工按天设置）。

## 9. 冻结决策增补（写入 AGENTS.md）

- **D19**：散客核销 = redemption/payment.appointment_id 为 NULL 的核销，必须指定服务（规则校验+金额=服务默认价）；一卡同事务扣次；撤销复用原链路但 **payment 作废与回放定位一律按 idempotency_key**（appointment 维度不变量只约束 appointment_id 非空行）。
- **D20**：预约窗口/间隔/容量/上下午分界以 settings 为准（键缺失用硬编码默认 09:00/20:00/30/1/12:00；cfg.Business 为次级兜底）；**逐槽并发 ≤ capacity 仅约束 SPECIFIC**（物理不撞车），**半日池（< 槽数×容量）对 SPECIFIC 与 HALF_DAY 一体适用**（模糊预约按用户公式从池扣减，不占逐槽并发）；校验一律 NamedLock+事务内。
- **D21**：核销码仅对持有 ACTIVE member_card 的顾客展示（前端门槛 + 商家端后端校验兜底）；核销码内容协议不变（ANMO-MEMBER:<ulid>）。
- **D22**：闭店 appointment_closure 按 (date, AM|PM) 粒度；创建闭店在 calendar 锁内统计 conflict_count，允许带预约（商家知情确认）；不自动改约。

## 10. 测试与验收

- go test 新增：appointment（HALF_DAY 创建/池容量/逐槽容量/闭店拒绝/模糊改具体排除自身）、transaction（散客核销幂等/撤销/规则拦截）、跨正午 SPECIFIC 占两池。
- E2E 脚本（经 5174 代理）：顾客模糊预约→商家列表徽标→商家改期成具体→开始服务→预约核销；散客核销→撤销；闭店创建带预约提示→顾客端该半天不可约；booking-options 剩余数正确。
- 前端：两端 vue-tsc + build。
- 回归：既有 13 测试包全绿（注意 validateWindow 改造对 repo_test 的影响——settings 无值时回落 cfg，老用例应不破坏；若用例直接依赖 cfg 值，保持行为一致）。

## 11. Phase 划分

- **Phase 1** 后端预约域：migration 011 + settings 业务配置读取（含缓存/兜底）+ validateBooking 重构 + HALF_DAY 创建/改期 + 容量 + closures CRUD + booking-options（api/admin 两端点）+ slot_type/day_part 输出 + 单测。
- **Phase 2** 后端散客核销：transaction 增 services 依赖 + RedeemWalkIn + 路由 + 结构体可空化 + 单测。
- **Phase 3** 顾客端：BookingPage 重做、MyAppointments 修复+模糊显示+改期选择层、QRCodePage 门槛/时钟/手机号。
- **Phase 4** 商家端：RescheduleDialog、AdjustCountDialog、ClosureDialog、ScanRedeemDialog 散客模式、设置页营业配置、列表徽标。
- **Phase 5** E2E + 文档（TUTORIAL/AGENTS/plan 增补/journal PHASE RESULT）+ 全量回归。
