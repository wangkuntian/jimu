#!/usr/bin/env bash
# OpenObserve 默认 dashboard 初始化脚本（幂等，v2：多面板 + 时间序列/统计/表格）
#
# 等待 OpenObserve 就绪后，创建 jimu 默认概览 dashboard：
#   - stat 卡片：错误日志 / 日志总量 / DB 连接池 / goroutines
#   - time-series：DB 连接池、goroutines、内存、调度任务、错误日志、日志量、HTTP in-flight、熔断状态
#   - table：最近错误日志
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
DASH_DESC="jimu 默认概览（日志/指标/追踪自动生成；面板查询可在 UI 中调整）"

API="$ZO_HTTP/api/$ZO_ORG"

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
EXISTING=$(curl -fsS --max-time 10 -u "$ZO_EMAIL:$ZO_PASSWORD" "$API/dashboards?page_size=100" 2>/dev/null || echo "")
if echo "$EXISTING" | grep -q "\"$DASH_TITLE\""; then
  echo "✅ dashboard 已存在，跳过"
  exit 0
fi

echo "==> 创建 dashboard 与面板（Python 驱动）"
ZO_HTTP="$ZO_HTTP" ZO_API="$API" ZO_EMAIL="$ZO_EMAIL" ZO_PASSWORD="$ZO_PASSWORD" ZO_DASH_TITLE="$DASH_TITLE" ZO_DASH_DESC="$DASH_DESC" python3 <<'PYEOF'
import json
import os
import sys
import time
import urllib.error
import urllib.request
import base64

API = os.environ["ZO_API"]
EMAIL = os.environ["ZO_EMAIL"]
PASSWORD = os.environ["ZO_PASSWORD"]
TITLE = os.environ["ZO_DASH_TITLE"]
DESC = os.environ["ZO_DASH_DESC"]
AUTH = "Basic " + base64.b64encode(f"{EMAIL}:{PASSWORD}".encode()).decode()


def req(method, path, body=None):
    r = urllib.request.Request(
        API + path,
        method=method,
        data=json.dumps(body).encode() if body is not None else None,
        headers={"Authorization": AUTH, "Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(r, timeout=20) as resp:
            return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode(errors="ignore")


def panel(pid, ptype, title, sql, stream, stype, layout, x_aliases=("count",), y_aliases=("count",)):
    return {
        "id": pid,
        "type": ptype,
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
                "x": [{"label": a, "alias": a, "column": a, "color": None} for a in x_aliases],
                "y": [{"label": a, "alias": a, "column": a, "color": None} for a in y_aliases],
                "z": [],
                "filter": {"filterType": "group", "logicalOperator": "AND", "conditions": []},
            },
            "config": {"promql_legend": ""},
        }],
        "layout": layout,
    }


def ts(stream, value_expr, alias="_value"):
    return (f'SELECT histogram(_timestamp) AS _time, {value_expr} AS {alias} '
            f'FROM "{stream}" GROUP BY _time ORDER BY _time')


def stat_sql(stream, expr, alias="count"):
    return f'SELECT {expr} AS {alias} FROM "{stream}"'


