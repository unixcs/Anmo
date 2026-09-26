# service 模块

## 职责
service_category / service 维护：分类、服务项目、价格（整数分）、时长、上下架、排序；顾客端服务列表（只出 ACTIVE）。

## 数据表
service_category, service。

## 公开 API（api.go）
- 后台：Category CRUD、Service CRUD（Create/Update/Enable/Disable/Sort）
- 顾客：`ListActive(ctx)`；`Get(ctx, id)` — 供 appointment 模块取 snapshot 与校验

## 事务规则
纯 CRUD，无跨模块事务。

## 测试
CRUD、上下架对顾客端可见性的影响、价格/时长校验（>0，>=0 分）。

## 禁止
不做商品/库存/商城语义——统一叫"服务项目"（§14）；不做图片上传存储（cover_image 存 URL 字符串）。
