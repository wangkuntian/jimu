#!/usr/bin/env bash
# Config contract checks: Compose wiring, observability lifecycle, and deployment manifests.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKDIRS=()

cleanup() {
  local dir
  for dir in "${WORKDIRS[@]}"; do
    [ -n "$dir" ] || continue
    rm -rf "$dir"
  done
}
trap cleanup EXIT

allocate_workdir() {
  local target="$1" dir
  dir="$(mktemp -d)"
  WORKDIRS+=("$dir")
  printf -v "$target" '%s' "$dir"
}

run_compose_contract() {
  local workdir core_secrets all_secrets core_json all_json poison_json
  local secrets_without_obs_env poison_dir
  allocate_workdir workdir
  core_secrets="$workdir/core-secrets"
  all_secrets="$workdir/all-secrets"
  core_json="$workdir/core.json"
  all_json="$workdir/all.json"
  poison_json="$workdir/poison.json"

  compose_core() {
    env -u JIMU_IMAGE \
      JIMU_SECRETS_DIR="$core_secrets" \
      docker compose --env-file /dev/null -f "$ROOT_DIR/docker-compose.yml" "$@"
  }
  compose_all() {
    env -u JIMU_IMAGE \
      JIMU_SECRETS_DIR="$all_secrets" \
      docker compose --env-file /dev/null \
        -f "$ROOT_DIR/docker-compose.yml" \
        -f "$ROOT_DIR/compose.observability.yml" "$@"
  }

  umask 077
  mkdir -p "$core_secrets" "$all_secrets"
  printf '%s' 'compose-root-password' >"$core_secrets/db_root_password.txt"
  printf '%s' 'compose-db-password' >"$core_secrets/db_password.txt"
  printf '%s' 'compose-jwt-secret-012345678901234567890123456789' >"$core_secrets/jwt_secret.txt"
  printf '%s' '01234567890123456789012345678901' >"$core_secrets/encryption_key.txt"
  printf '%s' 'compose-audit-secret-012345678901234567890123456789' >"$core_secrets/audit_hash_secret.txt"
  printf '%s' 'compose-admin-password' >"$core_secrets/admin_password.txt"
  cp "$core_secrets"/*.txt "$all_secrets/"
  printf '%s' 'compose-otel-password' >"$all_secrets/otel_auth_password.txt"
  printf '%s' 'b3BzQGV4YW1wbGUuY29tOmNvbXBvc2Utb3RlbC1wYXNzd29yZA==' >"$all_secrets/zo_auth_token.txt"
  cat >"$all_secrets/openobserve.env" <<'ENV'
ZO_ROOT_USER_EMAIL='ops@example.com'
ZO_ROOT_USER_PASSWORD='compose-otel-password'
ENV

  compose_core config --quiet
  compose_core config --format json >"$core_json"
  python3 - "$core_json" <<'PY'
import json
import sys

config = json.load(open(sys.argv[1], encoding="utf-8"))
services = config["services"]
assert "jimu-init" in services
assert services["server"]["depends_on"]["jimu-init"]["condition"] == "service_completed_successfully"
assert "openobserve" not in services
assert "otel-collector" not in services
assert set(config["secrets"]) == {
    "db_root_password", "db_password", "jwt_secret",
    "encryption_key", "audit_hash_secret", "admin_password",
}
serialized = json.dumps(config)
assert "/run/secrets/otel_auth_password" not in serialized
assert "/run/secrets/zo_auth_token" not in serialized
assert "openobserve.env" not in serialized

fixed_environment = {
    "DB_HOST": "mariadb", "DB_PORT": "3306", "DB_USER": "jimu",
    "DB_NAME": "jimu", "REDIS_ADDR": "redis:6379", "REDIS_DB": "0",
}
for service in ("jimu-init", "server"):
    environment = services[service]["environment"]
    for key, value in fixed_environment.items():
        assert environment[key] == value, f"{service} {key}"
    assert environment["ENCRYPTION_KEY_FILE"] == "/run/secrets/encryption_key"
    assert environment["AUDIT_HASH_SECRET_FILE"] == "/run/secrets/audit_hash_secret"
    assert environment["ADMIN_PASSWORD_FILE"] == "/run/secrets/admin_password"

assert services["jimu-init"]["restart"] == "no"
assert services["jimu-init"]["command"] == [
    "/bin/sh", "-c", "./jimu migrate up && ./jimu seed",
]
for dependency in ("mariadb", "redis"):
    assert services["jimu-init"]["depends_on"][dependency]["condition"] == "service_healthy"
for service, target in (("mariadb", 3306), ("redis", 6379)):
    assert any(
        port["host_ip"] == "127.0.0.1" and port["target"] == target
        for port in services[service]["ports"]
    ), f"{service} loopback port"
PY

  compose_all config --quiet
  compose_all config --format json >"$all_json"
  python3 - "$all_json" "$all_secrets/zo_auth_token.txt" <<'PY'
import base64
import json
import sys

services = json.load(open(sys.argv[1], encoding="utf-8"))["services"]
assert "openobserve" in services
assert "otel-collector" in services
assert services["server"]["environment"]["OTEL_ENABLED"] == "true"
assert services["server"]["environment"]["OTEL_ENDPOINT"] == "openobserve:5081"
assert services["server"]["environment"]["ZO_ROOT_USER_EMAIL"] == "ops@example.com"
assert services["server"]["environment"]["ZO_ROOT_USER_PASSWORD"] == "compose-otel-password"
assert services["openobserve"]["environment"]["ZO_ROOT_USER_PASSWORD"] == "compose-otel-password"
assert "Admin@12345" not in services["openobserve"]["environment"].values()
assert services["openobserve"].get("profiles") is None
assert services["otel-collector"].get("profiles") is None
for service in ("openobserve", "otel-collector"):
    assert services[service]["image"] != "poison:latest"

server_sources = {item["source"] for item in services["server"]["secrets"]}
assert {
    "db_password", "jwt_secret", "encryption_key",
    "audit_hash_secret", "admin_password",
}.issubset(server_sources)
collector_sources = {item["source"] for item in services["otel-collector"]["secrets"]}
assert {"db_root_password", "zo_auth_token"}.issubset(collector_sources)
token = base64.b64decode(open(sys.argv[2], encoding="utf-8").read(), validate=True).decode()
assert token == "ops@example.com:compose-otel-password"
PY

  secrets_without_obs_env="$workdir/secrets-without-obs-env"
  cp -R "$all_secrets" "$secrets_without_obs_env"
  rm "$secrets_without_obs_env/openobserve.env"
  env -u JIMU_IMAGE JIMU_SECRETS_DIR="$secrets_without_obs_env" \
    docker compose --env-file /dev/null \
      -f "$ROOT_DIR/docker-compose.yml" -f "$ROOT_DIR/compose.observability.yml" config --quiet

  poison_dir="$workdir/poison"
  mkdir -p "$poison_dir"
  cp "$ROOT_DIR/docker-compose.yml" "$ROOT_DIR/compose.observability.yml" "$poison_dir/"
  printf 'JIMU_IMAGE=poison:latest\n' >"$poison_dir/.env"
  (
    cd "$poison_dir"
    env -u JIMU_IMAGE JIMU_SECRETS_DIR="$all_secrets" \
      docker compose --env-file /dev/null \
        -f docker-compose.yml -f compose.observability.yml config --format json >"$poison_json"
  )
  python3 - "$poison_json" <<'PY'
import json
import sys

services = json.load(open(sys.argv[1], encoding="utf-8"))["services"]
for name in ("server", "jimu-init", "openobserve", "otel-collector"):
    assert services[name]["image"] != "poison:latest", f"{name} loaded JIMU_IMAGE from .env"
PY
  echo "compose contract passed"
}

run_observability_contract() {
  local workdir project_dir secrets_dir docker_log init_log hook_log
  allocate_workdir workdir
  project_dir="$workdir/project"
  secrets_dir="$project_dir/secrets"
  docker_log="$workdir/docker.log"
  init_log="$workdir/init.log"
  hook_log="$workdir/hook.log"
  mkdir -p "$workdir/bin" "$project_dir/scripts" "$project_dir/deploy/openobserve"
  cp "$ROOT_DIR/scripts/observability.sh" "$project_dir/scripts/observability.sh"

  cat >"$workdir/bin/docker" <<'DOCKER'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$DOCKER_LOG"
DOCKER
  chmod +x "$workdir/bin/docker"

  cat >"$project_dir/scripts/init-secrets.sh" <<'INIT'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$INIT_LOG"
mkdir -p "$SECRETS_DIR"
printf '%s' 'test-password' >"$SECRETS_DIR/otel_auth_password.txt"
printf '%s\n' "ZO_ROOT_USER_EMAIL='ops@example.com'" \
  "ZO_ROOT_USER_PASSWORD='test-password'" >"$SECRETS_DIR/openobserve.env"
INIT
  chmod +x "$project_dir/scripts/init-secrets.sh"
  cat >"$project_dir/deploy/openobserve/init-dashboard.sh" <<'HOOK'
#!/usr/bin/env bash
set -euo pipefail
printf 'dashboard:%s:%s\n' "$ZO_EMAIL" "$ZO_PASSWORD" >>"$HOOK_LOG"
HOOK
  cat >"$project_dir/deploy/openobserve/sync-alerts.sh" <<'HOOK'
#!/usr/bin/env bash
set -euo pipefail
printf 'alerts:%s:%s\n' "$ZO_EMAIL" "$ZO_PASSWORD" >>"$HOOK_LOG"
HOOK
  chmod +x "$project_dir"/deploy/openobserve/*.sh

  run_observability() {
    PATH="$workdir/bin:$PATH" JIMU_SECRETS_DIR="$secrets_dir" \
      DOCKER_LOG="$docker_log" INIT_LOG="$init_log" HOOK_LOG="$hook_log" \
      bash "$project_dir/scripts/observability.sh" "$@"
  }

  : >"$docker_log"
  run_observability status
  grep -Fq "compose --env-file /dev/null -f $project_dir/docker-compose.yml -f $project_dir/compose.observability.yml ps openobserve otel-collector" "$docker_log"
  [ ! -e "$secrets_dir/otel_auth_password.txt" ]

  : >"$docker_log"
  run_observability stop
  grep -Fq "compose --env-file /dev/null -f $project_dir/docker-compose.yml -f $project_dir/compose.observability.yml down openobserve otel-collector" "$docker_log"
  [ ! -e "$secrets_dir/otel_auth_password.txt" ]

  run_observability start
  [ "$(cat "$init_log")" = "--observability --no-legacy" ]
  grep -Fq "compose --env-file /dev/null -f $project_dir/docker-compose.yml -f $project_dir/compose.observability.yml up -d openobserve otel-collector" "$docker_log"
  grep -Fq 'dashboard:ops@example.com:test-password' "$hook_log"
  grep -Fq 'alerts:ops@example.com:test-password' "$hook_log"
  echo "observability script contract passed"
}

run_deployment_contract() {
  local helm_chart
  helm_chart="$ROOT_DIR/deploy/helm"
  fail() { echo "❌ $*" >&2; exit 1; }
  assert_contains() {
    local file="$1" needle="$2" label="$3"
    grep -Fq -- "$needle" "$file" || fail "${label}（未找到：${needle}）"
  }
  assert_not_contains() {
    local text="$1" needle="$2" label="$3"
    if grep -Fq -- "$needle" <<<"$text"; then
      fail "${label}（意外找到：${needle}）"
    fi
  }

  echo "== OTel Collector token 挂载 =="
  for file in "$helm_chart/templates/otel-collector.yaml" "$ROOT_DIR/deploy/k8s/otel-collector.yaml"; do
    assert_contains "$file" "/run/secrets/zo_auth_token" "collector 配置未读取 zo_auth_token"
    assert_contains "$file" "mountPath: /run/secrets/zo_auth_token" "collector 未挂载 zo_auth_token"
    assert_contains "$file" "subPath: zo_auth_token" "collector 挂载缺少 zo_auth_token subPath"
  done

  echo "== Helm 观测凭据条件渲染 =="
  assert_contains "$helm_chart/templates/configmap.yaml" '{{- if .Values.otel.enabled }}' "otel 密码 required 未按开关条件化"
  assert_contains "$helm_chart/templates/secret.yaml" '{{- if or .Values.otel.enabled .Values.otelCollector.enabled }}' "zo_auth_token 未按观测开关条件化"
  assert_contains "$helm_chart/templates/openobserve.yaml" '{{- if .Values.openobserve.enabled }}' "OpenObserve 资源未按开关条件化"

  if command -v helm >/dev/null 2>&1; then
    rendered="$(helm template review "$helm_chart" \
      --set otel.enabled=false \
      --set openobserve.enabled=false \
      --set otelCollector.enabled=false)"
    assert_not_contains "$rendered" "zo_auth_token:" "关闭观测后仍有 zo_auth_token Secret"
    assert_not_contains "$rendered" "ZO_ROOT_USER_PASSWORD" "关闭 OpenObserve 后仍有管理员密码环境变量"
    grep -Fq 'auth_password: ""' <<<"$rendered" \
      || fail "关闭观测后 auth_password 未渲染为空字符串"
    echo "helm disabled-mode render passed"
  else
    echo "helm 未安装：已完成模板静态断言"
  fi
  echo "deployment manifests contract passed"
}

case "${1:-all}" in
  compose) run_compose_contract ;;
  observability) run_observability_contract ;;
  deploy) run_deployment_contract ;;
  all)
    run_compose_contract
    run_observability_contract
    run_deployment_contract
    ;;
  *)
    echo "usage: $0 [all|compose|observability|deploy]" >&2
    exit 2
    ;;
esac
