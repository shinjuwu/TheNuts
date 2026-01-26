package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/shinjuwu/TheNuts/internal/infra/config"
	"go.uber.org/zap"
)

// RedisClient 封裝 Redis 客戶端
type RedisClient struct {
	Client *redis.Client
	logger *zap.Logger
}

// NewRedisClient 創建新的 Redis 客戶端
func NewRedisClient(ctx context.Context, cfg config.RedisConfig, logger *zap.Logger) (*RedisClient, error) {
	// 創建 Redis 客戶端
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// 測試連接
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("Redis client created",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.Int("db", cfg.DB),
		zap.Int("pool_size", cfg.PoolSize),
	)

	return &RedisClient{
		Client: client,
		logger: logger,
	}, nil
}

// HealthCheck 執行健康檢查
func (r *RedisClient) HealthCheck(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Close 關閉客戶端
func (r *RedisClient) Close() error {
	err := r.Client.Close()
	if err != nil {
		r.logger.Error("Failed to close Redis client", zap.Error(err))
		return err
	}
	r.logger.Info("Redis client closed")
	return nil
}

// Stats 返回連接池統計信息
func (r *RedisClient) Stats() *redis.PoolStats {
	return r.Client.PoolStats()
}
