package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const ticketKeyPrefix = "ticket:"

// RedisTicketStore 使用 Redis 儲存一次性票券
type RedisTicketStore struct {
	client *redis.Client
}

// NewRedisTicketStore 建立 Redis 票券儲存
func NewRedisTicketStore(client *redis.Client) *RedisTicketStore {
	return &RedisTicketStore{client: client}
}

// Generate 生成票券並存入 Redis（帶 TTL）
func (s *RedisTicketStore) Generate(ctx context.Context, playerID string, ttl time.Duration) (string, error) {
	ticket, err := generateRandomTicket(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate ticket: %w", err)
	}

	key := ticketKeyPrefix + ticket
	if err := s.client.Set(ctx, key, playerID, ttl).Err(); err != nil {
		return "", fmt.Errorf("failed to store ticket in redis: %w", err)
	}

	return ticket, nil
}

// Validate 驗證並銷毀票券（原子操作）
func (s *RedisTicketStore) Validate(ctx context.Context, ticket string) (string, error) {
	key := ticketKeyPrefix + ticket
	playerID, err := s.client.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("invalid ticket: not found or expired")
	}
	if err != nil {
		return "", fmt.Errorf("failed to validate ticket: %w", err)
	}

	return playerID, nil
}

// Close 關閉資源（Redis client 由外部管理，此處為 no-op）
func (s *RedisTicketStore) Close() error {
	return nil
}
