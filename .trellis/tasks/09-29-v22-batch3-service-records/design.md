# Design — V2.2 第三批：服务记录与散客快速结算

## 0. 事实基础（已核实，勿重复考古）

- `payment`（006+010 最终形态）：`member_id CHAR(26) NOT NULL`、`appointment_id NULL`、method CHECK IN ('CARD','WECHAT_TRANSFER','CASH','OTHER')、status VALID/VOIDED、`idempotency_key UNIQUE`、`valid_lock` VIRTUAL 生成列 + `uk_payment_valid_lock`、trg_payment_updated_at、idx_payment_appointment / idx_payment_status_recorded、FK 仅 fk_pay_appointment。**没有任何表 FK 指向 payment**（015 新建的 service_record 除外，同 migration 内后建）。
- `redemption`：member_card_id NOT NULL（卡维度专用，散客快速结算**不碰 redemption**）。
- ReverseRedemption（settle.go:398）已实现"撤销同事务置 payment VOIDED（按 idempotency_key 定位 CARD payment）"。
- 模块布线（app.go:47）：`transaction.New(db, cfg, cardMod, appointmentMod, memberMod, serviceMod)` —— transaction 已注入 member/service 句柄，直接调对方**导出方法**（api.go 注释即边界）。
- content settings：validateSetting + 下发清洗模式见 content/repo.go（amap/profile 键为先例）。
- migration 模式：Migrate 每文件单事务 → `PRAGMA foreign_keys=OFF` 在事务内无效；**重建表用 `PRAGMA defer_foreign_keys=ON` → DROP → 同名 CREATE → INSERT SELECT 回填**（014_member_password.sql 为先例，含升级路径测试 migrate_014_test）。

## 1. Migration `015_service_records.sql`

### 1.1 service_tag

```sql
CREATE TABLE service_tag (
  id         CHAR(26)    NOT NULL PRIMARY KEY,
  tag_group  VARCHAR(20) NOT NULL,
  name       VARCHAR(32) NOT NULL,
  sort       INTEGER     NOT NULL DEFAULT 0,
  status     VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME    NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at DATETIME    NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT uk_tag_group_name UNIQUE (tag_group, name),
  CONSTRAINT ck_tag_group  CHECK (tag_group IN ('BODY_PART','METHOD')),
  CONSTRAINT ck_tag_status CHECK (status IN ('ACTIVE','DISABLED'))
);
CREATE INDEX idx_tag_group_sort ON service_tag (tag_group, sort);
```

- 不建 FK；历史记录存**名称快照**，标签行删除/停用不影响历史（§五）。
- updated_at 触发器照 payment 先例。

### 1.2 service_record

```sql
CREATE TABLE service_record (
  id               CHAR(26)     NOT NULL PRIMARY KEY,
  member_id        CHAR(26)     NULL,            -- 散客不录手机号时无记录行，本列仍可空（防御）
  appointment_id   CHAR(26)     NULL,
  payment_id       CHAR(26)     NOT NULL,        -- 每条记录对应一笔 VALID payment
  redemption_id    CHAR(26)     NULL,            -- 卡核销时回填；散客现金/微信为 NULL
  service_id       CHAR(26)     NOT NULL,
  service_name     VARCHAR(128) NOT NULL,        -- 快照
  body_parts       VARCHAR(256) NOT NULL DEFAULT '[]', -- JSON 字符串数组（标签名快照）≤3
  service_method   VARCHAR(64)  NOT NULL DEFAULT '',   -- 方式名快照
  tech_note        VARCHAR(200) NOT NULL DEFAULT '',
  merchant_note    VARCHAR(500) NOT NULL DEFAULT '',
  merchant_note_by CHAR(26)     NULL,
  merchant_note_at DATETIME     NULL,
  communicated     INTEGER      NOT NULL DEFAULT 0,
  status           VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',  -- ACTIVE/REVERSED
  reversed_at      DATETIME     NULL,
  created_by       CHAR(26)     NULL,            -- operator（identity_user.id）
  created_at       DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  updated_at       DATETIME     NOT NULL DEFAULT (datetime('now','+8 hours')),
  CONSTRAINT ck_rec_status CHECK (status IN ('ACTIVE','REVERSED')),
  CONSTRAINT ck_rec_communicated CHECK (communicated IN (0,1)),
  CONSTRAINT fk_rec_member       FOREIGN KEY (member_id)      REFERENCES member (id),
  CONSTRAINT fk_rec_appointment  FOREIGN KEY (appointment_id) REFERENCES appointment (id),
  CONSTRAINT fk_rec_payment      FOREIGN KEY (payment_id)     REFERENCES payment (id),
  CONSTRAINT fk_rec_redemption   FOREIGN KEY (redemption_id)  REFERENCES redemption (id)
);
CREATE INDEX idx_rec_member_time ON service_record (member_id, created_at);
CREATE INDEX idx_rec_appointment ON service_record (appointment_id);
CREATE INDEX idx_rec_redemption  ON service_record (redemption_id);
CREATE INDEX idx_rec_payment     ON service_record (payment_id);
```

