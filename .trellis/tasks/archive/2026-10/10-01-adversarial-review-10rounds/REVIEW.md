# V2.2 十轮子代理对抗式审查报告（墨菲定律验收）

**日期**：2026-10-01 · **方法**：10 轮独立子代理（6 域并行 → 2 轮墨菲终审 → 2 轮对抗证伪），只审不改
**对象**：server（Go ~15.2k 行）、apps/customer（H5 ~4.5k 行）、apps/weapp（小程序 ~4.4k 行）、部署资产（compose/脚本）
**证伪结果**：26 条高优先级发现逐条复核，**0 条 REFUTED**（16 CONFIRMED / 10 PARTIAL），原报严重度系统性虚高——8 条 P1 仅 2 条维持，其余降 P2；另有 3 条经证伪轮**加重**（F5/F7/F11）。

## 执行摘要

**系统骨架是健康的**：单写者 + BEGIN IMMEDIATE 的并发模型、生成列三重唯一约束、快照纪律、token 取身份红线，在十轮轰炸后全部站住（证伪轮专门找过反证）。「一团乱麻」的感受不来自架构，而来自五个可批量修复的系统性模式：

| # | 根因模式 | 覆盖发现 | 修复量级 |
|---|---------|---------|---------|
| 1 | **配置缺省值朝开发友好兜底，无生产形态守卫** | F1 SMS dev 默认可接管任意会员、F2 wx 兜底无警示、F16 JWT secret 占位符 | config.validate 三条硬检查 + compose 一行 |
| 2 | **失败伪装成空态**（前端三页 + weapp 三入口，双端同病） | F21、cards 无错误态、home/services 目录、slot-picker 选项 | 每页复制 MyAppointments failed 态模板 |
| 3 | **部署运维层裸奔** | F4 空库静默启动、F5 备份静默失效链（cron 日志目录不存在→零日志）、F13 ops/daily cron 12h token、F14 healthz 不探库 | 脚本/compose 十余行 + `-daily` flag |
| 4 | **适老化硬指标未达标** | 对比度 3.3:1（AA 需 4.5:1）、44px 点击目标、iOS 15px 输入聚焦放大 | token 一处改 + 每端几行 |
| 5 | **双端行为漂移** | 退出确认、成功页单号、备注上限 200 vs 无界、改期窗口 14 vs 30 天、版本标识 H5 缺失 | 各 1-4 行对齐 |

## 一、必修清单（证伪后维持/加重）

