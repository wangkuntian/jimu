#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$(mktemp -d)"

cleanup() {
  rm -rf "$WORK"
}
trap cleanup EXIT

fail() {
  echo "❌ $*" >&2
  exit 1
}

run_make() {
  # APP_VERSION 来自外部 shell 时不应改变本测试对 Makefile 默认值的断言。
  env APP_VERSION=dev make -C "$ROOT" --no-print-directory -f "$ROOT/Makefile" "$@"
}

assert_contains() {
  local text="$1" needle="$2" label="$3"
  grep -Fq -- "$needle" <<<"$text" || fail "${label}（未找到：${needle}）"
}

assert_not_contains() {
  local text="$1" needle="$2" label="$3"
  if grep -Fq -- "$needle" <<<"$text"; then
    fail "${label}（意外找到：${needle}）"
  fi
}

assert_target_removed() {
  local target="$1"
  if grep -Eq "^${target}:" <<<"$MAKE_DATABASE"; then
    fail "旧公共 target 仍然存在：${target}"
  fi
}

echo "== 帮助与旧入口 =="
help="$(run_make help)"
assert_contains "$help" "make up " "help 缺少 up"
assert_contains "$help" "make up-observability " "help 缺少 up-observability"
assert_contains "$help" "make image " "help 缺少 image"
assert_contains "$help" "make check " "help 缺少 check"
assert_not_contains "$help" "make compose-up " "help 仍列出 compose-up"
assert_not_contains "$help" "make init-observability " "help 仍列出 init-observability"
assert_not_contains "$help" "make prepare-observability-secrets " "help 暴露了内部 observability Secret 入口"
if grep -Fq 'include .env' "$ROOT/Makefile"; then
  fail "Makefile 仍隐式加载 .env"
fi
MAKE_DATABASE="$(run_make -qp 2>/dev/null || true)"

for target in \
  run \
  secrets \
  docker-build docker-run docker-stop docker-logs \
  compose-up compose-down compose-restart compose-logs compose-migrate compose-seed \
  init-observability; do
  assert_target_removed "$target"
done

echo "== 镜像与启动命令 =="
image_plan="$(run_make -n image VERSION=v9.9.9)"
assert_contains "$image_plan" "docker build --build-arg PROFILE=full -t jimu:v9.9.9 ." "image 未构建版本标签"
assert_contains "$image_plan" "docker tag jimu:v9.9.9 jimu:latest" "image 未更新 latest 标签"

build_server_plan="$(run_make -n build-server VERSION=v9.9.9)"
assert_contains "$build_server_plan" '-X main.version=dev' "VERSION 意外注入本机 server 构建"
assert_not_contains "$build_server_plan" 'main.version=v9.9.9' "VERSION 仍影响本机 server 构建"
build_server_app_version_plan="$(run_make -n build-server APP_VERSION=v1.2.3)"
assert_contains "$build_server_app_version_plan" '-X main.version=v1.2.3' "APP_VERSION 未注入本机 server 构建"

up_plan="$(run_make -n up VERSION=v9.9.9)"
assert_contains "$up_plan" "JIMU_IMAGE=jimu:v9.9.9" "up 未使用版本镜像"
assert_contains "$up_plan" "docker compose --env-file /dev/null -f docker-compose.yml up -d --no-build" "up 未禁用隐式构建"
assert_contains "$up_plan" './scripts/init-secrets.sh --no-legacy' "up 未显式禁用旧 .env 迁移"
assert_not_contains "$up_plan" "LEGACY_ENV_FILE" "up 不得迁移旧 .env"
assert_not_contains "$up_plan" "./scripts/observability.sh start" "up 不得启动观测栈"
printf '%s\n' "$up_plan" >"$WORK/up.plan"
init_line="$(grep -nF './scripts/init-secrets.sh' "$WORK/up.plan" | head -n 1 | cut -d: -f1)"
image_line="$(grep -nF 'docker build --build-arg PROFILE=full' "$WORK/up.plan" | head -n 1 | cut -d: -f1)"
up_line="$(grep -nF 'up -d --no-build' "$WORK/up.plan" | head -n 1 | cut -d: -f1)"
[ -n "$init_line" ] && [ -n "$image_line" ] && [ -n "$up_line" ] || fail "up 计划缺少初始化、镜像或启动步骤"
[ "$init_line" -lt "$image_line" ] || fail "up 必须先初始化 secret 再构建镜像"
[ "$image_line" -lt "$up_line" ] || fail "up 必须先构建镜像再启动 Compose"
grep -Fq '.NOTPARALLEL: up up-observability' "$ROOT/Makefile" \
  || fail "up 与 up-observability 未声明 .NOTPARALLEL"