### 1.3 payment 重建（member_id 放宽为 NULL——散客不录手机号仅记账）

照 014 模式：`PRAGMA defer_foreign_keys=ON;` → `DROP TABLE payment;` → 按最终形态同名 CREATE（**列 = 006 全列 + idempotency_key + valid_lock 生成列内联（保持 VIRTUAL 语义）+ 全部 CHECK/FK/触发器**），唯一改动 `member_id CHAR(26) NULL` → `INSERT INTO payment (显式列, 不含生成列) SELECT ... FROM tmp_payment` 回填（先把旧表 RENAME 为 tmp_payment 再建新表，或 DROP 前建临时副本——参照 014 的实际写法）。重建后必须重建两个唯一索引 + 普通索引 + 触发器。**011 的 ALTER 历史不动**（只增不改）。

## 2. Server

### 2.1 service 模块（owner：service_tag + service_record）

新文件 `tags.go`、`records.go`；路由挂 handler.go：

| 路由 | 语义 |
|---|---|
| `GET /admin/service-tags?group=BODY_PART\|METHOD` | 全量（含 DISABLED），每项带 `used`（是否被 service_record 引用）；按 sort, created_at 排序 |
| `POST /admin/service-tags` `{group,name}` | 添加；sort = 组内 max+1 |
| `PUT /admin/service-tags/{id}` `{name?,sort?,status?}` | 改名/排序/停用启用 |
| `DELETE /admin/service-tags/{id}` | 仅未使用可删；used → 409 |
| `GET /admin/members/{id}/service-records` | 服务追踪（见下） |
| `PUT /admin/service-records/{id}/merchant-note` `{note}` | 追加/修改商家备注（记录 by/at） |

- 校验：name trim 后 1..12 字符、禁止含 `% _ " '`（保证 LIKE 定位安全与快照整洁）；`TAG_EXISTS` 409（uk 冲突）、`TAG_BAD_NAME`/`TAG_BAD_GROUP` 400、`TAG_IN_USE` 409。
- used 判定：METHOD 组 `EXISTS(service_record WHERE service_method = name AND …)`；BODY_PART 组 `body_parts LIKE '%"name"%'`（名称已禁引号，安全）。REVERSED 记录也算"使用过"（历史可查）。
- **跨模块入口（transaction 在其事务内调用，D6）**：
  - `InsertRecordTx(ctx, tx shared.Tx, in RecordInput) error` —— RecordInput{memberID, appointmentID, paymentID, redemptionID, serviceID, serviceName, bodyParts []string, method, techNote, communicated, operatorID}；内部校验：parts ≤3 且每项 1..32 字符、method ≤64、techNote ≤200，**communicated=false → 报错**（由 transaction 层提前校验亦可，此处兜底）。
  - `ReverseByRedemptionTx(ctx, tx, redemptionID) error` —— `UPDATE service_record SET status='REVERSED', reversed_at=… WHERE redemption_id=? AND status='ACTIVE'`（幂等：无行/已 REVERSED 均不报错）。
