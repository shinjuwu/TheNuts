package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinjuwu/TheNuts/internal/infra/config"
	"github.com/shinjuwu/TheNuts/internal/infra/database"
	"github.com/shinjuwu/TheNuts/internal/infra/logger"
	"github.com/shinjuwu/TheNuts/internal/infra/repository"
	"github.com/shinjuwu/TheNuts/internal/infra/repository/postgres"
	"golang.org/x/crypto/bcrypt"
)

// setupTest 初始化測試環境
func setupTest(t *testing.T) (*database.PostgresDB, repository.UnitOfWork, *postgres.TransactionRepo, repository.AccountRepository, repository.PlayerRepository, repository.WalletRepository) {
	// 載入配置
	cfg, err := config.LoadConfig("../../../../../config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 創建 logger
	log, err := logger.NewLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// 創建資料庫連接
	ctx := context.Background()
	db, err := database.NewPostgresPool(ctx, cfg.Database.Postgres, log)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 創建 Repositories
	uow := postgres.NewUnitOfWork(db.Pool)
	txRepo := postgres.NewTransactionRepository(db.Pool)
	accountRepo := postgres.NewAccountRepository(db.Pool)
	playerRepo := postgres.NewPlayerRepository(db.Pool)
	walletRepo := postgres.NewWalletRepository(db.Pool, txRepo)

	return db, uow, txRepo, accountRepo, playerRepo, walletRepo
}

// cleanupTest 清理測試資料
func cleanupTest(t *testing.T, db *database.PostgresDB, playerID uuid.UUID) {
	ctx := context.Background()

	// 先獲取 account_id
	var accountID uuid.UUID
	err := db.Pool.QueryRow(ctx, "SELECT account_id FROM players WHERE id = $1", playerID).Scan(&accountID)
	if err != nil {
		t.Logf("Warning: Could not get account_id: %v", err)
	}

	// 刪除測試資料（按外鍵依賴順序）
	queries := []string{
		"DELETE FROM transactions WHERE wallet_id IN (SELECT id FROM wallets WHERE player_id = $1)",
		"DELETE FROM wallets WHERE player_id = $1",
		"DELETE FROM players WHERE id = $1",
	}

	for _, query := range queries {
		_, err := db.Pool.Exec(ctx, query, playerID)
		if err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
	}

	// 刪除 account
	if accountID != uuid.Nil {
		_, err = db.Pool.Exec(ctx, "DELETE FROM accounts WHERE id = $1", accountID)
		if err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
	}

	db.Close()
}

// TestFullUserFlow 測試完整的用戶流程：註冊 -> 買入 -> 遊戲 -> 兌現
func TestFullUserFlow(t *testing.T) {
	db, uow, _, accountRepo, playerRepo, walletRepo := setupTest(t)
	ctx := context.Background()

	// 1. 創建帳號
	t.Log("=== Step 1: 創建帳號 ===")
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	account := &repository.Account{
		ID:           uuid.New(),
		Username:     fmt.Sprintf("testuser_%d", time.Now().Unix()),
		Email:        fmt.Sprintf("test_%d@example.com", time.Now().Unix()),
		PasswordHash: string(passwordHash),
		Status:       "active",
	}

	err = accountRepo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}
	t.Logf("✅ Account created: %s (ID: %s)", account.Username, account.ID)

	// 2. 創建玩家
	t.Log("\n=== Step 2: 創建玩家 ===")
	player := &repository.Player{
		ID:          uuid.New(),
		AccountID:   account.ID,
		DisplayName: "Test Player",
		Level:       1,
	}

	err = playerRepo.Create(ctx, player)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	t.Logf("✅ Player created: %s (ID: %s)", player.DisplayName, player.ID)

	// 清理測試資料
	defer cleanupTest(t, db, player.ID)

	// 3. 創建錢包
	t.Log("\n=== Step 3: 創建錢包 ===")
	wallet := &repository.Wallet{
		ID:       uuid.New(),
		PlayerID: player.ID,
		Balance:  0,
	}

	err = walletRepo.Create(ctx, wallet)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	t.Logf("✅ Wallet created (ID: %s, Balance: %d)", wallet.ID, wallet.Balance)

	// 4. 買入（存款）
	t.Log("\n=== Step 4: 買入 $100.00 ===")
	buyInAmount := int64(10000) // $100.00 in cents

	err = uow.WithTransaction(ctx, func(tx repository.Transaction) error {
		return walletRepo.Credit(ctx, tx, player.ID, buyInAmount, repository.TransactionTypeBuyIn, "Buy-in $100", "buyin_001")
	})
	if err != nil {
		t.Fatalf("Failed to buy-in: %v", err)
	}

	// 驗證餘額
	walletAfterBuyIn, err := walletRepo.GetByPlayerID(ctx, player.ID)
	if err != nil {
		t.Fatalf("Failed to get wallet: %v", err)
	}
	t.Logf("✅ Buy-in successful! Balance: $%.2f", float64(walletAfterBuyIn.Balance)/100)

	if walletAfterBuyIn.Balance != buyInAmount {
		t.Errorf("Expected balance %d, got %d", buyInAmount, walletAfterBuyIn.Balance)
	}

	// 5. 遊戲中贏錢
	t.Log("\n=== Step 5: 遊戲中贏得 $50.00 ===")
	winAmount := int64(5000) // $50.00

	err = uow.WithTransaction(ctx, func(tx repository.Transaction) error {
		return walletRepo.Credit(ctx, tx, player.ID, winAmount, repository.TransactionTypeWin, "Won a hand", "")
	})
	if err != nil {
		t.Fatalf("Failed to credit winnings: %v", err)
	}

	walletAfterWin, err := walletRepo.GetByPlayerID(ctx, player.ID)
	if err != nil {
		t.Fatalf("Failed to get wallet: %v", err)
	}
	t.Logf("✅ Win credited! Balance: $%.2f", float64(walletAfterWin.Balance)/100)

	expectedBalance := buyInAmount + winAmount
	if walletAfterWin.Balance != expectedBalance {
		t.Errorf("Expected balance %d, got %d", expectedBalance, walletAfterWin.Balance)
	}

	// 6. 遊戲中輸錢
	t.Log("\n=== Step 6: 遊戲中輸掉 $30.00 ===")
	lossAmount := int64(3000) // $30.00

	err = uow.WithTransaction(ctx, func(tx repository.Transaction) error {
		return walletRepo.Debit(ctx, tx, player.ID, lossAmount, repository.TransactionTypeLoss, "Lost a hand", "")
	})
	if err != nil {
		t.Fatalf("Failed to debit loss: %v", err)
	}

	walletAfterLoss, err := walletRepo.GetByPlayerID(ctx, player.ID)
	if err != nil {
		t.Fatalf("Failed to get wallet: %v", err)
	}
	t.Logf("✅ Loss debited! Balance: $%.2f", float64(walletAfterLoss.Balance)/100)

	expectedBalance = buyInAmount + winAmount - lossAmount
	if walletAfterLoss.Balance != expectedBalance {
		t.Errorf("Expected balance %d, got %d", expectedBalance, walletAfterLoss.Balance)
	}

	// 7. 兌現（提款）
	t.Log("\n=== Step 7: 兌現 $120.00 ===")
	cashOutAmount := int64(12000) // $120.00

	err = uow.WithTransaction(ctx, func(tx repository.Transaction) error {
		return walletRepo.Debit(ctx, tx, player.ID, cashOutAmount, repository.TransactionTypeCashOut, "Cash-out $120", "cashout_001")
	})
	if err != nil {
		t.Fatalf("Failed to cash-out: %v", err)
	}

	walletFinal, err := walletRepo.GetByPlayerID(ctx, player.ID)
	if err != nil {
		t.Fatalf("Failed to get wallet: %v", err)
	}
	t.Logf("✅ Cash-out successful! Final Balance: $%.2f", float64(walletFinal.Balance)/100)

	expectedBalance = buyInAmount + winAmount - lossAmount - cashOutAmount
	if walletFinal.Balance != expectedBalance {
		t.Errorf("Expected balance %d, got %d", expectedBalance, walletFinal.Balance)
	}

	// 8. 驗證淨盈虧
	netProfitLoss := walletFinal.Balance - buyInAmount
	t.Logf("\n=== Final Summary ===")
	t.Logf("Buy-in:      $%.2f", float64(buyInAmount)/100)
	t.Logf("Final:       $%.2f", float64(walletFinal.Balance)/100)
	t.Logf("Net P/L:     $%.2f", float64(netProfitLoss)/100)

	expectedNetPL := winAmount - lossAmount - cashOutAmount
	if netProfitLoss != expectedNetPL {
		t.Errorf("Expected net P/L %d, got %d", expectedNetPL, netProfitLoss)
	}
}

