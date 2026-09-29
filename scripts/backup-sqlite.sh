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

mkdir -p "$BACKUPS"

do_backup() {
  local ts dest
  ts="$(date +%Y%m%d-%H%M%S)"
  dest="$BACKUPS/anmo-$ts.db"
  docker exec "$CONTAINER" /app/anmo -backup "/backups/$(basename "$dest")"
  echo "[$(date '+%F %T')] backup -> $dest"
  # 保留策略：只留最近 KEEP 份
  ls -1t "$BACKUPS"/anmo-*.db 2>/dev/null | tail -n +$((KEEP + 1)) | while read -r old; do
    rm -f "$old" "$old-wal" "$old-shm"
    echo "[$(date '+%F %T')] pruned $old"
  done
}

do_restore_test() {
  # 取最近一份快照，用一次性容器以它为库跑 -migrate：
  # 能开库、读出全部 schema 并确认 13 个 migration 都在位，即恢复可用。
  local latest
  latest="$(ls -1t "$BACKUPS"/anmo-*.db | head -1)"
  echo "restore-test on $(basename "$latest")"
  # 快照本身保持只读：拷贝到临时目录后再做启动级演练（WAL 模式开库需要写权限）
  rm -rf /tmp/anmo-restore-test && mkdir -p /tmp/anmo-restore-test
  cp "$latest" /tmp/anmo-restore-test/anmo.db && chown -R 10001:10001 /tmp/anmo-restore-test
  docker run --rm -v /tmp/anmo-restore-test:/data \
    -e ANMO_DB_PATH=/data/anmo.db \
    -e ANMO_AUTH_JWT_SECRET=restore-test \
    --entrypoint /app/anmo anmo-server:v2 -migrate
  rc=$?; rm -rf /tmp/anmo-restore-test
  return $rc
  echo "restore-test OK: 最近快照可被服务进程打开并完成 schema 校验"
}

case "${1:-}" in
  --restore-test) do_restore_test ;;
  *) do_backup ;;
esac
