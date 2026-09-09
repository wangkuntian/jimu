#!/usr/bin/env bash
# OpenObserve 默认告警同步脚本
#
# 以 git 中 alerts/*.json 为准，将默认告警配置应用到 OpenObserve（幂等）：
#   1. 通知模板 jimu_alert_http（template-http.json）
#   2. webhook 通知目的地 jimu_webhook（destination-webhook.json，URL 由 ZO_ALERT_WEBHOOK_URL 注入）
#   3. 告警规则（jimu_*.json）：按 name 查找，存在则更新（PUT），否则创建（POST）
#
# 说明：
#   - OSS 版创建告警必须绑定至少一个已存在的 destination，因此整个同步依赖 ZO_ALERT_WEBHOOK_URL；
#     未设置时跳过（不影响 OpenObserve 使用，可在 UI 手工配置告警）
#   - 创建告警要求目标 stream 的 schema 已存在（即已有数据入库）。首次启动时数据可能尚未写入，
#     脚本按轮次重试；超限仍缺 stream 的规则跳过并提示（下次运行补齐）
#
# 用法:
#   ./deploy/openobserve/sync-alerts.sh
# 环境变量:
#   ZO_HTTP / ZO_ORG / ZO_EMAIL / ZO_PASSWORD（同 sync-dashboard.sh）
#   ZO_ALERT_WEBHOOK_URL  webhook 通知地址（未设置则跳过整个同步）
#   ALERTS_RETRIES        stream 未就绪时的重试轮数（默认 3）
#   ALERTS_RETRY_INTERVAL 每轮间隔秒数（默认 15）
set -euo pipefail

ZO_HTTP="${ZO_HTTP:-http://localhost:5080}"
ZO_ORG="${ZO_ORG:-default}"
ZO_EMAIL="${ZO_EMAIL:-admin@jimu.local}"
ZO_PASSWORD="${ZO_PASSWORD:-Admin@12345}"
ZO_ALERT_WEBHOOK_URL="${ZO_ALERT_WEBHOOK_URL:-}"
ALERTS_RETRIES="${ALERTS_RETRIES:-3}"
ALERTS_RETRY_INTERVAL="${ALERTS_RETRY_INTERVAL:-15}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ALERTS_DIR="$SCRIPT_DIR/alerts"
API="$ZO_HTTP/api/$ZO_ORG"

if [ -z "$ZO_ALERT_WEBHOOK_URL" ]; then
  echo "==> 跳过默认告警同步：未设置 ZO_ALERT_WEBHOOK_URL（告警需 webhook 通知目的地，可在 UI 手工配置）"
  exit 0
fi

echo "==> 等待 OpenObserve 就绪 ($ZO_HTTP)"
for i in $(seq 1 60); do
  if curl -fsS --max-time 2 "$ZO_HTTP/healthz" >/dev/null 2>&1; then
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "❌ OpenObserve 未就绪，退出" >&2
    exit 1
  fi
  sleep 2
done

echo "==> 同步通知模板 / webhook 目的地（幂等）"
ZO_API="$API" ZO_EMAIL="$ZO_EMAIL" ZO_PASSWORD="$ZO_PASSWORD" \
  ZO_WEBHOOK_URL="$ZO_ALERT_WEBHOOK_URL" ALERTS_DIR="$ALERTS_DIR" python3 <<'PYEOF'
import base64, json, os, sys, urllib.error, urllib.request

api = os.environ["ZO_API"]
email = os.environ["ZO_EMAIL"]; password = os.environ["ZO_PASSWORD"]
webhook_url = os.environ["ZO_WEBHOOK_URL"]
alerts_dir = os.environ["ALERTS_DIR"]
auth = "Basic " + base64.b64encode(f"{email}:{password}".encode()).decode()