| # | 严重度 | 发现 | 证据 | 最简修复 |
|---|-------|------|------|---------|
| F1 | **P1** | 生产部署载体 docker-compose.yml:15 显式 `ANMO_SMS_MODE: "dev"`，公开端点固定码 123456 → 任意手机号接管会员（EnsureByPhone 直发 Token） | config.go:145-147、customer.go:193-229、identity/module.go:11-12 | compose 改 `"off"`（一行）；可选 validate 生产形态拒绝 dev |
| F3 | **P1** | 公开认证端点零限速：register 三分支 409 可枚举手机号（文案是 plan 逐字口径，属知情）、login/claim 无尝试上限，bcrypt CPU 放大 | customer.go:54-96、119-148 | per-IP+per-phone 内存滑窗（复用 smsStore 模式 ~20 行）；文案不动 |
| F5 | **P2+** | 备份静默失效链**加重**：crontab 重定向 `var/backup.log` 但脚本只 mkdir BACKUPS——`/opt/anmo/var` 不存在时 cron **零日志零输出**；权限拒绝/卷挂/cron 停四条路径全静默 | backup-sqlite.sh:14,22,28 | 脚本 mkdir var + 快照体积下限断言 + 月度 --restore-test + chmod 600 |
| F7 | **P2+** | LOCK_RETRY 口径未达成**加重**：`_txlock=immediate` 下 BUSY 主发生点恰是 BeginTx——唯一未被 wrapTxErr 覆盖处；Begin/Commit 失败→裸 500 | shared/tx.go:50-54 | RunInTx 内 `IsBusy→Conflict("LOCK_RETRY")` 一处修全模块 |
| F4 | P2 | 空库静默启动：bind mount 丢失 → 驱动静默建库 → migration 全跑 → 服务"正常"，店主新账写入空库，备份链开始快照空库（原库仍在原处，属数据分叉） | database.go:60-76、main.go:33-55、compose:18 | 启动前 os.Stat 库文件，不存在且无 `ANMO_ALLOW_EMPTY_DB=1` 则拒启 |
| F6 | P2 | 幂等回放不校验请求体：同 key 不同预约/卡静默返回第一笔结果（前端 bug 时店主以为已收钱，目标卡未扣） | settle.go:131-140,202-214,395-402、walkin.go:62-73 | 回放命中比对 (appointment_id/method/amount)，不一致 409 IDEM_CONFLICT |
| F8 | P2 | 卡 valid_from 全链路不校验：预售卡发出当天可核销完；IssueCard 接受 valid_until 已过的模板 | redeem.go:32-58、249-250、template.go:219-251 | ValidateForRedeem + UsableCards 各加一行 valid_from 判定 |
| F15 | P2 | DecodeJSON 无 MaxBytesReader（47 个调用点含公开端点）→ 大 body 内存 DoS | respond.go:54-58 | 函数内一行 `http.MaxBytesReader(w, r.Body, 1<<20)` |
| F16 | P2 | JWT secret 仅非空校验，example/compose 占位符 `change-me-in-production` 可直接上生产；start-anmo.sh:76 还向控制台打印种子口令 | config.go:193-195 | validate 拒绝占位字面值且要求 ≥32 字符 |
| F17 | P2 | H5 保存资料后手机号/会员号**显示为空**：PUT /api/me/profile 后端返回裸对象、前端解 `{member}` | member/handler.go:171 vs endpoints.ts:84 | 后端包一层 `{member}`（与 GET 对称，前端零改动） |
| F18 | P2 | slot-picker 的 `slot` 是 WXML 保留属性，父页赋值恒失效 → 选中具体时间**永不高亮**（提交不受影响；官方文档确认保留字） | slot-picker.js:9、slot-picker.wxml:30/38、booking.wxml:88、appointments.wxml:80 | property 改名 pickedSlot，4 处机械替换 |
| F21 | P2 | H5 Cards/QRCode/Services 加载失败伪装空态无重试；QRCodePage 失败会向**持卡会员**谎报「暂无有效会员卡」（到店场景误导） | CardsPage.vue:61-71、QRCodePage.vue:83-87、ServicesPage.vue:13-19 | 三页复制 MyAppointments failed 态模板 |
| F22 | P2 | H5 游客访问 /about 被 401 全局 handler 踢登录（营销页），weapp 同接口处理正确（漏改） | AboutPage.vue:15、router.go:51 | onMounted 先判 currentToken()，2 行 |
| F23 | P2 | H5 登录失败 401 误触发会话过期 handler，redirect 链断裂（守卫兜底防住死循环） | http.ts:58、customer.go:88/93/96 | handler 加 `!path.startsWith('/api/auth/')` 豁免 |
| F24 | P2 | iOS Safari 输入框 15px 聚焦自动放大（<16px 阈值全链路坐实，无缓解） | style.css:47,78-83,398 | `.input,.textarea{font-size:16px}` 一行 |
| F25 | P2 | weapp 全部 5 处 .mask 无 catchtouchmove → 弹层滚动穿透（me 页三个长表单 sheet 最重） | booking.wxml:120、appointments.wxml:70、me.wxml:36/97/121 | 各加 catchtouchmove="noop" |
| F26 | P2 | weapp about 页登录态仅 onLoad 判定：登录返回仍游客态、下拉刷新被旧值短路空转 | about.js:22-30,67-70 | ready 判定移入 onShow |
| F19 | P2 | qrcode 登录失效循环跳转（游客实际无入口，真实触发=token 失效+静默登录失败；login 成功 navigateBack 可解环） | qrcode.js:27-36,57-60 | needLogin 改 redirectTo（1 行，杜绝栈累积） |
| F20 | P2 | booking 页价格渲染 `¥¥`（fmt.yuan 已含前缀） | booking.wxml:66 | 删一个字面量 |

## 二、确认为「规格如此」（不修，除非改冻结决策）

- **F9** IN_SERVICE 无出口：D8 冻结状态机原文如此；有退路（可结算收尾）。
- **F10** SettleByPay 拒 COMPLETED：§8/核心事务 2 只给卡核销开了 COMPLETED 放行；缝隙仅「卡核销撤销后想改现金」可达，可走散客结算兜底。
- **F11**（倾向修）：RevokeRecord 缺 appointment_id 守卫——但 UI 侧 MembersPage.vue:205 同样漏，误操作面大；作为 D1 下唯一现金错录更正路径又有存在价值。二选一：加守卫+补专门更正入口，或写入 D28 明示接受。
- **F12** JWT 无吊销：claim 半段有物理删除自然失效；如要加固，member 加 token_epoch 一列即可。
- 登录错误文案可枚举（plan 逐字口径）、D17 多时段无上限、核销码明文 ULID（D21）——知情接受。

## 三、墨菲定律验收（R7 十剧本判定）