// TestInsufficientBalance 測試餘額不足的情況
func TestInsufficientBalance(t *testing.T) {
	db, uow, _, accountRepo, playerRepo, walletRepo := setupTest(t)
	ctx := context.Background()

	// 創建測試玩家（使用納秒確保唯一性）
	timestamp := time.Now().UnixNano()
	account := &repository.Account{
		ID:           uuid.New(),
		Username:     fmt.Sprintf("testuser_insuf_%d", timestamp),
		Email:        fmt.Sprintf("test_insuf_%d@example.com", timestamp),
		PasswordHash: "hash",
		Status:       "active",
	}
	err := accountRepo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	player := &repository.Player{
		ID:          uuid.New(),
		AccountID:   account.ID,
		DisplayName: "Test Player Insuf",
		Level:       1,
	}
	err = playerRepo.Create(ctx, player)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	defer cleanupTest(t, db, player.ID)

	wallet := &repository.Wallet{
		ID:       uuid.New(),
		PlayerID: player.ID,
		Balance:  1000, // 只有 $10.00
	}
	err = walletRepo.Create(ctx, wallet)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// 嘗試扣除超過餘額的金額
	t.Log("=== Testing Insufficient Balance ===")
	err = uow.WithTransaction(ctx, func(tx repository.Transaction) error {
		return walletRepo.Debit(ctx, tx, player.ID, 2000, repository.TransactionTypeLoss, "Try to debit $20", "")
	})

	if err == nil {
		t.Fatal("Expected error for insufficient balance, got nil")
	}
	t.Logf("✅ Correctly rejected: %v", err)
}