- `RevokeRecord(ctx, recordID, operatorID)`：独立撤销入口，仅限 `redemption_id IS NULL` 的记录（散客现金/微信）：单事务 `UPDATE service_record → REVERSED` + `UPDATE payment SET status='VOIDED', remark=remark||'；记录撤销' WHERE id=? AND status='VALID'`（payment 非 VALID → 409 `PAY_ALREADY_VOIDED`）；redemption_id 非空 → 400 `REC_CARD_REVERSAL`（"卡核销记录请通过核销撤销入口撤销，以恢复卡次数"）。
- 服务追踪查询 `GET /admin/members/{id}/service-records?part=&method=&range=1m|3m|all&q=`：
  - WHERE member_id=? AND status='ACTIVE'（REVERSED 不出现在追踪流）+ 可选 `body_parts LIKE %"part"%`、`service_method=?`、`created_at >= 近1/3月起点（墙上时间，Asia/Shanghai）`、`(tech_note LIKE %q% OR merchant_note LIKE %q% OR service_name LIKE %q%)`；ORDER BY created_at DESC, id DESC。
  - 响应 `{items:[{id, service_name, body_parts(JSON解析为数组), service_method, tech_note, merchant_note, merchant_note_by_name, merchant_note_at, communicated, status, appointment_id, payment_method, payment_amount, payment_status, created_at, reversed_at}], summary:{total_all, total_3m, parts:[{name,count}], methods:[{name,count}]}}`。
  - summary 全库算（排除 REVERSED）：total_all=全部 ACTIVE 数、total_3m=近3月数、parts/methods=全部时间 top 若干（按 count DESC, name ASC）。merchant_note_by_name 经 JOIN identity_user 取 name。
  - GET 时同响应附 `filters:{parts:[…含 DISABLED 全量名], methods:[…]}` 供历史筛选器（§五：停用标签在历史筛选仍可见）。

### 2.2 member 模块

- 新增导出方法（api.go 注明边界）：`EnsureByPhoneTx(ctx, tx shared.Tx, phone, name string) (memberID string, created bool, err error)` —— 按 phone 查 member；命中返回；未命中创建最小档案（member_no 走 nextMemberNo 原子计数器、phone、可选 name、无密码无 openid）。校验 phone `^1\d{10}$`（复用 password.go 的 validatePhone）。若 password.go 的 CreateWithPhone 已有等价插入逻辑则复用其内部 helper，不复制 SQL。

### 2.3 transaction 模块

- **四个结算入口统一追加服务记录字段**（请求体新增：`body_parts []string≤3`、`service_method string`、`tech_note string≤200`、`communicated bool`）：
  - `POST /admin/appointments/{id}/redeem`（SettleByCard）
  - `POST /admin/cards/{id}/redeem`（RedeemWalkIn）
  - `POST /admin/appointments/{id}/payments`（SettleByPay）
  - `POST /admin/walkin/settle`（新增）
- **强制沟通确认**：`communicated != true` → 400 `TX_NEED_CONFIRM`（"请先勾选「服务前已完成沟通」"）。四个入口一律（§四.6 建议强制）。
- 事务内（payment/redemption 落库之后）调用 `service.InsertRecordTx`；`ReverseRedemption` 事务内追加 `service.ReverseByRedemptionTx`。service_record 快照的 service_name 用与 redemption/payment 相同的快照来源。
- **新增散客快速结算** `POST /admin/walkin/settle`：
  ```
  { phone?, name?, service_id, pay_method, amount, reference_no?, remark?,
    body_parts?, service_method?, tech_note?, communicated, idempotency_key }
  ```
  - 单 immediate 事务：服务项校验（存在且在架，快照 name+默认价；`TX_WALKIN_SERVICE`）；`pay_method ∈ {CASH, WECHAT_TRANSFER, OTHER}`（CARD → 400 `TX_WALKIN_NO_CARD`"无卡结算不支持卡核销方式"）；amount ≥0 整数分（`TX_WALKIN_AMOUNT`）；idempotency_key 幂等回放（uk_payment_idempotency，同 W2 模式）。
  - phone 非空 → trim + `member.EnsureByPhoneTx`（created=true 时响应 `member_created:true`）；payment.member_id = 该 member；`service.InsertRecordTx` 落记录。
  - phone 为空 → payment.member_id = NULL，**不落 service_record**（§四.4：仅记账）。请求里的记录字段被忽略；响应 `record:null` 并带 `record_skipped:true` 供 UI 提示。
  - 响应 `{payment:{…}, record:{…}|null, member_created:bool}`。
  - D19 的 RDM_WALKIN_BLOCKED（今日有预约会员拦卡核销）**不**套用到现金/微信散客结算（现金无卡无 D9 绕过问题），知悉即可。
- payment 列表/workbench 中 member 展示：member_id 为 NULL 的行显示"未登记"——检查 handleListPayments/workbench 的 member JOIN（若 INNER JOIN 改 LEFT JOIN + COALESCE 名称）。

