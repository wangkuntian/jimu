# 本地全栈 demo

`make demo` 用一条命令启动 **full server + jimu-web**，用于验证路线 B 的全栈体验。

> 这个 demo 使用 `full` profile 和当前 `jimu-web`，不是 `jimu new` 生成的 full-stack 项目，也不是生产采用证据。

## 前置条件

- Docker + Docker Compose v2
- Bun 1.4.x
- `curl`、`openssl`
- 与 `jimu` 同级的 `jimu-web` checkout（可用 `JIMU_WEB_DIR` 指向其他路径）

首次运行需要构建 full server 镜像和安装前端依赖，耗时取决于网络和镜像缓存；后续运行会明显更快。

## 启动

```bash
make demo
```

脚本会：

1. 用独立 Compose 项目 `jimu-demo` 启动 full server、MariaDB 和 Redis。
2. 等待管理端 `/readyz` 就绪。
3. 在 `jimu-web` 运行 `bun install --frozen-lockfile` 并启动 Vite。
4. 打印 demo 地址和管理员密码文件路径。

打开：

- UI：`http://127.0.0.1:5173`
- API：`http://127.0.0.1:18080/api/v1`
- 管理端就绪检查：`http://127.0.0.1:19090/readyz`

管理员账号是 `admin`，密码在 `secrets/admin_password.txt`。

## 演示路径

```text
admin 登录
  -> 租户列表 / 新建租户 / 分配套餐
  -> 用户列表 / 新建用户 / 分配角色
  -> 角色权限
  -> 账号安全
```

不需要手工连数据库；迁移和结构种子由 Compose 的 `jimu-init` 完成。

## 端口与隔离

默认端口刻意避开常规开发栈：

| 用途 | 默认端口 | 覆盖变量 |
|---|---:|---|
| UI | 5173 | `JIMU_DEMO_WEB_PORT` |
| HTTP API | 18080 | `JIMU_DEMO_HTTP_PORT` |
| 管理端 | 19090 | `JIMU_DEMO_MANAGEMENT_PORT` |
| MariaDB | 23306 | `JIMU_DEMO_DB_PORT` |
| Redis | 26379 | `JIMU_DEMO_REDIS_PORT` |

Compose 项目名和数据卷也可用 `JIMU_DEMO_PROJECT`、`JIMU_DEMO_MARIADB_VOLUME`、`JIMU_DEMO_REDIS_VOLUME` 覆盖。

## 停止与清理

- `Ctrl-C`：停止 Vite，并执行 `make down` 停止 demo Compose 栈。
- demo 数据卷默认保留，方便下次快速启动；需要重置时删除 `jimu-demo-mariadb-data` 和 `jimu-demo-redis-data`。

## 排查

- 端口被占用：脚本会在启动前报出冲突端口和对应变量，用上面的 `JIMU_DEMO_*` 变量换端口。
- 前端仍请求 8080：确认 `jimu-web` 已包含 `JIMU_API_PROXY_TARGET` 支持。
- 后端未就绪：查看 `make logs`，或手动访问 `http://127.0.0.1:19090/readyz`。