# ---- 面板定义（布局：24 列网格；time-series/stat 半宽两列，table 全宽）----
panels = [
    # 行0: stat 卡片
    panel("jimu-stat-error", "stat", "错误日志总数",
          'SELECT count(*) AS count FROM "default" WHERE severity = \'error\'',
          "default", "logs", {"x": 0, "y": 0, "w": 6, "h": 8, "i": 1}),
    panel("jimu-stat-logs", "stat", "日志总量",
          stat_sql("default", "count(*)", "count"),
          "default", "logs", {"x": 6, "y": 0, "w": 6, "h": 8, "i": 2}),
    panel("jimu-stat-db", "stat", "DB 使用连接",
          stat_sql("jimu_db_in_use_connections", "max(value)", "value"),
          "jimu_db_in_use_connections", "metrics", {"x": 12, "y": 0, "w": 6, "h": 8, "i": 3},
          x_aliases=("value",), y_aliases=("value",)),
    panel("jimu-stat-goroutines", "stat", "Goroutines",
          stat_sql("jimu_runtime_goroutines", "max(value)", "value"),
          "jimu_runtime_goroutines", "metrics", {"x": 18, "y": 0, "w": 6, "h": 8, "i": 4},
          x_aliases=("value",), y_aliases=("value",)),
    # 行1: 时间序列
    panel("jimu-ts-db", "time-series", "DB 连接池使用",
          ts("jimu_db_in_use_connections", "max(value)", "value"),
          "jimu_db_in_use_connections", "metrics", {"x": 0, "y": 8, "w": 12, "h": 9, "i": 5},
          x_aliases=("_time",), y_aliases=("value",)),
    panel("jimu-ts-goroutines", "time-series", "Goroutines 趋势",
          ts("jimu_runtime_goroutines", "max(value)", "value"),
          "jimu_runtime_goroutines", "metrics", {"x": 12, "y": 8, "w": 12, "h": 9, "i": 6},
          x_aliases=("_time",), y_aliases=("value",)),
    # 行2: 时间序列
    panel("jimu-ts-memory", "time-series", "内存分配",
          ts("jimu_runtime_memory_alloc_bytes", "max(value)", "value"),
          "jimu_runtime_memory_alloc_bytes", "metrics", {"x": 0, "y": 17, "w": 12, "h": 9, "i": 7},
          x_aliases=("_time",), y_aliases=("value",)),
    panel("jimu-ts-scheduler", "time-series", "调度任务执行数",
          ts("jimu_scheduler_jobs_total", "sum(value)", "value"),
          "jimu_scheduler_jobs_total", "metrics", {"x": 12, "y": 17, "w": 12, "h": 9, "i": 8},
          x_aliases=("_time",), y_aliases=("value",)),
    # 行3: 日志时间序列
    panel("jimu-ts-error-logs", "time-series", "错误日志趋势",
          'SELECT histogram(_timestamp) AS _time, count(*) AS count FROM "default" WHERE severity = \'error\' GROUP BY _time ORDER BY _time',
          "default", "logs", {"x": 0, "y": 26, "w": 12, "h": 9, "i": 9},
          x_aliases=("_time",), y_aliases=("count",)),
    panel("jimu-ts-logs", "time-series", "日志量趋势",
          'SELECT histogram(_timestamp) AS _time, count(*) AS count FROM "default" GROUP BY _time ORDER BY _time',
          "default", "logs", {"x": 12, "y": 26, "w": 12, "h": 9, "i": 10},
          x_aliases=("_time",), y_aliases=("count",)),
    # 行4: HTTP 与熔断
    panel("jimu-ts-inflight", "time-series", "HTTP in-flight",
          ts("jimu_http_requests_inflight", "max(value)", "value"),
          "jimu_http_requests_inflight", "metrics", {"x": 0, "y": 35, "w": 12, "h": 9, "i": 11},
          x_aliases=("_time",), y_aliases=("value",)),
    panel("jimu-ts-circuit", "time-series", "出站熔断状态",
          ts("jimu_httpclient_circuit_open", "max(value)", "value"),
          "jimu_httpclient_circuit_open", "metrics", {"x": 12, "y": 35, "w": 12, "h": 9, "i": 12},
          x_aliases=("_time",), y_aliases=("value",)),
    # 行5: MySQL / Redis（数据来自 otel-collector）
    panel("db-ts-mysql-threads", "time-series", "MySQL 线程数",
          ts("mysql_threads", "max(value)", "value"),
          "mysql_threads", "metrics", {"x": 0, "y": 53, "w": 12, "h": 9, "i": 14},
          x_aliases=("_time",), y_aliases=("value",)),
    panel("db-ts-redis-clients", "time-series", "Redis 客户端连接",
          ts("redis_clients_connected", "max(value)", "value"),
          "redis_clients_connected", "metrics", {"x": 12, "y": 53, "w": 12, "h": 9, "i": 15},
          x_aliases=("_time",), y_aliases=("value",)),
    # 行6: MySQL / Redis 内存与吞吐
    panel("db-ts-redis-memory", "time-series", "Redis 内存使用",
          ts("redis_memory_used", "max(value)", "value"),
          "redis_memory_used", "metrics", {"x": 0, "y": 62, "w": 12, "h": 9, "i": 16},
          x_aliases=("_time",), y_aliases=("value",)),
    panel("db-ts-redis-commands", "time-series", "Redis 处理指令数",
          ts("redis_commands_processed", "max(value)", "value"),
          "redis_commands_processed", "metrics", {"x": 12, "y": 62, "w": 12, "h": 9, "i": 17},
          x_aliases=("_time",), y_aliases=("value",)),
    # 行7: 最近错误日志表格
    panel("jimu-table-errors", "table", "最近错误日志",
          'SELECT _timestamp, severity, body, caller FROM "default" WHERE severity = \'error\' ORDER BY _timestamp DESC LIMIT 20',
          "default", "logs", {"x": 0, "y": 71, "w": 24, "h": 11, "i": 18},
          x_aliases=("_timestamp", "severity", "body", "caller"),
          y_aliases=()),
]

st, resp = req("POST", "/dashboards", {
    "version": 8, "title": TITLE, "description": DESC, "folder_id": "default",
    "tabs": [{"tabId": "default", "name": "Default", "panels": []}],
})
if st not in (200, 201):
    sys.exit(f"dashboard create failed HTTP {st}: {resp}")
did = resp["v8"]["dashboardId"]
hashv = resp.get("hash", "")
print(f"  dashboardId={did}")

for p in panels:
    st, resp = req("POST", f"/dashboards/{did}/panels?hash={hashv}&folder=default",
                   {"panel": p, "tabId": "default"})
    if st not in (200, 201):
        print(f"  ✗ panel '{p['title']}' failed: HTTP {st} {resp[:200]}", file=sys.stderr)
        continue
    hashv = resp.get("hash", hashv)
    print(f"  + {p['type']:11s} {p['title']}")

print(f"✅ dashboard '{TITLE}' 初始化完成（{len(panels)} 面板），打开 {os.environ['ZO_HTTP']}/dashboards 查看")
PYEOF