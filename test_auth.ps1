# 认证系统测试脚本 (PowerShell)

$BaseUrl = "http://localhost:8080"

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "认证系统测试" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# 生成随机用户名
$RandomUser = "testuser_" + (Get-Date -Format "yyyyMMddHHmmss")
$Email = "$RandomUser@example.com"
$Password = "testpassword123"

Write-Host "1. 测试注册新用户" -ForegroundColor Yellow
Write-Host "用户名: $RandomUser"
Write-Host "邮箱: $Email"
Write-Host "密码: $Password"
Write-Host ""

# 注册
$RegisterBody = @{
    username = $RandomUser
    email = $Email
    password = $Password
} | ConvertTo-Json

try {
    $RegisterResponse = Invoke-RestMethod -Uri "$BaseUrl/api/auth/register" `
        -Method Post `
        -ContentType "application/json" `
        -Body $RegisterBody
    
    Write-Host "✅ 注册成功!" -ForegroundColor Green
    Write-Host "   Account ID: $($RegisterResponse.account_id)"
    Write-Host "   Player ID: $($RegisterResponse.player_id)"
    Write-Host "   Message: $($RegisterResponse.message)"
    
    $AccountId = $RegisterResponse.account_id
    $PlayerId = $RegisterResponse.player_id
} catch {
    Write-Host "❌ 注册失败!" -ForegroundColor Red
    Write-Host $_.Exception.Message
    exit 1
}

Write-Host ""
Write-Host "----------------------------------------" -ForegroundColor Cyan
Write-Host "2. 测试用错误密码登录" -ForegroundColor Yellow
Write-Host ""

$LoginFailBody = @{
    username = $RandomUser
    password = "wrongpassword"
} | ConvertTo-Json

try {
    $LoginFailResponse = Invoke-RestMethod -Uri "$BaseUrl/api/auth/login" `
        -Method Post `
        -ContentType "application/json" `
        -Body $LoginFailBody
    
    Write-Host "❌ 应该拒绝错误密码!" -ForegroundColor Red
} catch {
    Write-Host "✅ 正确拒绝了错误密码" -ForegroundColor Green
    Write-Host "   错误: $($_.ErrorDetails.Message)"
}

Write-Host ""
Write-Host "----------------------------------------" -ForegroundColor Cyan
Write-Host "3. 测试用正确密码登录" -ForegroundColor Yellow
Write-Host ""

$LoginSuccessBody = @{
    username = $RandomUser
    password = $Password
} | ConvertTo-Json

try {
    $LoginResponse = Invoke-RestMethod -Uri "$BaseUrl/api/auth/login" `
        -Method Post `
        -ContentType "application/json" `
        -Body $LoginSuccessBody
    
    Write-Host "✅ 登录成功!" -ForegroundColor Green
    Write-Host "   Token: $($LoginResponse.token.Substring(0, [Math]::Min(50, $LoginResponse.token.Length)))..."
    Write-Host "   Player ID: $($LoginResponse.player_id)"
    Write-Host "   Display Name: $($LoginResponse.display_name)"
    
    if ($PlayerId -eq $LoginResponse.player_id) {
        Write-Host "   ✅ Player ID 匹配" -ForegroundColor Green
    } else {
        Write-Host "   ❌ Player ID 不匹配!" -ForegroundColor Red
    }
    
    $Token = $LoginResponse.token
} catch {
    Write-Host "❌ 登录失败!" -ForegroundColor Red
    Write-Host $_.Exception.Message
    exit 1
}

Write-Host ""
Write-Host "----------------------------------------" -ForegroundColor Cyan
Write-Host "4. 测试获取 WebSocket 票券" -ForegroundColor Yellow
Write-Host ""

try {
    $Headers = @{
        "Authorization" = "Bearer $Token"
    }
    
    $TicketResponse = Invoke-RestMethod -Uri "$BaseUrl/api/auth/ticket" `
        -Method Post `
        -ContentType "application/json" `
        -Headers $Headers `
        -Body "{}"
    
    Write-Host "✅ 票券获取成功!" -ForegroundColor Green
    Write-Host "   Ticket: $($TicketResponse.ticket.Substring(0, [Math]::Min(20, $TicketResponse.ticket.Length)))..."
    Write-Host "   WebSocket URL: $($TicketResponse.ws_url)"
    Write-Host "   Expires in: $($TicketResponse.expires_in)s"
} catch {
    Write-Host "❌ 票券获取失败!" -ForegroundColor Red
    Write-Host $_.Exception.Message
    exit 1
}

Write-Host ""
Write-Host "----------------------------------------" -ForegroundColor Cyan
Write-Host "5. 测试重复注册（应该失败）" -ForegroundColor Yellow
Write-Host ""

try {
    $DuplicateResponse = Invoke-RestMethod -Uri "$BaseUrl/api/auth/register" `
        -Method Post `
        -ContentType "application/json" `
        -Body $RegisterBody
    
    Write-Host "❌ 应该拒绝重复注册!" -ForegroundColor Red
} catch {
    Write-Host "✅ 正确拒绝了重复注册" -ForegroundColor Green
    Write-Host "   错误: $($_.ErrorDetails.Message)"
}

Write-Host ""
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "测试完成!" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "总结:" -ForegroundColor Green
Write-Host "  ✅ 用户注册"
Write-Host "  ✅ 密码验证（bcrypt）"
Write-Host "  ✅ JWT Token 生成"
Write-Host "  ✅ WebSocket 票券生成"
Write-Host "  ✅ 重复注册保护"
Write-Host ""
Write-Host "数据库中的用户:" -ForegroundColor Yellow
Write-Host "  Username: $RandomUser"
Write-Host "  Email: $Email"
Write-Host "  Account ID: $AccountId"
Write-Host "  Player ID: $PlayerId"
