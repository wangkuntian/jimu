#!/usr/bin/env bash
# OpenObserve dashboard 同步脚本
#
# 以 git 中 dashboards/*.json 为准，将 dashboard 配置应用到 OpenObserve（幂等）：
#   - 线上不存在同名 dashboard → 创建
#   - 已存在 → 删除后重建（保持与 git 一致；面板的手工 UI 调整会被覆盖）
# 也支持反方向导出：把线上 dashboard JSON 导出到本地文件（UI 微调后同步回 git）。
#
# 用法:
#   ./deploy/openobserve/sync-dashboard.sh                 # 应用 dashboards/*.json 到 OpenObserve
#   ./deploy/openobserve/sync-dashboard.sh --export 名称    # 导出线上同名 dashboard 为文件
# 环境变量:
#   ZO_HTTP / ZO_ORG / ZO_EMAIL / ZO_PASSWORD（同 init-dashboard.sh）
#   DASH_DIR  dashboard JSON 目录（默认 deploy/openobserve/dashboards）
set -euo pipefail

ZO_HTTP="${ZO_HTTP:-http://localhost:5080}"
ZO_ORG="${ZO_ORG:-default}"
ZO_EMAIL="${ZO_EMAIL:-admin@jimu.local}"
ZO_PASSWORD="${ZO_PASSWORD:-Admin@12345}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DASH_DIR="${DASH_DIR:-$SCRIPT_DIR/dashboards}"
API="$ZO_HTTP/api/$ZO_ORG"

if [ "${1:-}" = "--export" ]; then
  EXPORT_NAME="${2:?usage: sync-dashboard.sh --export 名称}"
  ZO_HTTP="$ZO_HTTP" ZO_API="$API" ZO_EMAIL="$ZO_EMAIL" ZO_PASSWORD="$ZO_PASSWORD" \
    ZO_NAME="$EXPORT_NAME" ZO_DIR="$DASH_DIR" python3 <<'PYEOF'
import base64, json, os, sys, urllib.request

api = os.environ["ZO_API"]; email = os.environ["ZO_EMAIL"]; password = os.environ["ZO_PASSWORD"]
name = os.environ["ZO_NAME"]; ddir = os.environ["ZO_DIR"]
auth = "Basic " + base64.b64encode(f"{email}:{password}".encode()).decode()
req = urllib.request.Request(api + "/dashboards?page_size=100",
                             headers={"Authorization": auth})
with urllib.request.urlopen(req, timeout=20) as resp:
    data = json.load(resp)
for d in data.get("dashboards", []):
    v8 = d.get("v8")
    if v8 and v8.get("title") == name:
        # 剥离服务端运行时元数据，仅保留可应用的配置字段
        for key in ("dashboardId", "role", "owner", "created", "updated", "latestHash"):
            v8.pop(key, None)
        out = os.path.join(ddir, name.lower().replace(" ", "-") + ".json")
        with open(out, "w", encoding="utf-8") as f:
            json.dump(v8, f, indent=2, ensure_ascii=False)
            f.write("\n")
        panels = len(v8.get("tabs", [{}])[0].get("panels", []))
        print(f"✅ 已导出 {panels} 面板 -> {out}")
        sys.exit(0)
print(f"❌ 未找到 dashboard '{name}'", file=sys.stderr)
sys.exit(1)
PYEOF
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

echo "==> 应用 dashboards/*.json（幂等：同名重建成 git 版本）"
ZO_API="$API" ZO_EMAIL="$ZO_EMAIL" ZO_PASSWORD="$ZO_PASSWORD" ZO_DIR="$DASH_DIR" python3 <<'PYEOF'
import base64, json, os, sys, urllib.error, urllib.request

api = os.environ["ZO_API"]; email = os.environ["ZO_EMAIL"]; password = os.environ["ZO_PASSWORD"]
ddir = os.environ["ZO_DIR"]
auth = "Basic " + base64.b64encode(f"{email}:{password}".encode()).decode()

def req(method, path, body=None):
    r = urllib.request.Request(api + path, method=method,
        data=json.dumps(body).encode() if body is not None else None,
        headers={"Authorization": auth, "Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(r, timeout=20) as resp:
            return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode(errors="ignore")

st, resp = req("GET", "/dashboards?page_size=100")
existing = {d.get("v8", {}).get("title"): d["v8"]["dashboardId"]
            for d in resp.get("dashboards", []) if d.get("v8")}

files = sorted(f for f in os.listdir(ddir) if f.endswith(".json"))
if not files:
    print("（dashboards 目录无 JSON 文件）")
    sys.exit(0)

for name in files:
    with open(os.path.join(ddir, name), encoding="utf-8") as f:
        dash = json.load(f)
    title = dash.get("title", name)
    did = existing.get(title)
    if did:
        st, _ = req("DELETE", f"/dashboards/{did}")
        if st not in (200, 204, 404):
            print(f"  ⚠ 删除旧版 '{title}' 失败 HTTP {st}，跳过")
            continue
    st, resp = req("POST", "/dashboards", dash)
    if st not in (200, 201):
        print(f"  ✗ 创建 '{title}' 失败 HTTP {st}: {str(resp)[:150]}")
        continue
    did = resp["v8"]["dashboardId"]
    # v8 create 支持内嵌 tabs[].panels 一次创建全部面板
    panels = dash.get("tabs", [{}])[0].get("panels", [])
    print(f"✅ dashboard '{title}' 就绪（{len(panels)} 面板，dashboardId={did}）")
PYEOF