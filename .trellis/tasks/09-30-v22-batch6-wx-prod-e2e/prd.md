# V2.2 第六批：真实微信凭据上线 + 小程序切新版本连云 + 全流程对抗审查

## Goal

① yun1 compose 配真实 ANMO_WX_APPID/SECRET 并重建容器，验证真实 code2session 模式（D24 兜底关闭）；② 小程序工程切新版本：project.config.json 填真实 appid（替换 touristappid）+ 版本标识可见（about 页）；③ 全流程对抗式审查与测试：wx 登录错误路径打磨（中文可行动文案/errcode 分流）、服务端+H5+weapp 测试、生产站 playwright 全流程回归

## Requirements

1. **真实凭据上线（yun1，凭据只进服务器 compose，不进仓库）**
   - `/opt/anmo/docker-compose.yml` server 增加 `ANMO_WX_APPID`（真实值）+ `ANMO_WX_SECRET`（真实值）
   - `docker compose up -d --force-recreate server` 后 **必须重启 anmo-nginx**（IP 陷阱）
   - 验证：容器启动日志正常、yun1→api.weixin.qq.com 连通、`POST /api/auth/wx/login` 垃圾 code 走真实
     code2session（返回微信 errcode 文案而非 dev: openid）＝ D24 兜底关闭
2. **weapp 切新版本**：project.config.json `appid` → 真实值；`config.js` 增 `VERSION` 常量；about 页页脚
   显示版本（肉眼确认新包加载）；README 补真实 appid 行为说明（devtools 登录也走真实 code2session）
3. **wx.go errcode 分流**（用户文案中文可行动，英文 errmsg 仅进日志）：40029/40163 → 登录状态已过期请重试；
   45011 → 操作太频繁；-1 → 微信服务繁忙；默认 → 微信登录失败请稍后再试。新增/更新单测
4. **测试工作流**：`go build/vet/test` 全绿；weapp `node --test` 全绿；生产站 playwright GUI 全流程回归
   （游客浏览→预约→未登录提交拦→注册→提交成功→我的预约→核销码门槛），测试会员/预约用后清理
5. **对抗审查**：凭据泄漏面（git/日志/镜像层）、wx 登录全路径、预约主路径；发现问题当批修复

## Deployment

- wx.go 变更 → WSL 构建 `anmo-server:v3` → save|gzip|scp|load 上 yun1 → compose 改 tag →
  force-recreate server → restart anmo-nginx → 公网冒烟（healthz / 匿名目录 / wx 实模式 40029）

## Acceptance Criteria

- [ ] yun1 容器带真实凭据运行，wx login 垃圾 code 返回微信 errcode 分流文案（非 dev: openid）
- [ ] project.config.json appid=真实值；about 页可见版本号
- [ ] wx errcode 分流有单测覆盖；go/vet/test、weapp node --test 全绿
- [ ] 生产站 GUI 全流程通过（8 项）且测试数据已清理
- [ ] secret 在 git 历史中零出现（grep 验证）；README/runbook 同步

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
