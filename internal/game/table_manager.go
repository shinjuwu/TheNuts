package game

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shinjuwu/TheNuts/internal/game/domain"
	"github.com/shinjuwu/TheNuts/internal/game/service"
	"go.uber.org/zap"
)

type TableManager struct {
	tables      map[string]*domain.Table
	mu          sync.RWMutex
	gameService *service.GameService
	logger      *zap.Logger
	tableLogger domain.Logger // 注入到每張 Table

	// 遊戲事件回調（由 main.go 注入，轉發到 WebSocket）
	onTableEvent func(event domain.TableEvent)
}

func NewTableManager(gs *service.GameService) *TableManager {
	return &TableManager{
		tables:      make(map[string]*domain.Table),
		gameService: gs,
	}
}

// SetLogger 設定日誌器
func (tm *TableManager) SetLogger(logger *zap.Logger) {
	tm.logger = logger
	tm.tableLogger = &zapDomainLogger{logger: logger}
}

// SetOnTableEvent 設定遊戲事件回調（應在建表前呼叫）
func (tm *TableManager) SetOnTableEvent(fn func(event domain.TableEvent)) {
	tm.onTableEvent = fn
}

func (tm *TableManager) GetOrCreateTable(id string) *domain.Table {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if t, ok := tm.tables[id]; ok {
		return t
	}

	t := domain.NewTable(id)
	t.AddOnHandComplete(tm.onHandComplete)
	if tm.onTableEvent != nil {
		t.AddOnEvent(tm.onTableEvent)
	}
	if tm.tableLogger != nil {
		t.Logger = tm.tableLogger
	}
	tm.tables[id] = t
	go t.Run()
	return t
}

func (tm *TableManager) GetTable(id string) *domain.Table {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tables[id]
}

// NotifyDisconnect 通知牌桌玩家斷線（實現 ws.TableNotifier）
func (tm *TableManager) NotifyDisconnect(playerID, tableID string) {
	table := tm.GetTable(tableID)
	if table == nil {
		return
	}
	select {
	case table.ActionCh <- domain.PlayerAction{
		Type:     domain.ActionDisconnect,
		PlayerID: playerID,
	}:
	default:
		tm.logWarn("notify disconnect: action queue full",
			zap.String("table_id", tableID), zap.String("player_id", playerID))
	}
}

// NotifyReconnect 通知牌桌玩家重連（實現 ws.TableNotifier）
func (tm *TableManager) NotifyReconnect(playerID, tableID string) {
	table := tm.GetTable(tableID)
	if table == nil {
		return
	}
	select {
	case table.ActionCh <- domain.PlayerAction{
		Type:     domain.ActionReconnect,
		PlayerID: playerID,
	}:
	default:
		tm.logWarn("notify reconnect: action queue full",
			zap.String("table_id", tableID), zap.String("player_id", playerID))
	}
}

func (tm *TableManager) onHandComplete(t *domain.Table) {
	// 同步快照玩家籌碼（在 Run() goroutine 中，安全讀取）
	playerChips := make(map[string]int64, len(t.Players))
	for id, player := range t.Players {
		playerChips[id] = player.Chips
	}

	// 異步同步到資料庫，不阻塞 Table.Run()
	go tm.syncPlayerChips(playerChips)
}

// syncPlayerChips 異步將玩家籌碼同步到資料庫
func (tm *TableManager) syncPlayerChips(playerChips map[string]int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for playerIDStr, chips := range playerChips {
		playerID, err := uuid.Parse(playerIDStr)
		if err != nil {
			tm.logError("failed to parse player ID",
				zap.String("player_id", playerIDStr), zap.Error(err))
			continue
		}

		session, err := tm.gameService.GetActiveSession(ctx, playerID)
		if err != nil {
			// 玩家可能已經登出或沒有會話（例如掉線），預期內
			continue
		}

		if err := tm.gameService.UpdateSessionChips(ctx, session.ID, chips); err != nil {
			tm.logError("failed to update chips",
				zap.String("player_id", playerIDStr), zap.Error(err))
		}
	}
}

// logWarn 安全地記錄警告（logger 可能為 nil）
func (tm *TableManager) logWarn(msg string, fields ...zap.Field) {
	if tm.logger != nil {
		tm.logger.Warn(msg, fields...)
	}
}

// logError 安全地記錄錯誤（logger 可能為 nil）
func (tm *TableManager) logError(msg string, fields ...zap.Field) {
	if tm.logger != nil {
		tm.logger.Error(msg, fields...)
	}
}

// zapDomainLogger 將 zap.Logger 適配為 domain.Logger 介面
type zapDomainLogger struct {
	logger *zap.Logger
}

func (z *zapDomainLogger) Info(msg string, keysAndValues ...interface{}) {
	z.logger.Sugar().Infow(msg, keysAndValues...)
}

func (z *zapDomainLogger) Warn(msg string, keysAndValues ...interface{}) {
	z.logger.Sugar().Warnw(msg, keysAndValues...)
}

func (z *zapDomainLogger) Error(msg string, keysAndValues ...interface{}) {
	z.logger.Sugar().Errorw(msg, keysAndValues...)
}
