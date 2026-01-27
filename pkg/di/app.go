package di

import (
	"context"

	"github.com/shinjuwu/TheNuts/internal/auth"
	"github.com/shinjuwu/TheNuts/internal/game"
	"github.com/shinjuwu/TheNuts/internal/game/adapter/ws"
	"github.com/shinjuwu/TheNuts/internal/game/service"
	"github.com/shinjuwu/TheNuts/internal/infra/config"
	"github.com/shinjuwu/TheNuts/internal/infra/database"
	"github.com/shinjuwu/TheNuts/internal/infra/repository"
	"github.com/shinjuwu/TheNuts/internal/infra/repository/postgres"
	"go.uber.org/zap"
)

// App 是伺服器的核心容器，封裝了所有組裝好的組件
type App struct {
	Config       *config.Config
	Logger       *zap.Logger
	TableManager *game.TableManager
	Hub          *ws.Hub
	WSHandler    *ws.Handler

	// 認證相關
	JWTService  *auth.JWTService
	TicketStore auth.TicketStore
	AuthService *auth.AuthService
	AuthHandler *auth.Handler

	// 資料庫相關
	PostgresDB  *database.PostgresDB
	RedisClient *database.RedisClient
	UnitOfWork  repository.UnitOfWork

	// Repository 相關
	AccountRepo     repository.AccountRepository
	PlayerRepo      repository.PlayerRepository
	WalletRepo      repository.WalletRepository
	TransactionRepo *postgres.TransactionRepo
	SessionRepo     repository.GameSessionRepository

	// Service 相關
	GameService    *service.GameService
	SessionManager *ws.SessionManager
}

func (a *App) Stop(ctx context.Context) {
	// 停止 SessionManager
	if a.SessionManager != nil {
		a.SessionManager.Stop()
	}

	// 關閉票券儲存
	if a.TicketStore != nil {
		_ = a.TicketStore.Close()
	}

	// 關閉 Redis 客戶端
	if a.RedisClient != nil {
		_ = a.RedisClient.Close()
		a.Logger.Info("Redis client closed")
	}

	// 關閉 PostgreSQL 連接池
	if a.PostgresDB != nil {
		a.PostgresDB.Close()
	}

	_ = a.Logger.Sync()
}
