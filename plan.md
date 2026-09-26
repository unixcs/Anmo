# 个人到店按摩店服务管理系统 V1
## 产品、数据库、核心业务、技术架构与 Codex 执行 Plan

---

# 0. 项目最高原则

这是一个：

**个人到店按摩店服务管理系统。**

不是：

- 上门按摩系统
- 美容院 SaaS
- 多技师排班系统
- 连锁门店系统
- 多租户 SaaS
- 综合按摩行业 ERP

当前真实经营模型：

```text
店主
＝
按摩师
＝
主要服务人员
```

未来可能：

```text
店主
+
家人
```

家人主要帮助：

- 管理预约
- 管理会员
- 发卡
- 核销
- 看后台

而不是建立一个复杂的员工管理体系。

---

# 1. 第一性原理

这个系统真正要解决的问题只有：

```text
今天谁来？
几点来？
做什么？
有没有预约冲突？
有没有会员卡？
做完以后扣没扣？
钱有没有记录？
```

因此核心业务只有：

```text
客户
 ↓
服务项目
 ↓
预约
 ↓
到店
 ↓
服务
 ↓
结算
 ↓
会员卡核销 / 微信转账 / 现金
 ↓
记录
```

会员卡、首页、日志、洞察都是外围能力。

真正的核心对象：

# 一次到店服务

---

# 2. V1 核心业务闭环

最终闭环：

```text
客户
 ↓
登录
 ↓
查看服务
 ↓
选择日期/时间
 ↓
提交预约
 ↓
老板确认
 ↓
客户到店
 ↓
开始服务
 ↓
服务完成
 ↓
选择结算方式
 ├── 会员卡
 │     ↓
 │   扣1次
 │
 ├── 微信转账
 │
 └── 现金
 ↓
预约完成
 ↓
形成服务/交易记录
```

如果客户没有会员卡：

也完全可以预约。

所以：

**预约 ≠ 办卡。**

---

# 3. V1 产品边界

## 3.1 V1 必须做

### 顾客端

- 手机号登录
- 首页
- 查看服务项目
- 查看会员卡
- 查看预约
- 创建预约
- 取消预约
- 改期
- 查看历史服务
- 管理基础个人资料

### 老板后台

- 今日工作台
- 会员管理
- 服务项目管理
- 卡模板管理
- 发卡
- 续卡
- 调整次数
- 预约管理
- 确认预约
- 开始服务
- 完成服务
- 核销
- 收款记录
- 撤销核销
- 操作日志
- 首页内容管理
- 基础客户洞察

---

# 4. V1 明确不做

绝对不要因为“以后可能需要”提前做：

- 上门服务
- 客户地址
- 技师派单
- 技师调度
- 技师排班
- 自动派单
- 地图
- GPS
- 路线
- 服务区域
- 距离计费
- 多门店
- 多租户
- 复杂员工管理
- 提成
- 佣金
- 分销
- 积分
- 优惠券
- 商城
- 商品库存
- 在线微信支付
- 微信支付回调
- AI 客服
- 复杂营销
- 服务人员 App

原则：

> V1 只解决“一个老板每天如何把店里的预约、会员和核销管理清楚”。

---

# 5. 未来扩展原则

以后如果店里真的出现：

```text
老板
+
技师 A
+
技师 B
```

再增加：

```text
staff
appointment.staff_id
staff_schedule
```

现在不要。

同样：

以后如果变成：

```text
一个店
 ↓
多个门店
```

再做：

```text
store
tenant
```

现在不要。

---

# 6. 开源项目借鉴原则

可以借鉴成熟项目的：

- 模块拆分
- 数据关系
- CRUD 组织
- 会员卡设计
- 交易流水
- 预约 UX
- 后台信息架构

不要直接复制代码。

## MMS

重点参考：

- members
- cards
- card_types
- transactions
- appointments
- services
- payment_methods
- audit_logs

它的模块划分比较接近本项目未来可能需要的能力。

特别参考：

**卡余额 + 交易流水 + 数据库事务。**

但它是 AGPL-3.0：

所以：

**只参考架构和业务思想，不直接复制代码。**

---

# 7. QingSi

重点参考：

- 会员
- 卡
- 服务项目
- 预约
- 后台管理
- 顾客自助预约



定位：

**产品 UX 参考。**

---

# 8. OpenSalon

重点参考：

```text
clients
services
appointments
appointment_services
```

特别值得参考：

**预约和服务项目分离。**



---

# 9. 系统角色

V1 不建立复杂员工体系。

## 角色一：Customer

顾客。

使用：

```text
H5
```

V2：

```text
微信小程序
```

---

## 角色二：Admin

老板。

拥有全部后台权限。

---

## 角色三：Operator

预留。

以后家里人帮忙操作后台时使用。

V1 可以先通过 RBAC 支持：

```text
OWNER
OPERATOR
```

但不建立：

```text
TECHNICIAN
DISPATCHER
SCHEDULER
```

---

# 10. 顾客身份

H5：

```text
手机号
 ↓
短信验证码
 ↓
登录
```

第一次登录：

如果不存在：

```text
创建 member
```

如果后台已经存在：

```text
绑定已有 member
```

---

# 11. 微信小程序

V2 再做。

目标：

