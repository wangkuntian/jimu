#!/bin/bash
# 安装数据库客户端工具（备份/恢复与集成测试用）：mariadb-client + PostgreSQL 客户端。
#
# 为什么不用发行版自带的 postgresql-client：pg_dump 要求客户端版本 >= 服务端版本，
# Ubuntu 24.04 自带 16.x，无法 dump PostgreSQL 17（报 "server version mismatch"），
# 因此从 PGDG 源安装指定大版本。
#
# 用法（需 root 或 sudo）：
#   sudo PG_CLIENT_VERSION=17 bash scripts/install_db_clients.sh
# CI（.github/workflows/ci.yml）与备份镜像（deploy/backup/Dockerfile）共用本脚本，
# 避免两处各写一份 PGDG 配置后再次漂移。
#
# 环境变量：
#   PG_CLIENT_VERSION  PostgreSQL 客户端大版本（默认 17，须与目标服务端一致或更高）

set -euo pipefail

PG_CLIENT_VERSION="${PG_CLIENT_VERSION:-17}"

if ! command -v apt-get >/dev/null 2>&1; then
    echo "❌ 本脚本仅支持 Debian/Ubuntu（apt-get）环境" >&2
    exit 1
fi

if [ "$(id -u)" -ne 0 ] && ! command -v sudo >/dev/null 2>&1; then
    echo "❌ 需要 root 权限或 sudo" >&2
    exit 1
fi
SUDO=""
if [ "$(id -u)" -ne 0 ]; then
    SUDO="sudo"
fi

# MariaDB 客户端：mariadb:12 官方镜像已无 mysqldump/mysql 软链，客户端跨版本可 dump
$SUDO apt-get update
$SUDO apt-get install -y --no-install-recommends ca-certificates curl gnupg mariadb-client

# PostgreSQL 客户端：从 PGDG 装指定大版本
$SUDO install -d /usr/share/postgresql-common/pgdg
curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc |
    $SUDO tee /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc >/dev/null
CODENAME="$(. /etc/os-release && echo "$VERSION_CODENAME")"
echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt ${CODENAME}-pgdg main" |
    $SUDO tee /etc/apt/sources.list.d/pgdg.list >/dev/null
$SUDO apt-get update
$SUDO apt-get install -y --no-install-recommends "postgresql-client-${PG_CLIENT_VERSION}"

# 安装即校验：客户端版本不匹配时应在构建/CI 阶段就失败，而不是备份运行期
mariadb-dump --version
"pg_dump" --version
echo "✅ 数据库客户端就绪（PostgreSQL 客户端大版本 ${PG_CLIENT_VERSION}）"
