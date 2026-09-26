# PRD — Phase 9: Customer H5

## Goal

plan.md §118/§101/§104-107/§130：Vue3 + Vite + TS 顾客端（首页/服务/预约/我的）。

## Requirements

1. 目录按 §101：core(api/models/logic/store/utils) + platform(auth/storage/notify) + pages/components
2. 页面：首页（品牌/服务推荐/预约入口/电话/营业时间）、服务列表、预约（选服务→日期→时间→提交）、我的（会员卡/我的预约/历史记录/个人资料）、登录（手机号+验证码）
3. 铁律 §102：core 禁止 window/document/localStorage；业务逻辑在 core/logic；页面只做展示交互；token 存取仅 platform/storage
4. 路由守卫：未登录访问受保护页跳登录
5. 冲突提示 §106："这个时间刚刚被预约，请重新选择"
6. 顾客详情不显示内部备注（§107）
7. npm run build（vue-tsc）通过

## Acceptance Criteria

- [ ] 四大页面 + 登录可用，核心闭环可操作（预约/取消/改期/查卡/查历史）
- [ ] 目录符合 §101、core 无 DOM 访问
- [ ] npm run build 通过
