package di

import (
	"context"

	"github.com/shinjuwu/TheNuts/internal/auth"
	"github.com/shinjuwu/TheNuts/internal/game"
	"github.com/shinjuwu/TheNuts/internal/game/adapter/ws"
	"github.com/shinjuwu/TheNuts/internal/infra/config"
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
	AuthHandler *auth.Handler
}

func (a *App) Stop(ctx context.Context) {
	// 關閉票券儲存
	if a.TicketStore != nil {
		_ = a.TicketStore.Close()
	}

	_ = a.Logger.Sync()
}
