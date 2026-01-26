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

// TransactionRepo 實作 repository.TransactionRepository
type TransactionRepo struct {
	pool *pgxpool.Pool
}

// NewTransactionRepository 創建新的 Transaction Repository
func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{pool: pool}
}

// Create 創建交易記錄（使用連接池）
func (r *TransactionRepo) Create(ctx context.Context, transaction *repository.WalletTransaction) error {
	return r.CreateWithTx(ctx, r.pool, transaction)
}

// CreateWithTx 創建交易記錄（使用指定的事務或連接池）
func (r *TransactionRepo) CreateWithTx(ctx context.Context, executor interface{}, transaction *repository.WalletTransaction) error {
	query := `
		INSERT INTO transactions (
			id, wallet_id, type, amount, balance_before, balance_after,
			description, idempotency_key, game_session_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	if transaction.ID == uuid.Nil {
		transaction.ID = uuid.New()
	}

	transaction.CreatedAt = time.Now()

	// 根據 executor 類型決定如何執行
	var err error
	switch ex := executor.(type) {
	case *pgxpool.Pool:
		_, err = ex.Exec(ctx, query,
			transaction.ID,
			transaction.WalletID,
			transaction.Type,
			transaction.Amount,
			transaction.BalanceBefore,
			transaction.BalanceAfter,
			transaction.Description,
			transaction.IdempotencyKey,
			transaction.GameSessionID,
			transaction.CreatedAt,
		)
	case pgx.Tx:
		_, err = ex.Exec(ctx, query,
			transaction.ID,
			transaction.WalletID,
			transaction.Type,
			transaction.Amount,
			transaction.BalanceBefore,
			transaction.BalanceAfter,
			transaction.Description,
			transaction.IdempotencyKey,
			transaction.GameSessionID,
			transaction.CreatedAt,
		)
	default:
		return fmt.Errorf("unsupported executor type")
	}

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// GetByID 根據 ID 查詢交易
func (r *TransactionRepo) GetByID(ctx context.Context, id uuid.UUID) (*repository.WalletTransaction, error) {
	query := `
		SELECT 
			id, wallet_id, type, amount, balance_before, balance_after,
			description, idempotency_key, game_session_id, created_at
		FROM transactions
		WHERE id = $1
	`

	tx := &repository.WalletTransaction{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tx.ID,
		&tx.WalletID,
		&tx.Type,
		&tx.Amount,
		&tx.BalanceBefore,
		&tx.BalanceAfter,
		&tx.Description,
		&tx.IdempotencyKey,
		&tx.GameSessionID,
		&tx.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("transaction not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return tx, nil
}

// GetByWalletID 根據錢包 ID 查詢交易記錄（分頁）
func (r *TransactionRepo) GetByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*repository.WalletTransaction, error) {
	query := `
		SELECT 
			id, wallet_id, type, amount, balance_before, balance_after,
			description, idempotency_key, game_session_id, created_at
		FROM transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, walletID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*repository.WalletTransaction
	for rows.Next() {
		tx := &repository.WalletTransaction{}
		err := rows.Scan(
			&tx.ID,
			&tx.WalletID,
			&tx.Type,
			&tx.Amount,
			&tx.BalanceBefore,
			&tx.BalanceAfter,
			&tx.Description,
			&tx.IdempotencyKey,
			&tx.GameSessionID,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return transactions, nil
}

// GetByIdempotencyKey 根據冪等性鍵查詢交易
func (r *TransactionRepo) GetByIdempotencyKey(ctx context.Context, key string) (*repository.WalletTransaction, error) {
	query := `
		SELECT 
			id, wallet_id, type, amount, balance_before, balance_after,
			description, idempotency_key, game_session_id, created_at
		FROM transactions
		WHERE idempotency_key = $1
	`

	tx := &repository.WalletTransaction{}
	err := r.pool.QueryRow(ctx, query, key).Scan(
		&tx.ID,
		&tx.WalletID,
		&tx.Type,
		&tx.Amount,
		&tx.BalanceBefore,
		&tx.BalanceAfter,
		&tx.Description,
		&tx.IdempotencyKey,
		&tx.GameSessionID,
		&tx.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("transaction not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return tx, nil
}

// GetByPlayerID 根據玩家 ID 查詢交易記錄（分頁）
func (r *TransactionRepo) GetByPlayerID(ctx context.Context, playerID uuid.UUID, limit, offset int) ([]*repository.WalletTransaction, error) {
	query := `
		SELECT 
			t.id, t.wallet_id, t.type, t.amount, t.balance_before, t.balance_after,
			t.description, t.idempotency_key, t.game_session_id, t.created_at
		FROM transactions t
		INNER JOIN wallets w ON w.id = t.wallet_id
		WHERE w.player_id = $1
		ORDER BY t.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, playerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*repository.WalletTransaction
	for rows.Next() {
		tx := &repository.WalletTransaction{}
		err := rows.Scan(
			&tx.ID,
			&tx.WalletID,
			&tx.Type,
			&tx.Amount,
			&tx.BalanceBefore,
			&tx.BalanceAfter,
			&tx.Description,
			&tx.IdempotencyKey,
			&tx.GameSessionID,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return transactions, nil
}
