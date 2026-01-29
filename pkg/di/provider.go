package di

import (
	"context"
	"time"

	"github.com/google/wire"
	"github.com/shinjuwu/TheNuts/internal/auth"
	"github.com/shinjuwu/TheNuts/internal/game"
	"github.com/shinjuwu/TheNuts/internal/game/adapter/ws"
	"github.com/shinjuwu/TheNuts/internal/game/service"
	"github.com/shinjuwu/TheNuts/internal/infra/config"
	"github.com/shinjuwu/TheNuts/internal/infra/database"
	"github.com/shinjuwu/TheNuts/internal/infra/logger"
	"github.com/shinjuwu/TheNuts/internal/infra/repository"
	"github.com/shinjuwu/TheNuts/internal/infra/repository/postgres"
	"go.uber.org/zap"
)

// InfrastructureSet 包含基礎設施模組的 Providers
var InfrastructureSet = wire.NewSet(
	config.LoadConfig,
	logger.NewLogger,
)

// DatabaseSet 包含資料庫相關的 Providers
var DatabaseSet = wire.NewSet(
	ProvidePostgresDB,
	ProvideRedisClient,
	ProvideUnitOfWork,
)

// RepositorySet 包含 Repository 相關的 Providers
var RepositorySet = wire.NewSet(
	ProvideAccountRepository,
	ProvidePlayerRepository,
	ProvideTransactionRepository,
	ProvideWalletRepository,
	ProvideGameSessionRepository,
)

// AuthSet 包含認證模組的 Providers
var AuthSet = wire.NewSet(
	ProvideJWTService,
	ProvideTicketStore,
	ProvideAuthService,
	ProvideAuthHandler,
)

// ServiceSet 包含业务服务的 Providers
var ServiceSet = wire.NewSet(
	ProvideGameService,
)

var GameSet = wire.NewSet(
	ProvideSessionManager,
	// game.NewTableManager 依賴 GameService
	ProvideTableManager,
	ws.NewHub,
	ProvideWSHandler,
)

// ProvideTableManager 提供 Table Manager (主要為了注入依賴)
func ProvideTableManager(gs *service.GameService) *game.TableManager {
	return game.NewTableManager(gs)
}

// ProvideJWTService 提供 JWT 服務
func ProvideJWTService(cfg *config.Config) *auth.JWTService {
	return auth.NewJWTService(cfg.Auth.JWTSecret)
}

// ProvideTicketStore 提供票券儲存（使用 Redis）
func ProvideTicketStore(redisClient *database.RedisClient) auth.TicketStore {
	return auth.NewRedisTicketStore(redisClient.Client)
}

// ProvideAuthService 提供认证服务
func ProvideAuthService(
	accountRepo repository.AccountRepository,
	playerRepo repository.PlayerRepository,
	logger *zap.Logger,
) *auth.AuthService {
	return auth.NewAuthService(accountRepo, playerRepo, logger)
}

// ProvideAuthHandler 提供認證 Handler
func ProvideAuthHandler(
	jwtService *auth.JWTService,
	ticketStore auth.TicketStore,
	authService *auth.AuthService,
	cfg *config.Config,
	logger *zap.Logger,
) *auth.Handler {
	handler := auth.NewHandler(jwtService, ticketStore, authService, logger)

	// 設定票券 TTL
	if cfg.Auth.TicketTTLSeconds > 0 {
		handler.SetTicketTTL(time.Duration(cfg.Auth.TicketTTLSeconds) * time.Second)
	}

	return handler
}

// ProvidePostgresDB 提供 PostgreSQL 連接池
func ProvidePostgresDB(cfg *config.Config, logger *zap.Logger) (*database.PostgresDB, error) {
	ctx := context.Background()
	db, err := database.NewPostgresPool(ctx, cfg.Database.Postgres, logger)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// ProvideRedisClient 提供 Redis 客戶端
func ProvideRedisClient(cfg *config.Config, logger *zap.Logger) (*database.RedisClient, error) {
	ctx := context.Background()
	client, err := database.NewRedisClient(ctx, cfg.Database.Redis, logger)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// ProvideUnitOfWork 提供工作單元
func ProvideUnitOfWork(db *database.PostgresDB) repository.UnitOfWork {
	return postgres.NewUnitOfWork(db.Pool)
}

// ProvideAccountRepository 提供 Account Repository
func ProvideAccountRepository(db *database.PostgresDB) repository.AccountRepository {
	return postgres.NewAccountRepository(db.Pool)
}

// ProvidePlayerRepository 提供 Player Repository
func ProvidePlayerRepository(db *database.PostgresDB) repository.PlayerRepository {
	return postgres.NewPlayerRepository(db.Pool)
}

// ProvideTransactionRepository 提供 Transaction Repository
func ProvideTransactionRepository(db *database.PostgresDB) *postgres.TransactionRepo {
	return postgres.NewTransactionRepository(db.Pool)
}

// ProvideWalletRepository 提供 Wallet Repository
func ProvideWalletRepository(db *database.PostgresDB, txRepo *postgres.TransactionRepo) repository.WalletRepository {
	return postgres.NewWalletRepository(db.Pool, txRepo)
}

// ProvideGameSessionRepository 提供 GameSession Repository
func ProvideGameSessionRepository(db *database.PostgresDB) repository.GameSessionRepository {
	return postgres.NewGameSessionRepository(db.Pool)
}

// ProvideGameService 提供 Game Service
func ProvideGameService(
	playerRepo repository.PlayerRepository,
	walletRepo repository.WalletRepository,
	sessionRepo repository.GameSessionRepository,
	uow repository.UnitOfWork,
	logger *zap.Logger,
) *service.GameService {
	return service.NewGameService(playerRepo, walletRepo, sessionRepo, uow, logger)
}

// ProvideSessionManager 提供 Session Manager
func ProvideSessionManager(
	gameService *service.GameService,
	logger *zap.Logger,
) *ws.SessionManager {
	return ws.NewSessionManager(gameService, logger)
}

// ProvideWSHandler 提供 WebSocket Handler
func ProvideWSHandler(
	hub *ws.Hub,
	tableMgr *game.TableManager,
	sessionMgr *ws.SessionManager,
	gameService *service.GameService,
	ticketStore auth.TicketStore,
	logger *zap.Logger,
) *ws.Handler {
	return ws.NewHandler(hub, tableMgr, sessionMgr, gameService, ticketStore, logger)
}
