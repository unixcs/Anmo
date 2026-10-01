#!/usr/bin/env bash
# backup-sqlite.sh — Anmo SQLite 定时备份 + 恢复验证（宿主机上运行，如 yun1 crontab）。
#
# 用法：
#   bash backup-sqlite.sh                # 快照一次（默认保留 14 份）
#   bash backup-sqlite.sh --restore-test # 用最近一份快照做启动级恢复演练
#
# 原理：
#   快照走容器内 `-backup`（VACUUM INTO）：在线一致、含 WAL 内容、
#   落盘后自动 integrity_check；备份目录是独立挂载卷（./backups:/backups），
#   即使容器/数据卷损坏备份仍在宿主机上。
#
# crontab 示例（每天 03:30）：
#   30 3 * * * cd /opt/anmo && bash scripts/backup-sqlite.sh >> var/backup.log 2>&1
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/anmo}"
KEEP=14
BACKUPS="$COMPOSE_DIR/backups"
CONTAINER="${CONTAINER:-anmo-server}"
# F5：镜像 tag 可用环境变量对齐实际部署（compose 默认 sqlite，yun1 现网 v2）。
IMAGE="${ANMO_IMAGE:-anmo-server:v2}"
# 快照体积下限：24 张表 + 种子数据远不止几 KB；低于此值视为快照失败，
# 绝不能让它顶替好备份进入保留轮换（F5/墨菲定律）。
MIN_SNAPSHOT_BYTES=4096

# crontab 重定向目标 var/backup.log —— 目录必须存在，否则整个 cron 静默无日志。
mkdir -p "$BACKUPS" "$COMPOSE_DIR/var"

do_backup() {
  local ts dest size
  ts="$(date +%Y%m%d-%H%M%S)"
  dest="$BACKUPS/anmo-$ts.db"
  docker exec "$CONTAINER" /app/anmo -backup "/backups/$(basename "$dest")"
  size="$(stat -c%s "$dest")"
  if (( size < MIN_SNAPSHOT_BYTES )); then
    echo "[$(date '+%F %T')] BACKUP FAILED: snapshot $dest is only ${size}B (< ${MIN_SNAPSHOT_BYTES}B) — removing, keeping previous backups"
    rm -f "$dest" "$dest-wal" "$dest-shm"
    return 1
  fi
  chmod 600 "$dest"   # 快照含全店会员/交易数据：宿主机其他用户不可读
  echo "[$(date '+%F %T')] backup -> $dest (${size}B)"
  # 保留策略：只留最近 KEEP 份
  ls -1t "$BACKUPS"/anmo-*.db 2>/dev/null | tail -n +$((KEEP + 1)) | while read -r old; do
    rm -f "$old" "$old-wal" "$old-shm"
    echo "[$(date '+%F %T')] pruned $old"
  done
}

do_restore_test() {
  # 取最近一份快照，用一次性容器以它为库跑 -migrate：
  # 能开库、读出全部 schema 并确认全部 migration 都在位，即恢复可用。
  local latest
  latest="$(ls -1t "$BACKUPS"/anmo-*.db | head -1)"
  echo "restore-test on $(basename "$latest")"
  # 快照本身保持只读：拷贝到临时目录后再做启动级演练（WAL 模式开库需要写权限）
  rm -rf /tmp/anmo-restore-test && mkdir -p /tmp/anmo-restore-test
  cp "$latest" /tmp/anmo-restore-test/anmo.db && chown -R 10001:10001 /tmp/anmo-restore-test
  docker run --rm -v /tmp/anmo-restore-test:/data \
    -e ANMO_DB_PATH=/data/anmo.db \
    -e ANMO_AUTH_JWT_SECRET=restore-test-only-not-a-real-secret-0000 \
    --entrypoint /app/anmo "$IMAGE" -migrate
  rc=$?; rm -rf /tmp/anmo-restore-test
  if (( rc == 0 )); then
    echo "restore-test OK: 最近快照可被服务进程打开并完成 schema 校验"
  fi
  return $rc
}

case "${1:-}" in
  --restore-test) do_restore_test ;;
  *) do_backup ;;
esac
