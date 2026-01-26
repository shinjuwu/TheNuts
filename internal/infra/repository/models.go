package repository

import (
	"time"

	"github.com/google/uuid"
)

// Account 代表用戶帳號（認證資訊）
type Account struct {
	ID                  uuid.UUID  `db:"id"`
	Username            string     `db:"username"`
	Email               string     `db:"email"`
	PasswordHash        string     `db:"password_hash"`
	Status              string     `db:"status"` // active, suspended, banned
	EmailVerified       bool       `db:"email_verified"`
	FailedLoginAttempts int        `db:"failed_login_attempts"`
	LockedUntil         *time.Time `db:"locked_until"`
	LastLoginAt         *time.Time `db:"last_login_at"`
	LastLoginIP         *string    `db:"last_login_ip"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}

// Player 代表玩家資料
type Player struct {
	ID               uuid.UUID  `db:"id"`
	AccountID        uuid.UUID  `db:"account_id"`
	DisplayName      string     `db:"display_name"`
	AvatarURL        *string    `db:"avatar_url"`
	Level            int        `db:"level"`
	Experience       int64      `db:"experience"`
	TotalGamesPlayed int        `db:"total_games_played"`
	TotalHandsPlayed int        `db:"total_hands_played"`
	TotalWinnings    int64      `db:"total_winnings"` // 單位：分（cents）
	VipLevel         int        `db:"vip_level"`
	VipExpiresAt     *time.Time `db:"vip_expires_at"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
}

// Wallet 代表玩家錢包
type Wallet struct {
	ID            uuid.UUID `db:"id"`
	PlayerID      uuid.UUID `db:"player_id"`
	Balance       int64     `db:"balance"`        // 可用餘額（分）
	LockedBalance int64     `db:"locked_balance"` // 鎖定餘額（分）
	Currency      string    `db:"currency"`       // 貨幣類型
	
	Version       int       `db:"version"`        // 樂觀鎖版本號
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// TotalBalance 返回總餘額（可用 + 鎖定）
func (w *Wallet) TotalBalance() int64 {
	return w.Balance + w.LockedBalance
}

// CanDebit 檢查是否可以扣款
func (w *Wallet) CanDebit(amount int64) bool {
	return w.Balance >= amount
}

// WalletTransaction 代表錢包交易記錄
type WalletTransaction struct {
	ID             uuid.UUID       `db:"id"`
	WalletID       uuid.UUID       `db:"wallet_id"`
	Type           TransactionType `db:"type"`
	Amount         int64           `db:"amount"`         // 金額（分）
	BalanceBefore  int64           `db:"balance_before"` // 交易前餘額
	BalanceAfter   int64           `db:"balance_after"`  // 交易後餘額
	Description    string          `db:"description"`
	IdempotencyKey *string         `db:"idempotency_key"` // 冪等性鍵
	GameSessionID  *uuid.UUID      `db:"game_session_id"` // 關聯的遊戲會話
	CreatedAt      time.Time       `db:"created_at"`
}

// GameSession 代表遊戲會話
type GameSession struct {
	ID           uuid.UUID  `db:"id"`
	PlayerID     uuid.UUID  `db:"player_id"`
	GameType     string     `db:"game_type"` // poker, slot, etc.
	TableID      string     `db:"table_id"`
	BuyInAmount  int64      `db:"buy_in_amount"` // 買入金額（分）
	CurrentChips int64      `db:"current_chips"` // 當前籌碼
	Status       string     `db:"status"`        // active, ended
	StartedAt    time.Time  `db:"started_at"`
	EndedAt      *time.Time `db:"ended_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

// NetProfitLoss 計算淨盈虧
func (g *GameSession) NetProfitLoss() int64 {
	if g.EndedAt == nil {
		return g.CurrentChips - g.BuyInAmount
	}
	return g.CurrentChips - g.BuyInAmount
}

// HandHistory 代表手牌歷史記錄
type HandHistory struct {
	ID             uuid.UUID `db:"id"`
	GameSessionID  uuid.UUID `db:"game_session_id"`
	HandNumber     int       `db:"hand_number"`
	PlayersData    []byte    `db:"players_data"`    // JSONB
	CommunityCards []byte    `db:"community_cards"` // JSONB
	Actions        []byte    `db:"actions"`         // JSONB
	Pots           []byte    `db:"pots"`            // JSONB
	Winners        []byte    `db:"winners"`         // JSONB
	RakeAmount     int64     `db:"rake_amount"`
	Duration       int       `db:"duration"` // 秒
	CreatedAt      time.Time `db:"created_at"`
}

// AuditLog 代表審計日誌
type AuditLog struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	Action     string    `db:"action"`      // login, buy_in, cash_out, etc.
	EntityType string    `db:"entity_type"` // account, wallet, game, etc.
	EntityID   uuid.UUID `db:"entity_id"`
	OldValue   []byte    `db:"old_value"` // JSONB
	NewValue   []byte    `db:"new_value"` // JSONB
	IPAddress  string    `db:"ip_address"`
	UserAgent  string    `db:"user_agent"`
	CreatedAt  time.Time `db:"created_at"`
}
