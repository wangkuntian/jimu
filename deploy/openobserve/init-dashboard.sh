#!/usr/bin/env bash
# OpenObserve 默认 dashboard 初始化脚本（幂等）
#
# 等待 OpenObserve 就绪后，创建 jimu 默认概览 dashboard（error/log 统计 + DB 连接池指标）。
# 已存在同名 dashboard 时直接跳过，可安全重复执行。
#
# 用法:
#   ./deploy/openobserve/init-dashboard.sh
# 环境变量:
#   ZO_HTTP      OpenObserve HTTP 地址   (默认 http://localhost:5080)
#   ZO_ORG       organization             (默认 default)
#   ZO_EMAIL     Basic Auth 邮箱          (默认 admin@jimu.local)
#   ZO_PASSWORD  Basic Auth 密码          (默认 Admin@12345)
set -euo pipefail

ZO_HTTP="${ZO_HTTP:-http://localhost:5080}"
ZO_ORG="${ZO_ORG:-default}"
ZO_EMAIL="${ZO_EMAIL:-admin@jimu.local}"
ZO_PASSWORD="${ZO_PASSWORD:-Admin@12345}"
DASH_TITLE="Jimu Overview"
DASH_DESC="jimu 默认概览（error/log 统计与 DB 连接池指标，自动生成；可在 UI 中调整面板查询）"

API="$ZO_HTTP/api/$ZO_ORG"
AUTH=(-u "$ZO_EMAIL:$ZO_PASSWORD")

echo "==> 等待 OpenObserve 就绪 ($ZO_HTTP)"
for i in $(seq 1 60); do
  if curl -fsS --max-time 2 "$ZO_HTTP/healthz" >/dev/null 2>&1; then
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "❌ OpenObserve 未在 60s 内就绪，跳过 dashboard 初始化" >&2
    exit 1
  fi
  sleep 2
done
echo "🟢 OpenObserve 就绪"

echo "==> 检查是否已存在 dashboard '$DASH_TITLE'"
EXISTING=$(curl -fsS --max-time 10 "${AUTH[@]}" "$API/dashboards?page_size=100" 2>/dev/null || echo "")
if echo "$EXISTING" | grep -q "\"$DASH_TITLE\""; then
  echo "✅ dashboard 已存在，跳过"
  exit 0
fi

echo "==> 创建 dashboard '$DASH_TITLE'"
CREATE_RESP=$(curl -fsS --max-time 15 "${AUTH[@]}" -X POST "$API/dashboards" \
  -H "Content-Type: application/json" \
  -d "{\"version\":8,\"title\":\"$DASH_TITLE\",\"description\":\"$DASH_DESC\",\"folder_id\":\"default\",\"tabs\":[{\"tabId\":\"default\",\"name\":\"Default\",\"panels\":[]}]}")
DASH_ID=$(echo "$CREATE_RESP" | python3 -c "import json,sys; print(json.load(sys.stdin)['v8']['dashboardId'])")
HASH=$(echo "$CREATE_RESP" | python3 -c "import json,sys; print(json.load(sys.stdin).get('hash',''))")
echo "   dashboardId=$DASH_ID"

# 逐个添加面板（与 openobserve 官方 AddPanel API 一致；每次写入返回新 hash）
add_panel() { # $1=panel_id  $2=title  $3=sql  $4=stream  $5=stream_type  $6=x_alias
  local pid="$1" title="$2" sql="$3" stream="$4" stype="$5" x_alias="$6"
  local panel_json
  panel_json=$(python3 - "$pid" "$title" "$sql" "$stream" "$stype" "$x_alias" <<'PYEOF'
import json, sys
pid, title, sql, stream, stype, x_alias = sys.argv[1:7]
panel = {
    "id": pid,
    "type": "bar",
    "title": title,
    "description": "",
    "config": {"show_legends": True, "legends_position": None},
    "queryType": "sql",
    "queries": [{
        "query": sql,
        "vrlFunctionQuery": None,
        "customQuery": True,
        "fields": {
            "stream": stream,
            "stream_type": stype,
            "x": [{"label": x_alias, "alias": x_alias, "column": x_alias, "color": None}],
            "y": [{"label": "count", "alias": "count", "column": "count", "color": None}],
            "z": [],
            "filter": {"filterType": "group", "logicalOperator": "AND", "conditions": []},
        },
        "config": {"promql_legend": ""},
    }],
    "layout": {"x": 0, "y": 0, "w": 24, "h": 9, "i": 1},
}
print(json.dumps(panel))
PYEOF
  )
  local resp
  resp=$(curl -fsS --max-time 15 "${AUTH[@]}" -X POST \
    "$API/dashboards/$DASH_ID/panels?hash=$HASH&folder=default" \
    -H "Content-Type: application/json" \
    -d "{\"panel\": $panel_json, \"tabId\": \"default\"}")
  HASH=$(echo "$resp" | python3 -c "import json,sys; print(json.load(sys.stdin)['hash'])")
  echo "   + panel '$title'"
}

add_panel "jimu-error-count" "错误日志总数" \
  'SELECT count(*) AS count FROM "default" WHERE severity_text = '\''error'\''' \
  "default" "logs" "count"

add_panel "jimu-log-count" "日志总量" \
  'SELECT count(*) AS count FROM "default"' \
  "default" "logs" "count"

add_panel "jimu-db-connections" "DB 连接池使用" \
  'SELECT value FROM "jimu_db_open_connections" ORDER BY _timestamp DESC LIMIT 1' \
  "jimu_db_open_connections" "metrics" "value"

echo "✅ dashboard '$DASH_TITLE' 初始化完成，打开 $ZO_HTTP/dashboards 查看"