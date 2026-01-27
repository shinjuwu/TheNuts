# WebSocket Game Flow Test Script
# Tests the complete flow: register → login → get ticket → connect WebSocket → buy-in → join table → cash-out

$ErrorActionPreference = "Stop"
$baseUrl = "http://localhost:8080"

Write-Host "===================================" -ForegroundColor Cyan
Write-Host "WebSocket Game Flow Test" -ForegroundColor Cyan
Write-Host "===================================" -ForegroundColor Cyan
Write-Host ""

# Generate unique username
$timestamp = Get-Date -Format "yyyyMMddHHmmss"
$username = "testplayer_$timestamp"
$password = "Test123456!"

Write-Host "Step 1: Register new user" -ForegroundColor Yellow
Write-Host "Username: $username" -ForegroundColor Gray
$registerBody = @{
    username = $username
    password = $password
} | ConvertTo-Json

try {
    $registerResponse = Invoke-RestMethod -Uri "$baseUrl/api/auth/register" `
        -Method Post `
        -Body $registerBody `
        -ContentType "application/json"
    Write-Host "✓ Registration successful" -ForegroundColor Green
    Write-Host "Player ID: $($registerResponse.player_id)" -ForegroundColor Gray
    Write-Host ""
} catch {
    Write-Host "✗ Registration failed: $_" -ForegroundColor Red
    exit 1
}

Start-Sleep -Seconds 1

Write-Host "Step 2: Login" -ForegroundColor Yellow
$loginBody = @{
    username = $username
    password = $password
} | ConvertTo-Json

try {
    $loginResponse = Invoke-RestMethod -Uri "$baseUrl/api/auth/login" `
        -Method Post `
        -Body $loginBody `
        -ContentType "application/json"
    Write-Host "✓ Login successful" -ForegroundColor Green
    Write-Host "Token: $($loginResponse.token.Substring(0, 20))..." -ForegroundColor Gray
    $token = $loginResponse.token
    Write-Host ""
} catch {
    Write-Host "✗ Login failed: $_" -ForegroundColor Red
    exit 1
}

Start-Sleep -Seconds 1

Write-Host "Step 3: Get WebSocket ticket" -ForegroundColor Yellow
try {
    $headers = @{
        "Authorization" = "Bearer $token"
    }
    $ticketResponse = Invoke-RestMethod -Uri "$baseUrl/api/auth/ticket" `
        -Method Post `
        -Headers $headers `
        -ContentType "application/json"
    Write-Host "✓ Ticket obtained" -ForegroundColor Green
    Write-Host "Ticket: $($ticketResponse.ticket.Substring(0, 20))..." -ForegroundColor Gray
    $ticket = $ticketResponse.ticket
    Write-Host ""
} catch {
    Write-Host "✗ Failed to get ticket: $_" -ForegroundColor Red
    exit 1
}

Write-Host "Step 4: Connect to WebSocket" -ForegroundColor Yellow
Write-Host "URL: ws://localhost:8080/ws?ticket=$ticket" -ForegroundColor Gray
Write-Host ""

# Create Node.js WebSocket test client
$wsTestScript = @"
const WebSocket = require('ws');

const ticket = process.argv[2];
const ws = new WebSocket('ws://localhost:8080/ws?ticket=' + ticket);

let connected = false;

ws.on('open', function open() {
    console.log('✓ WebSocket connected');
    connected = true;
    
    // Step 5: Buy-in with 100,000 chips
    console.log('\n--- Step 5: Buy-in (100,000 chips) ---');
    const buyInMsg = {
        action: 'BUY_IN',
        amount: 100000
    };
    ws.send(JSON.stringify(buyInMsg));
});

ws.on('message', function message(data) {
    try {
        const msg = JSON.parse(data);
        console.log('Received:', JSON.stringify(msg, null, 2));
        
        // Handle buy-in response
        if (msg.type === 'BUY_IN_SUCCESS') {
            console.log('✓ Buy-in successful!');
            console.log('Current chips:', msg.chips);
            
            // Step 6: Join table
            setTimeout(() => {
                console.log('\n--- Step 6: Join Table T1, Seat 3 ---');
                const joinMsg = {
                    action: 'JOIN_TABLE',
                    table_id: 'T1',
                    seat_no: 3
                };
                ws.send(JSON.stringify(joinMsg));
            }, 1000);
        }
        
        // Handle join table response
        if (msg.type === 'TABLE_STATE') {
            console.log('✓ Joined table successfully!');
            console.log('Table ID:', msg.table_id);
            console.log('Players:', msg.players);
            
            // Step 7: Get balance
            setTimeout(() => {
                console.log('\n--- Step 7: Get Balance ---');
                const balanceMsg = {
                    action: 'GET_BALANCE'
                };
                ws.send(JSON.stringify(balanceMsg));
            }, 1000);
        }
        
        // Handle balance response
        if (msg.type === 'BALANCE_INFO') {
            console.log('✓ Balance retrieved!');
            console.log('Wallet Balance:', msg.wallet_balance);
            console.log('Current Chips:', msg.current_chips);
            
            // Step 8: Cash out
            setTimeout(() => {
                console.log('\n--- Step 8: Cash Out ---');
                const cashOutMsg = {
                    action: 'CASH_OUT'
                };
                ws.send(JSON.stringify(cashOutMsg));
            }, 1000);
        }
        
        // Handle cash out response
        if (msg.type === 'CASH_OUT_SUCCESS') {
            console.log('✓ Cash out successful!');
            console.log('Final wallet balance:', msg.wallet_balance);
            console.log('Profit/Loss:', msg.profit_loss);
            
            // Close connection
            setTimeout(() => {
                console.log('\n--- Test Complete ---');
                ws.close();
                process.exit(0);
            }, 1000);
        }
        
        if (msg.type === 'ERROR') {
            console.error('✗ Error:', msg.message);
            ws.close();
            process.exit(1);
        }
    } catch (e) {
        console.log('Raw message:', data.toString());
    }
});

ws.on('error', function error(err) {
    console.error('✗ WebSocket error:', err.message);
    process.exit(1);
});

ws.on('close', function close() {
    if (connected) {
        console.log('WebSocket connection closed');
    }
});

// Timeout after 30 seconds
setTimeout(() => {
    console.error('✗ Test timeout');
    ws.close();
    process.exit(1);
}, 30000);
"@

$wsTestScript | Out-File -FilePath "test_ws_client.js" -Encoding UTF8

Write-Host "Step 5-8: Running WebSocket test client..." -ForegroundColor Yellow
Write-Host ""

try {
    node test_ws_client.js $ticket
} catch {
    Write-Host "✗ WebSocket test failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "===================================" -ForegroundColor Cyan
Write-Host "All Tests Completed Successfully!" -ForegroundColor Green
Write-Host "===================================" -ForegroundColor Cyan