parallel_up_plan="$(run_make -n -j8 up VERSION=v9.9.9)"
printf '%s\n' "$parallel_up_plan" >"$WORK/parallel-up.plan"
parallel_init_line="$(grep -nF './scripts/init-secrets.sh' "$WORK/parallel-up.plan" | head -n 1 | cut -d: -f1)"
parallel_image_line="$(grep -nF 'docker build --build-arg PROFILE=full' "$WORK/parallel-up.plan" | head -n 1 | cut -d: -f1)"
parallel_up_line="$(grep -nF 'up -d --no-build' "$WORK/parallel-up.plan" | head -n 1 | cut -d: -f1)"
[ "$parallel_init_line" -lt "$parallel_image_line" ] || fail "make -j up 未保证 secret 先于镜像"
[ "$parallel_image_line" -lt "$parallel_up_line" ] || fail "make -j up 未保证镜像先于 Compose"

observability_plan="$(run_make -n up-observability VERSION=v9.9.9)"
assert_contains "$observability_plan" './scripts/init-secrets.sh --observability --no-legacy' "up-observability 未初始化观测凭据"
assert_contains "$observability_plan" '-f compose.observability.yml' "up-observability 未叠加观测栈"
assert_contains "$observability_plan" "JIMU_IMAGE=jimu:v9.9.9" "up-observability 未使用版本镜像"
assert_contains "$observability_plan" '--no-build' "up-observability 未禁用隐式构建"
assert_contains "$observability_plan" './scripts/observability.sh start' "up-observability 未初始化 dashboard/告警"
assert_not_contains "$observability_plan" "LEGACY_ENV_FILE" "up-observability 不得迁移旧 .env"
printf '%s\n' "$observability_plan" >"$WORK/observability.plan"
observability_up_line="$(grep -nF 'up -d --no-build' "$WORK/observability.plan" | head -n 1 | cut -d: -f1)"
observability_start_line="$(grep -nF './scripts/observability.sh start' "$WORK/observability.plan" | head -n 1 | cut -d: -f1)"
[ -n "$observability_up_line" ] && [ -n "$observability_start_line" ] || fail "up-observability 计划缺少启动或观测初始化步骤"
[ "$observability_up_line" -lt "$observability_start_line" ] || fail "up-observability 必须先启动服务再初始化 dashboard/告警"

echo "== 生命周期与运维入口 =="
for target in down restart status logs; do
  plan="$(run_make -n "$target")"
  assert_contains "$plan" 'docker compose --env-file /dev/null -f docker-compose.yml -f compose.observability.yml' "$target 未使用完整 Compose 栈"
done

migrate_plan="$(run_make -n migrate PROFILE=minimal VERSION=v9.9.9)"
assert_contains "$migrate_plan" 'docker build --build-arg PROFILE=minimal -t jimu:v9.9.9 .' "migrate 未构建所选 PROFILE 镜像"
assert_contains "$migrate_plan" 'run --rm jimu-init ./jimu migrate up' "migrate 未通过 jimu-init 执行"
assert_contains "$migrate_plan" 'JIMU_IMAGE=jimu:v9.9.9' "migrate 未使用版本镜像"
printf '%s\n' "$migrate_plan" >"$WORK/migrate.plan"
migrate_image_line="$(grep -nF 'docker build --build-arg PROFILE=minimal' "$WORK/migrate.plan" | head -n 1 | cut -d: -f1)"
migrate_run_line="$(grep -nF 'run --rm jimu-init ./jimu migrate up' "$WORK/migrate.plan" | head -n 1 | cut -d: -f1)"
[ "$migrate_image_line" -lt "$migrate_run_line" ] || fail "migrate 必须先构建镜像再执行"

seed_plan="$(run_make -n seed PROFILE=minimal VERSION=v9.9.9)"
assert_contains "$seed_plan" 'docker build --build-arg PROFILE=minimal -t jimu:v9.9.9 .' "seed 未构建所选 PROFILE 镜像"
assert_contains "$seed_plan" 'run --rm jimu-init ./jimu seed' "seed 未通过 jimu-init 执行"
assert_contains "$seed_plan" 'JIMU_IMAGE=jimu:v9.9.9' "seed 未使用版本镜像"
printf '%s\n' "$seed_plan" >"$WORK/seed.plan"
seed_image_line="$(grep -nF 'docker build --build-arg PROFILE=minimal' "$WORK/seed.plan" | head -n 1 | cut -d: -f1)"
seed_run_line="$(grep -nF 'run --rm jimu-init ./jimu seed' "$WORK/seed.plan" | head -n 1 | cut -d: -f1)"
[ "$seed_image_line" -lt "$seed_run_line" ] || fail "seed 必须先构建镜像再执行"

