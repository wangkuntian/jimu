#!/bin/bash
# 备份/恢复往返测试：dump → 破坏数据 → restore → 数据还原。
# 通过 docker exec 调用 mariadb 容器内二进制（mariadb-dump/mariadb），
# 容器内走 socket 连接，不依赖主机安装 DB 客户端。
#
# 用法: ./scripts/test_backup_restore.sh [container] [db_name]
# 环境变量: DB_USER, DB_PASSWORD
# 默认 container=jimu-test-mysql，db=jimu_test。

set -euo pipefail

CONTAINER="${1:-jimu-test-mysql}"
DB_USER="${DB_USER:-root}"
DB_PASSWORD="${DB_PASSWORD:-root}"
DB_NAME="${2:-jimu_test}"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
DUMP_FILE="$WORKDIR/${DB_NAME}.sql.gz"

MARKER_TABLE="backup_restore_marker"
MARKER_VAL="roundtrip-$(date +%s)"

exec_sql() { docker exec "$CONTAINER" mariadb -u "$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" "$@"; }

echo "=== 备份/恢复往返测试 ==="
echo "container: $CONTAINER  db: $DB_NAME"

# 1. 建标记表 + 写入一行
exec_sql -e "DROP TABLE IF EXISTS ${MARKER_TABLE}"
exec_sql -e "CREATE TABLE ${MARKER_TABLE} (id INT PRIMARY KEY, val VARCHAR(255))"
exec_sql -e "INSERT INTO ${MARKER_TABLE} (id, val) VALUES (1, '${MARKER_VAL}')"
echo "✅ 写入标记数据: $MARKER_VAL"

# 2. 备份（容器内 mariadb-dump 输出 gz 到主机文件）
docker exec "$CONTAINER" mariadb-dump \
  -u "$DB_USER" -p"$DB_PASSWORD" \
  --single-transaction --routines --triggers \
  "$DB_NAME" | gzip > "$DUMP_FILE"
[ -s "$DUMP_FILE" ] || { echo "❌ dump 为空"; exit 1; }
echo "✅ 备份完成: $(du -h "$DUMP_FILE" | cut -f1)"

# 3. 破坏数据（清空标记表）
exec_sql -e "TRUNCATE TABLE ${MARKER_TABLE}"
COUNT=$(exec_sql -N -e "SELECT count(*) FROM ${MARKER_TABLE}")
[ "$COUNT" = "0" ] || { echo "❌ truncate 后应无数据，实际 $COUNT"; exit 1; }
echo "✅ 数据已清空"

# 4. 恢复
gunzip < "$DUMP_FILE" | docker exec -i "$CONTAINER" mariadb \
  -u "$DB_USER" -p"$DB_PASSWORD" "$DB_NAME"

# 5. 验证数据还原
RESTORED=$(exec_sql -N -e "SELECT val FROM ${MARKER_TABLE} WHERE id = 1")
if [ "$RESTORED" = "$MARKER_VAL" ]; then
  echo "✅ 恢复成功，数据还原: $RESTORED"
else
  echo "❌ 恢复失败，期望 '$MARKER_VAL'，实际 '$RESTORED'"
  exit 1
fi

# 清理
exec_sql -e "DROP TABLE ${MARKER_TABLE}"
echo "✅ 往返测试通过"
