#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INIT_SCRIPT="$ROOT/scripts/init-secrets.sh"
WORK="$(mktemp -d)"
FAKE_DOCKER_VOLUMES=""
FAKE_DOCKER_PROBE_EXIT=""
FAKE_DOCKER_PROBE_ERROR=""
TEST_ADMIN_PASSWORD=""
TEST_OBS_EMAIL=""
TEST_LEGACY_ENV_FILE=""
TEST_FORCE=""
TEST_ALLOW_NEW_SECURITY_KEYS=""

cleanup() {
  rm -rf "$WORK"
}
trap cleanup EXIT

fail() {
  echo "❌ $*" >&2
  exit 1
}

file_mode() {
  if stat -c %a "$1" >/dev/null 2>&1; then
    stat -c %a "$1"
  else
    stat -f %Lp "$1"
  fi
}

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

base64_decode() {
  if printf '' | base64 -d >/dev/null 2>&1; then
    base64 -d
  else
    base64 -D
  fi
}

env_value() {
  local file="$1" key="$2" line status
  [ -f "$file" ] || return 0
  if line="$(awk -v key="$key" '
    {
      line = $0
      sub(/\r$/, "", line)
      if (line ~ "^[[:space:]]*(export[[:space:]]+)?" key "[[:space:]]*=") {
        sub("^[[:space:]]*(export[[:space:]]+)?" key "[[:space:]]*=", "", line)
        count++
        value = line
      }
    }
    END {
      if (count > 1) exit 42
      if (count == 1) print value
    }
  ' "$file")"; then
    :
  else
    status=$?
    if [ "$status" = "42" ]; then
      fail "$file: $key 出现重复赋值"
    fi
    fail "$file: 解析 $key 失败"
  fi
  line="${line#"${line%%[![:space:]]*}"}"
  line="${line%"${line##*[![:space:]]}"}"
  if [ "${#line}" -ge 2 ]; then
    case "${line:0:1}${line: -1}" in
      "''"|'""') line="${line:1:${#line}-2}" ;;
    esac
  fi
  printf '%s' "$line"
}

assert_eq() {
  local expected="$1" actual="$2" label="$3"
  [ "$expected" = "$actual" ] || fail "$label: 期望 '$expected'，实际 '$actual'"
}

assert_file() {
  local path="$1"
  [ -f "$path" ] || fail "缺少文件：$path"
}

assert_mode() {
  local path="$1" expected="$2" actual
  actual="$(file_mode "$path")"
  assert_eq "$expected" "$actual" "权限 $path"
}

assert_contains() {
  local path="$1" needle="$2" label="$3"
  grep -Fq -- "$needle" "$path" || fail "${label}（未找到：${needle}）"
}

mkdir -p "$WORK/bin"
cat > "$WORK/bin/docker" <<'FAKE_DOCKER'
#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = "volume" ] && [ "${2:-}" = "inspect" ]; then
  if [ -n "${FAKE_DOCKER_PROBE_EXIT:-}" ]; then
    printf '%s\n' "${FAKE_DOCKER_PROBE_ERROR:-fake docker probe failure}" >&2
    exit "$FAKE_DOCKER_PROBE_EXIT"
  fi
  case ",${FAKE_DOCKER_VOLUMES:-}," in
    *",${3:-},"*) exit 0 ;;
  esac
  printf 'Error response from daemon: No such volume: %s\n' "${3:-}" >&2
fi
exit 1
FAKE_DOCKER
chmod 0755 "$WORK/bin/docker"

run_init() {
  local secrets_dir="$1"
  shift
  SECRETS_DIR="$secrets_dir" \
  LEGACY_ENV_FILE="$TEST_LEGACY_ENV_FILE" \
  ADMIN_PASSWORD="${TEST_ADMIN_PASSWORD:-}" \
  ZO_OBSERVE_ROOT_USER_EMAIL="${TEST_OBS_EMAIL:-}" \
  FORCE="$TEST_FORCE" \
  ALLOW_NEW_SECURITY_KEYS="$TEST_ALLOW_NEW_SECURITY_KEYS" \
  PATH="$WORK/bin:$PATH" \
  FAKE_DOCKER_VOLUMES="$FAKE_DOCKER_VOLUMES" \
  FAKE_DOCKER_PROBE_EXIT="$FAKE_DOCKER_PROBE_EXIT" \
  FAKE_DOCKER_PROBE_ERROR="$FAKE_DOCKER_PROBE_ERROR" \
  COMPOSE_MARIADB_VOLUME="jimu-mariadb-data" \
  COMPOSE_OPENOBSERVE_VOLUME="jimu-openobserve-data" \
  "$INIT_SCRIPT" "$@"
}

