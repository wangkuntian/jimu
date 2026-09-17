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
# 安装逻辑与 CI 共用 scripts/install_db_clients.sh（pg_dump 必须 >= 服务端版本，
# 发行版自带的 16.x 无法 dump PG 17，故走 PGDG 源），可用 --build-arg 指定大版本。
ARG PG_CLIENT_VERSION=17
COPY scripts/install_db_clients.sh /tmp/install_db_clients.sh
RUN PG_CLIENT_VERSION="${PG_CLIENT_VERSION}" bash /tmp/install_db_clients.sh \
    && rm -f /tmp/install_db_clients.sh


COPY scripts/backup.sh scripts/restore.sh scripts/db_common.sh /opt/jimu/scripts/
RUN chmod 0755 /opt/jimu/scripts/*.sh

WORKDIR /opt/jimu
ENTRYPOINT ["bash", "/opt/jimu/scripts/backup.sh"]
CMD ["/backups"]
