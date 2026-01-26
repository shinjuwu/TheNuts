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

// PlayerRepo 實作 repository.PlayerRepository
type PlayerRepo struct {
	pool *pgxpool.Pool
}

// NewPlayerRepository 創建新的 Player Repository
func NewPlayerRepository(pool *pgxpool.Pool) repository.PlayerRepository {
	return &PlayerRepo{pool: pool}
}

// Create 創建新玩家
func (r *PlayerRepo) Create(ctx context.Context, player *repository.Player) error {
	query := `
		INSERT INTO players (
			id, account_id, display_name, avatar_url, level, 
			experience, total_games_played, total_hands_played, 
			total_winnings, vip_level, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	now := time.Now()
	player.CreatedAt = now
	player.UpdatedAt = now

	if player.ID == uuid.Nil {
		player.ID = uuid.New()
	}

	_, err := r.pool.Exec(ctx, query,
		player.ID,
		player.AccountID,
		player.DisplayName,
		player.AvatarURL,
		player.Level,
		player.Experience,
		player.TotalGamesPlayed,
		player.TotalHandsPlayed,
		player.TotalWinnings,
		player.VipLevel,
		player.CreatedAt,
		player.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create player: %w", err)
	}

	return nil
}

// GetByID 根據 ID 查詢玩家
func (r *PlayerRepo) GetByID(ctx context.Context, id uuid.UUID) (*repository.Player, error) {
	query := `
		SELECT 
			id, account_id, display_name, avatar_url, level, 
			experience, total_games_played, total_hands_played, 
			total_winnings, vip_level, vip_expires_at, created_at, updated_at
		FROM players
		WHERE id = $1
	`

	player := &repository.Player{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&player.ID,
		&player.AccountID,
		&player.DisplayName,
		&player.AvatarURL,
		&player.Level,
		&player.Experience,
		&player.TotalGamesPlayed,
		&player.TotalHandsPlayed,
		&player.TotalWinnings,
		&player.VipLevel,
		&player.VipExpiresAt,
		&player.CreatedAt,
		&player.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("player not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	return player, nil
}

// GetByAccountID 根據帳號 ID 查詢玩家
func (r *PlayerRepo) GetByAccountID(ctx context.Context, accountID uuid.UUID) (*repository.Player, error) {
	query := `
		SELECT 
			id, account_id, display_name, avatar_url, level, 
			experience, total_games_played, total_hands_played, 
			total_winnings, vip_level, vip_expires_at, created_at, updated_at
		FROM players
		WHERE account_id = $1
	`

	player := &repository.Player{}
	err := r.pool.QueryRow(ctx, query, accountID).Scan(
		&player.ID,
		&player.AccountID,
		&player.DisplayName,
		&player.AvatarURL,
		&player.Level,
		&player.Experience,
		&player.TotalGamesPlayed,
		&player.TotalHandsPlayed,
		&player.TotalWinnings,
		&player.VipLevel,
		&player.VipExpiresAt,
		&player.CreatedAt,
		&player.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("player not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	return player, nil
}

// Update 更新玩家信息
func (r *PlayerRepo) Update(ctx context.Context, player *repository.Player) error {
	query := `
		UPDATE players SET
			display_name = $2,
			avatar_url = $3,
			level = $4,
			experience = $5,
			vip_level = $6,
			vip_expires_at = $7,
			updated_at = $8
		WHERE id = $1
	`

	player.UpdatedAt = time.Now()

	result, err := r.pool.Exec(ctx, query,
		player.ID,
		player.DisplayName,
		player.AvatarURL,
		player.Level,
		player.Experience,
		player.VipLevel,
		player.VipExpiresAt,
		player.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update player: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("player not found")
	}

	return nil
}

// UpdateStats 更新玩家統計數據
func (r *PlayerRepo) UpdateStats(ctx context.Context, id uuid.UUID, handsPlayed, handsWon int, totalWinnings int64) error {
	query := `
		UPDATE players SET
			total_hands_played = total_hands_played + $2,
			total_winnings = total_winnings + $3,
			updated_at = $4
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, handsPlayed, totalWinnings, now)
	if err != nil {
		return fmt.Errorf("failed to update player stats: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("player not found")
	}

	return nil
}

// GetTopPlayersByWinnings 查詢總贏利排行榜
func (r *PlayerRepo) GetTopPlayersByWinnings(ctx context.Context, limit int) ([]*repository.Player, error) {
	query := `
		SELECT 
			id, account_id, display_name, avatar_url, level, 
			experience, total_games_played, total_hands_played, 
			total_winnings, vip_level, vip_expires_at, created_at, updated_at
		FROM players
		ORDER BY total_winnings DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top players: %w", err)
	}
	defer rows.Close()

	var players []*repository.Player
	for rows.Next() {
		player := &repository.Player{}
		err := rows.Scan(
			&player.ID,
			&player.AccountID,
			&player.DisplayName,
			&player.AvatarURL,
			&player.Level,
			&player.Experience,
			&player.TotalGamesPlayed,
			&player.TotalHandsPlayed,
			&player.TotalWinnings,
			&player.VipLevel,
			&player.VipExpiresAt,
			&player.CreatedAt,
			&player.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan player: %w", err)
		}
		players = append(players, player)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return players, nil
}
