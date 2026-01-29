package ws

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shinjuwu/TheNuts/internal/game/domain"
	"github.com/shinjuwu/TheNuts/internal/game/service"
	"go.uber.org/zap"
)

// MessageHandler 处理各种 WebSocket 消息
type MessageHandler struct {
	sessionManager *SessionManager
	tableManager   interface{ GetOrCreateTable(id string) *domain.Table }
	gameService    *service.GameService
	logger         *zap.Logger
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(
	sessionMgr *SessionManager,
	tableMgr interface{ GetOrCreateTable(id string) *domain.Table },
	gameService *service.GameService,
	logger *zap.Logger,
) *MessageHandler {
	return &MessageHandler{
		sessionManager: sessionMgr,
		tableManager:   tableMgr,
		gameService:    gameService,
		logger:         logger,
	}
}

// HandleMessage 处理客户端消息
func (h *MessageHandler) HandleMessage(playerID uuid.UUID, message []byte) {
	var req Request
	if err := json.Unmarshal(message, &req); err != nil {
		h.logger.Warn("invalid message format",
			zap.String("player_id", playerID.String()),
			zap.Error(err),
		)
		h.sendError(playerID, "invalid_format", "Invalid message format")
		return
	}

	// 更新会话活动时间
	if session, exists := h.sessionManager.GetSession(playerID); exists {
		session.UpdateActivity()
	}

	// 根据动作类型路由
	switch req.Action {
	case "BUY_IN":
		h.handleBuyIn(playerID, req)
	case "CASH_OUT":
		h.handleCashOut(playerID, req)
	case "JOIN_TABLE":
		h.handleJoinTable(playerID, req)
	case "LEAVE_TABLE":
		h.handleLeaveTable(playerID, req)
	case "SIT_DOWN":
		h.handleSitDown(playerID, req)
	case "STAND_UP":
		h.handleStandUp(playerID, req)
	case "GAME_ACTION":
		h.handleGameAction(playerID, req)
	case "GET_BALANCE":
		h.handleGetBalance(playerID, req)
	default:
		h.logger.Warn("unknown action",
			zap.String("player_id", playerID.String()),
			zap.String("action", req.Action),
		)
		h.sendError(playerID, "unknown_action", "Unknown action: "+req.Action)
	}
}

// handleBuyIn 处理买入请求
func (h *MessageHandler) handleBuyIn(playerID uuid.UUID, req Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 验证金额
	if req.Amount <= 0 {
		h.sendError(playerID, "invalid_amount", "Amount must be positive")
		return
	}

	// 确保玩家有钱包
	if err := h.gameService.EnsureWalletExists(ctx, playerID, "USD"); err != nil {
		h.logger.Error("failed to ensure wallet exists",
			zap.String("player_id", playerID.String()),
			zap.Error(err),
		)
		h.sendError(playerID, "wallet_error", "Failed to access wallet")
		return
	}

	// 执行买入
	response, err := h.gameService.BuyIn(ctx, service.BuyInRequest{
		PlayerID: playerID,
		TableID:  req.TableID,
		GameType: "poker",
		Amount:   req.Amount,
	})

	if err != nil {
		h.logger.Warn("buy-in failed",
			zap.String("player_id", playerID.String()),
			zap.String("table_id", req.TableID),
			zap.Int64("amount", req.Amount),
			zap.Error(err),
		)

		// 根据错误类型返回不同消息
		switch err {
		case service.ErrInsufficientBalance:
			h.sendError(playerID, "insufficient_balance", "Insufficient balance")
		case service.ErrSessionAlreadyActive:
			h.sendError(playerID, "already_in_game", "Already have an active game session")
		default:
			h.sendError(playerID, "buy_in_failed", "Buy-in failed")
		}
		return
	}

	// 更新会话状态
	if session, exists := h.sessionManager.GetSession(playerID); exists {
		session.SetGameSession(response.SessionID, response.Chips)
	}

	// 发送成功响应
	h.sendResponse(playerID, "BUY_IN_SUCCESS", map[string]interface{}{
		"session_id":     response.SessionID.String(),
		"table_id":       response.TableID,
		"chips":          response.Chips,
		"wallet_balance": response.WalletBalance,
		"created_at":     response.CreatedAt,
	})

	h.logger.Info("buy-in successful",
		zap.String("player_id", playerID.String()),
		zap.String("table_id", req.TableID),
		zap.Int64("amount", req.Amount),
		zap.Int64("chips", response.Chips),
	)
}

// handleCashOut 处理兑现请求
func (h *MessageHandler) handleCashOut(playerID uuid.UUID, req Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取玩家会话
	session, exists := h.sessionManager.GetSession(playerID)
	if !exists {
		h.sendError(playerID, "no_session", "No active session")
		return
	}

	if session.GameSessionID == uuid.Nil {
		h.sendError(playerID, "no_game_session", "No active game session")
		return
	}

	// 执行兑现
	response, err := h.gameService.CashOut(ctx, service.CashOutRequest{
		PlayerID:  playerID,
		SessionID: session.GameSessionID,
		Chips:     session.GetChips(),
	})

	if err != nil {
		h.logger.Error("cash-out failed",
			zap.String("player_id", playerID.String()),
			zap.Error(err),
		)
		h.sendError(playerID, "cash_out_failed", "Cash-out failed")
		return
	}

	// 清理会话状态
	session.LeaveTable()
	session.GameSessionID = uuid.Nil
	session.Chips = 0

	// 发送成功响应
	h.sendResponse(playerID, "CASH_OUT_SUCCESS", map[string]interface{}{
		"session_id":     response.SessionID.String(),
		"buy_in_amount":  response.BuyInAmount,
		"cash_out":       response.CashOutAmount,
		"profit":         response.Profit,
		"wallet_balance": response.WalletBalance,
		"ended_at":       response.EndedAt,
	})

	h.logger.Info("cash-out successful",
		zap.String("player_id", playerID.String()),
		zap.Int64("profit", response.Profit),
	)
}

// sendTableCommand 透過 ActionCh 發送命令到 Table.Run() 並等待結果
func (h *MessageHandler) sendTableCommand(table *domain.Table, cmd domain.PlayerAction) domain.ActionResult {
	resultCh := make(chan domain.ActionResult, 1)
	cmd.ResultCh = resultCh

	select {
	case table.ActionCh <- cmd:
		// 等待結果（帶超時）
		select {
		case result := <-resultCh:
			return result
		case <-time.After(5 * time.Second):
			return domain.ActionResult{Err: errors.New("table command timeout")}
		}
	default:
		return domain.ActionResult{Err: errors.New("table action queue full")}
	}
}

// handleJoinTable 处理加入桌子请求
func (h *MessageHandler) handleJoinTable(playerID uuid.UUID, req Request) {
	session, exists := h.sessionManager.GetSession(playerID)
	if !exists {
		h.sendError(playerID, "no_session", "No active session")
		return
	}

	// 检查是否已有游戏会话
	if session.GameSessionID == uuid.Nil {
		h.sendError(playerID, "no_game_session", "Please buy-in first")
		return
	}

	// 获取或创建桌子
	table := h.tableManager.GetOrCreateTable(req.TableID)

	// 创建 domain.Player
	domainPlayer := &domain.Player{
		ID:         playerID.String(),
		SeatIdx:    req.SeatNo,
		Chips:      session.GetChips(),
		CurrentBet: 0,
		Status:     domain.StatusSittingOut,
		HoleCards:  []domain.Card{},
		HasActed:   false,
	}

	// 透過 ActionCh 發送，由 Table.Run() 統一處理
	result := h.sendTableCommand(table, domain.PlayerAction{
		Type:    domain.ActionJoinTable,
		Player:  domainPlayer,
		SeatIdx: req.SeatNo,
	})
	if result.Err != nil {
		h.logger.Warn("join table failed",
			zap.String("player_id", playerID.String()),
			zap.String("table_id", req.TableID),
			zap.Error(result.Err),
		)
		h.sendError(playerID, "join_table_failed", result.Err.Error())
		return
	}

	// 更新会话状态
	session.SetTable(req.TableID, req.SeatNo)

	// 广播桌子状态
	h.broadcastTableState(req.TableID, table)

	// 发送成功响应
	h.sendResponse(playerID, "JOIN_TABLE_SUCCESS", map[string]interface{}{
		"table_id": req.TableID,
		"seat_no":  req.SeatNo,
		"chips":    session.GetChips(),
	})

	h.logger.Info("player joined table",
		zap.String("player_id", playerID.String()),
		zap.String("table_id", req.TableID),
		zap.Int("seat_no", req.SeatNo),
	)
}

// handleLeaveTable 处理离开桌子请求
func (h *MessageHandler) handleLeaveTable(playerID uuid.UUID, req Request) {
	session, exists := h.sessionManager.GetSession(playerID)
	if !exists {
		h.sendError(playerID, "no_session", "No active session")
		return
	}

	tableID := session.GetTableID()
	if tableID == "" {
		h.sendError(playerID, "not_at_table", "Not at any table")
		return
	}

	// 获取桌子
	table := h.tableManager.GetOrCreateTable(tableID)

	// 透過 ActionCh 移除玩家
	result := h.sendTableCommand(table, domain.PlayerAction{
		Type:     domain.ActionLeaveTable,
		PlayerID: playerID.String(),
	})
	if result.Err != nil {
		h.logger.Warn("leave table failed",
			zap.String("player_id", playerID.String()),
			zap.String("table_id", tableID),
			zap.Error(result.Err),
		)
		h.sendError(playerID, "leave_table_failed", result.Err.Error())
		return
	}

	// 更新会话状态
	session.LeaveTable()

	// 广播桌子状态
	h.broadcastTableState(tableID, table)

	// 发送成功响应
	h.sendResponse(playerID, "LEAVE_TABLE_SUCCESS", map[string]interface{}{
		"table_id": tableID,
	})

	h.logger.Info("player left table",
		zap.String("player_id", playerID.String()),
		zap.String("table_id", tableID),
	)
}

// handleSitDown 处理坐下请求
func (h *MessageHandler) handleSitDown(playerID uuid.UUID, req Request) {
	session, exists := h.sessionManager.GetSession(playerID)
	if !exists {
		h.sendError(playerID, "no_session", "No active session")
		return
	}

	tableID := session.GetTableID()
	if tableID == "" {
		h.sendError(playerID, "not_at_table", "Not at any table")
		return
	}

	table := h.tableManager.GetOrCreateTable(tableID)

	result := h.sendTableCommand(table, domain.PlayerAction{
		Type:     domain.ActionSitDown,
		PlayerID: playerID.String(),
	})
	if result.Err != nil {
		h.logger.Warn("sit down failed",
			zap.String("player_id", playerID.String()),
			zap.String("table_id", tableID),
			zap.Error(result.Err),
		)
		h.sendError(playerID, "sit_down_failed", result.Err.Error())
		return
	}

	h.broadcastTableState(tableID, table)

	h.sendResponse(playerID, "SIT_DOWN_SUCCESS", map[string]interface{}{
		"table_id": tableID,
		"seat_no":  session.SeatNo,
	})

	h.logger.Info("player sat down",
		zap.String("player_id", playerID.String()),
		zap.String("table_id", tableID),
		zap.Int("seat_no", session.SeatNo),
	)
}

// handleStandUp 处理站起请求
func (h *MessageHandler) handleStandUp(playerID uuid.UUID, req Request) {
	session, exists := h.sessionManager.GetSession(playerID)
	if !exists {
		h.sendError(playerID, "no_session", "No active session")
		return
	}

	tableID := session.GetTableID()
	if tableID == "" {
		h.sendError(playerID, "not_at_table", "Not at any table")
		return
	}

	table := h.tableManager.GetOrCreateTable(tableID)

	result := h.sendTableCommand(table, domain.PlayerAction{
		Type:     domain.ActionStandUp,
		PlayerID: playerID.String(),
	})
	if result.Err != nil {
		h.logger.Warn("stand up failed",
			zap.String("player_id", playerID.String()),
			zap.String("table_id", tableID),
			zap.Error(result.Err),
		)
		h.sendError(playerID, "stand_up_failed", result.Err.Error())
		return
	}

	h.broadcastTableState(tableID, table)

	h.sendResponse(playerID, "STAND_UP_SUCCESS", map[string]interface{}{
		"table_id":    tableID,
		"was_in_hand": result.WasInHand,
	})

	h.logger.Info("player stood up",
		zap.String("player_id", playerID.String()),
		zap.String("table_id", tableID),
		zap.Bool("was_in_hand", result.WasInHand),
	)
}

// handleGameAction 处理游戏动作
func (h *MessageHandler) handleGameAction(playerID uuid.UUID, req Request) {
	session, exists := h.sessionManager.GetSession(playerID)
	if !exists {
		h.sendError(playerID, "no_session", "No active session")
		return
	}

	tableID := session.GetTableID()
	if tableID == "" {
		h.sendError(playerID, "not_at_table", "Not at any table")
		return
	}

	table := h.tableManager.GetOrCreateTable(tableID)

	// 转换为 domain 动作
	actionType := MapActionType(req.GameAction)
	playerAction := domain.PlayerAction{
		PlayerID: playerID.String(),
		Type:     actionType,
		Amount:   req.Amount,
	}

	// 发送到桌子的动作通道
	select {
	case table.ActionCh <- playerAction:
		h.logger.Debug("game action sent",
			zap.String("player_id", playerID.String()),
			zap.String("action", req.GameAction),
		)
	default:
		h.sendError(playerID, "action_failed", "Failed to process action")
	}
}

// handleGetBalance 处理查询余额请求
func (h *MessageHandler) handleGetBalance(playerID uuid.UUID, req Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wallet, err := h.gameService.GetPlayerBalance(ctx, playerID)
	if err != nil {
		h.sendError(playerID, "balance_error", "Failed to get balance")
		return
	}

	session, _ := h.sessionManager.GetSession(playerID)
	chips := int64(0)
	if session != nil {
		chips = session.GetChips()
	}

	h.sendResponse(playerID, "BALANCE_INFO", map[string]interface{}{
		"wallet_balance": wallet.Balance,
		"locked_balance": wallet.LockedBalance,
		"current_chips":  chips,
		"total_balance":  wallet.TotalBalance(),
		"currency":       wallet.Currency,
	})
}

// broadcastTableState 广播桌子状态
func (h *MessageHandler) broadcastTableState(tableID string, table *domain.Table) {
	// 构建桌子状态快照
	snapshot := h.buildTableSnapshot(table)

	// 广播给桌子上的所有玩家
	h.sessionManager.BroadcastToTable(tableID, Response{
		Type:      "TABLE_STATE",
		Payload:   snapshot,
		Timestamp: time.Now(),
	})
}

// buildTableSnapshot 构建桌子状态快照
func (h *MessageHandler) buildTableSnapshot(table *domain.Table) map[string]interface{} {
	players := make([]map[string]interface{}, 0)

	for _, player := range table.Players {
		if player != nil {
			players = append(players, map[string]interface{}{
				"id":          player.ID,
				"seat_idx":    player.SeatIdx,
				"chips":       player.Chips,
				"current_bet": player.CurrentBet,
				"status":      player.Status,
				"has_acted":   player.HasActed,
			})
		}
	}

	communityCards := make([]string, 0)
	for _, card := range table.CommunityCards {
		communityCards = append(communityCards, card.String())
	}

	return map[string]interface{}{
		"table_id":        table.ID,
		"state":           table.State,
		"players":         players,
		"community_cards": communityCards,
		"dealer_pos":      table.DealerPos,
		"current_pos":     table.CurrentPos,
		"min_bet":         table.MinBet,
		"pot_total":       table.Pots.Total(),
	}
}

// sendResponse 发送响应消息
func (h *MessageHandler) sendResponse(playerID uuid.UUID, msgType string, payload interface{}) {
	response := Response{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	h.sessionManager.SendToPlayer(playerID, response)
}

// sendError 发送错误消息
func (h *MessageHandler) sendError(playerID uuid.UUID, code, message string) {
	response := Response{
		Type: "ERROR",
		Payload: ErrorPayload{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now(),
	}

	h.sessionManager.SendToPlayer(playerID, response)
}

// MapActionType 将字符串转换为 ActionType
