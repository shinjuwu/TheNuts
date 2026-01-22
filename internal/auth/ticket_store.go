package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// TicketStore 管理一次性票券的儲存與驗證
type TicketStore interface {
	// Generate 生成一個新的票券，關聯到指定的玩家 ID
	Generate(ctx context.Context, playerID string, ttl time.Duration) (string, error)

	// Validate 驗證票券並返回關聯的玩家 ID（驗證成功後立即銷毀票券）
	Validate(ctx context.Context, ticket string) (string, error)

	// Close 關閉儲存資源
	Close() error
}

// MemoryTicketStore 記憶體版本的票券儲存（開發/測試用）
// 生產環境建議使用 RedisTicketStore
type MemoryTicketStore struct {
	mu      sync.RWMutex
	tickets map[string]*ticketData
}

type ticketData struct {
	PlayerID  string
	ExpiresAt time.Time
}

// NewMemoryTicketStore 創建記憶體票券儲存
func NewMemoryTicketStore() *MemoryTicketStore {
	store := &MemoryTicketStore{
		tickets: make(map[string]*ticketData),
	}

	// 啟動過期票券清理 Goroutine
	go store.cleanupExpired()

	return store
}

// Generate 生成票券
func (s *MemoryTicketStore) Generate(ctx context.Context, playerID string, ttl time.Duration) (string, error) {
	// 生成 32 字元的隨機票券
	ticket, err := generateRandomTicket(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate ticket: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.tickets[ticket] = &ticketData{
		PlayerID:  playerID,
		ExpiresAt: time.Now().Add(ttl),
	}

	return ticket, nil
}

// Validate 驗證並銷毀票券
func (s *MemoryTicketStore) Validate(ctx context.Context, ticket string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, exists := s.tickets[ticket]
	if !exists {
		return "", fmt.Errorf("invalid ticket: not found")
	}

	// 檢查是否過期
	if time.Now().After(data.ExpiresAt) {
		delete(s.tickets, ticket)
		return "", fmt.Errorf("invalid ticket: expired")
	}

	playerID := data.PlayerID

	// 立即銷毀票券（防止重放攻擊）
	delete(s.tickets, ticket)

	return playerID, nil
}

// Close 清理資源
func (s *MemoryTicketStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tickets = nil
	return nil
}

// cleanupExpired 定期清理過期票券
func (s *MemoryTicketStore) cleanupExpired() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for ticket, data := range s.tickets {
			if now.After(data.ExpiresAt) {
				delete(s.tickets, ticket)
			}
		}
		s.mu.Unlock()
	}
}

// generateRandomTicket 生成密碼學安全的隨機票券
func generateRandomTicket(length int) (string, error) {
	bytes := make([]byte, length/2) // hex 編碼會加倍長度
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
