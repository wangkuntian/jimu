# 数据库备份/恢复任务镜像：内置 MySQL/MariaDB 与 PostgreSQL 客户端，并带上仓库内的脚本
# （与 compose 里容器内执行的是同一份 scripts/backup.sh / restore.sh，避免逻辑漂移）。
#
# 构建（仓库根目录）：
#   docker build -f deploy/backup/Dockerfile -t jimu-backup:1.0.0 .
#   make build-backup-image
# 运行（ENTRYPOINT 已固定为 backup.sh，参数即输出目录）：
#   docker run --rm -v backups:/backups -e DB_DRIVER=mysql \
#     -e DB_HOST=... -e DB_USER=... -e DB_PASSWORD=... jimu-backup:1.0.0 /backups
# 恢复（需覆盖 ENTRYPOINT，把 restore.sh 作为命令）：
#   docker run --rm --entrypoint bash -v backups:/backups -e FORCE=1 -e DB_DRIVER=postgres \
#     -e DB_HOST=... jimu-backup:1.0.0 /opt/jimu/scripts/restore.sh /backups/jimu_20250101_120000.sql.gz
FROM mariadb:12.1.2-noble

# PostgreSQL 客户端与 mariadb-dump 共存：同一个镜像可备份两种方言。
# pg_dump 要求客户端版本 ≥ 服务端版本（Ubuntu 自带的 16.x 无法 dump PG 17），
# 因此从 PGDG 源安装，可用 --build-arg PG_CLIENT_VERSION=18 跟随服务端升级。
ARG PG_CLIENT_VERSION=17
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl gnupg \
    && install -d /usr/share/postgresql-common/pgdg \
    && curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc \
        -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc \
    && echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt $(. /etc/os-release && echo "$VERSION_CODENAME")-pgdg main" \
        > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends "postgresql-client-${PG_CLIENT_VERSION}" \
    && apt-get purge -y --auto-remove curl gnupg \
    && rm -rf /var/lib/apt/lists/*
# 安装后校验：客户端主版本必须与服务端一致或更高，否则备份会在运行期才失败
RUN pg_dump --version && mariadb-dump --version

COPY scripts/backup.sh scripts/restore.sh scripts/db_common.sh /opt/jimu/scripts/
RUN chmod 0755 /opt/jimu/scripts/*.sh

WORKDIR /opt/jimu
ENTRYPOINT ["bash", "/opt/jimu/scripts/backup.sh"]
CMD ["/backups"]
