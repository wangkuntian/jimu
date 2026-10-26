#!/usr/bin/env bash
# 隔离 Compose 运行时验证：先检查管理端与安全边界，再跑认证/租户 API 契约。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# 端口用专用变量：调用方环境可能设置 HTTP_HOST_PORT=8080 并泄漏进来，
# 沿用同名变量会与本机正在运行的栈抢端口，隔离检查因而失去意义。
HTTP_HOST_PORT=${CHECK_HTTP_HOST_PORT:-18081}
MANAGEMENT_HOST_PORT=${CHECK_MANAGEMENT_HOST_PORT:-19091}
BASE_URL="http://127.0.0.1:${HTTP_HOST_PORT}/api/v1"
PROJECT_NAME="jimu-api-${RANDOM}${RANDOM}"
JIMU_IMAGE="${PROJECT_NAME}:latest"
WORKDIR=$(mktemp -d)
SECRET_DIR="$WORKDIR/secrets"

# 显式覆盖所有运行时输入，避免调用方环境把隔离检查接到本地数据卷或镜像上。
compose() {
  APP_ENV=dev \
  PROFILE=full \
  JIMU_SECRETS_DIR="$SECRET_DIR" \
  JIMU_IMAGE="$JIMU_IMAGE" \
  COMPOSE_MARIADB_VOLUME="${PROJECT_NAME}-mariadb" \
  COMPOSE_REDIS_VOLUME="${PROJECT_NAME}-redis" \
  JIMU_DB_HOST_PORT=0 \
  JIMU_REDIS_HOST_PORT=0 \
  HTTP_HOST_PORT="$HTTP_HOST_PORT" \
  MANAGEMENT_HOST_PORT="$MANAGEMENT_HOST_PORT" \
  docker compose --project-name "$PROJECT_NAME" --env-file /dev/null \
    -f "$ROOT_DIR/docker-compose.yml" "$@"
}

