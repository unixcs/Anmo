# Design — Phase 1 工程骨架

## 技术选型（冻结）

| 项 | 选择 | 理由 |
|----|------|------|
| Web | stdlib net/http + Go 1.22 ServeMux | 零框架依赖，简单稳定 |
| DB 访问 | database/sql + go-sql-driver/mysql | 无 ORM；SQL 显式可审计，核销事务需要 FOR UPDATE |
| DB 连接参数 | parseTime=true&loc=Asia/Shanghai&time_zone=+08:00&charset=utf8mb4 | §69 时间统一 |
| ID | oklog/ulid/v2，CHAR(26) | §68 |
| JWT | golang-jwt/jwt/v5 | Phase 2 |
| 配置 | gopkg.in/yaml.v3 + ANMO_ 环境变量覆盖 | 简单 |
| 日志 | log/slog（JSON handler） | stdlib |
| 校验 | 手写 validate 函数 | 引入 validator 属过度设计 |

依赖总计 4 个第三方包（mysql driver/ulid/jwt/yaml）。

## 目录

```
server/
  go.mod  cmd/anmo/main.go  config.example.yaml
  migrations/001_identity.sql … 008_ops.sql  schema/README.md
  internal/
    config/config.go
    logger/logger.go
    database/{database.go,migrate.go,migrate_test.go}
    shared/{errors.go,tx.go,ids.go,pagination.go,respond.go}
    middleware/{requestid.go,logging.go,recover.go,auth.go}
    router/router.go
    modules/{identity,member,service,card,appointment,transaction,content,ops}/
      AGENTS.md  api.go  module.go(handler 挂载, V1 最小)
```

## 模块挂载协议（D6 预埋）

```go
type Module interface { Mount(mux *http.ServeMux, r RouteDeps) }
```
每个模块提供 `New(deps) *Provider`，Provider 暴露 api.go 中定义的公开方法（V1 骨架阶段为空/最小实现），main.go 依赖注入并 Mount。跨模块调用一律 `*other.Provider` 的公开方法；事务执行器 `shared.Tx` 为 `interface{ ExecContext/QueryContext/... }`（*sql.Tx 满足）。

## Migrations（24 表分布）

- 001_identity: identity_user, identity_role, identity_permission + 种子(role/permission/OWNER 账号, 密码 bcrypt, 手机号来自 config)
- 002_member: member, member_tag, member_tag_rel
- 003_service: service_category, service
- 004_card: card_template, card_service_rule, member_card, card_transaction
- 005_appointment: appointment, appointment_service, appointment_status_log
- 006_transaction: payment, redemption(含 active_lock 生成列+UNIQUE), redemption_reversal
- 007_content: content_page_config, content_banner, content_announcement, content_system_setting
- 008_ops: ops_operation_log, ops_insight_snapshot

关键约束：member.phone UNIQUE、member_no UNIQUE、appointment_no UNIQUE、redemption.idempotency_key UNIQUE、redemption UNIQUE(active_lock)、card_transaction(member_card_id,created_at)、appointment(status,scheduled_start)、appointment(scheduled_start,scheduled_end)、appointment CHECK(scheduled_end>scheduled_start)、member_card CHECK(remaining_count>=0)、FK 按计划 §67、money BIGINT、状态列 VARCHAR(32)。

## Migration Runner

schema_migrations(filename PK, applied_at)。启动时读 migrations/*.sql 排序，未应用的逐个执行（多语句 Exec），成功后记录。DDL 隐式提交无法事务化 → 单文件执行后立即记录，失败即退出启动（fail-fast）。

## 配置项

server.addr(:8080)、mysql.dsn、auth.jwt_secret、auth.admin_phone/admin_password_seed、sms.mode(dev)、log.level、business.*(营业时间/提前量/取消窗口——读取侧 Phase 6 用)。

## 测试

- migrate_test：对临时库执行 runner 两次断言幂等，断言 24 张表存在
- shared/errors、ids、pagination 单测
- healthz 冒烟：main 启动后 httptest 检查（router 单测）

## 不做

无 ORM、无框架、无 docker 化后端镜像、无 CI、无 graceful 做秀（保留基本 http.Server Shutdown）。
