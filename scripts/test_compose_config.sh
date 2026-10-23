#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKDIR="$(mktemp -d)"
CORE_SECRETS_DIR="$WORKDIR/core-secrets"
ALL_SECRETS_DIR="$WORKDIR/all-secrets"
CORE_JSON="$WORKDIR/core.json"
ALL_JSON="$WORKDIR/all.json"
POISON_JSON="$WORKDIR/poison.json"

cleanup() {
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

compose_core() {
  env -u JIMU_IMAGE \
    JIMU_SECRETS_DIR="$CORE_SECRETS_DIR" \
    docker compose --env-file /dev/null -f "$ROOT_DIR/docker-compose.yml" "$@"
}

compose_all() {
  env -u JIMU_IMAGE \
    JIMU_SECRETS_DIR="$ALL_SECRETS_DIR" \
    docker compose --env-file /dev/null \
      -f "$ROOT_DIR/docker-compose.yml" \
      -f "$ROOT_DIR/compose.observability.yml" "$@"
}

mkdir -p "$CORE_SECRETS_DIR" "$ALL_SECRETS_DIR"
printf '%s' 'compose-root-password' > "$CORE_SECRETS_DIR/db_root_password.txt"
printf '%s' 'compose-db-password' > "$CORE_SECRETS_DIR/db_password.txt"
printf '%s' 'compose-jwt-secret-012345678901234567890123456789' > "$CORE_SECRETS_DIR/jwt_secret.txt"
printf '%s' '01234567890123456789012345678901' > "$CORE_SECRETS_DIR/encryption_key.txt"
printf '%s' 'compose-audit-secret-012345678901234567890123456789' > "$CORE_SECRETS_DIR/audit_hash_secret.txt"
printf '%s' 'compose-admin-password' > "$CORE_SECRETS_DIR/admin_password.txt"
cp "$CORE_SECRETS_DIR"/*.txt "$ALL_SECRETS_DIR/"
printf '%s' 'compose-otel-password' > "$ALL_SECRETS_DIR/otel_auth_password.txt"
printf '%s' 'compose-zo-token' > "$ALL_SECRETS_DIR/zo_auth_token.txt"
cat > "$ALL_SECRETS_DIR/openobserve.env" <<'ENV'
ZO_ROOT_USER_EMAIL='admin@jimu.local'
ZO_ROOT_USER_PASSWORD='compose-otel-password'
ENV

compose_core config --quiet
compose_core config --format json > "$CORE_JSON"
python3 - "$CORE_JSON" <<'PY'
import json
import sys

config = json.load(open(sys.argv[1], encoding="utf-8"))
services = config["services"]
assert "jimu-init" in services
assert services["server"]["depends_on"]["jimu-init"]["condition"] == "service_completed_successfully"
assert "openobserve" not in services
assert "otel-collector" not in services
assert set(config["secrets"]) == {
    "db_root_password",
    "db_password",
    "jwt_secret",
    "encryption_key",
    "audit_hash_secret",
    "admin_password",
}
serialized = json.dumps(config)
assert "/run/secrets/otel_auth_password" not in serialized
assert "/run/secrets/zo_auth_token" not in serialized
assert "openobserve.env" not in serialized

fixed_environment = {
    "DB_HOST": "mariadb",
    "DB_PORT": "3306",
    "DB_USER": "jimu",
    "DB_NAME": "jimu",
    "REDIS_ADDR": "redis:6379",
    "REDIS_DB": "0",
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
    "/bin/sh",
    "-c",
    "./jimu migrate up && ./jimu seed",
]
for dependency in ("mariadb", "redis"):
    assert services["jimu-init"]["depends_on"][dependency]["condition"] == "service_healthy"

for service, target in (("mariadb", 3306), ("redis", 6379)):
    assert any(
        port["host_ip"] == "127.0.0.1" and port["target"] == target
        for port in services[service]["ports"]
    ), f"{service} loopback port"

assert services["server"]["environment"]["ENCRYPTION_KEY_FILE"] == "/run/secrets/encryption_key"
assert services["server"]["environment"]["AUDIT_HASH_SECRET_FILE"] == "/run/secrets/audit_hash_secret"
PY

compose_all config --quiet
compose_all config --format json > "$ALL_JSON"
python3 - "$ALL_JSON" <<'PY'
import json
import sys

services = json.load(open(sys.argv[1], encoding="utf-8"))["services"]
assert "openobserve" in services
assert "otel-collector" in services
assert services["server"]["environment"]["OTEL_ENABLED"] == "true"
assert services["server"]["environment"]["OTEL_ENDPOINT"] == "openobserve:5081"
assert services["server"]["environment"]["OTEL_AUTH_EMAIL"] == "admin@jimu.local"
assert services["server"]["environment"]["OTEL_AUTH_PASSWORD_FILE"] == "/run/secrets/otel_auth_password"
assert services["openobserve"]["environment"]["ZO_ROOT_USER_PASSWORD"] == "compose-otel-password"
assert "Admin@12345" not in services["openobserve"]["environment"].values()
assert services["openobserve"].get("profiles") is None
assert services["otel-collector"].get("profiles") is None
for service in ("openobserve", "otel-collector"):
    assert services[service]["image"] != "poison:latest"

server_secret_sources = {item["source"] for item in services["server"]["secrets"]}
assert {
    "db_password",
    "jwt_secret",
    "encryption_key",
    "audit_hash_secret",
    "admin_password",
    "otel_auth_password",
}.issubset(server_secret_sources)
collector_secret_sources = {item["source"] for item in services["otel-collector"]["secrets"]}
assert {"db_root_password", "zo_auth_token"}.issubset(collector_secret_sources)
PY

SECRETS_WITHOUT_OBS_ENV="$WORKDIR/secrets-without-obs-env"
cp -R "$ALL_SECRETS_DIR" "$SECRETS_WITHOUT_OBS_ENV"
rm "$SECRETS_WITHOUT_OBS_ENV/openobserve.env"
env -u JIMU_IMAGE \
  JIMU_SECRETS_DIR="$SECRETS_WITHOUT_OBS_ENV" \
  docker compose --env-file /dev/null \
    -f "$ROOT_DIR/docker-compose.yml" \
    -f "$ROOT_DIR/compose.observability.yml" config --quiet

POISON_DIR="$WORKDIR/poison"
mkdir -p "$POISON_DIR"
cp "$ROOT_DIR/docker-compose.yml" "$ROOT_DIR/compose.observability.yml" "$POISON_DIR/"
printf 'JIMU_IMAGE=poison:latest\n' > "$POISON_DIR/.env"
(
  cd "$POISON_DIR"
  env -u JIMU_IMAGE \
    JIMU_SECRETS_DIR="$ALL_SECRETS_DIR" \
    docker compose --env-file /dev/null \
      -f docker-compose.yml -f compose.observability.yml config --format json > "$POISON_JSON"
)
python3 - "$POISON_JSON" <<'PY'
import json
import sys

services = json.load(open(sys.argv[1], encoding="utf-8"))["services"]
for name in ("server", "jimu-init", "openobserve", "otel-collector"):
    assert services[name]["image"] != "poison:latest", f"{name} loaded JIMU_IMAGE from .env"
PY

echo "compose contract passed"