| 剧本 | 判定 |
|------|------|
| 断电重启 | 防了大半（WAL 不损坏库；restart 策略有）；synchronous=NORMAL 丢尾部已提交事务，换 FULL 一词或知情接受 |
| 磁盘满 | **没防住**：SQLITE_FULL→英文 "internal error"；docker 日志无上限加剧吃盘 |
| 备份失效链 | **没防住**（F5，四条静默路径） |
| Tunnel 断连 | 半防：顾客文案合格，店主零感知 → 免费外部探针打公网 /healthz 全覆盖 |
| 证书/密钥 | 防住了（dev key 600+gitignore；换 secret=全端重登可接受） |
| 时间/撞号 | **防住了**：per-day 序列名永不归零，跨年不撞号；NTP 装 chrony 一行 |
| 升级窗口 | 半防：migration 单事务可回滚；runbook 缺「升级前先备份」一行 |
| 微信凭据失效 | 半防：全端登录挂且报 "internal error"，应按 errcode 映射友好文案 |
| 单点 | 见下 |
| 监控盲区 | **没防住**：healthz 静态 ok、无 healthcheck、无外部探针 |

**单点清单**：容器挂死不重启（healthcheck 一行修，不接受）；单文件库（正确选型，接受）；备份同宿主（核实独立盘）；单 tunnel（接受+探针）；单 VPS 1.6G（接受）；店主=唯一运维（runbook 补两条核验）。

## 四、双端一致性矩阵要点（R8，10 项全核对）

一致：核销码门槛（D21）、卡徽标映射、空态文案（服务/卡/码页）、日期跨度 30 天、booking-options 同源。
不一致（择要）：退出确认（weapp 有/H5 无）；成功页单号（weapp 有/H5 无）；备注上限（weapp 200/H5 无界，服务端也不校验）；改期窗口（14 vs 30 天）；H5 性别设过就改不回「不透露」（patch 跳过空值）；游客首页营业时间 H5 硬编码 09:00-20:00（配置不同即误导）；版本标识仅 weapp 有；weapp 无历史服务页；**weapp 出现禁用词「技师」**（booking.wxml:124，V1 硬约束）；称谓店主/店家/商家三种混用；banner/activity/richtext 块 weapp 静默不渲染（商家无感知）。

## 五、性能/架构优化空间（按 ROI 排序，全部「最简」）

1. **工作台/今日接口 N+1**：/admin/workbench ≈1+3N 查询且是店主整日轮询屏——ServicesOfMany 已存在未用（queries.go:336-347、workbench.go:33-36）。
2. **审计日志范围失控**：所有非 GET 全落 ops_operation_log（与自身注释矛盾），登录爆破即刷爆单写者库——一行 path 前缀过滤（middleware.go:93）。
3. http.Server 补 Read/Write/IdleTimeout 三行（main.go:67-71）。
4. smsStore 两 map 只增不减（send 时顺带清理过期项 ~5 行）。
5. 优雅关停补 `wal_checkpoint(TRUNCATE)`+db.Close()（main.go:83-86）。
6. Insights 会员 N+1、PublicSettings 双查、卡流水/服务记录无 LIMIT（量级小，随手修）。
7. 收款列表无 status 过滤时全表排序（一条 migration 补两索引，或放着）。
8. **不建议做**：缓存层、连接池调优、读写分离、消息队列——当前量级（单店、单写者、QPS≈0）下收益为零，符合「最简/不做」。

## 六、十轮共同确认的优点

事务纪律（外部 HTTP 一律事务外）、生成列+CHECK 数据库级兜底不变量、锁序注释与撤销幂等撞键设计、快照彻底（服务/卡名/标签全快照）、身份红线零违反、SQL 全参数化零拼接、公开面白名单式最小化、format.js/booking.ts 北京时间锚定单一口径、H5 契约层 15 端点 14 个精确匹配、零 console.log/any。

## 七、建议修复批次（第九批候选，按依赖排序）

1. **安全闸**（F1+F16+F2+F15，~30 行）：compose off + validate 生产形态三检查 + MaxBytesReader + wx 启动警示
2. **运维防线**（F4+F5+F13+F14，~40 行）：空库守卫 + 备份脚本四防护 + `-daily` flag + healthz SELECT 1/compose healthcheck
3. **后端逻辑**（F7+F6+F8+F17+F3 限速，~80 行）
4. **weapp 体验**（F18+F19+F20+F25+F26+错误态三处+「技师」文案，~60 行）
5. **H5 体验**（F21+F22+F23+F24+退出确认+单号+备注上限+对比度/44px token，~80 行）
6. 双端对齐杂项（改期窗口统一、性别回退、术语统一）——可与 5 合并