def req(method, path, body=None):
    r = urllib.request.Request(api + path, method=method,
        data=json.dumps(body).encode() if body is not None else None,
        headers={"Authorization": auth, "Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(r, timeout=20) as resp:
            raw = resp.read()
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode(errors="ignore")


def upsert(kind, name, payload, path, query=""):
    st, resp = req("GET", f"{path}/{name}")
    if st == 200:
        st, resp = req("PUT", f"{path}/{name}{query}", payload)
        action = "更新"
    else:
        st, resp = req("POST", f"{path}{query}", payload)
        action = "创建"
    if st not in (200, 201):
        print(f"  ✗ {kind} '{name}' {action}失败 HTTP {st}: {str(resp)[:150]}", file=sys.stderr)
        sys.exit(1)
    print(f"  ✅ {kind} '{name}' 已{action}")


with open(os.path.join(alerts_dir, "template-http.json"), encoding="utf-8") as f:
    tpl = json.load(f)
upsert("通知模板", tpl["name"], tpl, "/alerts/templates")

with open(os.path.join(alerts_dir, "destination-webhook.json"), encoding="utf-8") as f:
    dest = json.load(f)
    raw = json.dumps(dest).replace("__ZO_ALERT_WEBHOOK_URL__", webhook_url)
    upsert("通知目的地", dest["name"], json.loads(raw), "/alerts/destinations")
PYEOF

echo "==> 应用 alerts/jimu_*.json（幂等：存在则更新，否则创建；stream 未就绪自动重试）"
ZO_API="$API" ZO_EMAIL="$ZO_EMAIL" ZO_PASSWORD="$ZO_PASSWORD" \
  ALERTS_DIR="$ALERTS_DIR" RETRIES="$ALERTS_RETRIES" INTERVAL="$ALERTS_RETRY_INTERVAL" python3 <<'PYEOF'
import base64, glob, json, os, sys, time, urllib.error, urllib.request

api = os.environ["ZO_API"]
email = os.environ["ZO_EMAIL"]; password = os.environ["ZO_PASSWORD"]
alerts_dir = os.environ["ALERTS_DIR"]; retries = int(os.environ["RETRIES"]); interval = int(os.environ["INTERVAL"])
auth = "Basic " + base64.b64encode(f"{email}:{password}".encode()).decode()


def req(method, path, body=None):
    r = urllib.request.Request(api + path, method=method,
        data=json.dumps(body).encode() if body is not None else None,
        headers={"Authorization": auth, "Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(r, timeout=20) as resp:
            raw = resp.read()
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode(errors="ignore")


def existing_ids():
    st, resp = req("GET", "/v2/alerts")
    if st != 200:
        print(f"  ✗ 拉取告警列表失败 HTTP {st}: {str(resp)[:150]}", file=sys.stderr)
        sys.exit(1)
    return {item.get("name"): str(item.get("alert_id")) for item in resp.get("list", [])}


files = sorted(glob.glob(os.path.join(alerts_dir, "jimu_*.json")))
if not files:
    print("（alerts 目录无 jimu_*.json 规则文件）")
    sys.exit(0)

pending = {f: json.load(open(f, encoding="utf-8")) for f in files}
ok_count = 0
skipped = []
for attempt in range(1, retries + 1):
    ids = existing_ids()
    retry_next = {}
    for f, rule in pending.items():
        name = rule["name"]
        if name in ids:
            st, resp = req("PUT", f"/v2/alerts/{ids[name]}", rule)
            action = "更新"
        else:
            st, resp = req("POST", "/v2/alerts", rule)
            action = "创建"
        if st in (200, 201):
            print(f"  ✅ {name} 已{action}")
            ok_count += 1
            continue
        detail = str(resp)
        # stream schema 未就绪（无数据入库）时可重试；其余错误直接放弃该规则
        if attempt < retries and ("stream" in detail.lower() or "not found" in detail.lower()):
            retry_next[f] = rule
            continue
        skipped.append(f"  ⚠ {name} 失败 HTTP {st}: {detail[:150]}")
    pending = retry_next
    if not pending:
        break
    print(f"  … 第 {attempt} 轮后 {len(pending)} 条待重试（{interval}s 后重试）")
    time.sleep(interval)

for line in skipped:
    print(line, file=sys.stderr)
if pending:
    for f, rule in pending.items():
        print(f"  ⚠ {rule['name']} 多次重试失败（目标 stream 可能尚无数据），已跳过；稍后重新运行本脚本可补齐", file=sys.stderr)
print(f"✅ 告警同步完成：成功 {ok_count}，跳过 {len(skipped) + len(pending)}")
PYEOF
