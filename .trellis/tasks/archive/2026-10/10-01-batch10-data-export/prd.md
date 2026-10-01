# V2.2 第十批：商家后台数据导出（XLSX + TXT-Markdown）

## Goal

按 docs/proposals/data-export-plan.md v2（P1）实施：老板可从后台导出 会员总表 / 收款流水 / 服务记录，格式 XLSX（对账/全量）与 TXT（Markdown 内容，手机直开/喂 AI/手机号打码）。

## Requirements

- shared/export 工具包：格式分发、Content-Disposition（RFC5987 中文名）、公式注入防护、分→元、10 万行硬顶、Markdown 表格/卡片双排版 + 手机号打码
- 端点 POST /admin/export/{members|payments|service-records}，body {format:"xlsx"|"txt"}；模块内查询，member 域聚合经 card/transaction 的 api.go（守模块边界 D6/D7）
- 审计复用 OperationLog（POST 非 GET 天然落日志）
- admin 前端 downloadBlob（Authorization 头 + blob + 文件名解析）+ MembersPage/RecordsPage 导出按钮
- 单测 + e2e：xlsx 魔数 PK、txt 结构与打码、幂等/权限/行数上限

## Acceptance Criteria

- [ ] go build/vet/test 全绿；admin vite build 通过
- [ ] 会员总表含 累计消费(VALID)/有效卡数/剩余总次数/最近到店 聚合
- [ ] TXT 手机号全部 138**** 形态；XLSX 全量
- [ ] 每次 POST /admin/export/* 在 ops_operation_log 可见
- [ ] 审查子代理结论中的 P0/P1 问题修复后验收

## Notes

- 方案审查：两个并行子代理（后端架构 / 前端横切+对抗完整性）
