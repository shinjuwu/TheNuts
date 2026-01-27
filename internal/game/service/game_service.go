package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shinjuwu/TheNuts/internal/infra/repository"
	"go.uber.org/zap"
)

var (
	// ErrInsufficientBalance 余额不足
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrPlayerNotFound 玩家未找到
	ErrPlayerNotFound = errors.New("player not found")
	// ErrWalletNotFound 钱包未找到
	ErrWalletNotFound = errors.New("wallet not found")
	// ErrSessionNotFound 会话未找到
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionAlreadyActive 玩家已有活跃会话
	ErrSessionAlreadyActive = errors.New("player already has active session")
	// ErrInvalidAmount 无效金额
	ErrInvalidAmount = errors.New("invalid amount")
)

// GameService 游戏服务，处理游戏业务逻辑和资金操作
type GameService struct {
	playerRepo  repository.PlayerRepository
	walletRepo  repository.WalletRepository
	sessionRepo repository.GameSessionRepository
	uow         repository.UnitOfWork
	logger      *zap.Logger
}

// NewGameService 创建游戏服务
func NewGameService(
	playerRepo repository.PlayerRepository,
	walletRepo repository.WalletRepository,
	sessionRepo repository.GameSessionRepository,
	uow repository.UnitOfWork,
	logger *zap.Logger,
) *GameService {
	return &GameService{
		playerRepo:  playerRepo,
		walletRepo:  walletRepo,
		sessionRepo: sessionRepo,
		uow:         uow,
		logger:      logger,
	}
}

// BuyInRequest 买入请求
type BuyInRequest struct {
	PlayerID uuid.UUID
	TableID  string
	GameType string
	Amount   int64 // 单位：分（cents）
}

// BuyInResponse 买入响应
type BuyInResponse struct {
	SessionID     uuid.UUID
	PlayerID      uuid.UUID
	TableID       string
	Chips         int64
	WalletBalance int64
	CreatedAt     time.Time
}

