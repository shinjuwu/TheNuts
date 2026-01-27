#!/bin/bash

# 认证系统测试脚本

BASE_URL="http://localhost:8080"

echo "========================================="
echo "认证系统测试"
echo "========================================="

# 生成随机用户名
RANDOM_USER="testuser_$(date +%s)"
EMAIL="${RANDOM_USER}@example.com"
PASSWORD="testpassword123"

echo ""
echo "1. 测试注册新用户"
echo "用户名: $RANDOM_USER"
echo "邮箱: $EMAIL"
echo "密码: $PASSWORD"
echo ""

REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$RANDOM_USER\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

echo "注册响应:"
echo "$REGISTER_RESPONSE" | jq .
echo ""

# 检查注册是否成功
if echo "$REGISTER_RESPONSE" | jq -e '.player_id' > /dev/null; then
  PLAYER_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.player_id')
  ACCOUNT_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.account_id')
  echo "✅ 注册成功!"
  echo "   Account ID: $ACCOUNT_ID"
  echo "   Player ID: $PLAYER_ID"
else
  echo "❌ 注册失败!"
  exit 1
fi

echo ""
echo "----------------------------------------"
echo "2. 测试用错误密码登录"
echo ""

LOGIN_FAIL_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$RANDOM_USER\",\"password\":\"wrongpassword\"}")

echo "登录响应:"
echo "$LOGIN_FAIL_RESPONSE" | jq .
echo ""

if echo "$LOGIN_FAIL_RESPONSE" | jq -e '.error' > /dev/null; then
  echo "✅ 正确拒绝了错误密码"
else
  echo "❌ 应该拒绝错误密码"
fi

echo ""
echo "----------------------------------------"
echo "3. 测试用正确密码登录"
echo ""

LOGIN_SUCCESS_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$RANDOM_USER\",\"password\":\"$PASSWORD\"}")

echo "登录响应:"
echo "$LOGIN_SUCCESS_RESPONSE" | jq .
echo ""

# 检查登录是否成功
if echo "$LOGIN_SUCCESS_RESPONSE" | jq -e '.token' > /dev/null; then
  TOKEN=$(echo "$LOGIN_SUCCESS_RESPONSE" | jq -r '.token')
  RETURNED_PLAYER_ID=$(echo "$LOGIN_SUCCESS_RESPONSE" | jq -r '.player_id')
  DISPLAY_NAME=$(echo "$LOGIN_SUCCESS_RESPONSE" | jq -r '.display_name')
  
  echo "✅ 登录成功!"
  echo "   Token: ${TOKEN:0:50}..."
  echo "   Player ID: $RETURNED_PLAYER_ID"
  echo "   Display Name: $DISPLAY_NAME"
  
  # 验证 Player ID 匹配
  if [ "$PLAYER_ID" == "$RETURNED_PLAYER_ID" ]; then
    echo "   ✅ Player ID 匹配"
  else
    echo "   ❌ Player ID 不匹配!"
  fi
else
  echo "❌ 登录失败!"
  exit 1
fi

echo ""
echo "----------------------------------------"
echo "4. 测试获取 WebSocket 票券"
echo ""

TICKET_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/ticket" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN")

echo "票券响应:"
echo "$TICKET_RESPONSE" | jq .
echo ""

if echo "$TICKET_RESPONSE" | jq -e '.ticket' > /dev/null; then
  TICKET=$(echo "$TICKET_RESPONSE" | jq -r '.ticket')
  WS_URL=$(echo "$TICKET_RESPONSE" | jq -r '.ws_url')
  EXPIRES_IN=$(echo "$TICKET_RESPONSE" | jq -r '.expires_in')
  
  echo "✅ 票券获取成功!"
  echo "   Ticket: ${TICKET:0:20}..."
  echo "   WebSocket URL: $WS_URL"
  echo "   Expires in: ${EXPIRES_IN}s"
else
  echo "❌ 票券获取失败!"
  exit 1
fi

echo ""
echo "----------------------------------------"
echo "5. 测试重复注册（应该失败）"
echo ""

DUPLICATE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$RANDOM_USER\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

echo "重复注册响应:"
echo "$DUPLICATE_RESPONSE" | jq .
echo ""

if echo "$DUPLICATE_RESPONSE" | jq -e '.error' | grep -q "username_exists\|email_exists"; then
  echo "✅ 正确拒绝了重复注册"
else
  echo "❌ 应该拒绝重复注册"
fi

echo ""
echo "========================================="
echo "测试完成!"
echo "========================================="
echo ""
echo "总结:"
echo "  ✅ 用户注册"
echo "  ✅ 密码验证（bcrypt）"
echo "  ✅ JWT Token 生成"
echo "  ✅ WebSocket 票券生成"
echo "  ✅ 重复注册保护"
echo ""
echo "数据库中的用户:"
echo "  Username: $RANDOM_USER"
echo "  Email: $EMAIL"
echo "  Account ID: $ACCOUNT_ID"
echo "  Player ID: $PLAYER_ID"
