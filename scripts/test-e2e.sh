#!/bin/bash
#
# 端到端测试脚本 — 模拟完整告警→工单→推送→授权流程
#
# 使用方法：
#   ./scripts/test-e2e.sh              # 完整测试
#   ./scripts/test-e2e.sh --step 4     # 只执行第 4 步
#

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
echo "========================================="
echo " 网络工单平台 — 端到端测试"
echo " API: $BASE_URL"
echo "========================================="

API_KEY="ak_test_client_001"
HMAC_SECRET="hs_test_secret_key_32chars_long!!"

# ---------------------------------------------------------------------------
# Helper: 登录获取 token
# ---------------------------------------------------------------------------
get_token() {
  curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{"username":"admin","password":"admin123"}' | \
    python3 -c "import sys,json; print(json.load(sys.stdin)['token'])"
}

# ---------------------------------------------------------------------------
# Helper: 计算 HMAC 签名
#   Go 后端: hmac.Write(timestamp_string) + hmac.Write(body_bytes)
#   即: HMAC-SHA256(secret, timestamp_string + body_bytes)
# ---------------------------------------------------------------------------
compute_hmac() {
  local secret="$1" timestamp="$2" body="$3"
  printf '%s' "${timestamp}${body}" | openssl dgst -sha256 -hmac "$secret" -hex 2>/dev/null | awk '{print $NF}'
}

# ---------------------------------------------------------------------------
# Step 1: 登录
# ---------------------------------------------------------------------------
step_login() {
  echo ""
  echo "=== Step 1: 登录 ==="
  TOKEN=$(get_token)
  echo "Token: ${TOKEN:0:30}..."
}