migrate_status_plan="$(run_make -n migrate-status PROFILE=minimal VERSION=v9.9.9)"
assert_contains "$migrate_status_plan" 'run --rm jimu-init ./jimu migrate status' "migrate-status 未通过 jimu-init 执行"
assert_contains "$migrate_status_plan" 'JIMU_IMAGE=jimu:v9.9.9' "migrate-status 未使用版本镜像"
assert_not_contains "$migrate_status_plan" 'go run -overlay=' "migrate-status 不得绕过容器 Secret 使用本机 CLI"

migrate_down_plan="$(run_make -n migrate-down PROFILE=minimal VERSION=v9.9.9)"
assert_contains "$migrate_down_plan" 'run --rm jimu-init ./jimu migrate down' "migrate-down 未通过 jimu-init 执行"
assert_contains "$migrate_down_plan" 'JIMU_IMAGE=jimu:v9.9.9' "migrate-down 未使用版本镜像"
assert_not_contains "$migrate_down_plan" 'go run -overlay=' "migrate-down 不得绕过容器 Secret 使用本机 CLI"

adminer_plan="$(run_make -n adminer)"
assert_contains "$adminer_plan" '--profile dev up -d adminer' "adminer 未显式启用 dev profile"

echo "== init 旧 .env 条件迁移 =="
NO_ENV_DIR="$WORK/init-no-env"
WITH_ENV_DIR="$WORK/init-with-env"
mkdir -p "$NO_ENV_DIR/scripts" "$WITH_ENV_DIR/scripts"
cp "$ROOT/Makefile" "$NO_ENV_DIR/Makefile"
cp "$ROOT/Makefile" "$WITH_ENV_DIR/Makefile"
cat >"$NO_ENV_DIR/scripts/init-secrets.sh" <<'FAKE_INIT'
#!/usr/bin/env bash
set -euo pipefail
printf 'legacy=%s secrets_dir=%s allow_new_security_keys=%s force=%s args=%s\n' \
  "${LEGACY_ENV_FILE-}" "${SECRETS_DIR-}" "${ALLOW_NEW_SECURITY_KEYS-}" "${FORCE-}" "$*"
FAKE_INIT
cp "$NO_ENV_DIR/scripts/init-secrets.sh" "$WITH_ENV_DIR/scripts/init-secrets.sh"
chmod +x "$NO_ENV_DIR/scripts/init-secrets.sh" "$WITH_ENV_DIR/scripts/init-secrets.sh"
touch "$WITH_ENV_DIR/.env"
NO_ENV_REAL="$(cd "$NO_ENV_DIR" && pwd -P)"
WITH_ENV_REAL="$(cd "$WITH_ENV_DIR" && pwd -P)"

no_env_output="$(env -u LEGACY_ENV_FILE make -C "$NO_ENV_DIR" --no-print-directory init)"
assert_contains "$no_env_output" "legacy=" "无旧 .env 时未调用初始化脚本"
assert_not_contains "$no_env_output" "legacy=$NO_ENV_REAL/.env" "无旧 .env 时仍传入 LEGACY_ENV_FILE"

with_env_output="$(env -u LEGACY_ENV_FILE make -C "$WITH_ENV_DIR" --no-print-directory init)"
assert_contains "$with_env_output" "legacy=$WITH_ENV_REAL/.env" "存在旧 .env 时未传入 LEGACY_ENV_FILE"
CUSTOM_LEGACY_FILE="$WORK/custom-legacy.env"
touch "$CUSTOM_LEGACY_FILE"
custom_legacy_output="$(LEGACY_ENV_FILE="$CUSTOM_LEGACY_FILE" make -C "$WITH_ENV_DIR" --no-print-directory init)"
assert_contains "$custom_legacy_output" "legacy=$CUSTOM_LEGACY_FILE" "显式 LEGACY_ENV_FILE 未传入"
assert_not_contains "$custom_legacy_output" "legacy=$WITH_ENV_REAL/.env" "显式 LEGACY_ENV_FILE 被根 .env 覆盖"
with_env_init_plan="$(env -u LEGACY_ENV_FILE make -C "$WITH_ENV_DIR" --no-print-directory -n init)"
assert_contains "$with_env_init_plan" "LEGACY_ENV_FILE" "make -n init 未展示旧 .env 迁移入口"
with_env_up_plan="$(env -u LEGACY_ENV_FILE make -C "$WITH_ENV_DIR" --no-print-directory -n up)"
assert_not_contains "$with_env_up_plan" "LEGACY_ENV_FILE" "存在旧 .env 时 make -n up 仍迁移"
with_env_observability_plan="$(env -u LEGACY_ENV_FILE make -C "$WITH_ENV_DIR" --no-print-directory -n up-observability)"
assert_not_contains "$with_env_observability_plan" "LEGACY_ENV_FILE" "存在旧 .env 时 make -n up-observability 仍迁移"

