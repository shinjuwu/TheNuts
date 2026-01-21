package di

import (
	"context"

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
}

func (a *App) Stop(ctx context.Context) {
	_ = a.Logger.Sync()
}