echo "== 默认不隐式读取仓库根 .env =="
NO_IMPLICIT_REPO="$WORK/no-implicit-repo"
mkdir -p "$NO_IMPLICIT_REPO/scripts"
cp "$INIT_SCRIPT" "$NO_IMPLICIT_REPO/scripts/init-secrets.sh"
cat > "$NO_IMPLICIT_REPO/.env" <<'ENV'
DB_ROOT_PASSWORD=implicit-root-sentinel-1234567890
DB_PASSWORD=implicit-db-sentinel-12345678901
JWT_SECRET=implicit-jwt-sentinel-12345678901234567890123456789012
ADMIN_PASSWORD=implicit-admin-sentinel
ENV
if env -u LEGACY_ENV_FILE \
  SECRETS_DIR="$WORK/no-implicit-secrets" \
  ADMIN_PASSWORD="" \
  FORCE="" \
  PATH="$WORK/bin:$PATH" \
  FAKE_DOCKER_VOLUMES="" \
  FAKE_DOCKER_PROBE_EXIT="" \
  COMPOSE_MARIADB_VOLUME="jimu-mariadb-data" \
  COMPOSE_OPENOBSERVE_VOLUME="jimu-openobserve-data" \
  bash "$NO_IMPLICIT_REPO/scripts/init-secrets.sh" </dev/null >"$WORK/no-implicit.out" 2>&1; then
  :
else
  fail "未设置 LEGACY_ENV_FILE 时初始化不应失败"
fi
if grep -Fq "已停用" "$WORK/no-implicit.out"; then
  fail "未设置 LEGACY_ENV_FILE 时不得读取仓库根 .env"
fi
for pair in \
  "db_root_password.txt:implicit-root-sentinel-1234567890" \
  "db_password.txt:implicit-db-sentinel-12345678901" \
  "jwt_secret.txt:implicit-jwt-sentinel-12345678901234567890123456789012" \
  "admin_password.txt:implicit-admin-sentinel"; do
  file="${pair%%:*}"
  sentinel="${pair#*:}"
  [ "$(cat "$WORK/no-implicit-secrets/$file")" != "$sentinel" ] \
    || fail "未设置 LEGACY_ENV_FILE 时迁移了 $file"
done

echo "== 核心 secret、权限与幂等 =="
CORE_DIR="$WORK/core"
run_init "$CORE_DIR" </dev/null
assert_mode "$CORE_DIR" 700
for file in db_root_password.txt db_password.txt jwt_secret.txt \
            encryption_key.txt audit_hash_secret.txt admin_password.txt; do
  assert_file "$CORE_DIR/$file"
  assert_mode "$CORE_DIR/$file" 600
done
[ "$(wc -c < "$CORE_DIR/encryption_key.txt" | tr -d ' ')" = "32" ] || fail "encryption_key 必须恰好 32 字节"
LC_ALL=C grep -Eq '^[ -~]{32}$' "$CORE_DIR/encryption_key.txt" || fail "encryption_key 必须是 32 个可打印 ASCII 字符"
[ "$(wc -c < "$CORE_DIR/jwt_secret.txt" | tr -d ' ')" -ge 32 ] || fail "jwt_secret 少于 32 字节"
[ "$(wc -c < "$CORE_DIR/db_root_password.txt" | tr -d ' ')" -ge 24 ] || fail "db_root_password 少于 24 字符"
[ "$(wc -c < "$CORE_DIR/db_password.txt" | tr -d ' ')" -ge 24 ] || fail "db_password 少于 24 字符"
[ "$(wc -c < "$CORE_DIR/audit_hash_secret.txt" | tr -d ' ')" -ge 32 ] || fail "audit_hash_secret 少于 32 字节"
first_jwt="$(sha256 "$CORE_DIR/jwt_secret.txt")"
first_admin="$(cat "$CORE_DIR/admin_password.txt")"
run_init "$CORE_DIR" </dev/null
assert_eq "$first_jwt" "$(sha256 "$CORE_DIR/jwt_secret.txt")" "重复初始化不得覆盖 jwt_secret"
assert_eq "$first_admin" "$(cat "$CORE_DIR/admin_password.txt")" "重复初始化不得覆盖 admin_password"
assert_eq "" "$(find "$CORE_DIR" -maxdepth 1 -name '.*.tmp.*' -print -quit)" "成功初始化后不得残留临时文件"

echo "== 既有 encryption_key 校验 =="
ENCRYPTION_31="$(printf 'x%.0s' {1..31})"
ENCRYPTION_32='abcdefghijklmnopqrstuvwxyz012345'
ENCRYPTION_33="$(printf 'y%.0s' {1..33})"
for spec in "31:$ENCRYPTION_31" "33:$ENCRYPTION_33"; do
  label="${spec%%:*}"
  value="${spec#*:}"
  dir="$WORK/encryption-$label"
  mkdir -p "$dir"
  printf '%s' "$value" > "$dir/encryption_key.txt"
  chmod 0600 "$dir/encryption_key.txt"
  if run_init "$dir" </dev/null >"$WORK/encryption-$label.out" 2>&1; then
    fail "${label} 字节 encryption_key 必须失败"
  fi
  assert_contains "$WORK/encryption-$label.out" "encryption_key" "${label} 字节 encryption_key 错误提示"
  assert_eq "$value" "$(cat "$dir/encryption_key.txt")" "${label} 字节 encryption_key 不得被覆盖"
done

NEWLINE_DIR="$WORK/encryption-newline"
mkdir -p "$NEWLINE_DIR"
printf '%s\n' "$(printf 'z%.0s' {1..31})" > "$NEWLINE_DIR/encryption_key.txt"
chmod 0600 "$NEWLINE_DIR/encryption_key.txt"
if run_init "$NEWLINE_DIR" </dev/null >"$WORK/encryption-newline.out" 2>&1; then
  fail "带换行的 32 字节 encryption_key 必须失败"
