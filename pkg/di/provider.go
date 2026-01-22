package di

import (
	"time"

	"github.com/google/wire"
	"github.com/shinjuwu/TheNuts/internal/auth"
	"github.com/shinjuwu/TheNuts/internal/game"
	"github.com/shinjuwu/TheNuts/internal/game/adapter/ws"
	"github.com/shinjuwu/TheNuts/internal/infra/config"
	"github.com/shinjuwu/TheNuts/internal/infra/logger"
	"go.uber.org/zap"
)

// InfrastructureSet 包含基礎設施模組的 Providers
var InfrastructureSet = wire.NewSet(
	config.LoadConfig,
	logger.NewLogger,
)

// AuthSet 包含認證模組的 Providers
var AuthSet = wire.NewSet(
	ProvideJWTService,
	ProvideTicketStore,
	ProvideAuthHandler,
)

var GameSet = wire.NewSet(
	game.NewTableManager,
	ws.NewHub,
	ws.NewHandler,
)

// ProvideJWTService 提供 JWT 服務
func ProvideJWTService(cfg *config.Config) *auth.JWTService {
	return auth.NewJWTService(cfg.Auth.JWTSecret)
}

// ProvideTicketStore 提供票券儲存
func ProvideTicketStore() auth.TicketStore {
	// 使用記憶體版本（開發/測試）
	// 生產環境應該使用 Redis 版本
	return auth.NewMemoryTicketStore()
}

// ProvideAuthHandler 提供認證 Handler
func ProvideAuthHandler(
	jwtService *auth.JWTService,
	ticketStore auth.TicketStore,
	cfg *config.Config,
	logger *zap.Logger,
) *auth.Handler {
	handler := auth.NewHandler(jwtService, ticketStore, logger)

	// 設定票券 TTL
	if cfg.Auth.TicketTTLSeconds > 0 {
		handler.SetTicketTTL(time.Duration(cfg.Auth.TicketTTLSeconds) * time.Second)
	}

	return handler
}