```text
微信登录
 ↓
绑定手机号
 ↓
找到同一个 member_id
```

H5 和小程序最终：

```text
同一个会员
```

---

# 12. Member

核心表：

```text
member
```

字段：

```text
id
member_no

name
phone
gender
birthday

status

remark

created_at
updated_at
last_visit_at
```

手机号默认唯一。

---

# 13. Member Tag

V1 可以做简单标签：

```text
VIP
老客户
肩颈
腰部
敏感
```

数据：

```text
member_tag
member_tag_rel
```

不要做复杂标签规则引擎。

---

# 14. Service

所有按摩项目统一叫：

# 服务项目

不要叫：

```text
商品
```

例如：

```text
全身按摩 60 分钟
全身按摩 90 分钟
肩颈按摩 60 分钟
足疗
拔罐
热敷
针灸
```

以后店主可以自己增加。

---

# 15. service_category

```text
id
name
sort
status
created_at
updated_at
```

---

# 16. service

```text
id
category_id

name
description

duration_minutes
default_price

cover_image

status
sort

created_at
updated_at
```

价格：

统一使用：

**整数分。**

例如：

128 元：

```text
12800
```

禁止 float。

---

# 17. 为什么 Service 要独立

因为未来可能：

```text
按摩
 ↓
肩颈
 ↓
拔罐
 ↓
热敷
 ↓
针灸
```

预约系统不应该写死：

```text
massage_type = 1
```

而应该：

```text
appointment
 ↓
appointment_service
 ↓
service
```

这样以后增加服务不用改预约系统。

---

# 18. V1 预约模型

因为：

**店主就是唯一按摩师。**

所以预约实际上是在预约：

# 店主自己的服务时间。

不需要：

- 技师选择
- 技师派单
- 技师容量
- 员工排班

---

# 19. 时间模型

V1 推荐：

# 固定营业时间 + 时间段预约

例如：

营业时间：

```text
09:00 - 21:00
```

默认预约间隔：

```text
30 分钟
```

顾客看到：

```text
09:00
09:30
10:00
10:30
11:00
...
```

服务时长不同：

预约时根据服务项目计算结束时间。

例如：

```text
15:00
+
60 分钟
=
16:00
```

---

# 20. 为什么不做复杂 Slot Inventory

原 Plan 里的：

```text
booking_template
booking_time_slot
booking_slot_inventory
SlotInventoryService
```

是为了支持：

```text
多个技师
多个服务位
多人同时预约
容量
```

但当前真实业务：

```text
一个老板
一个按摩位
```

所以 V1 不需要复杂库存模型。

直接根据：

```text
appointment
+
scheduled_start
+
scheduled_end
```

判断冲突。

---

# 21. appointment

核心表：

```text
appointment
```

字段：

```text
id
appointment_no

member_id

scheduled_start
scheduled_end

status

customer_note
internal_note

created_at
updated_at

confirmed_at
started_at
completed_at
cancelled_at
```

---

# 22. appointment_service

```text
id
appointment_id
service_id

service_name_snapshot
duration_minutes_snapshot
price_snapshot

quantity

created_at
```

V1：

**一个预约默认一个服务项目。**

但数据库保持：

```text
appointment
1
 ↓
N appointment_service
```

以后可以扩展组合服务。

---

# 23. 为什么保存 Snapshot

例如：

今天：

```text
肩颈按摩
128 元
60 分钟
```

三个月后：

```text
138 元
70 分钟
```

历史预约不能自动变成：

```text
138 元
70 分钟
```

所以预约创建时保存：

```text
service_name_snapshot
duration_minutes_snapshot
price_snapshot
```

---

# 24. 预约状态

V1：

```text
PENDING_CONFIRM
CONFIRMED
IN_SERVICE
COMPLETED

CANCELLED
NO_SHOW
```

---

# 25. 状态流转

正常：

```text
PENDING_CONFIRM
      ↓
CONFIRMED
      ↓
IN_SERVICE
      ↓
COMPLETED
```

取消：

```text
PENDING_CONFIRM → CANCELLED
CONFIRMED → CANCELLED
```

爽约：

```text
CONFIRMED → NO_SHOW
```

---

# 26. 为什么保留待确认

个人店也不一定适合所有预约自动确认。

例如老板可能：

```text
正在按摩
```

客户突然预约：

```text
15:30
```

老板可能需要确认。

所以 V1 默认：

# 手动确认

后台可以预留：

```text
AUTO_CONFIRM
MANUAL_CONFIRM
```

但默认：

```text
MANUAL_CONFIRM
```

---

# 27. 时间冲突

这是整个预约系统最重要的规则。

因为只有一个按摩师。

所以：

```text
同一时间只能存在一个有效服务。
```

判断：

```text
existing.start < new.end
AND
existing.end > new.start
```

如果成立：

```text
冲突
```

不能确认。

---

# 28. 示例

已有：

```text
15:00 - 16:00
```

客户预约：

```text
15:30 - 16:30
```

冲突。

拒绝。

客户预约：

```text
16:00 - 17:00
```

不冲突。

允许。

---

# 29. 是否需要间隔时间

V1 默认：

```text
Buffer = 0
```

如果店主实际需要：

```text
15 分钟整理/休息
```

以后增加：

