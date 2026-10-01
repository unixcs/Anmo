# V2.2 第九批：十轮审查修复（安全闸/运维防线/后端逻辑/双端体验对齐）

## Goal

按 REVIEW.md §7 六组修复约 290 行：安全闸（SMS off/wx 警示/body 限幅/secret 校验/认证限速/审计过滤）、运维防线（空库守卫/备份四防护/-daily/healthz）、后端逻辑（LOCK_RETRY/幂等体校验/valid_from/profile 包裹）、weapp 与 H5 体验、双端对齐；含单测与 automator 回归

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