fi
assert_contains "$WORK/encryption-newline.out" "encryption_key" "带换行 encryption_key 错误提示"

VALID_ENCRYPTION_DIR="$WORK/encryption-valid"
mkdir -p "$VALID_ENCRYPTION_DIR"
printf '%s' "$ENCRYPTION_32" > "$VALID_ENCRYPTION_DIR/encryption_key.txt"
chmod 0600 "$VALID_ENCRYPTION_DIR/encryption_key.txt"
valid_encryption_hash="$(sha256 "$VALID_ENCRYPTION_DIR/encryption_key.txt")"
run_init "$VALID_ENCRYPTION_DIR" </dev/null
assert_eq "$valid_encryption_hash" "$(sha256 "$VALID_ENCRYPTION_DIR/encryption_key.txt")" "有效 encryption_key 不得被覆盖"

echo "== 非 TTY 随机管理员密码与显式 ADMIN_PASSWORD =="
run_init "$WORK/admin-random-a" </dev/null
run_init "$WORK/admin-random-b" </dev/null
admin_a="$(cat "$WORK/admin-random-a/admin_password.txt")"
admin_b="$(cat "$WORK/admin-random-b/admin_password.txt")"
[ -n "$admin_a" ] || fail "非 TTY 未生成 admin_password"
[ "$admin_a" != "$admin_b" ] || fail "非 TTY admin_password 应为随机值"
[ "${#admin_a}" -eq 32 ] || fail "默认 admin_password 必须为 32 位，实际 ${#admin_a}"
[[ "$admin_a" =~ [A-Za-z] ]] || fail "默认 admin_password 缺少字母"
[[ "$admin_a" =~ [0-9] ]] || fail "默认 admin_password 缺少数字"
TEST_ADMIN_PASSWORD="explicit-admin1-password"
run_init "$WORK/admin-explicit" </dev/null
assert_eq "explicit-admin1-password" "$(cat "$WORK/admin-explicit/admin_password.txt")" "ADMIN_PASSWORD 非交互输入"
TEST_ADMIN_PASSWORD=""

for spec in "too-short|Abc1234" "too-long|A1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" "no-letter|12345678" "no-digit|abcdefgh"; do
  label="${spec%%|*}"
  password="${spec#*|}"
  TEST_ADMIN_PASSWORD="$password"
  if run_init "$WORK/admin-invalid-$label" </dev/null >"$WORK/admin-$label.out" 2>&1; then
    fail "ADMIN_PASSWORD $label 应被拒绝"
  fi
  assert_contains "$WORK/admin-$label.out" "管理员密码" "ADMIN_PASSWORD $label 错误"
done
TEST_ADMIN_PASSWORD=""

INVALID_EXISTING_DIR="$WORK/admin-invalid-existing"
mkdir -p "$INVALID_EXISTING_DIR"
printf '%s' 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' > "$INVALID_EXISTING_DIR/admin_password.txt"
if run_init "$INVALID_EXISTING_DIR" </dev/null >"$WORK/admin-invalid-existing.out" 2>&1; then
  fail "已有的无效 admin_password 应被拒绝"
fi
assert_contains "$WORK/admin-invalid-existing.out" "管理员密码" "已有 admin_password 错误"

echo "== OpenObserve 派生文件与一致性校验 =="
OBS_DIR="$WORK/observability"
run_init "$OBS_DIR" --observability </dev/null
for file in otel_auth_password.txt openobserve.env zo_auth_token.txt; do
  assert_file "$OBS_DIR/$file"
  assert_mode "$OBS_DIR/$file" 600
done
obs_email="$(env_value "$OBS_DIR/openobserve.env" ZO_ROOT_USER_EMAIL)"
obs_password="$(env_value "$OBS_DIR/openobserve.env" ZO_ROOT_USER_PASSWORD)"
assert_eq "admin@jimu.local" "$obs_email" "OpenObserve 默认邮箱"
assert_eq "$(cat "$OBS_DIR/otel_auth_password.txt")" "$obs_password" "OpenObserve 密码源"
[ "${#obs_password}" -ge 8 ] && [ "${#obs_password}" -le 128 ] || fail "OpenObserve 密码长度不符合 8-128"
[[ "$obs_password" =~ [a-z] ]] || fail "OpenObserve 密码缺少小写字母"
[[ "$obs_password" =~ [A-Z] ]] || fail "OpenObserve 密码缺少大写字母"
[[ "$obs_password" =~ [0-9] ]] || fail "OpenObserve 密码缺少数字"
[[ "$obs_password" =~ [^a-zA-Z0-9] ]] || fail "OpenObserve 密码缺少特殊字符"
decoded_token="$(base64_decode < "$OBS_DIR/zo_auth_token.txt")"
assert_eq "$obs_email:$obs_password" "$decoded_token" "zo_auth_token 内容"
printf '%s' 'custom-token-must-survive' > "$OBS_DIR/zo_auth_token.txt"
chmod 0600 "$OBS_DIR/zo_auth_token.txt"
if run_init "$OBS_DIR" --observability </dev/null >"$WORK/obs-token-mismatch.out" 2>&1; then
  fail "不一致的 zo_auth_token 必须显式失败"
fi
assert_contains "$WORK/obs-token-mismatch.out" "zo_auth_token" "不一致 token 的错误提示"
assert_eq "custom-token-must-survive" "$(cat "$OBS_DIR/zo_auth_token.txt")" "不得静默覆盖自定义 zo_auth_token"