```text
buffer_minutes
```

即可。

不要一开始把它设计成复杂排班系统。

---

# 30. 提前预约规则

默认：

```text
最早提前 30 天
最晚提前 2 小时
```

也就是说：

当前：

```text
12:00
```

最早可以预约：

```text
14:00
```

这些都是后台配置项。

---

# 31. 取消规则

默认：

```text
提前 2 小时可取消
```

不足 2 小时：

顾客端不允许直接取消。

老板后台可以人工处理。

---

# 32. 改期

改期本质：

```text
旧预约
 ↓
取消/释放旧时间
 ↓
检查新时间
 ↓
创建新的时间安排
```

推荐最终实现：

**修改同一个 appointment 的 scheduled_start/end，并记录状态日志。**

不要为了改期创建大量无意义的订单。

---

# 33. appointment_status_log

```text
id
appointment_id

from_status
to_status

operator_type
operator_id

remark

created_at
```

例如：

```text
老板
09:20

PENDING_CONFIRM
→
CONFIRMED
```

---

# 34. 会员卡

V1：

# 次卡

例如：

```text
按摩10次卡
```

---

# 35. card_template

```text
id

name
type

total_count

validity_type
valid_from
valid_until

price

status

created_at
updated_at
```

类型：

```text
COUNT
ACTIVITY
```

V2：

```text
DURATION
```

---

# 36. 次卡

例如：

```text
按摩10次卡
10 次
永久有效
```

---

# 37. 活动卡

例如：

```text
周年庆活动卡
10 次
2026-10-01
~
2026-12-31
```

---

# 38. member_card

```text
id
member_id
card_template_id

total_count
remaining_count

valid_from
valid_until

status

issued_at
issued_by

created_at
updated_at
```

---

# 39. 卡状态

```text
ACTIVE
USED_UP
EXPIRED
CANCELLED
```

---

# 40. Card Transaction

这是会员卡系统最重要的数据。

```text
card_transaction
```

所有次数变化必须有流水。

类型：

```text
ISSUE
REDEEM
REVERSAL
ADJUSTMENT
REFUND
```

---

# 41. 示例

办卡：

```text
0 → 10

ISSUE
+10
```

消费：

```text
10 → 9

REDEEM
-1
```

撤销：

```text
9 → 10

REVERSAL
+1
```

人工调整：

```text
9 → 11

ADJUSTMENT
+2
```

---

# 42. Card Transaction 字段

```text
id

member_card_id
member_id

type
quantity

before_count
after_count

reference_type
reference_id

remark
operator_id

created_at
```

---

# 43. 卡与服务项目

增加：

```text
card_service_rule
```

表示：

这张卡可以用于什么服务。

例如：

```text
按摩10次卡
 ↓
肩颈按摩
全身按摩
```

如果以后：

```text
拔罐
```

不能使用这张卡。

核销时拒绝。

---

# 44. 为什么现在就做 Card Service Rule

因为否则以后业务很容易变成：

```text
if card.type == massage:
    ...
```

这种硬编码。

现在用简单关系表解决：

```text
card_type
   ↓
card_service_rule
   ↓
service
```

足够。

---

# 45. 权益模型

未来可能有：

```text
积分
优惠券
折扣
赠送
会员等级
```

但 V1 不做。

当前：

```text
Member
 ↓
Count Card
```

就够。

未来如果真正出现这些需求：

再抽象：

```text
Entitlement
```

不要现在创建一堆空模块。

---

# 46. Payment

虽然当前不接在线支付，但必须记录：

# 钱是怎么收的。

payment：

```text
id
appointment_id
member_id

amount

method
status

reference_no
remark

recorded_by
recorded_at

created_at
updated_at
```

---

# 47. Payment Method

V1：

```text
CARD
WECHAT_TRANSFER
CASH
OTHER
```

---

# 48. 为什么 CARD 也属于结算方式

例如：

客户做：

```text
肩颈按摩
```

价格：

```text
128
```

客户使用：

```text
10次卡
```

系统：

```text
payment.method = CARD
```

同时：

```text
redemption
↓
member_card
↓
-1
```

---

# 49. 微信转账

客户：

```text
微信转账 128 元
```

店主确认到账。

后台：

```text
记录支付
```

不需要微信支付 API。

---

# 50. 现金

同样：

```text
payment.method = CASH
```

记录：

```text
128 元
```

---

# 51. Appointment != Payment

必须保持分离。

例如：

```text
预约
```

可以存在：

```text
payment = 未记录
```

直到服务完成。

---

# 52. Redemption

核销记录：

```text
redemption
```

字段：

```text
id

appointment_id
member_id
member_card_id
service_id

quantity

before_count
after_count

status

idempotency_key

operator_id
created_at
```

---

# 53. 核销核心事务

这是 V1 最重要的事务。

执行：

```text
BEGIN

锁定 member_card

检查 card status

检查有效期

检查 card_service_rule

检查 remaining_count

扣次数

写 card_transaction

写 redemption

写 payment

更新 appointment = COMPLETED

写 appointment_status_log

COMMIT
```

任意一步失败：

```text
ROLLBACK
```

---

# 54. 为什么不能用异步事件扣卡

原 Plan 中曾经设计：

