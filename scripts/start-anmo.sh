#!/usr/bin/env bash
# 一键启动 Anmo 本地服务（后端 :8080 + H5 :5173），供局域网验收使用。
# 用法: bash scripts/start-anmo.sh
set -euo pipefail
cd "$(dirname "$0")/.."

ROOT="$(pwd)"

# 1) MySQL 容器
if ! docker ps --format '{{.Names}}' | grep -q '^anmo-mysql$'; then
  echo "[1/4] 启动 MySQL 容器..."
  docker compose up -d
  sleep 3
else
  echo "[1/4] MySQL 已在运行"
fi

# 2) 后端二进制（无则现场编译）
if [ ! -x /tmp/anmo-bin ] || [ "$ROOT/server/cmd/anmo/main.go" -nt /tmp/anmo-bin ]; then
  echo "[2/4] 编译后端..."
  (cd "$ROOT/server" && go build -o /tmp/anmo-bin ./cmd/anmo)
fi

# 3) 后端（已监听则跳过）
if ! curl -sf -o /dev/null http://127.0.0.1:8080/healthz; then
  echo "[3/4] 启动后端 :8080 ..."
  nohup /tmp/anmo-bin -config "$ROOT/server/config.example.yaml" \
    >> /tmp/anmo-server.log 2>&1 &
  for i in $(seq 1 20); do
    curl -sf -o /dev/null http://127.0.0.1:8080/healthz && break
    sleep 0.5
  done
  curl -sf -o /dev/null http://127.0.0.1:8080/healthz \
    || { echo "后端启动失败，查看 /tmp/anmo-server.log"; exit 1; }
else
  echo "[3/4] 后端已在运行"
fi

# 4) H5 dev server（已监听则跳过）
if ! curl -sf -o /dev/null http://127.0.0.1:5173/; then
  echo "[4/5] 启动 H5 :5173 ..."
  nohup npm --prefix "$ROOT/apps/customer" run dev \
    >> /tmp/anmo-h5.log 2>&1 &
  sleep 3
else
  echo "[4/5] H5 已在运行"
fi

# 5) 商家端 dev server（已监听则跳过）
if ! curl -sf -o /dev/null http://127.0.0.1:5174/; then
  echo "[5/5] 启动商家端 :5174 ..."
  nohup npm --prefix "$ROOT/apps/admin" run dev \
    >> /tmp/anmo-admin.log 2>&1 &
  sleep 3
else
  echo "[5/5] 商家端已在运行"
fi

LAN_IP=$(hostname -I | awk '{print $1}')
echo
echo "✅ 全部就绪"
echo "   手机/局域网访问 H5:  http://${LAN_IP}:5173"
echo "   商家端管理界面:      http://${LAN_IP}:5174"
echo "   API:                 http://${LAN_IP}:8080"
echo "   后台账号: 13800000000 / anmo-admin-2026"
echo "   顾客短信验证码(dev): 123456"
echo "   日志: /tmp/anmo-server.log  /tmp/anmo-h5.log  /tmp/anmo-admin.log"