CUSTOM_SECRETS_DIR="$WORK/custom-secrets"
custom_init_output="$(env -u LEGACY_ENV_FILE -u SECRETS_DIR make -C "$NO_ENV_DIR" --no-print-directory init JIMU_SECRETS_DIR="$CUSTOM_SECRETS_DIR")"
assert_contains "$custom_init_output" "secrets_dir=$CUSTOM_SECRETS_DIR" "init 未传递自定义 JIMU_SECRETS_DIR"
custom_observability_output="$(env -u LEGACY_ENV_FILE -u SECRETS_DIR make -C "$NO_ENV_DIR" --no-print-directory prepare-observability-secrets JIMU_SECRETS_DIR="$CUSTOM_SECRETS_DIR")"
assert_contains "$custom_observability_output" "secrets_dir=$CUSTOM_SECRETS_DIR" "observability 准备目标未传递自定义 JIMU_SECRETS_DIR"
assert_contains "$custom_observability_output" "args=--observability --no-legacy" "observability 准备目标未禁用旧 .env 迁移"
allow_new_output="$(env -u LEGACY_ENV_FILE -u SECRETS_DIR -u ALLOW_NEW_SECURITY_KEYS make -C "$NO_ENV_DIR" --no-print-directory init ALLOW_NEW_SECURITY_KEYS=1 FORCE=1)"
assert_contains "$allow_new_output" "allow_new_security_keys=1" "make init 未传递 ALLOW_NEW_SECURITY_KEYS"
assert_contains "$allow_new_output" "force=1" "make init 未传递 FORCE"

echo "== dev 入口 =="
dev_plan="$(run_make -n dev)"
assert_contains "$dev_plan" 'ENV=dev PROFILE=full ./scripts/dev.sh' "dev 未按 ENV/PROFILE 调用本地脚本"
[ -f "$ROOT/scripts/dev.sh" ] || fail "缺少 scripts/dev.sh"
bash -n "$ROOT/scripts/dev.sh"
grep -Fq './scripts/init-secrets.sh --no-legacy' "$ROOT/scripts/dev.sh" \
  || fail "dev.sh 仍可能迁移旧 .env"
grep -Fq 'cmd = "go build -overlay=\"$JIMU_OVERLAY\" -o ./tmp/server ./cmd/server"' "$ROOT/.air.toml" \
  || fail ".air.toml 未使用 JIMU_OVERLAY 构建"

echo "== dev.sh 本机编排 =="
DEV_BIN="$WORK/dev-bin"
DEV_LOG="$WORK/dev.log"
DEV_OVERLAY="$WORK/overlay.json"
DEV_SECRETS="$(cd "$(mktemp -d "$WORK/dev-secrets.XXXXXX")" && pwd -P)"
mkdir -p "$DEV_BIN"
: >"$DEV_LOG"

cat >"$DEV_BIN/docker" <<'FAKE_DOCKER'
#!/usr/bin/env bash
set -euo pipefail
printf 'docker %s\n' "$*" >>"$JIMU_TEST_LOG"
if [ "${1:-}" = "volume" ] && [ "${2:-}" = "inspect" ]; then
  printf 'Error response from daemon: No such volume: %s\n' "${3:-}" >&2
  exit 1
fi
if [ "${1:-}" = "compose" ]; then
  exit 0
fi
printf 'unexpected docker invocation: %s\n' "$*" >&2
exit 1
FAKE_DOCKER

cat >"$DEV_BIN/go" <<'FAKE_GO'
#!/usr/bin/env bash
set -euo pipefail
printf 'go %s\n' "$*" >>"$JIMU_TEST_LOG"
case "$*" in
  "run ./tools/profileoverlay full")
    printf '%s\n' "$JIMU_TEST_OVERLAY"
    ;;
  "run -overlay=$JIMU_TEST_OVERLAY ./cmd/cli migrate up")
    ;;
  "run -overlay=$JIMU_TEST_OVERLAY ./cmd/cli seed")
    ;;
  *)
    printf 'unexpected go invocation: %s\n' "$*" >&2
    exit 1
    ;;