```text
RedemptionCompleted
 ↓
card listener
 ↓
扣次数
```

这个设计本项目取消。

原因：

**扣卡是核心交易。**

不能：

```text
核销成功
 ↓
事件排队
 ↓
稍后扣卡
```

必须：

```text
核销
+
扣卡
+
流水
+
结算
+
完成预约
```

在一个数据库事务里完成。

EventBus 只能处理：

- 通知
- 统计
- 洞察
- 非关键日志

不能处理核心扣次。

---

# 55. 幂等

顾客或者老板可能：

连续点击两次。

或者：

网络重试。

所以：

```text
idempotency_key
```

必须存在。

推荐：

```text
redeem:{appointment_id}
```

并建立唯一约束。

---

# 56. 一次预约只能一次有效核销

数据库必须保证：

```text
appointment_id
+
SUCCESS
```

只能存在一个有效核销。

---

# 57. 核销撤销

表：

```text
redemption_reversal
```

字段：

```text
id
redemption_id

reason

operator_id
created_at
```

撤销：

```text
锁卡
 ↓
确认未撤销
 ↓
余额 +1
 ↓
写 REVERSAL
 ↓
写 reversal
 ↓
COMMIT
```

---

# 58. 撤销不删除原数据

禁止：

```text
DELETE redemption
```

必须：

```text
原核销
+
撤销记录
+
恢复流水
```

形成完整历史。

---

# 59. 撤销后预约状态

默认：

仍然：

```text
COMPLETED
```

因为：

撤销核销不等于服务没有发生。

例如：

老板误扣了客户 A 的卡。

服务其实已经完成。

应该：

```text
服务 = 完成
核销 = 撤销
```

然后重新正确结算。

---

# 60. 爽约

默认：

```text
NO_SHOW
```

不扣次数。

但是记录。

未来如果真实业务需要：

```text
爽约扣次
爽约次数限制
黑名单
```

再增加。

---

# 61. 预约是否必须有会员

不必须。

可以：

```text
普通客户
 ↓
预约
 ↓
到店
 ↓
微信付款
```

做完以后：

老板可以询问是否办理会员卡。

---

# 62. 预约与办卡解耦

绝对不要：

```text
只有会员才能预约
```

V1 默认：

任何登录客户都可以预约。

会员卡是：

**一种权益/结算方式。**

---

# 63. 数据库总表

最终 V1：

```text
identity_user
identity_role
identity_permission

member
member_tag
member_tag_rel

service_category
service

card_template
card_service_rule
member_card
card_transaction

appointment
appointment_service
appointment_status_log

payment

redemption
redemption_reversal

content_page_config
content_banner
content_announcement
content_system_setting

ops_operation_log
ops_insight_snapshot
```

---

# 64. 明确删除的旧表

不再需要：

```text
booking_template
booking_time_slot
booking_slot_inventory
```

也不需要：

```text
staff
staff_schedule
service_area
member_address
```

V1 都不要。

---

# 65. 核心关系

```text
member
 │
 ├── member_tag
 │
 ├── member_card
 │      │
 │      ├── card_transaction
 │      │
 │      └── card_service_rule
 │
 └── appointment
        │
        ├── appointment_service
        │        │
        │        └── service
        │
        ├── payment
        │
        └── redemption
                 │
                 └── redemption_reversal
```

---

# 66. Appointment 与 Service

```text
appointment
    1
    │
    N
appointment_service
    │
    N
    │
    1
service
```

V1 UI 限制：

```text
一次预约只能选择一个服务
```

但数据库保留多服务能力。

---

# 67. 外键

核心表建立外键。

例如：

```text
member_card.member_id → member.id

appointment.member_id → member.id

appointment_service.appointment_id
→ appointment.id

appointment_service.service_id
→ service.id

payment.appointment_id
→ appointment.id

redemption.appointment_id
→ appointment.id

redemption.member_card_id
→ member_card.id
```

核心历史数据：

禁止物理删除。

---

# 68. ID

统一：

UUID / ULID。

另外生成：

```text
member_no
appointment_no
```

例如：

```text
APT202609280001
```

数据库 ID 和业务编号分开。

---

# 69. 时间

统一使用：

```text
Asia/Shanghai
```

因为当前业务是中国本地实体店。

所有：

```text
appointment
payment
redemption
log
```

必须统一时间规则。

---

# 70. 索引

至少：

```text
member(phone)

member_card(member_id, status)

card_transaction(member_card_id, created_at)

appointment(member_id, scheduled_start)

appointment(status, scheduled_start)

appointment(scheduled_start, scheduled_end)

appointment_service(appointment_id)

payment(appointment_id)

redemption(appointment_id)

member_tag_rel(member_id)
```

---

# 71. 预约冲突查询

核心 SQL 逻辑：

```text
existing.scheduled_start < new.scheduled_end
AND
existing.scheduled_end > new.scheduled_start
```

只检查：

```text
PENDING_CONFIRM
CONFIRMED
IN_SERVICE
```

取消：

```text
CANCELLED
```

不参与冲突。

爽约：

```text
NO_SHOW
```

也不参与。

---

# 72. 并发预约

这是个人店也必须处理的问题。

两个客户同时抢：

```text
15:00
```

必须只有一个成功。

推荐：

