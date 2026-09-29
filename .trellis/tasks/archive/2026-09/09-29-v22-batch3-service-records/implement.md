# Implement — V2.2 第三批

执行顺序（每步可编译可测）：

1. **migration 015** `server/migrations/015_service_records.sql`：service_tag + service_record 建表、payment 重建（member_id → NULL，保留全部约束/索引/触发器/生成列）。附 `migrate_015_test.go`（照 migrate_014_test 模式，含升级路径 + 约束仍生效断言）。
2. **service 模块**：tags.go（CRUD + used 判定）、records.go（InsertRecordTx / ReverseByRedemptionTx / RevokeRecord / 追踪查询+summary）、handler 路由（6 条，见 design §2.1）+ 单测。
3. **member 模块**：EnsureByPhoneTx（api.go 注明边界；复用现有插入 helper）+ 单测。
4. **content 模块**：service_note_presets 校验 + 下发清洗 + 单测。
5. **transaction 模块**：四结算入口接入记录字段 + TX_NEED_CONFIRM；ReverseRedemption 联动；新增 POST /admin/walkin/settle（WalkInSettle + handler + 幂等）；payment 列表/workbench member NULL 兜底；settle_test/walkin_test 扩展与适配。
6. **admin 前端**：ServiceTagsPage 新页 + 路由菜单；三处结算弹窗扩展（部位/方式/技师备注+快捷短语/沟通必勾）；散客结算弹窗；MembersPage 服务追踪区；RecordsPage 未登记兜底。`npm run build`（--prefix apps/admin）。
7. **AGENTS.md**：追加 D28/D29 冻结决策（§ 冻结决策表末尾）。
8. 全量验证：`go build ./... && go vet ./... && go test ./...`（-C server）+ admin build。
9. 不 commit——留给主会话 Murphy 验收后统一提交。

边界提醒：

- Bash 会话 cwd 必须保持在仓库根；目录相关命令用子 shell `(cd … && …)` 或 `go -C`/`npm --prefix` 形式。
- migration 只增不改；禁 float 金额（整数分）；时间一律墙上时间字符串。
- 跨模块只能调对方导出方法（api.go 边界）；事务一律 shared.RunInTx（_txlock=immediate）。
- 错误码风格照既有（模块前缀_大写蛇形），文案全中文、可直接展示。
