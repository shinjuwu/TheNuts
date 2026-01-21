package di

import (
	"github.com/google/wire"
	"github.com/shinjuwu/TheNuts/internal/game"
	"github.com/shinjuwu/TheNuts/internal/game/adapter/ws"
	"github.com/shinjuwu/TheNuts/internal/infra/config"
	"github.com/shinjuwu/TheNuts/internal/infra/logger"
)

// InfrastructureSet 包含基礎設施模組的 Providers
var InfrastructureSet = wire.NewSet(
	config.LoadConfig,
	logger.NewLogger,
)

var GameSet = wire.NewSet(
	game.NewTableManager,
	ws.NewHub,
	ws.NewHandler,
)
