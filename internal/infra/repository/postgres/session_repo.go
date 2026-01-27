package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shinjuwu/TheNuts/internal/infra/repository"
)

// GameSessionRepo 实现 repository.GameSessionRepository
type GameSessionRepo struct {
	pool *pgxpool.Pool
}

// NewGameSessionRepository 创建新的 GameSession Repository
func NewGameSessionRepository(pool *pgxpool.Pool) repository.GameSessionRepository {
	return &GameSessionRepo{pool: pool}
}

// Create 创建新游戏会话
func (r *GameSessionRepo) Create(ctx context.Context, session *repository.GameSession) error {
	query := `
		INSERT INTO game_sessions (
			id, player_id, game_type, table_id, buy_in_amount,
			current_chips, status, started_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}

	_, err := r.pool.Exec(ctx, query,
		session.ID,
		session.PlayerID,
		session.GameType,
		session.TableID,
		session.BuyInAmount,
		session.CurrentChips,
		session.Status,
		session.StartedAt,
		session.CreatedAt,
		session.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create game session: %w", err)
	}

	return nil
}

// GetByID 根据 ID 查询游戏会话
func (r *GameSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*repository.GameSession, error) {
	query := `
		SELECT 
			id, player_id, game_type, table_id, buy_in_amount,
			current_chips, status, started_at, ended_at,
			created_at, updated_at
		FROM game_sessions
		WHERE id = $1
	`

	session := &repository.GameSession{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.PlayerID,
		&session.GameType,
		&session.TableID,
		&session.BuyInAmount,
		&session.CurrentChips,
		&session.Status,
		&session.StartedAt,
		&session.EndedAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("game session not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get game session: %w", err)
	}

	return session, nil
}

// GetActiveByPlayerID 查询玩家当前活跃的会话
func (r *GameSessionRepo) GetActiveByPlayerID(ctx context.Context, playerID uuid.UUID) (*repository.GameSession, error) {
	query := `
		SELECT 
			id, player_id, game_type, table_id, buy_in_amount,
			current_chips, status, started_at, ended_at,
			created_at, updated_at
		FROM game_sessions
		WHERE player_id = $1 AND status = 'active'
		ORDER BY started_at DESC
		LIMIT 1
	`

	session := &repository.GameSession{}
	err := r.pool.QueryRow(ctx, query, playerID).Scan(
		&session.ID,
		&session.PlayerID,
		&session.GameType,
		&session.TableID,
		&session.BuyInAmount,
		&session.CurrentChips,
		&session.Status,
		&session.StartedAt,
		&session.EndedAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // 没有活跃会话不是错误
		}
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	return session, nil
}

// Update 更新会话信息
func (r *GameSessionRepo) Update(ctx context.Context, session *repository.GameSession) error {
	query := `
		UPDATE game_sessions SET
			current_chips = $2,
			status = $3,
			updated_at = $4
		WHERE id = $1
	`

	session.UpdatedAt = time.Now()

	result, err := r.pool.Exec(ctx, query,
		session.ID,
		session.CurrentChips,
		session.Status,
		session.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update game session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("game session not found")
	}

	return nil
}

// End 结束会话
func (r *GameSessionRepo) End(ctx context.Context, id uuid.UUID, finalChips int64) error {
	query := `
		UPDATE game_sessions SET
			current_chips = $2,
			status = 'ended',
			ended_at = $3,
			updated_at = $3
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, finalChips, now)
	if err != nil {
		return fmt.Errorf("failed to end game session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("game session not found")
	}

	return nil
}