OBS_EMPTY_DIR="$WORK/obs-empty-env"
mkdir -p "$OBS_EMPTY_DIR"
: > "$OBS_EMPTY_DIR/openobserve.env"
chmod 0600 "$OBS_EMPTY_DIR/openobserve.env"
if run_init "$OBS_EMPTY_DIR" --observability </dev/null >"$WORK/obs-empty-env.out" 2>&1; then
  fail "空 openobserve.env 必须失败"
fi
assert_contains "$WORK/obs-empty-env.out" "openobserve.env" "空 openobserve.env 的错误提示"
assert_contains "$WORK/obs-empty-env.out" "空" "空 openobserve.env 必须明确报错"

OBS_ENV_MISMATCH_DIR="$WORK/obs-env-mismatch"
run_init "$OBS_ENV_MISMATCH_DIR" --observability </dev/null
printf '%s\n' 'ZO_ROOT_USER_EMAIL=admin@jimu.local' 'ZO_ROOT_USER_PASSWORD=DifferentObs@123' > "$OBS_ENV_MISMATCH_DIR/openobserve.env"
chmod 0600 "$OBS_ENV_MISMATCH_DIR/openobserve.env"
if run_init "$OBS_ENV_MISMATCH_DIR" --observability </dev/null >"$WORK/obs-env-mismatch.out" 2>&1; then
  fail "openobserve.env 与 otel_auth_password.txt 不一致时必须失败"
fi
assert_contains "$WORK/obs-env-mismatch.out" "openobserve.env" "不一致 openobserve.env 的错误提示"

OBS_DUPLICATE_DIR="$WORK/obs-duplicate-password"
mkdir -p "$OBS_DUPLICATE_DIR"
printf '%s' 'CanonicalObs@123' > "$OBS_DUPLICATE_DIR/otel_auth_password.txt"
printf '%s\n' \
  'ZO_ROOT_USER_EMAIL=admin@jimu.local' \
  'ZO_ROOT_USER_PASSWORD=CanonicalObs@123' \
  'ZO_ROOT_USER_PASSWORD=DifferentObs@123' > "$OBS_DUPLICATE_DIR/openobserve.env"
