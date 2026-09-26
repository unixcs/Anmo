# content 模块

## 职责
首页内容（content_page_config JSON blocks，D16 引用素材 id 不复制正文）、Banner、公告 CRUD、系统设置（content_system_setting；业务默认值以 config 为底）。

## 数据表
content_page_config, content_banner, content_announcement, content_system_setting。

## 公开 API（api.go）
- 顾客：`Home(ctx)` — 返回页面 blocks + ACTIVE banners/announcements；`Settings(ctx)` — 营业时间等公开项
- 后台：PageConfig Get/Update、Banner CRUD、Announcement CRUD、Settings Get/Update

## 事务规则
纯 CRUD；不做拖拽 CMS（§82）。

## 测试
blocks JSON 读写、上下架过滤、settings 键唯一。

## 禁止
不做多页面 CMS/模板引擎；不做富媒体存储。
