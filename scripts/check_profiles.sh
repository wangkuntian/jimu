#!/usr/bin/env bash
# 全部形态（profile）的构建门禁 + 依赖闭包裁剪门禁 + 可选启动/健康检查。
#
# 形态闭包口径 = `./cmd/server` + 该形态的构建期 overlay（`tools/profileoverlay` 把
# internal/profiles/active 的选点替换成「只选该形态」的版本），即出货二进制的真实 import 图；
# 形态名由 `go run ./tools/profileoverlay -list`（registry 唯一来源）派生，不再写死。
#
# 默认（不设 JIMU_PROFILES_SMOKE=1）：构建 + 依赖闭包裁剪门禁 —— 这是硬要求，任一形态
# 构建失败、或闭包里出现被禁止/非预期的能力即非零退出；每个形态仍打印一行 SKIP，说明
# 启动与健康检查需要 DB+Redis，不静默跳过。
#
# 依赖闭包裁剪门禁（「裁剪是真的」这件事的唯一可测形式）：
#   1) 生产包不得 import catalog —— `go list -f '{{join .Imports "\n"}}' ./cmd/server
#      ./internal/profiles/...` 里出现 capabilities/catalog 即失败：种子若再退回
#      catalog.All()，形态裁剪立刻失效。
#   2) 每个形态的能力根包闭包必须逐值等于 EXPECTED_<profile>（golden）：任何能力泄漏
#      （无论是经 catalog 还是新加的 import）都会失败。
#   3) 每个形态的闭包必须不含 FORBIDDEN_<profile>（本次修复要求的逐形态禁止清单）。
#      其中 ALLOWED_<profile> 是**既有的类型级传递残留**，从禁止清单里显式豁免且逐值列明：
#      user/auth 的生产代码直接 import capabilities/outbox（*outbox.Outbox / outbox.Event）
#      与 capabilities/notification（notification.Message / Dispatcher），outbox 又 import
#      capabilities/queue —— 这是软依赖在编译期的类型残留，去除需要把共享类型迁到
#      contract/kernel（另一次改动，见 AGENTS.md「能力边界」），不在本次种子修复范围内。
#      这些能力仍被 (2) 的 golden 锁定：不会再无声明地增减。
#
# JIMU_PROFILES_SMOKE=1：构建后逐个以 APP_ENV=dev 启动，轮询管理端 readiness
# （GET /readyz 检查 DB+Redis 可达，即内核的 /health 语义）直到就绪，然后关停；
# 任一形态启动失败或超时即非零退出。为避开本机已运行的服务，默认用隔离端口
# 18080/19090（HTTP_PORT/MANAGEMENT_PORT），可用 JIMU_PROFILES_HTTP_PORT、
# JIMU_PROFILES_MGMT_PORT、JIMU_PROFILES_READY_TIMEOUT_SEC 覆盖。
#
# DB/Redis 连接沿用应用配置与环境变量（DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/REDIS_ADDR…）。
# 注意：环境里设了 ADMIN_PASSWORD 时，启动会先执行结构性种子（见 internal/profiles
# 的 StructuralSeed），需要先跑过 `jimu migrate up`；未设置时跳过种子。
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

# 形态名来自 registry（唯一来源）：-list 失败即非零退出（set -e），不静默用空列表。
PROFILES=($(go run ./tools/profileoverlay -list))
HTTP_PORT="${JIMU_PROFILES_HTTP_PORT:-18080}"
MGMT_PORT="${JIMU_PROFILES_MGMT_PORT:-19090}"
READY_TIMEOUT_SEC="${JIMU_PROFILES_READY_TIMEOUT_SEC:-30}"

BIN_DIR="$(mktemp -d "${TMPDIR:-/tmp}/jimu-profiles-XXXXXX")"
LOG_DIR="$BIN_DIR/logs"
mkdir -p "$LOG_DIR"
trap 'rm -rf "$BIN_DIR"' EXIT

echo "==> 构建各形态（overlay 构建 ./cmd/server）"
for p in "${PROFILES[@]}"; do
  ov="$(go run ./tools/profileoverlay "$p")"
  go build -overlay="$ov" -o "$BIN_DIR/jimu-$p" ./cmd/server
  echo "    ok  形态 ${p}（overlay 构建 cmd/server）"
done

# ---------------------------------------------------------------------------
# 依赖闭包裁剪门禁
# ---------------------------------------------------------------------------

# 全量能力根包清单（catalog 18 + 非 catalog 7；新增能力请同步此表与 EXPECTED_*）。
ALL_CAPS="access apidocs apikey audit auth breach captcha console dataops encryption feature grpc mfa notification oauth outbox passkey queue retention search storage tenant uploadsec user ws"

# EXPECTED_<profile>：该形态二进制依赖闭包里允许出现的**能力根包**，逐值锁定（golden）。
# 来源＝ internal/profiles/<profile>/assembly.go 的清单 + 传递必需的编译期依赖。
EXPECTED_full="access apidocs apikey audit auth breach captcha console dataops encryption feature grpc mfa notification oauth outbox passkey queue retention search storage tenant uploadsec user ws"
EXPECTED_minimal="access auth encryption notification outbox queue user"
EXPECTED_saas="access audit auth encryption notification outbox queue tenant user"
EXPECTED_enterprise="access audit auth console dataops encryption notification oauth outbox queue storage user ws"
EXPECTED_machine="access apikey encryption grpc notification outbox queue user"

