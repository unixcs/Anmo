# Phase 0: Plan Review

## Goal

对 plan.md 做只读审查（业务闭环/数据库/状态机/事务/并发/权限/模块边界/V1范围），输出 REVIEW 并给出 PASS/BLOCKED

## Requirements

- 只审查、只输出报告，不写任何业务代码
- 审查对象：`plan.md`（项目唯一需求源）
- 检查维度：业务闭环（§1-2）、数据库设计（§63-64/§124-127）、状态机（§24-25）、事务完整性（§53/§127）、并发（§72/§98）、权限（§100）、模块边界（§89-93）、V1 范围（§3-5/§131）

## Acceptance Criteria

- [ ] `REVIEW.md` 存在于本任务目录，严格按 plan.md §133 五段格式
- [ ] DECISION = PASS（若 BLOCKED 只修阻断项）
- [ ] 未修改任何业务代码

## Notes

- 本任务为 lightweight，PRD-only
