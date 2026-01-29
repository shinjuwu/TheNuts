package auth

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// newTestRedisClient 建立測試用 Redis 客戶端，若連線失敗則跳過測試
func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6382",
		DB:   1, // 使用 DB 1 避免影響開發資料
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("skipping redis integration test: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

func TestRedisTicketStore_GenerateAndValidate(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewRedisTicketStore(client)

	ctx := context.Background()
	playerID := "player-123"

	ticket, err := store.Generate(ctx, playerID, 30*time.Second)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(ticket) != 32 {
		t.Fatalf("expected ticket length 32, got %d", len(ticket))
	}

	got, err := store.Validate(ctx, ticket)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if got != playerID {
		t.Fatalf("expected playerID %q, got %q", playerID, got)
	}
}

func TestRedisTicketStore_ValidateInvalidTicket(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewRedisTicketStore(client)

	ctx := context.Background()

	_, err := store.Validate(ctx, "nonexistent-ticket")
	if err == nil {
		t.Fatal("expected error for invalid ticket, got nil")
	}
}

func TestRedisTicketStore_ValidateConsumedTicket(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewRedisTicketStore(client)

	ctx := context.Background()
	playerID := "player-456"

	ticket, err := store.Generate(ctx, playerID, 30*time.Second)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 第一次驗證應成功
	_, err = store.Validate(ctx, ticket)
	if err != nil {
		t.Fatalf("first Validate failed: %v", err)
	}

	// 第二次驗證應失敗（防重放）
	_, err = store.Validate(ctx, ticket)
	if err == nil {
		t.Fatal("expected error for consumed ticket, got nil")
	}
}

func TestRedisTicketStore_TTLExpiry(t *testing.T) {
	client := newTestRedisClient(t)
	store := NewRedisTicketStore(client)

	ctx := context.Background()
	playerID := "player-789"

	// 使用 1 秒 TTL
	ticket, err := store.Generate(ctx, playerID, 1*time.Second)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 等待過期
	time.Sleep(1500 * time.Millisecond)

	// 驗證應失敗
	_, err = store.Validate(ctx, ticket)
	if err == nil {
		t.Fatal("expected error for expired ticket, got nil")
	}
}