printf '%s' 'admin@jimu.local:CanonicalObs@123' | base64 | tr -d '\n' > "$OBS_DUPLICATE_DIR/zo_auth_token.txt"
chmod 0600 "$OBS_DUPLICATE_DIR"/*.txt "$OBS_DUPLICATE_DIR/openobserve.env"
obs_duplicate_otel_hash="$(sha256 "$OBS_DUPLICATE_DIR/otel_auth_password.txt")"
obs_duplicate_token_hash="$(sha256 "$OBS_DUPLICATE_DIR/zo_auth_token.txt")"
if run_init "$OBS_DUPLICATE_DIR" --observability </dev/null >"$WORK/obs-duplicate-password.out" 2>&1; then
  fail "openobserve.env 重复 ZO_ROOT_USER_PASSWORD 必须失败"
fi
assert_contains "$WORK/obs-duplicate-password.out" "ZO_ROOT_USER_PASSWORD" "重复 OpenObserve 密码错误提示包含键名"
assert_contains "$WORK/obs-duplicate-password.out" "重复" "重复 OpenObserve 密码错误提示"
assert_eq "$obs_duplicate_otel_hash" "$(sha256 "$OBS_DUPLICATE_DIR/otel_auth_password.txt")" "重复 OpenObserve 密码不得改写密码源"
assert_eq "$obs_duplicate_token_hash" "$(sha256 "$OBS_DUPLICATE_DIR/zo_auth_token.txt")" "重复 OpenObserve 密码不得改写 token"
[ ! -f "$OBS_DUPLICATE_DIR/db_root_password.txt" ] || fail "重复 OpenObserve 密码必须在生成核心 secret 前失败"

OBS_DUPLICATE_EMAIL_DIR="$WORK/obs-duplicate-email"
mkdir -p "$OBS_DUPLICATE_EMAIL_DIR"
printf '%s' 'CanonicalObs@123' > "$OBS_DUPLICATE_EMAIL_DIR/otel_auth_password.txt"
printf '%s\n' \
  'ZO_ROOT_USER_EMAIL=first@jimu.local' \
  'ZO_ROOT_USER_EMAIL=second@jimu.local' \
  'ZO_ROOT_USER_PASSWORD=CanonicalObs@123' > "$OBS_DUPLICATE_EMAIL_DIR/openobserve.env"
chmod 0600 "$OBS_DUPLICATE_EMAIL_DIR"/*.txt "$OBS_DUPLICATE_EMAIL_DIR/openobserve.env"
obs_duplicate_email_hash="$(sha256 "$OBS_DUPLICATE_EMAIL_DIR/openobserve.env")"
if run_init "$OBS_DUPLICATE_EMAIL_DIR" --observability </dev/null >"$WORK/obs-duplicate-email.out" 2>&1; then
  fail "openobserve.env 重复 ZO_ROOT_USER_EMAIL 必须失败"
fi
assert_contains "$WORK/obs-duplicate-email.out" "ZO_ROOT_USER_EMAIL" "重复 OpenObserve 邮箱错误提示包含键名"
assert_contains "$WORK/obs-duplicate-email.out" "重复" "重复 OpenObserve 邮箱错误提示"
assert_eq "$obs_duplicate_email_hash" "$(sha256 "$OBS_DUPLICATE_EMAIL_DIR/openobserve.env")" "重复 OpenObserve 邮箱不得改写配置"
[ ! -f "$OBS_DUPLICATE_EMAIL_DIR/db_root_password.txt" ] || fail "重复 OpenObserve 邮箱必须在生成核心 secret 前失败"

echo "== 已有 MariaDB 卷保护 =="
PROBE_FAIL_DIR="$WORK/probe-fail"
FAKE_DOCKER_PROBE_EXIT="2"
FAKE_DOCKER_PROBE_ERROR="Cannot connect to the Docker daemon"
if run_init "$PROBE_FAIL_DIR" </dev/null >"$WORK/probe-fail.out" 2>&1; then
  fail "Docker 探测未知错误时必须失败"
fi
assert_contains "$WORK/probe-fail.out" "无法确认" "Docker 探测失败的明确提示"
assert_contains "$WORK/probe-fail.out" "Docker" "Docker 探测失败提示包含 Docker"
[ ! -f "$PROBE_FAIL_DIR/db_root_password.txt" ] || fail "Docker 探测失败时不得生成 DB root 密码"
[ ! -f "$PROBE_FAIL_DIR/db_password.txt" ] || fail "Docker 探测失败时不得生成 DB 密码"
FAKE_DOCKER_PROBE_EXIT=""
FAKE_DOCKER_PROBE_ERROR=""

MISSING_DB_DIR="$WORK/missing-db"
FAKE_DOCKER_VOLUMES="jimu-mariadb-data"
if run_init "$MISSING_DB_DIR" </dev/null >"$WORK/missing-db.out" 2>&1; then
  fail "MariaDB 卷存在且缺 DB secret 时必须失败"
fi
assert_contains "$WORK/missing-db.out" "数据库" "缺 DB secret 的错误提示"
[ ! -f "$MISSING_DB_DIR/db_root_password.txt" ] || fail "缺 DB secret 时不得自动生成 root 密码"

SECURITY_DIR="$WORK/missing-security"
mkdir -p "$SECURITY_DIR"
printf '%s' 'existing-root-password-1234567890' > "$SECURITY_DIR/db_root_password.txt"
printf '%s' 'existing-app-password-12345678901' > "$SECURITY_DIR/db_password.txt"
chmod 0600 "$SECURITY_DIR/db_root_password.txt" "$SECURITY_DIR/db_password.txt"
if run_init "$SECURITY_DIR" </dev/null >"$WORK/missing-security.out" 2>&1; then
  fail "已有数据卷且缺安全 secret 时，未授权迁移必须失败"
fi
assert_contains "$WORK/missing-security.out" "--allow-new-security-keys" "缺安全 secret 的错误提示"
[ ! -f "$SECURITY_DIR/encryption_key.txt" ] || fail "未授权迁移时不得生成 encryption_key"
[ ! -f "$SECURITY_DIR/admin_password.txt" ] || fail "未授权迁移时不得生成 admin_password"

run_init "$SECURITY_DIR" --allow-new-security-keys </dev/null >"$WORK/allow-security.out" 2>&1
assert_contains "$WORK/allow-security.out" "警告" "--allow-new-security-keys 必须打印警告"
assert_file "$SECURITY_DIR/jwt_secret.txt"
assert_file "$SECURITY_DIR/encryption_key.txt"
assert_file "$SECURITY_DIR/audit_hash_secret.txt"
assert_file "$SECURITY_DIR/admin_password.txt"
assert_eq "32" "$(wc -c < "$SECURITY_DIR/encryption_key.txt" | tr -d ' ')" "授权迁移后的 encryption_key 长度"

FORCE_DB_DIR="$WORK/force-existing-mariadb"
mkdir -p "$FORCE_DB_DIR"
printf '%s' 'keep-this-root-password-1234567890' > "$FORCE_DB_DIR/db_root_password.txt"
printf '%s' 'keep-this-db-password-12345678901' > "$FORCE_DB_DIR/db_password.txt"
printf '%s' "$(printf 'j%.0s' {1..64})" > "$FORCE_DB_DIR/jwt_secret.txt"
printf '%s' "$ENCRYPTION_32" > "$FORCE_DB_DIR/encryption_key.txt"
printf '%s' "$(printf 'a%.0s' {1..64})" > "$FORCE_DB_DIR/audit_hash_secret.txt"
printf '%s' 'existing-admin1-password' > "$FORCE_DB_DIR/admin_password.txt"
chmod 0600 "$FORCE_DB_DIR"/*.txt
force_root_hash="$(sha256 "$FORCE_DB_DIR/db_root_password.txt")"
force_db_hash="$(sha256 "$FORCE_DB_DIR/db_password.txt")"
force_jwt_hash="$(sha256 "$FORCE_DB_DIR/jwt_secret.txt")"
force_encryption_hash="$(sha256 "$FORCE_DB_DIR/encryption_key.txt")"
force_audit_hash="$(sha256 "$FORCE_DB_DIR/audit_hash_secret.txt")"
force_admin_hash="$(sha256 "$FORCE_DB_DIR/admin_password.txt")"
TEST_FORCE="1"
FAKE_DOCKER_VOLUMES="jimu-mariadb-data"
run_init "$FORCE_DB_DIR" </dev/null
TEST_FORCE=""
assert_eq "$force_root_hash" "$(sha256 "$FORCE_DB_DIR/db_root_password.txt")" "FORCE 不得覆盖已有卷的 DB root 密码"
assert_eq "$force_db_hash" "$(sha256 "$FORCE_DB_DIR/db_password.txt")" "FORCE 不得覆盖已有卷的 DB 应用密码"
assert_eq "$force_jwt_hash" "$(sha256 "$FORCE_DB_DIR/jwt_secret.txt")" "FORCE 未授权时不得覆盖已有卷的 JWT secret"
assert_eq "$force_encryption_hash" "$(sha256 "$FORCE_DB_DIR/encryption_key.txt")" "FORCE 未授权时不得覆盖已有卷的 encryption key"
assert_eq "$force_audit_hash" "$(sha256 "$FORCE_DB_DIR/audit_hash_secret.txt")" "FORCE 未授权时不得覆盖已有卷的 audit secret"
assert_eq "$force_admin_hash" "$(sha256 "$FORCE_DB_DIR/admin_password.txt")" "FORCE 未授权时不得覆盖已有卷的 admin password"

FORCE_PARTIAL_DIR="$WORK/force-partial-security"
mkdir -p "$FORCE_PARTIAL_DIR"
printf '%s' 'existing-root-password-1234567890' > "$FORCE_PARTIAL_DIR/db_root_password.txt"
printf '%s' 'existing-app-password-12345678901' > "$FORCE_PARTIAL_DIR/db_password.txt"
printf '%s' "$(printf 'j%.0s' {1..64})" > "$FORCE_PARTIAL_DIR/jwt_secret.txt"
printf '%s' 'existing-admin1-password' > "$FORCE_PARTIAL_DIR/admin_password.txt"
chmod 0600 "$FORCE_PARTIAL_DIR"/*.txt
partial_jwt_hash="$(sha256 "$FORCE_PARTIAL_DIR/jwt_secret.txt")"
partial_admin_hash="$(sha256 "$FORCE_PARTIAL_DIR/admin_password.txt")"
TEST_FORCE="1"
TEST_ALLOW_NEW_SECURITY_KEYS="1"
FAKE_DOCKER_VOLUMES="jimu-mariadb-data"
run_init "$FORCE_PARTIAL_DIR" </dev/null >"$WORK/force-partial-security.out" 2>&1
TEST_FORCE=""
TEST_ALLOW_NEW_SECURITY_KEYS=""
assert_eq "$partial_jwt_hash" "$(sha256 "$FORCE_PARTIAL_DIR/jwt_secret.txt")" "FORCE + 授权迁移不得覆盖已有 JWT secret"
assert_eq "$partial_admin_hash" "$(sha256 "$FORCE_PARTIAL_DIR/admin_password.txt")" "FORCE + 授权迁移不得覆盖已有 admin password"
assert_file "$FORCE_PARTIAL_DIR/encryption_key.txt"
assert_file "$FORCE_PARTIAL_DIR/audit_hash_secret.txt"
assert_contains "$WORK/force-partial-security.out" "加密字段" "授权补齐缺失安全 secret 必须提示影响"

echo "== 已有 OpenObserve 卷保护与旧密码恢复 =="
OBS_VOLUME_DIR="$WORK/obs-volume-missing"
FAKE_DOCKER_VOLUMES="jimu-openobserve-data"
if run_init "$OBS_VOLUME_DIR" --observability </dev/null >"$WORK/obs-volume-missing.out" 2>&1; then
  fail "OpenObserve 卷存在且无旧密码时必须失败"
fi
assert_contains "$WORK/obs-volume-missing.out" "OpenObserve" "缺 OpenObserve 旧密码的错误提示"
[ ! -f "$OBS_VOLUME_DIR/otel_auth_password.txt" ] || fail "缺旧密码时不得生成新的 OpenObserve 密码"

OBS_RECOVER_DIR="$WORK/obs-volume-recover"
mkdir -p "$OBS_RECOVER_DIR"
printf '%s\n' 'ZO_ROOT_USER_EMAIL=existing@jimu.local' 'ZO_ROOT_USER_PASSWORD=ExistingObs@123' > "$OBS_RECOVER_DIR/openobserve.env"
chmod 0600 "$OBS_RECOVER_DIR/openobserve.env"
run_init "$OBS_RECOVER_DIR" --observability </dev/null
assert_eq "ExistingObs@123" "$(cat "$OBS_RECOVER_DIR/otel_auth_password.txt")" "从旧 openobserve.env 恢复密码"
assert_eq "existing@jimu.local:ExistingObs@123" "$(base64_decode < "$OBS_RECOVER_DIR/zo_auth_token.txt")" "从旧邮箱和密码派生 token"

echo "== 旧 .env 仅迁移 secret 键 =="
FAKE_DOCKER_VOLUMES=""
LEGACY_DIR="$WORK/legacy"
LEGACY_FILE="$WORK/legacy.env"
cat > "$LEGACY_FILE" <<'ENV'
DB_ROOT_PASSWORD="quoted-root-password"
DB_PASSWORD=literal-password # inline-comment
JWT_SECRET='quoted-jwt-secret # literal'
ADMIN_PASSWORD="quoted-admin1-password"
ZO_OBSERVE_ROOT_USER_PASSWORD='LegacyObs@123'
ZO_OBSERVE_ROOT_USER_EMAIL='legacy-admin@example.com'
DB_HOST=conflicting-db-host-value
HTTP_HOST_PORT=19999
COMPOSE_MARIADB_VOLUME=conflicting-volume-name
ENCRYPTION_KEY=conflicting-encryption-key
AUDIT_HASH_SECRET=conflicting-audit-secret
ZO_OBSERVE_AUTH_TOKEN=conflicting-zo-token
ENV
TEST_LEGACY_ENV_FILE="$LEGACY_FILE"
run_init "$LEGACY_DIR" --observability </dev/null >"$WORK/legacy.out" 2>&1
TEST_LEGACY_ENV_FILE=""
assert_contains "$WORK/legacy.out" "已停用" "旧 .env 停用提示"
assert_file "$LEGACY_FILE"
assert_eq "quoted-root-password" "$(cat "$LEGACY_DIR/db_root_password.txt")" "迁移带引号的 DB_ROOT_PASSWORD"
assert_eq "literal-password" "$(cat "$LEGACY_DIR/db_password.txt")" "迁移带行内注释的 DB_PASSWORD"
assert_eq "quoted-jwt-secret # literal" "$(cat "$LEGACY_DIR/jwt_secret.txt")" "保留引号内的 #"
assert_eq "quoted-admin1-password" "$(cat "$LEGACY_DIR/admin_password.txt")" "迁移带引号的 ADMIN_PASSWORD"
assert_eq "LegacyObs@123" "$(cat "$LEGACY_DIR/otel_auth_password.txt")" "迁移 ZO_OBSERVE_ROOT_USER_PASSWORD"
assert_eq "LegacyObs@123" "$(env_value "$LEGACY_DIR/openobserve.env" ZO_ROOT_USER_PASSWORD)" "迁移值写入 openobserve.env"
assert_eq "legacy-admin@example.com" "$(env_value "$LEGACY_DIR/openobserve.env" ZO_ROOT_USER_EMAIL)" "迁移自定义 OpenObserve 邮箱"
assert_eq "legacy-admin@example.com:LegacyObs@123" "$(base64_decode < "$LEGACY_DIR/zo_auth_token.txt")" "自定义邮箱参与 token 派生"
if grep -R -Fq "conflicting-db-host-value" "$LEGACY_DIR"; then
  fail "DB_HOST 值不得迁入任何生成文件"
fi
if grep -R -Fq "conflicting-volume-name" "$LEGACY_DIR"; then
  fail "COMPOSE_* 值不得迁入任何生成文件"
fi
[ "$(cat "$LEGACY_DIR/encryption_key.txt")" != "conflicting-encryption-key" ] || fail "旧 .env 的 ENCRYPTION_KEY 不得迁移"
[ "$(cat "$LEGACY_DIR/audit_hash_secret.txt")" != "conflicting-audit-secret" ] || fail "旧 .env 的 AUDIT_HASH_SECRET 不得迁移"
[ "$(cat "$LEGACY_DIR/zo_auth_token.txt")" != "conflicting-zo-token" ] || fail "旧 .env 的 ZO_OBSERVE_AUTH_TOKEN 不得迁移"

OBS_EMAIL_CONFLICT_DIR="$WORK/obs-email-conflict"
mkdir -p "$OBS_EMAIL_CONFLICT_DIR"
printf '%s' 'CanonicalObs@123' > "$OBS_EMAIL_CONFLICT_DIR/otel_auth_password.txt"
printf '%s\n' \
  'ZO_ROOT_USER_EMAIL=canonical@example.com' \
  'ZO_ROOT_USER_PASSWORD=CanonicalObs@123' > "$OBS_EMAIL_CONFLICT_DIR/openobserve.env"
chmod 0600 "$OBS_EMAIL_CONFLICT_DIR"/*
OBS_EMAIL_CONFLICT_ENV="$WORK/obs-email-conflict.env"
cat >"$OBS_EMAIL_CONFLICT_ENV" <<'ENV'
ZO_OBSERVE_ROOT_USER_EMAIL=different@example.com
ZO_OBSERVE_ROOT_USER_PASSWORD=CanonicalObs@123
ENV
TEST_LEGACY_ENV_FILE="$OBS_EMAIL_CONFLICT_ENV"
if run_init "$OBS_EMAIL_CONFLICT_DIR" --observability </dev/null >"$WORK/obs-email-conflict.out" 2>&1; then
  fail "旧 .env 邮箱与 openobserve.env 不一致时必须失败"
fi
TEST_LEGACY_ENV_FILE=""
assert_contains "$WORK/obs-email-conflict.out" "邮箱" "邮箱冲突必须明确报错"
assert_eq "canonical@example.com" "$(env_value "$OBS_EMAIL_CONFLICT_DIR/openobserve.env" ZO_ROOT_USER_EMAIL)" "邮箱冲突不得改写 openobserve.env"

echo "== --no-legacy 忽略显式旧 .env =="
NO_LEGACY_DIR="$WORK/no-legacy"
TEST_LEGACY_ENV_FILE="$LEGACY_FILE"
run_init "$NO_LEGACY_DIR" --observability --no-legacy </dev/null >"$WORK/no-legacy.out" 2>&1
TEST_LEGACY_ENV_FILE=""
if grep -Fq "已停用" "$WORK/no-legacy.out"; then
  fail "--no-legacy 仍读取旧 .env"
fi
for pair in \
  "db_root_password.txt:quoted-root-password" \
  "db_password.txt:literal-password" \
  "jwt_secret.txt:quoted-jwt-secret # literal" \
  "admin_password.txt:quoted-admin1-password" \
  "otel_auth_password.txt:LegacyObs@123"; do
  file="${pair%%:*}"
  sentinel="${pair#*:}"
  [ -s "$NO_LEGACY_DIR/$file" ] || fail "--no-legacy 未生成 $file"
  [ "$(cat "$NO_LEGACY_DIR/$file")" != "$sentinel" ] || fail "--no-legacy 迁移了 $file"
done

echo "== 旧 .env 重复键拒绝 =="
DUP_DB_DIR="$WORK/legacy-duplicate-db"
mkdir -p "$DUP_DB_DIR"
printf '%s' 'existing-db-password-must-not-change' > "$DUP_DB_DIR/db_password.txt"
chmod 0600 "$DUP_DB_DIR/db_password.txt"
dup_db_hash="$(sha256 "$DUP_DB_DIR/db_password.txt")"
DUP_DB_ENV="$WORK/legacy-duplicate-db.env"
cat > "$DUP_DB_ENV" <<'ENV'
DB_PASSWORD=first-password
DB_PASSWORD=second-password
ENV
TEST_LEGACY_ENV_FILE="$DUP_DB_ENV"
if run_init "$DUP_DB_DIR" </dev/null >"$WORK/legacy-duplicate-db.out" 2>&1; then
  fail "旧 .env 重复 DB_PASSWORD 必须失败"
fi
TEST_LEGACY_ENV_FILE=""
assert_contains "$WORK/legacy-duplicate-db.out" "DB_PASSWORD" "重复 DB_PASSWORD 错误提示包含键名"
assert_contains "$WORK/legacy-duplicate-db.out" "重复" "重复 DB_PASSWORD 错误提示"
assert_eq "$dup_db_hash" "$(sha256 "$DUP_DB_DIR/db_password.txt")" "重复 DB_PASSWORD 不得改写已有密码"
[ ! -f "$DUP_DB_DIR/db_root_password.txt" ] || fail "重复 DB_PASSWORD 必须在生成其他 secret 前失败"

DUP_OBS_LEGACY_DIR="$WORK/legacy-duplicate-observability"
mkdir -p "$DUP_OBS_LEGACY_DIR"
printf '%s' 'existing-obs-password-must-not-change' > "$DUP_OBS_LEGACY_DIR/otel_auth_password.txt"
chmod 0600 "$DUP_OBS_LEGACY_DIR/otel_auth_password.txt"
dup_obs_legacy_hash="$(sha256 "$DUP_OBS_LEGACY_DIR/otel_auth_password.txt")"
DUP_OBS_LEGACY_ENV="$WORK/legacy-duplicate-observability.env"
cat > "$DUP_OBS_LEGACY_ENV" <<'ENV'
ZO_OBSERVE_ROOT_USER_PASSWORD=first-observability-password
ZO_OBSERVE_ROOT_USER_PASSWORD=second-observability-password
ENV
TEST_LEGACY_ENV_FILE="$DUP_OBS_LEGACY_ENV"
if run_init "$DUP_OBS_LEGACY_DIR" --observability </dev/null >"$WORK/legacy-duplicate-observability.out" 2>&1; then
  fail "旧 .env 重复 ZO_OBSERVE_ROOT_USER_PASSWORD 必须失败"
fi
TEST_LEGACY_ENV_FILE=""
assert_contains "$WORK/legacy-duplicate-observability.out" "ZO_OBSERVE_ROOT_USER_PASSWORD" "重复可观测密码错误提示包含键名"
assert_contains "$WORK/legacy-duplicate-observability.out" "重复" "重复可观测密码错误提示"
assert_eq "$dup_obs_legacy_hash" "$(sha256 "$DUP_OBS_LEGACY_DIR/otel_auth_password.txt")" "重复可观测密码不得改写已有密码"
[ ! -f "$DUP_OBS_LEGACY_DIR/db_root_password.txt" ] || fail "重复可观测密码必须在生成其他 secret 前失败"

BAD_LEGACY_DIR="$WORK/legacy-unsupported"
BAD_LEGACY_FILE="$WORK/legacy-unsupported.env"
cat > "$BAD_LEGACY_FILE" <<'ENV'
DB_PASSWORD=$(printf unsupported)
ENV
TEST_LEGACY_ENV_FILE="$BAD_LEGACY_FILE"
if run_init "$BAD_LEGACY_DIR" </dev/null >"$WORK/legacy-unsupported.out" 2>&1; then
  fail "不支持的旧 .env 语法必须失败"
fi
TEST_LEGACY_ENV_FILE=""
assert_contains "$WORK/legacy-unsupported.out" "不支持" "不支持语法的明确提示"
[ ! -f "$BAD_LEGACY_DIR/db_password.txt" ] || fail "不支持语法时不得写入错误密码"

echo "✅ test_init_secrets.sh: 所有场景通过"
