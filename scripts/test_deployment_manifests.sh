#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HELM_CHART="$ROOT/deploy/helm"

fail() {
  echo "❌ $*" >&2
  exit 1
}

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
for file in \
  "$HELM_CHART/templates/otel-collector.yaml" \
  "$ROOT/deploy/k8s/otel-collector.yaml"; do
  assert_contains "$file" "/run/secrets/zo_auth_token" "collector 配置未读取 zo_auth_token"
  assert_contains "$file" "mountPath: /run/secrets/zo_auth_token" "collector 未挂载 zo_auth_token"
  assert_contains "$file" "subPath: zo_auth_token" "collector 挂载缺少 zo_auth_token subPath"
done

echo "== Helm 观测凭据条件渲染 =="
assert_contains "$HELM_CHART/templates/configmap.yaml" '{{- if .Values.otel.enabled }}' "otel 密码 required 未按开关条件化"
assert_contains "$HELM_CHART/templates/secret.yaml" '{{- if or .Values.otel.enabled .Values.otelCollector.enabled }}' "zo_auth_token 未按观测开关条件化"
assert_contains "$HELM_CHART/templates/openobserve.yaml" '{{- if .Values.openobserve.enabled }}' "OpenObserve 资源未按开关条件化"

if command -v helm >/dev/null 2>&1; then
  rendered="$(helm template review "$HELM_CHART" \
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