esac
FAKE_GO

cat >"$DEV_BIN/air" <<'FAKE_AIR'
#!/usr/bin/env bash
set -euo pipefail
printf 'air-env APP_ENV=%s DB_HOST=%s DB_PORT=%s REDIS_ADDR=%s DB_PASSWORD_FILE=%s JWT_SECRET_FILE=%s ENCRYPTION_KEY_FILE=%s AUDIT_HASH_SECRET_FILE=%s ADMIN_PASSWORD_FILE=%s JIMU_OVERLAY=%s JIMU_SECRETS_DIR=%s\n' \
  "${APP_ENV-}" "${DB_HOST-}" "${DB_PORT-}" "${REDIS_ADDR-}" "${DB_PASSWORD_FILE-}" "${JWT_SECRET_FILE-}" \
  "${ENCRYPTION_KEY_FILE-}" "${AUDIT_HASH_SECRET_FILE-}" "${ADMIN_PASSWORD_FILE-}" \
  "${JIMU_OVERLAY-}" "${JIMU_SECRETS_DIR-}" >>"$JIMU_TEST_LOG"
FAKE_AIR

chmod +x "$DEV_BIN/docker" "$DEV_BIN/go" "$DEV_BIN/air"
env -u LEGACY_ENV_FILE -u APP_ENV \
  PATH="$DEV_BIN:$PATH" \
  ENV=dev \
  PROFILE=full \
  JIMU_SECRETS_DIR="$DEV_SECRETS" \
  JIMU_DB_HOST_PORT=13306 \
  JIMU_REDIS_HOST_PORT=16379 \
  JIMU_TEST_LOG="$DEV_LOG" \
  JIMU_TEST_OVERLAY="$DEV_OVERLAY" \
  "$ROOT/scripts/dev.sh" >"$WORK/dev.out" 2>&1 \
  || fail "dev.sh 执行失败：$(tail -n 5 "$WORK/dev.out")"

grep -Fq 'docker volume inspect jimu-mariadb-data' "$DEV_LOG" \
  || fail "dev.sh 未通过初始化脚本探测 Secret 状态"
grep -Fq 'docker compose --env-file /dev/null -f docker-compose.yml up -d --wait mariadb redis' "$DEV_LOG" \
  || fail "dev.sh 未等待 mariadb/redis 健康"
grep -Fq 'go run ./tools/profileoverlay full' "$DEV_LOG" \
  || fail "dev.sh 未生成 PROFILE overlay"
grep -Fq "go run -overlay=$DEV_OVERLAY ./cmd/cli migrate up" "$DEV_LOG" \
  || fail "dev.sh 未执行 CLI migrate up"
grep -Fq "go run -overlay=$DEV_OVERLAY ./cmd/cli seed" "$DEV_LOG" \
  || fail "dev.sh 未执行 CLI seed"
grep -Fq "air-env APP_ENV=dev DB_HOST=127.0.0.1 DB_PORT=13306 REDIS_ADDR=127.0.0.1:16379 DB_PASSWORD_FILE=$DEV_SECRETS/db_password.txt JWT_SECRET_FILE=$DEV_SECRETS/jwt_secret.txt ENCRYPTION_KEY_FILE=$DEV_SECRETS/encryption_key.txt AUDIT_HASH_SECRET_FILE=$DEV_SECRETS/audit_hash_secret.txt ADMIN_PASSWORD_FILE=$DEV_SECRETS/admin_password.txt JIMU_OVERLAY=$DEV_OVERLAY JIMU_SECRETS_DIR=$DEV_SECRETS" "$DEV_LOG" \
  || fail "air 未收到完整的本地开发环境"

line_number() {
  awk -v needle="$1" 'index($0, needle) { print NR; exit }' "$DEV_LOG"
}
compose_line="$(line_number 'up -d --wait mariadb redis')"
overlay_line="$(line_number 'go run ./tools/profileoverlay full')"
migrate_line="$(line_number 'cmd/cli migrate up')"
seed_line="$(line_number 'cmd/cli seed')"
air_line="$(line_number 'air-env ')"
[ "$compose_line" -lt "$overlay_line" ] || fail "dev.sh 必须先启动依赖再生成 overlay"
[ "$overlay_line" -lt "$migrate_line" ] || fail "dev.sh 必须先生成 overlay 再迁移"
[ "$migrate_line" -lt "$seed_line" ] || fail "dev.sh 必须先迁移再 seed"
[ "$seed_line" -lt "$air_line" ] || fail "dev.sh 必须完成 seed 后才执行 air"

echo "✅ Makefile contract passed"