// BuyIn 玩家买入游戏
func (s *GameService) BuyIn(ctx context.Context, req BuyInRequest) (*BuyInResponse, error) {
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	_, err := s.playerRepo.GetByID(ctx, req.PlayerID)
	if err != nil {
		s.logger.Error("failed to get player",
			zap.String("player_id", req.PlayerID.String()),
			zap.Error(err),
		)
		return nil, ErrPlayerNotFound
	}

	existingSession, err := s.sessionRepo.GetActiveByPlayerID(ctx, req.PlayerID)
	if err == nil && existingSession != nil {
		s.logger.Warn("player already has active session",
			zap.String("player_id", req.PlayerID.String()),
			zap.String("session_id", existingSession.ID.String()),
		)
		return nil, ErrSessionAlreadyActive
	}

	var response *BuyInResponse

	err = s.uow.WithTransaction(ctx, func(tx repository.Transaction) error {
		wallet, err := s.walletRepo.GetWithLock(ctx, tx, req.PlayerID)
		if err != nil {
			return fmt.Errorf("failed to get wallet: %w", err)
		}

		if !wallet.CanDebit(req.Amount) {
			return ErrInsufficientBalance
		}

		idempotencyKey := fmt.Sprintf("buyin-%s-%s-%d",
			req.PlayerID.String(),
			req.TableID,
			time.Now().UnixNano(),
		)

		err = s.walletRepo.Debit(
			ctx,
			tx,
			req.PlayerID,
			req.Amount,
			repository.TransactionTypeBuyIn,
			fmt.Sprintf("Buy-in to table %s", req.TableID),
			idempotencyKey,
		)
		if err != nil {
			return fmt.Errorf("failed to debit wallet: %w", err)
		}

		session := &repository.GameSession{
			ID:           uuid.New(),
			PlayerID:     req.PlayerID,
			GameType:     req.GameType,
			TableID:      req.TableID,
			BuyInAmount:  req.Amount,
			CurrentChips: req.Amount,
			Status:       "active",
			StartedAt:    time.Now(),
		}

		if err := s.sessionRepo.Create(ctx, session); err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}

		wallet, err = s.walletRepo.GetByPlayerID(ctx, req.PlayerID)
		if err != nil {
			return fmt.Errorf("failed to get updated wallet: %w", err)
		}

		response = &BuyInResponse{
			SessionID:     session.ID,
			PlayerID:      req.PlayerID,
			TableID:       req.TableID,
			Chips:         session.CurrentChips,
			WalletBalance: wallet.Balance,
			CreatedAt:     session.StartedAt,
		}

		return nil
	})

	if err != nil {
		s.logger.Error("buy-in failed",
			zap.String("player_id", req.PlayerID.String()),
			zap.String("table_id", req.TableID),
			zap.Int64("amount", req.Amount),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info("buy-in successful",
		zap.String("player_id", req.PlayerID.String()),
		zap.String("session_id", response.SessionID.String()),
		zap.String("table_id", req.TableID),
		zap.Int64("amount", req.Amount),
		zap.Int64("wallet_balance", response.WalletBalance),
	)

	return response, nil
}

// CashOutRequest 兑现请求
type CashOutRequest struct {
	PlayerID  uuid.UUID
	SessionID uuid.UUID
	Chips     int64
}

// CashOutResponse 兑现响应
type CashOutResponse struct {
	SessionID     uuid.UUID
	PlayerID      uuid.UUID
	BuyInAmount   int64
	CashOutAmount int64
	Profit        int64
	WalletBalance int64
	EndedAt       time.Time
}

// CashOut 玩家兑现离开游戏
func (s *GameService) CashOut(ctx context.Context, req CashOutRequest) (*CashOutResponse, error) {
	if req.Chips < 0 {
		return nil, ErrInvalidAmount
	}

	session, err := s.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		s.logger.Error("failed to get session",
			zap.String("session_id", req.SessionID.String()),
			zap.Error(err),
		)
		return nil, ErrSessionNotFound
	}

	if session.PlayerID != req.PlayerID {
		return nil, errors.New("session does not belong to player")
	}

	if session.Status != "active" {
		return nil, errors.New("session is not active")
	}

	var response *CashOutResponse

	err = s.uow.WithTransaction(ctx, func(tx repository.Transaction) error {
		if req.Chips > 0 {
			idempotencyKey := fmt.Sprintf("cashout-%s-%s-%d",
				req.PlayerID.String(),
				req.SessionID.String(),
				time.Now().UnixNano(),
			)

			err = s.walletRepo.Credit(
				ctx,
				tx,
				req.PlayerID,
				req.Chips,
				repository.TransactionTypeCashOut,
				fmt.Sprintf("Cash-out from table %s", session.TableID),
				idempotencyKey,
			)
			if err != nil {
				return fmt.Errorf("failed to credit wallet: %w", err)
			}
		}

		if err := s.sessionRepo.End(ctx, session.ID, req.Chips); err != nil {
			return fmt.Errorf("failed to end session: %w", err)
		}

		profit := req.Chips - session.BuyInAmount

		wallet, err := s.walletRepo.GetByPlayerID(ctx, req.PlayerID)
		if err != nil {
			return fmt.Errorf("failed to get wallet: %w", err)
		}

		response = &CashOutResponse{
			SessionID:     session.ID,
			PlayerID:      req.PlayerID,
			BuyInAmount:   session.BuyInAmount,
			CashOutAmount: req.Chips,
			Profit:        profit,
			WalletBalance: wallet.Balance,
			EndedAt:       time.Now(),
		}

		return nil
	})

	if err != nil {
		s.logger.Error("cash-out failed",
			zap.String("player_id", req.PlayerID.String()),
			zap.String("session_id", req.SessionID.String()),
			zap.Int64("chips", req.Chips),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info("cash-out successful",
		zap.String("player_id", req.PlayerID.String()),
		zap.String("session_id", response.SessionID.String()),
		zap.Int64("buy_in", response.BuyInAmount),
		zap.Int64("cash_out", response.CashOutAmount),
		zap.Int64("profit", response.Profit),
		zap.Int64("wallet_balance", response.WalletBalance),
	)

	return response, nil
}

// GetPlayerBalance 获取玩家余额
func (s *GameService) GetPlayerBalance(ctx context.Context, playerID uuid.UUID) (*repository.Wallet, error) {
	wallet, err := s.walletRepo.GetByPlayerID(ctx, playerID)
	if err != nil {
		s.logger.Error("failed to get player balance",
			zap.String("player_id", playerID.String()),
			zap.Error(err),
		)
		return nil, ErrWalletNotFound
	}

	return wallet, nil
}

// GetActiveSession 获取玩家活跃会话
func (s *GameService) GetActiveSession(ctx context.Context, playerID uuid.UUID) (*repository.GameSession, error) {
	session, err := s.sessionRepo.GetActiveByPlayerID(ctx, playerID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// UpdateSessionChips 更新会话筹码（游戏过程中）
func (s *GameService) UpdateSessionChips(ctx context.Context, sessionID uuid.UUID, chips int64) error {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	session.CurrentChips = chips
	session.UpdatedAt = time.Now()

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		s.logger.Error("failed to update session chips",
			zap.String("session_id", sessionID.String()),
			zap.Int64("chips", chips),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// EnsureWalletExists 确保玩家钱包存在，如果不存在则创建
func (s *GameService) EnsureWalletExists(ctx context.Context, playerID uuid.UUID, currency string) error {
	_, err := s.walletRepo.GetByPlayerID(ctx, playerID)
	if err == nil {
		return nil
	}

	wallet := &repository.Wallet{
		ID:            uuid.New(),
		PlayerID:      playerID,
		Balance:       0,
		LockedBalance: 0,
		Currency:      currency,
		Version:       1,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.walletRepo.Create(ctx, wallet); err != nil {
		s.logger.Error("failed to create wallet",
			zap.String("player_id", playerID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	s.logger.Info("wallet created",
		zap.String("player_id", playerID.String()),
		zap.String("wallet_id", wallet.ID.String()),
		zap.String("currency", currency),
	)

	return nil
}
