# Anmo 接手指南（HANDOFF）

> 任何 Agent / 开发者在任何机器上接手本项目，从这份文件开始。
> 最高约束：根目录 [AGENTS.md](AGENTS.md)（含 D1–D18 冻结决策与 V1 边界）；完整规划：[plan.md](plan.md)；开发流程：`.trellis/workflow.md`。

## 一、项目现状（2026-09-27）

- **V1 全量交付完毕**（Phase 0–11 + 对抗式审查，13 个 Go 测试包全绿）。
- V1 后按用户需求追加：商家端 Web UI（`apps/admin`，8 页面覆盖全部 /admin 端点）、顾客端核销二维码 ↔ 商家扫码/拍照核销闭环、手机适配、HTTPS、一键启动脚本、[TUTORIAL.md](TUTORIAL.md) 小白教程。
- 历史：`.trellis/workspace/anmo-agent/journal-1.md`（PHASE RESULT + POST-V1 全记录，接手前扫一眼末尾 6 条）。

## 二、机器与仓库拓扑

| 机器 | 角色 | 路径 | 说明 |
|------|------|------|------|
| WSL (feng-1) | 开发机 A（原始） | `/mnt/Projects/Anmo` | 四件套：8080 / 5173 / 5174 / 5175 |
| Tencent | 开发机 B（新主力） | `/mnt/Projects/Anmo` | `ssh tencent`（root）；工具链见下 |
| yun1 | 部署演示机 | `/opt/anmo` | `ssh yun1`（root）；公网 121.41.206.32 |
| GitHub | 唯一事实源 | `unixcs/Anmo`（main） | 两台开发机 + 部署更新都走它 |

**多端协作工作流（约定）**：
1. 动手前 `git pull --rebase origin main`；
2. 小步提交，一次 Phase/一个修复一条 commit，push 回 GitHub；
3. `.trellis/`（journal、spec、task）随仓库走 —— **journal 就是交接日志**，跨机器无缝续接；
4. 严禁两台机器同时改未推送的同一区域；换机前先 push。
5. 生产凭据/证书/本地配置永不入库（`.gitignore` 已覆盖）。

## 三、各机环境备忘

### WSL（开发机 A）
- Go 1.27.1 在 `/usr/local/go`（需 `export PATH=$PATH:/usr/local/go/bin`）。
- 一键启动：`bash scripts/start-anmo.sh`（MySQL→后端→5173/5174/5175 全拉起并巡检）。
- 后端必须从 `server/` 目录或用 `-config` 指路启动（migrations 相对路径）；现行为 `-config server/config.example.yaml`。
- 自签证书 `scripts/certs/`（gitignored），换网络重跑 `scripts/gen-dev-cert.sh`。

### Tencent（开发机 B）
- 仓库 `/mnt/Projects/Anmo`；GitHub 已配 SSH key（`~/.ssh/id_ed25519_github_unixcs` → unixcs）。
- Go 1.27.1 在 **`/usr/local/go1.27`**（系统 go 1.23 未动）；npm 全局 bin 在 `/usr/local/node/bin`；两者已由 `/etc/profile.d/dev-tools.sh` 注入 PATH + `GOPROXY=https://goproxy.cn,direct`。**非登录 shell 需手动 `source /etc/profile.d/dev-tools.sh`**。
- npm 安装用 `--registry=https://registry.npmmirror.com`。
- MySQL：项目根 `docker compose up -d`（anmo-mysql，33306，与 WSL 同凭据）。仅内网/tailnet 可达（公网只开 22/80/443，UFW）。
- 已装：trellis 0.6.17、codex、node 20。启动后端：`cd server && go build -o /tmp/anmo-dev-bin ./cmd/anmo && nohup /tmp/anmo-dev-bin -config /mnt/Projects/Anmo/server/config.example.yaml > /tmp/anmo-dev.log 2>&1 &`（从 server/ 目录起，或直接跑 `bash scripts/start-anmo.sh`）。

### yun1（部署演示机）
- 入口：`http://121.41.206.32:18090` ／ `https://121.41.206.32:18091`（自签证书 SAN 含公网 IP，手机信任一次即可用摄像头实时扫码）。
- 商家后台 `/admin-ui/`；凭据在 yun1:`/opt/anmo/credentials.txt`；运维与更新流程见 yun1:`/opt/anmo/README.md`（5 步：WSL 构建 → save|scp|load → force-recreate）。
- 红线：1.6G 内存机器，勿在本机 build 镜像；MySQL 已限内存；改完跑 `bash "/mnt/vps/tencent/Remote AI Coding/ops/health-yun.sh" yun1` 须全绿。

## 四、开发必读顺序（每 Phase）

1. 根 `AGENTS.md` → 2. 模块 `AGENTS.md` → 3. 模块 `api.go` → 4. 相关代码 → 5. 改 → 6. 测试。
- 验证命令：`go build ./... && go vet ./... && go test ./...`（测试需 `ANMO_TEST_MYSQL_DSN`，写法见 journal 与 yun1 README；Tencent 上已验证全绿）。
- 前端：`./node_modules/.bin/vue-tsc --noEmit` + `vite build`（勿用 npx 全局版）。
- schema 只增不改（`server/migrations/NNN_*.sql`）；歧义一律"选不做"；禁 Plan 外实体。

## 五、已知约束 / 下一步候选

- 短信验证码 dev 固定 123456（正式营业前必须接真实通道）；admin 种子密码上线环境已随机化（yun1 credentials.txt）。
- 扫码核销：HTTP 页面用「拍照识别」，HTTPS（5175 / yun1 18091）可用摄像头实时扫。
- 候选需求（用户未拍板，勿擅自做）：正式短信通道、多端冲突更强的通知、营业数据报表增强。
