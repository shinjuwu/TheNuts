package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shinjuwu/TheNuts/internal/infra/repository"
)

// PgTransaction 實作 repository.Transaction 介面
type PgTransaction struct {
	tx pgx.Tx
}

// Commit 提交事務
func (t *PgTransaction) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback 回滾事務
func (t *PgTransaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// GetTx 返回底層的 pgx.Tx（內部使用）
func (t *PgTransaction) GetTx() pgx.Tx {
	return t.tx
}

// PgUnitOfWork 實作 repository.UnitOfWork 介面
type PgUnitOfWork struct {
	pool *pgxpool.Pool
}

// NewUnitOfWork 創建新的工作單元
func NewUnitOfWork(pool *pgxpool.Pool) repository.UnitOfWork {
	return &PgUnitOfWork{
		pool: pool,
	}
}

// Begin 開始新事務
func (u *PgUnitOfWork) Begin(ctx context.Context) (repository.Transaction, error) {
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &PgTransaction{tx: tx}, nil
}

// WithTransaction 在事務中執行函數
func (u *PgUnitOfWork) WithTransaction(ctx context.Context, fn func(tx repository.Transaction) error) error {
	// 開始事務
	tx, err := u.Begin(ctx)
	if err != nil {
		return err
	}

	// 延遲處理：確保事務被正確提交或回滾
	defer func() {
		if p := recover(); p != nil {
			// 如果 panic，回滾事務
			_ = tx.Rollback(ctx)
			panic(p) // 重新拋出 panic
		}
	}()

	// 執行業務邏輯
	err = fn(tx)
	if err != nil {
		// 如果出錯，回滾事務
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	// 提交事務
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
