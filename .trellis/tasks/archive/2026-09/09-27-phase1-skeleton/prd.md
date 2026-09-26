# PRD — Phase 1: 工程骨架（数据库冻结 + Go 骨架）

## Goal

按 plan.md §110 建立可运行的工程骨架并完成数据库冻结。本 Phase 结束后：`docker compose up` + `go run ./cmd/anmo` 可启动并对 /healthz 返回 200；migrations 001-008 全部执行成功，24 张表与索引/约束与 AGENTS.md 冻结决策一致。

## Requirements

1. 迁移系统：`server/migrations/NNN_*.sql`（001_identity → 008_ops，按 §96），自建 runner（schema_migrations 表记录已应用文件，按文件名序执行，幂等可重入）
2. 数据库 schema：覆盖 §124 全部 24 张表；主键 CHAR(26) ULID；价格整数分 BIGINT；utf8mb4；DATETIME 业务时间；索引按 §70 + 冻结决策 D14 唯一约束；核销不变量用生成列 active_lock + UNIQUE（W-E）；payment.status 枚举含 VALID/VOIDED（D1）；identity_user.role 枚举（D2）
3. Go 骨架：cmd/anmo/main.go + internal/{config,logger,database,router,middleware,shared} + internal/modules/{identity,member,service,card,appointment,transaction,content,ops}
4. 每模块 AGENTS.md（职责/表/公开API/事务规则/测试/禁止事项）+ api.go（Provider 结构与依赖注入挂载点，V1 先空实现）
5. shared：错误类型(AppError with code/http status)、TxRunner(D6)、ULID 生成、分页参数、响应封装
6. config：YAML + 环境变量覆盖（ANMO_ 前缀），含 mysql dsn / jwt secret / sms 模式 / 日志级别
7. middleware：RequestID、日志(slog)、Recover、Auth 占位（Phase 2 实装）、操作日志钩子占位（D7）
8. 静态种子：identity_role/identity_permission 种子、OWNER 账号种子（管理员手机号+初始密码，通过配置注入）
9. 验收命令全过：`go build ./... && go vet ./... && go test ./...`（含 1 个 db 迁移集成测试）+ migration 在干净库上跑通

## Acceptance Criteria

- [ ] 干净 MySQL 库上 migrate up 全成功，24 表齐全，重复执行幂等
- [ ] go build/vet/test 全过
- [ ] /healthz 200
- [ ] 8 个模块目录各有 AGENTS.md 与可挂载 api.go，router 无业务路由
- [ ] 无 §131 禁止实体表/字段
