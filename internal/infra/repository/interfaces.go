package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AccountRepository 定義帳號認證相關的資料庫操作
type AccountRepository interface {
	// Create 創建新帳號
	Create(ctx context.Context, account *Account) error

	// GetByID 根據 ID 查詢帳號
	GetByID(ctx context.Context, id uuid.UUID) (*Account, error)

	// GetByUsername 根據用戶名查詢帳號
	GetByUsername(ctx context.Context, username string) (*Account, error)

	// GetByEmail 根據郵箱查詢帳號
	GetByEmail(ctx context.Context, email string) (*Account, error)

	// Update 更新帳號信息
	Update(ctx context.Context, account *Account) error

	// UpdateLastLogin 更新最後登入時間和 IP
	UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error

	// IncrementFailedAttempts 增加失敗登入次數
	IncrementFailedAttempts(ctx context.Context, id uuid.UUID) error

	// ResetFailedAttempts 重置失敗登入次數
	ResetFailedAttempts(ctx context.Context, id uuid.UUID) error

	// LockAccount 鎖定帳號直到指定時間
	LockAccount(ctx context.Context, id uuid.UUID, until time.Time) error
}

// PlayerRepository 定義玩家資料相關的資料庫操作
type PlayerRepository interface {
	// Create 創建新玩家
	Create(ctx context.Context, player *Player) error

	// GetByID 根據 ID 查詢玩家
	GetByID(ctx context.Context, id uuid.UUID) (*Player, error)

	// GetByAccountID 根據帳號 ID 查詢玩家
	GetByAccountID(ctx context.Context, accountID uuid.UUID) (*Player, error)

	// Update 更新玩家信息
	Update(ctx context.Context, player *Player) error

	// UpdateStats 更新玩家統計數據
	UpdateStats(ctx context.Context, id uuid.UUID, handsPlayed, handsWon int, totalWinnings int64) error

	// GetTopPlayersByWinnings 查詢總贏利排行榜
	GetTopPlayersByWinnings(ctx context.Context, limit int) ([]*Player, error)
}

// WalletRepository 定義錢包相關的資料庫操作（最重要）
type WalletRepository interface {
	// Create 創建新錢包
	Create(ctx context.Context, wallet *Wallet) error

	// GetByPlayerID 根據玩家 ID 查詢錢包
	GetByPlayerID(ctx context.Context, playerID uuid.UUID) (*Wallet, error)

	// GetWithLock 使用行鎖查詢錢包（用於事務中）
	GetWithLock(ctx context.Context, tx Transaction, playerID uuid.UUID) (*Wallet, error)

	// Credit 入帳（加錢）
	// idempotencyKey 用於防止重複入帳
	Credit(ctx context.Context, tx Transaction, playerID uuid.UUID, amount int64, txType TransactionType, description string, idempotencyKey string) error

	// Debit 出帳（扣錢）
	// idempotencyKey 用於防止重複扣款
	Debit(ctx context.Context, tx Transaction, playerID uuid.UUID, amount int64, txType TransactionType, description string, idempotencyKey string) error

	// LockBalance 鎖定餘額（用於下注等場景）
	LockBalance(ctx context.Context, tx Transaction, playerID uuid.UUID, amount int64) error

	// UnlockBalance 解鎖餘額（遊戲結束後）
	UnlockBalance(ctx context.Context, tx Transaction, playerID uuid.UUID, amount int64) error
}

// TransactionRepository 定義交易記錄相關的資料庫操作
type TransactionRepository interface {
	// Create 創建交易記錄
	Create(ctx context.Context, transaction *WalletTransaction) error

	// GetByID 根據 ID 查詢交易
	GetByID(ctx context.Context, id uuid.UUID) (*WalletTransaction, error)

	// GetByWalletID 根據錢包 ID 查詢交易記錄（分頁）
	GetByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*WalletTransaction, error)

	// GetByIdempotencyKey 根據冪等性鍵查詢交易
	GetByIdempotencyKey(ctx context.Context, key string) (*WalletTransaction, error)

	// GetByPlayerID 根據玩家 ID 查詢交易記錄（分頁）
	GetByPlayerID(ctx context.Context, playerID uuid.UUID, limit, offset int) ([]*WalletTransaction, error)
}

// GameSessionRepository 定義遊戲會話相關的資料庫操作
type GameSessionRepository interface {
	// Create 創建新遊戲會話
	Create(ctx context.Context, session *GameSession) error

	// GetByID 根據 ID 查詢會話
	GetByID(ctx context.Context, id uuid.UUID) (*GameSession, error)

	// GetActiveByPlayerID 查詢玩家當前活躍的會話
	GetActiveByPlayerID(ctx context.Context, playerID uuid.UUID) (*GameSession, error)

	// Update 更新會話信息
	Update(ctx context.Context, session *GameSession) error

	// End 結束會話
	End(ctx context.Context, id uuid.UUID, finalChips int64) error
}

// HandHistoryRepository 定義手牌歷史相關的資料庫操作
type HandHistoryRepository interface {
	// Create 創建手牌歷史記錄
	Create(ctx context.Context, history *HandHistory) error

	// GetByID 根據 ID 查詢手牌歷史
	GetByID(ctx context.Context, id uuid.UUID) (*HandHistory, error)

	// GetByGameSessionID 根據遊戲會話 ID 查詢手牌歷史
	GetByGameSessionID(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*HandHistory, error)

	// GetByPlayerID 根據玩家 ID 查詢手牌歷史
	GetByPlayerID(ctx context.Context, playerID uuid.UUID, limit, offset int) ([]*HandHistory, error)
}

// Transaction 定義資料庫事務介面
type Transaction interface {
	// Commit 提交事務
	Commit(ctx context.Context) error

	// Rollback 回滾事務
	Rollback(ctx context.Context) error
}

// UnitOfWork 定義工作單元模式（用於事務管理）
type UnitOfWork interface {
	// Begin 開始新事務
	Begin(ctx context.Context) (Transaction, error)

	// WithTransaction 在事務中執行函數
	WithTransaction(ctx context.Context, fn func(tx Transaction) error) error
}

// TransactionType 定義交易類型
type TransactionType string

const (
	TransactionTypeBuyIn    TransactionType = "buy_in"    // 買入
	TransactionTypeCashOut  TransactionType = "cash_out"  // 兌現
	TransactionTypeWin      TransactionType = "game_win"  // 贏錢
	TransactionTypeLoss     TransactionType = "game_loss" // 輸錢
	TransactionTypeDeposit  TransactionType = "deposit"   // 存款
	TransactionTypeWithdraw TransactionType = "withdraw"  // 提款
	TransactionTypeRefund   TransactionType = "refund"    // 退款
	TransactionTypeBonus    TransactionType = "bonus"     // 獎金
)