### 2.4 content 模块（settings KV）

- `service_note_presets`：值为 JSON 字符串数组（≤20 项，每项 trim 后 1..50 字符；非法 → `SETTING_BAD_VALUE`）；下发清洗同 amap 先例（仅合法数组下发，空数组合法=清空）。写入走既有 SaveSetting（注意批量保存通道也要过校验）。
- AGENTS.md 冻结决策追加：**D28**（服务标签/服务记录：名称快照、撤销联动、商家内部数据用户不可见）、**D29**（散客快速结算：payment.member_id 放宽 NULL；不录手机号仅记账无记录；沟通确认强制；CARD 拒绝）。

## 3. Admin 前端（apps/admin）

### 3.1 新页 `ServiceTagsPage.vue`（路由 `/service-tags`，菜单「服务标签管理」放设置分组）

- 两个卡片：调理部位（BODY_PART）/ 服务方式（METHOD）：输入框添加；列表行 = 名称 + 上移/下移（swap sort，PUT）+ 停用/启用（PUT status）+ 删除（DELETE；409 → toast"该标签已被服务记录使用，仅可停用"）。停用行置灰可见。
- 第三卡片「技师备注快捷短语」：chip 编辑（添加/删除，≤20 条）→ 保存 `service_note_presets`。

### 3.2 结算弹窗扩展（AppointmentsPage 的卡核销/现金微信结算弹窗、MembersPage 的直接卡核销弹窗）

- 追加区块（全部选填，除强制勾选）：部位多选 chips（≤3，超出禁选）、方式单选 chips、技师备注 textarea + 快捷短语 chips 点击填入（数据来自 settings `service_note_presets`，经 admin settings 读取接口）、底部必勾「服务前已完成沟通」checkbox（未勾 → 提交按钮禁用）。
- 标签数据源：GET /admin/service-tags（仅 ACTIVE 出现在录入 chips）。

### 3.3 散客结算

- AppointmentsPage 工具栏加「散客结算」按钮 → 弹窗：手机号（选填，填了才显示姓名输入，新建时提示"将创建会员档案"）、服务项下拉（在架服务，选中后金额默认填默认价，可改）、收款方式 radio（现金/微信/其他）、备注；手机号为空时记录区块整体禁用并提示"不录手机号仅记账，不生成服务记录"；沟通确认必勾；提交 → POST /admin/walkin/settle；成功 toast（含"已创建会员档案"/"已记账（未关联会员）"）。

### 3.4 会员详情「服务追踪」（MembersPage）

- 详情抽屉/面板加「服务追踪」区：汇总卡（近3个月共 N 次；部位/方式 counts）；筛选行（部位/方式下拉——含停用标签、时间 radio 近1月/近3月/全部、关键词输入）；列表倒序：日期 + 服务名 + 金额/方式（payment）+ 部位 + 方式 + 技师备注 + 商家备注（无则「+ 商家备注」按钮 → 弹窗输入 PUT merchant-note，显示操作人+时间）；REVERSED 不展示。

### 3.5 RecordsPage

- 收款列表 member 名为空显示"未登记"（LEFT JOIN 语义的兜底 UI）。

## 4. 测试

- migration：migrate_015_test（照 014 模式：旧库注入 payment 行 → 升级 → 行保留、member_id NULL 可写、valid_lock 唯一仍生效（同 appointment 两笔 VALID 第二笔 dup 报错）、FK 强制（孤儿 member_id payment…注意 member 无 FK，改测孤儿 service_record.payment_id 插入失败）、触发器工作）。
- service：tags CRUD/去重/used 删除断言；InsertRecordTx 校验；RevokeRecord 分支；ReverseByRedemptionTx 幂等；追踪筛选与 summary（构造含 REVERSED 数据）。
- transaction：四入口 communicated 缺省拒绝；walkin 有/无手机号、CARD 拒绝、幂等重放、服务快照与金额校验；ReverseRedemption 后 record REVERSED。
- content：service_note_presets 合法/非法/清洗。
- 既有 settle 测试签名适配（新增记录参数传合法值）。

## 5. 风险与回退

- payment 重建是本批唯一高危点：014 已验证重建模式；015 测试必须含升级路径。
- 回退 = 该批 commit revert；015 前进兼容（旧代码读新库：payment 多了 NULL member_id 行，列表查询需容忍）。
