# V2.2 第七批：真实驱动 Windows 微信开发者工具全流程测试 + 对抗审查

## Goal

miniprogram-automator 经 WSL 连接 Windows 开发者工具自动化端口(9420)，对新包(appid/VERSION=V2.2.6)跑真实全流程：游客浏览→微信一键登录(真实 code2session)→预约提交→我的预约→核销码门槛→关于页版本标识→测试预约 UI 取消清理；对抗审查 weapp 代码与运行时 console；发现 bug 当批修复

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
