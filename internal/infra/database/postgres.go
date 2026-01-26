package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shinjuwu/TheNuts/internal/infra/config"
	"go.uber.org/zap"
)

// PostgresDB 封裝 PostgreSQL 連接池
type PostgresDB struct {
	Pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresPool 創建新的 PostgreSQL 連接池
func NewPostgresPool(ctx context.Context, cfg config.PostgresConfig, logger *zap.Logger) (*PostgresDB, error) {
	// 構建連接字符串
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	// 配置連接池
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	// 設置連接池參數
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.GetMaxConnLifetime()
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// 創建連接池
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// 健康檢查
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("PostgreSQL connection pool created",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
		zap.Int32("max_conns", cfg.MaxConns),
		zap.Int32("min_conns", cfg.MinConns),
	)

	return &PostgresDB{
		Pool:   pool,
		logger: logger,
	}, nil
}

// HealthCheck 執行健康檢查
func (db *PostgresDB) HealthCheck(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Close 關閉連接池
func (db *PostgresDB) Close() {
	db.Pool.Close()
	db.logger.Info("PostgreSQL connection pool closed")
}

// Stats 返回連接池統計信息
func (db *PostgresDB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}