# FORBIDDEN_<profile>：该形态闭包里明确禁止出现的能力（本次修复的逐形态「不得包含」清单）。
# 未列出的形态按「ALL_CAPS − EXPECTED」即全部非预期能力处理（见下方 FORBIDDEN_* 逐条列明）。
FORBIDDEN_full=""
FORBIDDEN_minimal="tenant passkey oauth dataops audit console grpc storage queue outbox search feature uploadsec breach retention apidocs ws"
FORBIDDEN_saas="apidocs apikey breach captcha console dataops feature grpc mfa oauth passkey retention search storage uploadsec ws"
FORBIDDEN_enterprise="apidocs apikey breach captcha feature grpc mfa passkey retention search tenant uploadsec"
FORBIDDEN_machine="auth tenant mfa passkey oauth console audit dataops storage queue outbox notification"

# ALLOWED_<profile>：从 FORBIDDEN 中显式豁免的既有类型级传递残留（见文件头注释），
# 它们仍是 EXPECTED 的成员，只豁免「禁止」不改「锁定」。
ALLOWED_full=""
ALLOWED_minimal="outbox queue"
ALLOWED_saas=""
ALLOWED_enterprise=""
ALLOWED_machine="notification outbox queue"

# cap_roots <overlay>：该形态（./cmd/server + overlay）闭包里的能力**根包**名
# （不含 access/domain 之类的子包），排序去重。
cap_roots() {
  go list -overlay="$1" -deps ./cmd/server | sed -n 's#^jimu/internal/capabilities/\([^/]*\)$#\1#p' | sort -u
}

# subtract_words <words> <remove>：$1 去掉 $2 中出现过的词。
subtract_words() {
  local w out=""
  for w in $1; do
    case " $2 " in *" $w "*) ;; *) out="$out $w" ;; esac
  done
  printf '%s' "${out# }"
}

echo "==> 生产包不得 import catalog"
catalog_importers="$(go list -f '{{join .Imports "\n"}}' ./cmd/server ./internal/profiles/... | grep 'jimu/internal/capabilities/catalog' || true)"
if [[ -n "$catalog_importers" ]]; then
  echo "❌ ./cmd/server 或 ./internal/profiles/... 的生产包 import 了 capabilities/catalog：形态裁剪会失效" >&2
  echo "$catalog_importers" >&2
  exit 1
fi
echo "    ok  ./cmd/server 与 ./internal/profiles/... 无 capabilities/catalog"

echo "==> 依赖闭包裁剪门禁（能力根包）"
closure_fail=0
for p in "${PROFILES[@]}"; do
  ov="$(go run ./tools/profileoverlay "$p")"
  roots="$(cap_roots "$ov")"
  expected_var="EXPECTED_$p"
  expected="${!expected_var}"
  forbidden_var="FORBIDDEN_$p"
  forbidden="${!forbidden_var}"
  allowed_var="ALLOWED_$p"
  allowed="${!allowed_var}"
  profile_fail=0

  effective_forbidden="$(subtract_words "$forbidden" "$allowed")"
  hits="$(comm -12 <(printf '%s\n' $effective_forbidden | sed '/^$/d' | sort -u) \
                    <(printf '%s\n' $roots | sed '/^$/d' | sort -u) | tr '\n' ' ')"
  hits="${hits% }"
  if [[ -n "$hits" ]]; then
    echo "❌ 形态 $p 闭包含被禁止的能力：$hits" >&2
    closure_fail=1
    profile_fail=1
  fi

  if [[ "$roots" != "$(printf '%s\n' $expected | sort -u)" ]]; then
    echo "❌ 形态 $p 闭包与 golden 期望不一致" >&2
    echo "    实际：$(printf '%s' "$roots" | tr '\n' ' ')" >&2
    echo "    期望：$expected" >&2
    closure_fail=1
    profile_fail=1
  fi
  if [[ "$profile_fail" == "0" ]]; then
    echo "    ok  形态 $p 能力根包：$(printf '%s' "$roots" | tr '\n' ' ')"
  fi
done
if [[ "$closure_fail" != "0" ]]; then
  exit 1
fi

if [[ "${JIMU_PROFILES_SMOKE:-}" != "1" ]]; then
  for p in "${PROFILES[@]}"; do
    echo "SKIP 形态 $p 启动与健康检查：需要 DB+Redis（设置 JIMU_PROFILES_SMOKE=1 启用；APP_ENV=dev，端口 ${HTTP_PORT}/${MGMT_PORT}）"
  done
  echo "✅ 各形态（overlay 构建 cmd/server）+ 依赖闭包裁剪门禁通过（启动与健康检查已跳过：未设置 JIMU_PROFILES_SMOKE=1）"
  exit 0
fi

echo "==> 启动与健康检查（APP_ENV=dev，HTTP ${HTTP_PORT} / management ${MGMT_PORT}）"
for p in "${PROFILES[@]}"; do
  log="$LOG_DIR/$p.log"
  APP_ENV=dev HTTP_PORT="$HTTP_PORT" MANAGEMENT_PORT="$MGMT_PORT" MANAGEMENT_HOST=127.0.0.1 \
    "$BIN_DIR/jimu-$p" >"$log" 2>&1 &
  pid=$!

  ready=0
  for _ in $(seq 1 "$READY_TIMEOUT_SEC"); do
    if ! kill -0 "$pid" 2>/dev/null; then
      break # 进程已退出：直接看日志
    fi
    if curl -fsS "http://127.0.0.1:${MGMT_PORT}/readyz" >/dev/null 2>&1; then
      ready=1
      break
    fi
    sleep 1
  done

  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true

  if [[ "$ready" != "1" ]]; then
    echo "❌ 形态 $p 未在 ${READY_TIMEOUT_SEC}s 内就绪，日志：" >&2
    sed 's/^/    /' "$log" >&2
    exit 1
  fi
  echo "    ok  形态 $p 就绪（GET /readyz）"
done

echo "✅ 各形态（overlay 构建 cmd/server）+ 依赖闭包裁剪门禁 + 启动 + 健康检查通过"
