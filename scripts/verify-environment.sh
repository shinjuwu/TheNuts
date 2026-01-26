#!/bin/bash

# TheNuts 环境验证脚本
# 用途：快速检查 Docker 环境是否正常运行

echo "🔍 TheNuts 环境验证开始..."
echo "================================"
echo ""

# 检查 Docker 是否运行
echo "1️⃣ 检查 Docker..."
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker 未运行，请先启动 Docker Desktop"
    exit 1
fi
echo "✅ Docker 运行正常"
echo ""

# 检查容器状态
echo "2️⃣ 检查容器状态..."
CONTAINERS=("thenuts-postgres" "thenuts-redis" "thenuts-pgadmin" "thenuts-redis-commander")
for container in "${CONTAINERS[@]}"; do
    if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
        STATUS=$(docker inspect --format='{{.State.Health.Status}}' ${container} 2>/dev/null || echo "running")
        if [ "$STATUS" = "healthy" ] || [ "$STATUS" = "running" ]; then
            echo "  ✅ ${container}: ${STATUS}"
        else
            echo "  ⚠️  ${container}: ${STATUS}"
        fi
    else
        echo "  ❌ ${container}: not running"
    fi
done
echo ""

# 检查 PostgreSQL
echo "3️⃣ 检查 PostgreSQL 连接..."
if docker exec thenuts-postgres psql -U thenuts -d thenuts -c "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ PostgreSQL 连接成功"
    
    # 检查表数量
    TABLE_COUNT=$(docker exec thenuts-postgres psql -U thenuts -d thenuts -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';" | tr -d ' ')
    echo "  📊 数据库表数量: ${TABLE_COUNT}"
    
    # 检查管理员账号
    if docker exec thenuts-postgres psql -U thenuts -d thenuts -t -c "SELECT username FROM accounts WHERE username='admin';" | grep -q "admin"; then
        echo "  👤 管理员账号: 已创建"
    else
        echo "  ⚠️  管理员账号: 未找到"
    fi
else
    echo "❌ PostgreSQL 连接失败"
fi
echo ""

# 检查 Redis
echo "4️⃣ 检查 Redis 连接..."
if docker exec thenuts-redis redis-cli ping > /dev/null 2>&1; then
    echo "✅ Redis 连接成功"
    
    # 检查内存使用
    REDIS_MEMORY=$(docker exec thenuts-redis redis-cli info memory | grep "used_memory_human" | cut -d: -f2 | tr -d '\r')
    echo "  💾 内存使用: ${REDIS_MEMORY}"
else
    echo "❌ Redis 连接失败"
fi
echo ""

# 检查端口
echo "5️⃣ 检查端口..."
PORTS=("5432:PostgreSQL" "6382:Redis" "5050:pgAdmin" "8081:Redis-Commander")
for port_info in "${PORTS[@]}"; do
    PORT=$(echo $port_info | cut -d: -f1)
    NAME=$(echo $port_info | cut -d: -f2)
    if netstat -ano 2>/dev/null | grep -q ":${PORT} " || ss -tuln 2>/dev/null | grep -q ":${PORT} "; then
        echo "  ✅ ${PORT} (${NAME}): 正在监听"
    else
        echo "  ⚠️  ${PORT} (${NAME}): 未监听"
    fi
done
echo ""

# 检查数据卷
echo "6️⃣ 检查数据卷..."
VOLUMES=("thenuts_postgres_data" "thenuts_redis_data" "thenuts_pgadmin_data")
for volume in "${VOLUMES[@]}"; do
    if docker volume ls | grep -q "${volume}"; then
        echo "  ✅ ${volume}"
    else
        echo "  ❌ ${volume}: 不存在"
    fi
done
echo ""

# 最终总结
echo "================================"
echo "🎉 环境验证完成！"
echo ""
echo "📋 访问地址:"
echo "  • PostgreSQL: localhost:5432"
echo "  • Redis:      localhost:6382"
echo "  • pgAdmin:    http://localhost:5050"
echo "  • Redis-Cmd:  http://localhost:8081"
echo ""
echo "🔐 默认凭证:"
echo "  • PostgreSQL: thenuts / devpassword"
echo "  • pgAdmin:    admin@thenuts.com / admin"
echo ""