数据库事务 + 冲突检查 + 适当锁/唯一约束策略。

不要只在前端判断。

---

# 73. 后台首页

后台默认首页：

# 今日

而不是：

```text
Dashboard
```

里面直接显示：

```text
今日预约：8

待确认：2
待服务：3
服务中：1
已完成：2
```

---

# 74. 今日预约卡片

例如：

```text
15:00

张三

肩颈按摩
60分钟

128元

待服务
```

按钮：

```text
确认
开始
完成
核销
收款
取消
爽约
```

根据状态显示对应按钮。

---

# 75. 今日工作台原则

老板打开后台：

**不需要思考去哪。**

第一屏直接告诉他：

```text
现在要做什么。
```

这是整个后台 UX 最重要的原则。

---

# 76. 会员页面

列表：

```text
姓名
手机号
卡数量
剩余次数
最后到店
状态
```

会员详情：

```text
基本资料
会员卡
预约
服务记录
交易记录
标签
备注
```

---

# 77. 服务项目页面

支持：

```text
新增
编辑
停用
启用
排序
价格
时长
介绍
```

例如：

```text
肩颈按摩
60分钟
128元
```

---

# 78. 卡管理

后台：

```text
卡模板
 ↓
新增
编辑
上下架
```

客户：

```text
会员
 ↓
发卡
 ↓
10次
```

后台可以：

```text
续卡
调整次数
作废
查看流水
```

---

# 79. 预约管理

筛选：

```text
日期
会员
服务
状态
```

状态：

```text
待确认
已确认
服务中
已完成
已取消
爽约
```

---

# 80. 收款/核销

服务完成以后：

```text
结算
```

显示：

```text
客户
服务
价格
可用会员卡
```

选择：

```text
会员卡
微信转账
现金
其他
```

---

# 81. 一个完整示例

客户：

```text
张三
```

预约：

```text
9月28日
15:00
肩颈按摩60分钟
128元
```

后台：

```text
确认
```

15:00：

```text
开始服务
```

16:00：

```text
服务完成
```

客户说：

```text
用10次卡
```

系统：

```text
卡余额 6
 ↓
锁卡
 ↓
检查服务权限
 ↓
6 → 5
 ↓
card_transaction -1
 ↓
redemption SUCCESS
 ↓
payment CARD
 ↓
appointment COMPLETED
```

完成。

---

# 82. 首页内容

V1 保留原 Plan 的 JSON 驱动思路。

可以配置：

```text
轮播
公告
服务推荐
活动卡片
图文
```

但不要做：

```text
拖拽式 CMS
```

---

# 83. content_page_config

可以按区块保存 JSON。

例如：

```text
{
  "blocks": [
    {
      "type": "banner",
      "data": {}
    },
    {
      "type": "service_list",
      "data": {}
    }
  ]
}
```

前后端共享 Schema。

---

# 84. 系统设置

至少：

```text
营业时间
预约模式
最早提前预约时间
最晚提前预约时间
取消规则
Buffer
```

例如：

```text
营业时间：
09:00-21:00

提前预约：
30天

最晚预约：
2小时

取消：
2小时前

Buffer：
0分钟
```

---

# 85. 操作日志

记录：

```text
登录
新增会员
修改会员
发卡
续卡
调整次数
作废卡
创建预约
确认预约
取消预约
改期
开始服务
完成服务
核销
撤销核销
记录收款
修改服务
修改系统配置
```

---

# 86. 顾客洞察

V1 做最简单的三个：

### 低余额

```text
remaining_count <= 2
```

### 沉睡客户

```text
60天没有到店
```

### 即将过期

```text
7天内过期
```

这些只是后台提醒。

不要做复杂 AI 推荐。

---

# 87. EventBus

保留，但降级为：

# 辅助机制

例如：

```text
AppointmentCompleted
PaymentRecorded
MemberCreated
RedemptionCompleted
```

可以用于：

```text
统计
洞察
通知
日志
```

但：

**不能用 EventBus 执行核心扣卡。**

---

# 88. 后端技术架构

使用：

# Go Modular Monolith

一个后端。

一个数据库。

多个业务模块。

不要微服务。

---

# 89. 后端模块

```text
internal/modules/

identity
member
service
card
appointment
transaction
content
ops
```

---

# 90. 模块职责

### identity

登录、RBAC。

### member

会员、标签。

### service

服务项目。

### card

卡、余额、流水、卡服务规则。

### appointment

预约、时间冲突、状态。

### transaction

payment、redemption、reversal。

### content

首页和系统配置。

### ops

日志、洞察、定时任务。

---

# 91. 为什么把 Redemption 放到 Transaction

原 Plan：

```text
redemption
```

单独作为一个模块。

现在推荐：

```text
transaction/
    payment
    redemption
    reversal
```

因为它们实际上都属于：

# 服务完成后的结算。

这样核心业务边界更清楚。

---

# 92. 模块目录

```text
modules/card/

AGENTS.md
api.go
handler.go
service.go
repo.go
model.go
```

其他模块：

```text
member/
service/
appointment/
transaction/
content/
ops/
identity/
```

---

# 93. 模块边界

允许：

```text
module
 ↓
另一个 module/api.go
```

禁止：

```text
module A
 ↓
module B/repo.go
```