cleanup() {
  compose down --volumes --remove-orphans >/dev/null 2>&1 || true
  docker image rm "$JIMU_IMAGE" >/dev/null 2>&1 || true
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

mkdir -p "$SECRET_DIR"
chmod 0700 "$SECRET_DIR"
printf '%s' 'jimu-api-root-password' > "$SECRET_DIR/db_root_password.txt"
printf '%s' 'jimu-api-db-password' > "$SECRET_DIR/db_password.txt"
printf '%s' 'jimu-api-jwt-secret-012345678901234567890123456789' > "$SECRET_DIR/jwt_secret.txt"
printf '%s' '01234567890123456789012345678901' > "$SECRET_DIR/encryption_key.txt"
printf '%s' 'jimu-api-audit-secret-012345678901234567890123456789' > "$SECRET_DIR/audit_hash_secret.txt"
printf '%s' 'smoke-admin1-pass' > "$SECRET_DIR/admin_password.txt"

compose config --quiet
compose up -d --build

for endpoint in /livez /readyz; do
  ok=0
  for _ in $(seq 1 60); do
    if curl -fsS "http://127.0.0.1:${MANAGEMENT_HOST_PORT}${endpoint}" >/dev/null; then
      ok=1
      break
    fi
    sleep 2
  done
  if [ "$ok" != "1" ]; then
    echo "management endpoint failed: ${endpoint}" >&2
    exit 1
  fi
done

if curl -fsS "http://127.0.0.1:${HTTP_HOST_PORT}/debug/pprof/" >/dev/null 2>&1; then
  echo "public pprof endpoint is exposed" >&2
  exit 1
fi

login() {
  local username="$1" password="$2" response
  for _ in $(seq 1 60); do
    if response="$(curl -fsS "${BASE_URL}/auth/login" -X POST \
      -H "Content-Type: application/json" \
      -d "{\"username\":\"${username}\",\"password\":\"${password}\"}" 2>/dev/null)"; then
      printf '%s' "$response"
      return 0
    fi
    sleep 2
  done
  return 1
}

registered=0
for _ in $(seq 1 60); do
  status=$(curl -sS -o /tmp/jimu-register.json -w "%{http_code}" "${BASE_URL}/auth/register" -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"smoke_user","password":"secret123"}' || true)
  if [ "$status" = "200" ]; then
    grep -q '"code":0' /tmp/jimu-register.json
    registered=1
    break
  fi
  if [ "$status" = "409" ]; then
    grep -q '"code":2002' /tmp/jimu-register.json
    registered=1
    break
  fi
  sleep 2
done
if [ "$registered" != "1" ]; then
  cat /tmp/jimu-register.json
  exit 1
fi

LOGIN="$(login smoke_user secret123)" || {
  echo "smoke user login failed" >&2
  exit 1
}
ACCESS=$(printf "%s" "$LOGIN" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
REFRESH=$(printf "%s" "$LOGIN" | sed -n 's/.*"refresh_token":"\([^"]*\)".*/\1/p')
test -n "$ACCESS"
test -n "$REFRESH"

curl -fsS "${BASE_URL}/auth/refresh" -X POST -H "Content-Type: application/json" -d "{\"refresh_token\":\"${REFRESH}\"}" >/tmp/jimu-refresh.json

missing_status=$(curl -sS -o /tmp/jimu-logout-missing-token.json -w "%{http_code}" "${BASE_URL}/auth/logout" -X POST || true)
if [ "$missing_status" != "401" ] || ! grep -q '"code":1002' /tmp/jimu-logout-missing-token.json; then
  cat /tmp/jimu-logout-missing-token.json
  exit 1
fi

curl -fsS "${BASE_URL}/auth/logout" -X POST -H "Authorization: Bearer ${ACCESS}" >/tmp/jimu-logout.json

grep -q '"code":0' /tmp/jimu-refresh.json
grep -q '"code":0' /tmp/jimu-logout.json

# ===== 联动链路：认证开关 → 会话引导 → 租户筛选/创建 → 套餐分配 → 用量 =====
ADMIN="$(login admin smoke-admin1-pass)" || {
  echo "admin login failed" >&2
  exit 1
}
ADMIN_ACCESS=$(printf "%s" "$ADMIN" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
test -n "$ADMIN_ACCESS"

# 1) 认证开关对客户端可见（dev 配置：允许公开注册、不开通式建租户）
curl -fsS "${BASE_URL}/auth/config" >/tmp/jimu-auth-config.json
grep -q '"public_registration":true' /tmp/jimu-auth-config.json
grep -q '"provisioning_enabled":false' /tmp/jimu-auth-config.json

# 2) 管理员会话引导：非空角色 + 非空权限点，且含租户列表读权限
curl -fsS -H "Authorization: Bearer ${ADMIN_ACCESS}" "${BASE_URL}/auth/me" >/tmp/jimu-me.json
grep -q '"code":0' /tmp/jimu-me.json
grep -q '"roles":\["' /tmp/jimu-me.json
grep -q '"resource":"/api/v1/tenants","action":"GET"' /tmp/jimu-me.json

# 3) 零角色注册用户的会话引导是 200 + 空数组（证明 /auth/me 不在权限中间件下）
smoke_me_status=$(curl -sS -o /tmp/jimu-me-smoke.json -w "%{http_code}" \
  -H "Authorization: Bearer ${ACCESS}" "${BASE_URL}/auth/me" || true)
test "$smoke_me_status" = "200"
grep -q '"roles":\[\]' /tmp/jimu-me-smoke.json
grep -q '"permissions":\[\]' /tmp/jimu-me-smoke.json

# 4) 同一注册用户调管理端点必须 403（与第 3 步成对照：有身份但无角色）
smoke_tenants_status=$(curl -sS -o /tmp/jimu-tenants-smoke.json -w "%{http_code}" \
  -H "Authorization: Bearer ${ACCESS}" "${BASE_URL}/tenants" || true)
test "$smoke_tenants_status" = "403"

# 5) 租户列表筛选：keyword 命中 code + status 生效
curl -fsS -H "Authorization: Bearer ${ADMIN_ACCESS}" "${BASE_URL}/tenants?keyword=default&status=1" >/tmp/jimu-tenants.json
grep -q '"code":0' /tmp/jimu-tenants.json
grep -q '"code":"default"' /tmp/jimu-tenants.json

# 筛选真的生效：不存在的关键字必须 0 条（不假设库里只有默认租户一行）
curl -fsS -H "Authorization: Bearer ${ADMIN_ACCESS}" "${BASE_URL}/tenants?keyword=__no_such_tenant__" >/tmp/jimu-tenants-none.json
grep -q '"code":0' /tmp/jimu-tenants-none.json
grep -q '"total":0' /tmp/jimu-tenants-none.json

# 6) 建租户 → 取套餐 → 分配套餐 → 用量
curl -fsS -H "Authorization: Bearer ${ADMIN_ACCESS}" -H "Content-Type: application/json" \
  -X POST "${BASE_URL}/tenants" -d '{"code":"smoke-tenant","name":"Smoke Tenant"}' >/tmp/jimu-tenant-create.json
grep -q '"code":0' /tmp/jimu-tenant-create.json
TENANT_ID=$(sed -n 's/.*"id":"\([^"]*\)".*/\1/p' /tmp/jimu-tenant-create.json | head -1)
test -n "$TENANT_ID"

curl -fsS -H "Authorization: Bearer ${ADMIN_ACCESS}" "${BASE_URL}/tenant-plans?page_size=1" >/tmp/jimu-plans.json
PLAN_ID=$(sed -n 's/.*"id":\([0-9]*\).*/\1/p' /tmp/jimu-plans.json | head -1)
test -n "$PLAN_ID"

curl -fsS -H "Authorization: Bearer ${ADMIN_ACCESS}" -H "Content-Type: application/json" \
  -X PUT "${BASE_URL}/tenants/${TENANT_ID}/plan" -d "{\"plan_id\":${PLAN_ID}}" >/tmp/jimu-plan-assign.json
grep -q '"code":0' /tmp/jimu-plan-assign.json

curl -fsS -H "Authorization: Bearer ${ADMIN_ACCESS}" "${BASE_URL}/tenants/usage" >/tmp/jimu-usage.json
grep -q '"code":0' /tmp/jimu-usage.json