// TestIdempotency 測試冪等性保證
func TestIdempotency(t *testing.T) {
	db, uow, _, accountRepo, playerRepo, walletRepo := setupTest(t)
	ctx := context.Background()

	// 創建測試玩家（使用納秒確保唯一性）
	timestamp := time.Now().UnixNano()
	account := &repository.Account{
		ID:           uuid.New(),
		Username:     fmt.Sprintf("testuser_idem_%d", timestamp),
		Email:        fmt.Sprintf("test_idem_%d@example.com", timestamp),
		PasswordHash: "hash",
		Status:       "active",
	}
	err := accountRepo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	player := &repository.Player{
		ID:          uuid.New(),
		AccountID:   account.ID,
		DisplayName: "Test Player Idem",
		Level:       1,
	}
	err = playerRepo.Create(ctx, player)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	defer cleanupTest(t, db, player.ID)

	wallet := &repository.Wallet{
		ID:       uuid.New(),
		PlayerID: player.ID,
		Balance:  0,
	}
	err = walletRepo.Create(ctx, wallet)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// 使用相同的冪等性鍵執行兩次買入
	t.Log("=== Testing Idempotency ===")
	idempotencyKey := fmt.Sprintf("buyin_idempotent_%d", timestamp)
	amount := int64(10000) // $100.00

	// 第一次買入
	err = uow.WithTransaction(ctx, func(tx repository.Transaction) error {
		return walletRepo.Credit(ctx, tx, player.ID, amount, repository.TransactionTypeBuyIn, "Buy-in $100", idempotencyKey)
	})
	if err != nil {
		t.Fatalf("First buy-in failed: %v", err)
	}

	walletAfterFirst, _ := walletRepo.GetByPlayerID(ctx, player.ID)
	t.Logf("After first buy-in: $%.2f", float64(walletAfterFirst.Balance)/100)

	// 第二次買入（使用相同的冪等性鍵）
	err = uow.WithTransaction(ctx, func(tx repository.Transaction) error {
		return walletRepo.Credit(ctx, tx, player.ID, amount, repository.TransactionTypeBuyIn, "Buy-in $100 (duplicate)", idempotencyKey)
	})
	if err != nil {
		t.Fatalf("Second buy-in failed: %v", err)
	}

	walletAfterSecond, _ := walletRepo.GetByPlayerID(ctx, player.ID)
	t.Logf("After second buy-in: $%.2f", float64(walletAfterSecond.Balance)/100)

	// 驗證餘額只增加了一次
	if walletAfterSecond.Balance != amount {
		t.Errorf("Expected balance %d (credited once), got %d (credited twice)", amount, walletAfterSecond.Balance)
	} else {
		t.Logf("✅ Idempotency works! Balance only credited once: $%.2f", float64(walletAfterSecond.Balance)/100)
	}
}