禁止直接读取别人内部 model。

---

# 94. AGENTS.md

根目录：

```text
AGENTS.md
```

写：

```text
项目定位
核心业务闭环
数据库规则
状态机
模块地图
V1边界
禁止事项
```

每个模块：

```text
modules/card/AGENTS.md
```

写：

```text
模块职责
数据表
公开 API
事务规则
测试
禁止事项
```

---

# 95. Codex 开发规则

每次：

```text
读取根 AGENTS
 ↓
判断涉及模块
 ↓
读取模块 AGENTS
 ↓
读取 api.go
 ↓
读取相关代码
 ↓
修改
 ↓
测试
```

禁止：

```text
一次性读取整个项目
```

---

# 96. 数据库 Migration

所有数据库变化必须 migration。

例如：

```text
001_identity.sql
002_member.sql
003_service.sql
004_card.sql
005_appointment.sql
006_transaction.sql
007_content.sql
008_ops.sql
```

禁止：

```text
直接修改生产数据库
```

---

# 97. 测试

至少：

### Unit Test

测试：

- 状态转换
- 时间冲突
- 卡有效期
- 卡服务权限
- 余额扣减

### DB Test

测试：

- 事务
- 锁
- 唯一约束
- 外键

### E2E

完整跑：

```text
注册
→
预约
→
确认
→
开始
→
完成
→
核销
→
余额减少
```

---

# 98. 必测并发案例

## Case 1

卡余额：

```text
1
```

同时两个核销。

结果：

```text
一个成功
一个失败
```

---

## Case 2

同时两个人抢：

```text
15:00
```

结果：

```text
一个成功
一个失败
```

---

# 99. 必测业务案例

### Case 3

余额：

```text
0
```

核销：

失败。

### Case 4

卡过期：

失败。

### Case 5

卡不允许服务：

失败。

### Case 6

重复点击：

不能重复扣。

### Case 7

撤销核销：

次数恢复。

### Case 8

重复撤销：

失败。

### Case 9

取消后的预约：

不占用时间。

### Case 10

已完成预约：

不能重复完成。

---

# 100. 顾客数据权限

客户只能：

```text
读取自己的：

会员资料
会员卡
预约
服务记录
```

禁止：

```text
/member/123
```

就看到别人数据。

后端必须根据：

```text
token
 ↓
member_id
```

确定身份。

不能相信前端传来的 member_id。

---

# 101. H5 架构

V1：

```text
Vue 3
Vite
TypeScript
```

目录：

```text
apps/customer/

src/

core/
  api/
  models/
  logic/
  store/
  utils/

platform/
  auth/
  storage/
  notify/

pages/
components/
```

---

# 102. 前端铁律

`core`：

禁止：

```text
window
document
localStorage
```

业务逻辑：

放：

```text
core/logic
```

页面：

只做：

```text
展示
交互
调用业务
```

---

# 103. 微信小程序

V2：

把顾客端迁移/统一到：

```text
uni-app
```

目标：

```text
同一套业务
 ↓
H5
+
微信小程序
```

不要维护两套业务代码。

---

# 104. 顾客端页面

V1：

```text
首页
服务
预约
我的
```

我的里面：

```text
会员卡
我的预约
历史记录
个人资料
```

---

# 105. 首页

首页只需要：

```text
品牌/店铺介绍
服务推荐
活动
预约入口
联系电话
营业时间
```

不要做复杂商城首页。

---

# 106. 预约页面

流程：

```text
选择服务
 ↓
选择日期
 ↓
选择时间
 ↓
确认
 ↓
提交预约
```

如果冲突：

直接：

```text
这个时间刚刚被预约，请重新选择
```

---

# 107. 顾客预约详情

显示：

```text
服务
日期
时间
状态
金额
备注
```

不显示：

```text
后台内部备注
```

---

# 108. V1 开发阶段

## Phase 0：Review

Codex：

只 Review。

检查：

```text
业务闭环
数据库
状态机
事务
并发
权限
模块边界
V1范围
```

输出：

```text
# REVIEW

## BLOCKER

## WARNING

## RECOMMENDATION

## DECISION

PASS / BLOCKED
```

---

# 109. Review 通过后

如果：

```text
PASS
```

才开始写代码。

如果：

```text
BLOCKED
```

只修改阻断项。

不要重新设计整个项目。

---

# 110. Phase 1：工程骨架

建立：

```text
Go
数据库
migration
config
logging
error
auth
modules
shared
AGENTS
```

验收：

```text
go build
go test
migration
```

全部通过。

---

# 111. Phase 2：Identity

实现：

```text
后台登录
RBAC
Customer Token
```

验收：

老板可以进入后台。

顾客可以登录。

---

# 112. Phase 3：Member

实现：

```text
会员
标签
备注
```

验收：

能新增会员。

能查询会员。

---

# 113. Phase 4：Service

实现：

```text
分类
服务项目
价格
时长
上下架
排序
```

验收：

后台创建：

```text
肩颈按摩
60分钟
128元
```

顾客端能看到。

---

# 114. Phase 5：Card

实现：

```text
card_template
member_card
card_service_rule
card_transaction
```

验收：

发：

```text
10次卡
```

看到：

```text
10
```

并产生：