# ---------------------------------------------------------------------------
# Step 2: 创建告警源
# ---------------------------------------------------------------------------
step_create_source() {
  echo ""
  echo "=== Step 2: 创建告警源 ==="
  SOURCE_RESP=$(curl -s -X POST "$BASE_URL/api/v1/alert-sources" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d '{"name":"测试告警源","type":"generic"}')
  echo "$SOURCE_RESP" | python3 -m json.tool 2>/dev/null || echo "$SOURCE_RESP"

  SOURCE_ID=$(echo "$SOURCE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || echo "")
  if [ -z "$SOURCE_ID" ]; then
    echo "告警源可能已存在，获取已有..."
    SOURCE_ID=$(curl -s "$BASE_URL/api/v1/alert-sources" \
      -H "Authorization: Bearer $TOKEN" | \
      python3 -c "import sys,json; items=json.load(sys.stdin).get('items',[]); print(items[0]['id'] if items else '')" 2>/dev/null)
  fi
  echo "Source ID: $SOURCE_ID"
}

# ---------------------------------------------------------------------------
# Step 3: 创建客户（模拟客户系统）
# ---------------------------------------------------------------------------
step_create_client() {
  echo ""
  echo "=== Step 3: 创建客户 ==="
  CLIENT_RESP=$(curl -s -X POST "$BASE_URL/api/v1/clients" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{
      \"name\": \"测试客户系统\",
      \"api_endpoint\": \"http://host.docker.internal:9999/ticket\",
      \"api_key\": \"$API_KEY\",
      \"hmac_secret\": \"$HMAC_SECRET\",
      \"status\": \"active\"
    }")
  echo "$CLIENT_RESP" | python3 -m json.tool 2>/dev/null || echo "$CLIENT_RESP"
  CLIENT_ID=$(echo "$CLIENT_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || echo "")
  if [ -z "$CLIENT_ID" ]; then
    echo "客户可能已存在，获取已有..."
    CLIENT_ID=$(curl -s "$BASE_URL/api/v1/clients" \
      -H "Authorization: Bearer $TOKEN" | \
      python3 -c "import sys,json; items=json.load(sys.stdin).get('items',[]); print(items[0]['id'] if items else '')" 2>/dev/null)
  fi
  echo "Client ID: $CLIENT_ID"
}

# ---------------------------------------------------------------------------
# Step 4: 发送告警 → 自动创建工单
# ---------------------------------------------------------------------------
step_send_alert() {
  echo ""
  echo "=== Step 4: 发送告警 (Webhook) ==="
  if [ -z "${SOURCE_ID:-}" ]; then
    echo "没有 Source ID，跳过"
    return
  fi
  ALERT_RESP=$(curl -s -X POST "$BASE_URL/api/v1/alerts/webhook/$SOURCE_ID" \
    -H 'Content-Type: application/json' \
    -d '{
      "title": "CPU 使用率过高 - server-01",
      "description": "server-01 CPU 使用率持续超过 90%，已持续 5 分钟",
      "severity": "critical",
      "source_ip": "192.168.1.100",
      "device_name": "server-01"
    }')
  echo "$ALERT_RESP" | python3 -m json.tool 2>/dev/null || echo "$ALERT_RESP"
  TICKET_NO=$(echo "$ALERT_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('ticket_no',''))" 2>/dev/null || echo "")
  echo "Ticket No: $TICKET_NO"
}

# ---------------------------------------------------------------------------
# Step 5: 手动建单
# ---------------------------------------------------------------------------
step_manual_ticket() {
  echo ""
  echo "=== Step 5: 手动建单 ==="
  TT_ID=$(curl -s "$BASE_URL/api/v1/ticket-types" \
    -H "Authorization: Bearer $TOKEN" | \
    python3 -c "import sys,json; items=json.load(sys.stdin).get('items',[]); print(items[0]['id'] if items else 1)" 2>/dev/null)

  local client_json="${CLIENT_ID:+\"client_id\": $CLIENT_ID,}"
  MANUAL_RESP=$(curl -s -X POST "$BASE_URL/api/v1/tickets/manual" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{
      \"title\": \"链路中断 - 核心交换机\",
      \"description\": \"核心交换机 core-switch-01 端口 GigabitEthernet0/1 状态 down\",
      \"severity\": \"critical\",
      \"ticket_type_id\": $TT_ID,
      ${client_json}
      \"description_detail\": \"\"
    }")
  echo "$MANUAL_RESP" | python3 -m json.tool 2>/dev/null || echo "$MANUAL_RESP"
  MANUAL_TICKET_NO=$(echo "$MANUAL_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('ticket_no',''))" 2>/dev/null || echo "")
  echo "Manual Ticket No: $MANUAL_TICKET_NO"
}

# ---------------------------------------------------------------------------
# Step 6: 模拟客户回调 — 授权
# ---------------------------------------------------------------------------
step_callback_authorize() {
  echo ""
  echo "=== Step 6: 客户回调 — 授权 ==="
  local ticket_no="${MANUAL_TICKET_NO:-${TICKET_NO:-}}"
  if [ -z "$ticket_no" ]; then
    echo "没有可用的工单号，跳过"
    return
  fi

  local callback_body
  callback_body="{\"ticket_no\":\"$ticket_no\",\"action\":\"authorize\",\"operator\":\"张三\",\"comment\":\"已确认，开始处理\"}"

  local timestamp
  timestamp=$(date +%s)
  local signature
  signature=$(compute_hmac "$HMAC_SECRET" "$timestamp" "$callback_body")
  local nonce
  nonce=$(uuidgen 2>/dev/null || echo "test-nonce-$$")

  echo "  工单: $ticket_no"
  echo "  签名: $signature"
  echo ""

  CB_RESP=$(curl -s -X POST "$BASE_URL/api/v1/callback/authorization" \
    -H "Content-Type: application/json" \
    -H "X-Api-Key: $API_KEY" \
    -H "X-Timestamp: $timestamp" \
    -H "X-Signature: $signature" \
    -H "X-Nonce: $nonce" \
    -d "$callback_body")
  echo "回调响应:"
  echo "$CB_RESP" | python3 -m json.tool 2>/dev/null || echo "$CB_RESP"
}

# ---------------------------------------------------------------------------
# Step 7: 查看工单最终状态
# ---------------------------------------------------------------------------
step_check_status() {
  echo ""
  echo "=== Step 7: 查看工单最终状态 ==="
  curl -s "$BASE_URL/api/v1/tickets" \
    -H "Authorization: Bearer $TOKEN" | \
    python3 -c "
import sys, json
data = json.load(sys.stdin)
for t in data.get('items', [])[:5]:
    status_map = {
        'pending': '🟡 等待',
        'in_progress': '🔵 处理中',
        'completed': '🟢 已完成',
        'failed': '🔴 失败',
        'cancelled': '⚪ 已取消',
        'rejected': '🟠 已拒绝',
    }
    s = status_map.get(t['status'], t['status'])
    print(f\"  {t['ticket_no']}  {s}  {t['severity']:8s}  {t['title']}\")
" 2>/dev/null
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
STEP="${2:-all}"

case "${1:-}" in
  --step)
    TOKEN=$(get_token)
    case "$STEP" in
      1) step_login ;;
      2) step_create_source ;;
      3) step_create_client ;;
      4) step_send_alert ;;
      5) step_manual_ticket ;;
      6) step_callback_authorize ;;
      7) step_check_status ;;
    esac
    ;;
  *)
    step_login
    step_create_source
    step_create_client
    step_send_alert
    step_manual_ticket
    step_callback_authorize
    step_check_status
    echo ""
    echo "========================================="
    echo " 测试完成!"
    echo "========================================="
    ;;
esac
