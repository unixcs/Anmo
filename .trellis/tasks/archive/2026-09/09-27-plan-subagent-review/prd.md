# 子代理 Plan 独立审查（编码前门禁）

## Goal

在 Trellis 接管后、写任何业务代码之前，由**独立子代理**（主会话之外）对 plan.md 做对抗式审查。通过后才能执行 Phase 1 及后续所有 Phase。

## Requirements

- 审查者必须是子代理，不得由主会话自审替代
- 审查输入：plan.md 全文、AGENTS.md、`.trellis/tasks/archive/2026-09/09-27-phase0-review/REVIEW.md`（Phase 0 冻结口径）
- 审查维度按 plan.md §132：业务、数据库、并发、权限、技术，另加 §131 范围违禁检查
- 输出必须落盘：`{本任务目录}/SUBAGENT-REVIEW.md`，含 BLOCKER/WARNING/RECOMMENDATION/SCOPE CHECK/DECISION(PASS|BLOCKED)
- 结论必须明确；BLOCKED 时列出可修复的阻断项

## Acceptance Criteria

- [ ] SUBAGENT-REVIEW.md 存在且含明确 DECISION
- [ ] DECISION=PASS（或 BLOCKED 项已修复并复审通过）
- [ ] 主会话据此决定是否进入 Phase 1
