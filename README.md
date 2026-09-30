# Anmo — 个人到店按摩店服务管理系统

个人到店按摩店的预约与服务管理：顾客在线预约 → 店主（兼唯一按摩师）接单服务 → 会员卡核销 / 现金·微信转账结算。顾客端提供 **H5（Vue3）** 与 **微信小程序（原生）** 双端，共享同一 Go 后端与会员身份。

- 生产环境：[https://anmo.oiob.cn](https://anmo.oiob.cn)（H5 + API；小程序同源对接）
- 技术栈：Go 1.27（Modular Monolith）· SQLite（modernc 纯 Go + WAL）· Vue3 + Vite + TS · 微信小程序原生（零构建）

## 仓库结构（monorepo，按端分目录）

```
Anmo/
├── server/            Go 后端（单服务单库，全部业务模块）
│   ├── cmd/anmo/          入口，默认监听 :8080
│   ├── internal/modules/  identity / member / service / card / appointment
│   │                      / transaction / content / ops（模块化单体）
│   └── migrations/        00N_*.sql（只增不改）
├── apps/
│   ├── customer/      顾客 H5（Vue3 + Vite + TS）
│   ├── weapp/         顾客微信小程序（原生，源码即产物，零构建）
│   └── admin/         店主后台前端（Vite）
├── docs/              部署 runbook、运维手册、品牌规范
├── scripts/           备份 / 证书 / 启动脚本
└── plan.md            全项目计划（AGENTS.md 为其工程化摘要）
```

## 快速开始

**后端**（Go 1.27+，SQLite 零外部依赖）：

```bash
cd server
go run ./cmd/anmo        # 监听 :8080，库文件默认 data/anmo.db（可用 ANMO_DB_PATH 覆盖）
```

验证：`go build ./... && go vet ./... && go test ./...`（测试用内存/临时 SQLite，无需外部服务）。

**顾客 H5**：

```bash
cd apps/customer
npm install && npm run dev     # Vite dev server，API 代理/基址见 src/config 或 vite 配置
```

**微信小程序**：用微信开发者工具直接导入 `apps/weapp/` 即可（原生零构建）。开发期后端基址见该目录 `config.js`（默认本机 `:8080`）；真实生产链路的 DevTools 自动化测试工作流（副本目录 + miniprogram-automator 16 断言）见 [docs/guides/wechat-devtools-automation.md](docs/guides/wechat-devtools-automation.md)。

**店主后台**：

```bash
cd apps/admin
npm install && npm run dev
```

## 文档索引

| 文档 | 内容 |
|------|------|
| [AGENTS.md](AGENTS.md) | 全项目约束：业务闭环、状态机、冻结决策（D1–D29）、模块边界 |
| [plan.md](plan.md) | 项目总计划（权威需求源） |
| [apps/weapp/README.md](apps/weapp/README.md) | 小程序页面 ↔ 后端端点对照、使用说明 |
| [docs/guides/wechat-devtools-automation.md](docs/guides/wechat-devtools-automation.md) | 微信开发者工具自动化：两层自动化机制、11 条踩坑实录、16 断言回归剧本 |
| [docs/guides/](docs/guides/) | 生产部署 runbook、品牌规范等 |

## 开发约定（摘要）

- 所有 schema 变更走 `server/migrations/NNN_*.sql`，历史 migration 禁改
- 预约状态机冻结：`WAITING → IN_SERVICE → COMPLETED`（异常 → CANCELLED / NO_SHOW）
- 核销/撤销/收款均为单事务，`remaining_count >= 0` 与"一预约一笔 VALID payment"为硬不变量
- 金额一律整数分；时间存 `YYYY-MM-DD HH:MM:SS` 墙上时间（Asia/Shanghai）