```text
ISSUE +10
```

---

# 115. Phase 6：Appointment

实现：

```text
appointment
appointment_service
appointment_status_log
```

实现：

```text
时间冲突
确认
取消
改期
爽约
```

验收：

两个客户不能预约同一服务时间。

---

# 116. Phase 7：Transaction

实现：

```text
payment
redemption
redemption_reversal
```

重点：

**数据库事务 + 并发。**

这是核心阶段。

---

# 117. Phase 8：今日工作台

实现：

```text
今日预约
待确认
待服务
服务中
已完成
```

快速操作：

```text
确认
开始
完成
核销
收款
取消
爽约
```

---

# 118. Phase 9：Customer H5

实现：

```text
首页
服务
预约
会员卡
我的预约
历史记录
个人中心
```

---

# 119. Phase 10：Content + Ops

实现：

```text
首页配置
Banner
公告
操作日志
客户洞察
```

---

# 120. Phase 11：E2E

完整跑：

```text
顾客登录
 ↓
选择按摩
 ↓
选择时间
 ↓
预约
 ↓
老板确认
 ↓
开始服务
 ↓
完成服务
 ↓
选择会员卡
 ↓
核销
 ↓
卡余额减少
 ↓
交易记录
```

---

# 121. Phase 12：上线

检查：

```text
HTTPS
数据库备份
日志
错误监控
migration
回滚方案
```

---

# 122. V2

只有 V1 稳定以后才进入。

候选：

```text
微信小程序
微信登录
微信通知
在线支付
会员卡时长
更多会员权益
```

---

# 123. V3

根据真实经营需求决定：

```text
多技师
员工账号
排班
技师绩效
多门店
多租户
积分
优惠券
商品
库存
营销
```

不要现在实现。

---

# 124. V1 数据库最终版本

最终只保留：

```text
identity_user
identity_role
identity_permission

member
member_tag
member_tag_rel

service_category
service

card_template
card_service_rule
member_card
card_transaction

appointment
appointment_service
appointment_status_log

payment
redemption
redemption_reversal

content_page_config
content_banner
content_announcement
content_system_setting

ops_operation_log
ops_insight_snapshot
```

---

# 125. 最重要的数据库原则

### 原则一

余额不是唯一真相。

```text
member_card.remaining_count
```

是当前缓存。

真正历史：

```text
card_transaction
```

---

### 原则二

预约不是支付。

```text
appointment
payment
```

分开。

---

### 原则三

支付不是核销。

```text
payment
redemption
```

分开。

---

### 原则四

核销不是删除。

撤销必须留下历史。

---

### 原则五

服务名称/价格/时长要保存 Snapshot。

---

# 126. 最重要的业务不变量

系统任何时候都必须满足：

```text
remaining_count >= 0
```

并且：

```text
有效核销
必须对应
一个预约
+
一个会员
+
一张有效卡
+
一个服务
```

同时：

```text
一个预约最多一次有效核销
```

以及：

```text
一个时间段只能有一个有效服务
```

---

# 127. 最重要的事务

## 发卡

```text
创建 member_card
+
card_transaction
```

一个事务。

---

## 核销

```text
锁 card
+
扣次数
+
card_transaction
+
redemption
+
payment
+
appointment completed
+
status log
```

一个事务。

---

## 撤销

```text
锁 card
+
恢复次数
+
card_transaction
+
reversal
+
operation log
```

一个事务。

---

# 128. 最重要的 UX 原则

这个系统不是给专业运营团队使用。

它是给：

# 一个按摩师老板

使用的。

所以后台必须：

```text
少菜单
少按钮
少概念
少填写
```

老板每天最常做的事情：

```text
看今天
确认预约
开始服务
完成服务
核销
收款
```

这些必须最快。

---

# 129. 最终后台结构

```text
今日
会员
服务项目
会员卡
预约
内容
设置
```

不需要：

```text
排班
调度
员工
门店
库存
营销中心
财务中心
```

---

# 130. 最终顾客端结构

```text
首页
服务
预约
我的
```

足够。

---

# 131. Codex 禁止事项

Codex 不得自行增加：

```text
staff
staff_schedule
member_address
service_area
tenant_id
points
coupon
mall
inventory
commission
online_payment
dispatch
map
gps
```

除非未来 Plan 明确进入对应版本。

---

# 132. Codex Review 时重点检查

### 业务

1. 是否仍然把系统理解成上门按摩？
2. 是否错误引入技师派单？
3. 是否错误引入员工排班？
4. 是否错误引入多门店？
5. 是否把会员卡和预约强绑定？
6. 是否允许普通客户预约？

### 数据库

7. appointment 是否足够表达一次到店服务？
8. service 是否与 appointment 分离？
9. card_transaction 是否完整？
10. payment 和 redemption 是否分离？
11. 历史 Snapshot 是否保存？

### 并发

12. 同一个时间能否被两个客户同时预约？
13. 同一张卡能否被同时扣两次？
14. 重复提交是否幂等？

### 权限

15. 顾客能否读取其他顾客数据？
16. Operator 权限是否合理？

### 技术

17. 模块是否存在循环依赖？
18. 是否存在不必要的复杂抽象？
19. migration 是否完整？
20. E2E 是否覆盖核心闭环？

