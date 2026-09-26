# PRD — Phase 11: E2E

## Goal

plan.md §120：完整闭环 E2E。基于真实 HTTP 栈（全部模块挂载 + 真实 MySQL）。

## Requirements

1. 抽取 internal/app 装配（main 复用），E2E 用 httptest.Server 驱动
2. 主链路：admin 登录→建服务/卡模板/规则→顾客登录→建卡→预约→确认→开始→完成→核销→余额 9→流水/收款可查
3. 负路径：并发抢时段 API 级 1 成功、重复核销幂等、撤销恢复+现金重结不双计、越权 403、取消释放时段
4. 全部 §136 核心正确性项在 E2E 断言

## Acceptance Criteria

- [ ] E2E 测试全过
- [ ] go build/vet/test 全过
