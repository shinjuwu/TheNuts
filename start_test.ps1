# TheNuts 认证系统快速测试启动脚本

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "TheNuts 认证系统测试启动" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# 1. 检查 Docker 是否运行
Write-Host "[1/6] 检查 Docker 状态..." -ForegroundColor Yellow
$dockerRunning = docker ps 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker 未运行，请先启动 Docker Desktop" -ForegroundColor Red
    exit 1
}
Write-Host "✅ Docker 正在运行" -ForegroundColor Green
Write-Host ""

# 2. 启动数据库服务
Write-Host "[2/6] 启动数据库服务..." -ForegroundColor Yellow
docker-compose up -d
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ 启动数据库失败" -ForegroundColor Red
    exit 1
}
Write-Host "✅ 数据库服务已启动" -ForegroundColor Green
Write-Host ""

# 3. 等待数据库就绪
Write-Host "[3/6] 等待数据库就绪..." -ForegroundColor Yellow
Start-Sleep -Seconds 10
$dbReady = docker exec thenuts-postgres pg_isready -U thenuts 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ 数据库就绪" -ForegroundColor Green
} else {
    Write-Host "⚠️  数据库可能还未完全启动，但继续进行..." -ForegroundColor Yellow
    Start-Sleep -Seconds 5
}
Write-Host ""

# 4. 创建测试用户
Write-Host "[4/6] 创建测试用户 (testuser1)..." -ForegroundColor Yellow
Get-Content scripts/create_test_user.sql | docker exec -i thenuts-postgres psql -U thenuts -d thenuts 2>&1 | Out-Null
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ 测试用户已创建" -ForegroundColor Green
    Write-Host "   Username: testuser1" -ForegroundColor Cyan
    Write-Host "   Password: password123" -ForegroundColor Cyan
    Write-Host "   Balance: $1000.00" -ForegroundColor Cyan
} else {
    Write-Host "⚠️  测试用户可能已存在" -ForegroundColor Yellow
}
Write-Host ""

# 5. 编译服务器
Write-Host "[5/6] 编译游戏服务器..." -ForegroundColor Yellow
go build -o game-server.exe ./cmd/game-server
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ 编译失败" -ForegroundColor Red
    exit 1
}
Write-Host "✅ 编译成功" -ForegroundColor Green
Write-Host ""

# 6. 启动服务器（后台）
Write-Host "[6/6] 启动游戏服务器..." -ForegroundColor Yellow
$serverJob = Start-Job -ScriptBlock {
    Set-Location $using:PWD
    ./game-server.exe
}
Start-Sleep -Seconds 3

# 检查服务器是否启动
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080" -TimeoutSec 2 -ErrorAction SilentlyContinue
    Write-Host "✅ 服务器已启动" -ForegroundColor Green
    Write-Host "   URL: http://localhost:8080" -ForegroundColor Cyan
} catch {
    Write-Host "⚠️  服务器可能还在启动中..." -ForegroundColor Yellow
}
Write-Host ""

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "环境准备完成！" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "可用的测试账号：" -ForegroundColor Yellow
Write-Host "  Username: testuser1" -ForegroundColor White
Write-Host "  Password: password123" -ForegroundColor White
Write-Host ""
Write-Host "API 端点：" -ForegroundColor Yellow
Write-Host "  POST http://localhost:8080/api/auth/register" -ForegroundColor White
Write-Host "  POST http://localhost:8080/api/auth/login" -ForegroundColor White
Write-Host "  POST http://localhost:8080/api/auth/ticket" -ForegroundColor White
Write-Host ""
Write-Host "管理界面：" -ForegroundColor Yellow
Write-Host "  pgAdmin:         http://localhost:5050" -ForegroundColor White
Write-Host "  Redis Commander: http://localhost:8081" -ForegroundColor White
Write-Host ""
Write-Host "运行测试：" -ForegroundColor Yellow
Write-Host "  ./test_auth.ps1" -ForegroundColor Cyan
Write-Host ""
Write-Host "查看日志：" -ForegroundColor Yellow
Write-Host "  Receive-Job $($serverJob.Id)" -ForegroundColor Cyan
Write-Host ""
Write-Host "停止服务器：" -ForegroundColor Yellow
Write-Host "  Stop-Job $($serverJob.Id)" -ForegroundColor Cyan
Write-Host "  docker-compose down" -ForegroundColor Cyan
Write-Host ""

# 询问是否运行测试
Write-Host "是否立即运行认证测试？(Y/N): " -NoNewline -ForegroundColor Yellow
$runTest = Read-Host

if ($runTest -eq "Y" -or $runTest -eq "y") {
    Write-Host ""
    Start-Sleep -Seconds 2
    & ./test_auth.ps1
    Write-Host ""
    Write-Host "测试完成！" -ForegroundColor Green
}

Write-Host ""
Write-Host "服务器正在后台运行 (Job ID: $($serverJob.Id))" -ForegroundColor Cyan
Write-Host "按任意键停止服务器..." -ForegroundColor Yellow
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

Write-Host ""
Write-Host "正在停止服务器..." -ForegroundColor Yellow
Stop-Job $serverJob
Remove-Job $serverJob

Write-Host "✅ 服务器已停止" -ForegroundColor Green
Write-Host ""
Write-Host "数据库服务仍在运行，使用以下命令停止：" -ForegroundColor Yellow
Write-Host "  docker-compose down" -ForegroundColor Cyan
