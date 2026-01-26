@echo off
REM TheNuts 环境验证脚本 (Windows)
REM 用途：快速检查 Docker 环境是否正常运行

echo.
echo 🔍 TheNuts 环境验证开始...
echo ================================
echo.

REM 检查 Docker 是否运行
echo 1️⃣  检查 Docker...
docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Docker 未运行，请先启动 Docker Desktop
    exit /b 1
)
echo ✅ Docker 运行正常
echo.

REM 检查容器状态
echo 2️⃣  检查容器状态...
for %%c in (thenuts-postgres thenuts-redis thenuts-pgadmin thenuts-redis-commander) do (
    docker ps --format "{{.Names}}" | findstr /c:"%%c" >nul 2>&1
    if !errorlevel! equ 0 (
        echo   ✅ %%c: running
    ) else (
        echo   ❌ %%c: not running
    )
)
echo.

REM 检查 PostgreSQL
echo 3️⃣  检查 PostgreSQL 连接...
docker exec thenuts-postgres psql -U thenuts -d thenuts -c "SELECT 1;" >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ PostgreSQL 连接成功
    
    REM 检查表数量
    for /f "delims=" %%i in ('docker exec thenuts-postgres psql -U thenuts -d thenuts -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';"') do set TABLE_COUNT=%%i
    echo   📊 数据库表数量: %TABLE_COUNT%
    
    REM 检查管理员账号
    docker exec thenuts-postgres psql -U thenuts -d thenuts -t -c "SELECT username FROM accounts WHERE username='admin';" | findstr "admin" >nul 2>&1
    if !errorlevel! equ 0 (
        echo   👤 管理员账号: 已创建
    ) else (
        echo   ⚠️  管理员账号: 未找到
    )
) else (
    echo ❌ PostgreSQL 连接失败
)
echo.

REM 检查 Redis
echo 4️⃣  检查 Redis 连接...
docker exec thenuts-redis redis-cli ping >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ Redis 连接成功
) else (
    echo ❌ Redis 连接失败
)
echo.

REM 检查端口
echo 5️⃣  检查端口...
netstat -ano | findstr ":5432 " >nul 2>&1
if %errorlevel% equ 0 (
    echo   ✅ 5432 ^(PostgreSQL^): 正在监听
) else (
    echo   ⚠️  5432 ^(PostgreSQL^): 未监听
)

netstat -ano | findstr ":6382 " >nul 2>&1
if %errorlevel% equ 0 (
    echo   ✅ 6382 ^(Redis^): 正在监听
) else (
    echo   ⚠️  6382 ^(Redis^): 未监听
)

netstat -ano | findstr ":5050 " >nul 2>&1
if %errorlevel% equ 0 (
    echo   ✅ 5050 ^(pgAdmin^): 正在监听
) else (
    echo   ⚠️  5050 ^(pgAdmin^): 未监听
)

netstat -ano | findstr ":8081 " >nul 2>&1
if %errorlevel% equ 0 (
    echo   ✅ 8081 ^(Redis-Commander^): 正在监听
) else (
    echo   ⚠️  8081 ^(Redis-Commander^): 未监听
)
echo.

REM 检查数据卷
echo 6️⃣  检查数据卷...
docker volume ls | findstr "thenuts_postgres_data" >nul 2>&1
if %errorlevel% equ 0 (
    echo   ✅ thenuts_postgres_data
) else (
    echo   ❌ thenuts_postgres_data: 不存在
)

docker volume ls | findstr "thenuts_redis_data" >nul 2>&1
if %errorlevel% equ 0 (
    echo   ✅ thenuts_redis_data
) else (
    echo   ❌ thenuts_redis_data: 不存在
)

docker volume ls | findstr "thenuts_pgadmin_data" >nul 2>&1
if %errorlevel% equ 0 (
    echo   ✅ thenuts_pgadmin_data
) else (
    echo   ❌ thenuts_pgadmin_data: 不存在
)
echo.

REM 最终总结
echo ================================
echo 🎉 环境验证完成！
echo.
echo 📋 访问地址:
echo   • PostgreSQL: localhost:5432
echo   • Redis:      localhost:6382
echo   • pgAdmin:    http://localhost:5050
echo   • Redis-Cmd:  http://localhost:8081
echo.
echo 🔐 默认凭证:
echo   • PostgreSQL: thenuts / devpassword
echo   • pgAdmin:    admin@thenuts.com / admin
echo.
pause
