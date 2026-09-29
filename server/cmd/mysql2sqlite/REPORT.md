# MySQL → SQLite 数据迁移校验报告

- 日期：2026-09-28（本报告 = 工具本地实测；yun1 生产执行记录见 `.trellis/workspace/anmo-agent/journal-1.md` P4 节）
- 工具：`server/cmd/mysql2sqlite`（独立 go.mod，主服务模块零 MySQL 依赖）
- 源：本地 docker `anmo-mysql`（MySQL 8.4，33306，TZ +08:00）`anmo` 库全量 26 张表
- 目标：全新 SQLite 文件（migrations 001-013 自动建终态 schema）

## 复制结果（逐表行数 MySQL→SQLite 全部一致）

| 表 | 行数 | 表 | 行数 |
|---|---|---|---|
| identity_role | 2 | appointment_closure | 1 |
| identity_permission | 7 | payment | 5 |
| identity_user | 1 | redemption | 5 |
| member | 18 | redemption_reversal | 3 |
| member_tag | 5 | content_page_config | 1 |
| member_tag_rel | 0 | content_banner | 1 |
| service_category | 9 | content_announcement | 1 |
| service | 10 | content_system_setting | 4 |
| card_template | 8 | ops_operation_log | 265 |
| card_service_rule | 7 | ops_insight_snapshot | 0 |
| member_card | 7 | sys_sequence | 4 |
| card_transaction | 15 | appointment | 29 |
| appointment_service | 29 | appointment_status_log | 53 |

生成列 `payment.valid_lock` / `redemption.active_lock` 不做物理复制，由 SQLite 按行自动计算（工具已按 pragma_table_xinfo hidden 过滤）。

## 校验项（全部通过）

1. **行数**：26 表 MySQL vs SQLite 逐一比对 = 0 差异
2. **foreign_key_check**：0 违规
3. **PRAGMA integrity_check**：ok
4. **业务不变量抽查**：一预约多笔 VALID 收款 = 0；一预约多笔有效核销 = 0
5. **时间格式**：DATETIME → `YYYY-MM-DD HH:MM:SS`（Asia/Shanghai 墙上时间），DATE → 纯 `YYYY-MM-DD`（逐行抽样核对）
6. **序列延续**：迁移后建单编号 `APT202609280022` 紧接源库最大序号 `...0021`，sys_sequence 计数器无缝
7. **应用级冒烟**（真实二进制 + 迁移库）：
   - SMS 登录既有会员 → token 正常
   - `GET /api/appointments` 迁移数据完整，`scheduled_start: "2026-10-05T10:00:00+08:00"`（与 MySQL `loc=Asia/Shanghai` 的 JSON 完全一致）
   - `POST /api/appointments` 写入成功（编号/时间格式正确）

## yun1 生产执行步骤（P4 部署时）

```bash
# 1) 停写入：docker stop anmo-server
# 2) 在 WSL 交叉准备工具二进制（或 yun1 上 go run；工具模块独立无 CGO）
cd server/cmd/mysql2sqlite && go build -o /tmp/mysql2sqlite .
scp /tmp/mysql2sqlite yun1:/opt/anmo/
# 3) yun1 执行（DSN 在 /opt/anmo/docker-compose.yml 内）
ssh yun1 '/opt/anmo/mysql2sqlite \
  -mysql "anmo:<PASS>@tcp(127.0.0.1:33306)/anmo?parseTime=true&loc=Asia%2FShanghai" \
  -sqlite /opt/anmo/data/anmo.db -migrations /opt/anmo/migrations'
# 4) 校验全绿后按新 compose（无 MySQL，挂 ./data）启动
# 5) 保留 mysql-data 卷至少两周作为最终回退手段
```

## 回退方案

迁移失败或异常：旧 `anmo-mysql` 容器与 `mysql-data` 卷原样保留（只停不删），
compose 切回旧版即可回滚；SQLite 侧数据文件丢弃重来（迁移是幂等新建）。