// TestConcurrentTransactions 測試並發交易
func TestConcurrentTransactions(t *testing.T) {
	db, uow, _, accountRepo, playerRepo, walletRepo := setupTest(t)
	ctx := context.Background()

	// 創建測試玩家（使用納秒確保唯一性）
	timestamp := time.Now().UnixNano()
	account := &repository.Account{
		ID:           uuid.New(),
		Username:     fmt.Sprintf("testuser_conc_%d", timestamp),
		Email:        fmt.Sprintf("test_conc_%d@example.com", timestamp),
		PasswordHash: "hash",
		Status:       "active",
	}
	err := accountRepo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	player := &repository.Player{
		ID:          uuid.New(),
		AccountID:   account.ID,
		DisplayName: "Test Player Conc",
		Level:       1,
	}
	err = playerRepo.Create(ctx, player)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	defer cleanupTest(t, db, player.ID)

	wallet := &repository.Wallet{
		ID:       uuid.New(),
		PlayerID: player.ID,
		Balance:  100000, // $1000.00 初始餘額
	}
	err = walletRepo.Create(ctx, wallet)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	t.Log("=== Testing Concurrent Transactions ===")
	t.Logf("Initial balance: $%.2f", float64(wallet.Balance)/100)

	// 並發執行 10 次扣款，每次 $10.00
	concurrency := 10
	debitAmount := int64(1000) // $10.00 each

	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := uow.WithTransaction(ctx, func(tx repository.Transaction) error {
				return walletRepo.Debit(ctx, tx, player.ID, debitAmount, repository.TransactionTypeLoss, fmt.Sprintf("Concurrent debit #%d", idx), "")
			})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 檢查錯誤
	for err := range errors {
		t.Logf("Transaction error: %v", err)
	}

	// 驗證最終餘額
	walletFinal, _ := walletRepo.GetByPlayerID(ctx, player.ID)
	expectedBalance := wallet.Balance - (debitAmount * int64(concurrency))
	t.Logf("Final balance: $%.2f", float64(walletFinal.Balance)/100)
	t.Logf("Expected balance: $%.2f", float64(expectedBalance)/100)

	if walletFinal.Balance != expectedBalance {
		t.Errorf("Expected balance %d, got %d (possible race condition!)", expectedBalance, walletFinal.Balance)
	} else {
		t.Logf("✅ Concurrent transactions handled correctly!")
	}
}

// TestLockAndUnlockBalance 測試餘額鎖定/解鎖
func TestLockAndUnlockBalance(t *testing.T) {
	db, uow, _, accountRepo, playerRepo, walletRepo := setupTest(t)
	ctx := context.Background()

	// 創建測試玩家（使用納秒確保唯一性）
	timestamp := time.Now().UnixNano()
	account := &repository.Account{
		ID:           uuid.New(),
		Username:     fmt.Sprintf("testuser_lock_%d", timestamp),
		Email:        fmt.Sprintf("test_lock_%d@example.com", timestamp),
		PasswordHash: "hash",
		Status:       "active",
	}
	err := accountRepo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	player := &repository.Player{
		ID:          uuid.New(),
		AccountID:   account.ID,
		DisplayName: "Test Player Lock",
		Level:       1,
	}
	err = playerRepo.Create(ctx, player)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}

	defer cleanupTest(t, db, player.ID)

	wallet := &repository.Wallet{
		ID:       uuid.New(),
		PlayerID: player.ID,
		Balance:  10000, // $100.00
	}
	err = walletRepo.Create(ctx, wallet)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	t.Log("=== Testing Lock/Unlock Balance ===")
	lockAmount := int64(5000) // $50.00

	// 鎖定餘額
	err = uow.WithTransaction(ctx, func(tx repository.Transaction) error {
		return walletRepo.LockBalance(ctx, tx, player.ID, lockAmount)
	})
	if err != nil {
		t.Fatalf("Failed to lock balance: %v", err)
	}

	walletAfterLock, _ := walletRepo.GetByPlayerID(ctx, player.ID)
	t.Logf("After lock - Balance: $%.2f, Locked: $%.2f",
		float64(walletAfterLock.Balance)/100,
		float64(walletAfterLock.LockedBalance)/100)

	if walletAfterLock.Balance != 5000 || walletAfterLock.LockedBalance != 5000 {
		t.Errorf("Expected balance=5000, locked=5000, got balance=%d, locked=%d",
			walletAfterLock.Balance, walletAfterLock.LockedBalance)
	}

	// 解鎖餘額
	err = uow.WithTransaction(ctx, func(tx repository.Transaction) error {
		return walletRepo.UnlockBalance(ctx, tx, player.ID, lockAmount)
	})
	if err != nil {
		t.Fatalf("Failed to unlock balance: %v", err)
	}

	walletAfterUnlock, _ := walletRepo.GetByPlayerID(ctx, player.ID)
	t.Logf("After unlock - Balance: $%.2f, Locked: $%.2f",
		float64(walletAfterUnlock.Balance)/100,
		float64(walletAfterUnlock.LockedBalance)/100)

	if walletAfterUnlock.Balance != 10000 || walletAfterUnlock.LockedBalance != 0 {
		t.Errorf("Expected balance=10000, locked=0, got balance=%d, locked=%d",
			walletAfterUnlock.Balance, walletAfterUnlock.LockedBalance)
	}

	t.Logf("✅ Lock/Unlock works correctly!")
}