---

# 133. Review 输出

Codex 必须严格输出：

```text
# REVIEW

## 1. BLOCKER

无
或：

- xxx

## 2. WARNING

无
或：

- xxx

## 3. RECOMMENDATION

无
或：

- xxx

## 4. SCOPE CHECK

确认当前系统是：

个人到店按摩店
单老板/单按摩师
家庭成员可辅助后台
无上门服务
无技师调度
无复杂排班
无多门店
无多租户

## 5. DECISION

PASS
或
BLOCKED
```

---

# 134. Review PASS 后执行规则

如果：

```text
DECISION = PASS
```

直接：

```text
Phase 1
↓
Phase 2
↓
Phase 3
...
```

执行。

每完成一个 Phase：

必须：

```text
代码
+
测试
+
migration
+
文档
```

一起完成。

---

# 135. 每个 Phase 完成后的 Codex 汇报

格式：

```text
# PHASE RESULT

## Completed

- xxx
- xxx

## Tests

- go test: PASS
- integration test: PASS

## Database

- migration xxx

## Files Changed

- xxx

## Remaining

- xxx

## Next

Phase X
```

---

# 136. 最终 V1 验收

## 顾客

- [ ] 手机登录
- [ ] 查看服务
- [ ] 查看会员卡
- [ ] 创建预约
- [ ] 取消预约
- [ ] 改期
- [ ] 查看历史预约

## 老板

- [ ] 今日工作台
- [ ] 管会员
- [ ] 管服务
- [ ] 发卡
- [ ] 续卡
- [ ] 调整次数
- [ ] 看预约
- [ ] 确认预约
- [ ] 开始服务
- [ ] 完成服务
- [ ] 核销
- [ ] 收款
- [ ] 撤销核销
- [ ] 看交易流水
- [ ] 看操作日志

## 核心正确性

- [ ] 同一时间不能重复预约
- [ ] 卡余额不能小于 0
- [ ] 卡扣次必须有流水
- [ ] 重复核销不能重复扣次
- [ ] 撤销核销能恢复次数
- [ ] 重复撤销失败
- [ ] 过期卡不能使用
- [ ] 不适用服务的卡不能使用
- [ ] 已取消预约不能完成
- [ ] 已完成预约不能重复完成
- [ ] 客户不能读取其他客户数据
- [ ] 所有重要后台操作有日志

---

# 137. 最终架构图

```text
                  ┌──────────────┐
                  │    顾客      │
                  └──────┬───────┘
                         │
                         ↓
                ┌────────────────┐
                │ H5 / 小程序    │
                └───────┬────────┘
                        │
                        ↓
                 ┌────────────┐
                 │   Member   │
                 └─────┬──────┘
                       │
              ┌────────┴────────┐
              ↓                 ↓
          Service           Member Card
              │                 │
              └────────┬────────┘
                       ↓
                 Appointment
                       │
                       ↓
                  到店服务
                       │
                       ↓
                  服务完成
                       │
              ┌────────┴────────┐
              ↓                 ↓
          Payment           Redemption
              │                 │
              │                 ↓
              │          Card Transaction
              │
              └────────┬────────┘
                       ↓
                    完成记录
```

---

# 138. 最终第一性原理

不要把这个项目想成：

“我要做一个完整的按摩店 SaaS。”

而应该想成：

> **我要给一个自己亲自按摩的老板做一个数字化小助手。**

这个小助手每天只帮他做好几件事情：

```text
1. 告诉我今天谁来。

2. 告诉我几点来。

3. 告诉我做什么。

4. 不要让我撞预约。

5. 到店做完以后，点一下完成。

6. 如果客户用次卡，帮我正确扣一次。

7. 如果客户微信转账，帮我记下来。

8. 以后我想知道这个客户还有几次，也能马上看到。
```

如果这 8 件事情做得足够简单、稳定、可靠：

# V1 就成功。

其他东西：

以后再说。

---

# 139. 最终开发顺序

```text
业务冻结
    ↓
数据库冻结
    ↓
状态机冻结
    ↓
Go 骨架
    ↓
Identity
    ↓
Member
    ↓
Service
    ↓
Card
    ↓
Appointment
    ↓
Transaction
    ↓
今日工作台
    ↓
Customer H5
    ↓
Content / Ops
    ↓
E2E
    ↓
上线
    ↓
真实使用
    ↓
根据真实问题决定 V2
```

---

# 140. 项目最终定义

**项目名称：**

个人到店按摩店服务管理系统

**V1 核心：**

```text
会员
+
服务
+
预约
+
次卡
+
核销
+
收款
```

**经营模型：**

```text
一个店主
+
一个主要按摩师
+
未来家人辅助后台
```

**核心业务：**

```text
预约到店
→
按摩服务
→
完成
→
核销/收款
```

**核心技术：**

```text
Go Modular Monolith
+
单数据库
+
Vue3 H5
+
未来 uni-app 小程序
```

**核心原则：**

```text
简单
稳定
低学习成本
低资源
不提前过度设计
```

**V1 最终目标：**

> 让一个自己每天给客户按摩的老板，打开后台就能知道今天谁来、几点来、做什么，服务结束以后点几下就能完成核销和收款记录。

这就是本项目 V1 的全部核心。