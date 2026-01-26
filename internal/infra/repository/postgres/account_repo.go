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

// AccountRepo 實作 repository.AccountRepository
type AccountRepo struct {
	pool *pgxpool.Pool
}

// NewAccountRepository 創建新的 Account Repository
func NewAccountRepository(pool *pgxpool.Pool) repository.AccountRepository {
	return &AccountRepo{pool: pool}
}

// Create 創建新帳號
func (r *AccountRepo) Create(ctx context.Context, account *repository.Account) error {
	query := `
		INSERT INTO accounts (
			id, username, email, password_hash, status, 
			email_verified, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now

	if account.ID == uuid.Nil {
		account.ID = uuid.New()
	}

	_, err := r.pool.Exec(ctx, query,
		account.ID,
		account.Username,
		account.Email,
		account.PasswordHash,
		account.Status,
		account.EmailVerified,
		account.CreatedAt,
		account.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	return nil
}

// GetByID 根據 ID 查詢帳號
func (r *AccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*repository.Account, error) {
	query := `
		SELECT 
			id, username, email, password_hash, status, 
			email_verified, failed_login_attempts, locked_until,
			last_login_at, last_login_ip, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`

	account := &repository.Account{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&account.ID,
		&account.Username,
		&account.Email,
		&account.PasswordHash,
		&account.Status,
		&account.EmailVerified,
		&account.FailedLoginAttempts,
		&account.LockedUntil,
		&account.LastLoginAt,
		&account.LastLoginIP,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("account not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return account, nil
}

// GetByUsername 根據用戶名查詢帳號
func (r *AccountRepo) GetByUsername(ctx context.Context, username string) (*repository.Account, error) {
	query := `
		SELECT 
			id, username, email, password_hash, status, 
			email_verified, failed_login_attempts, locked_until,
			last_login_at, last_login_ip, created_at, updated_at
		FROM accounts
		WHERE username = $1
	`

	account := &repository.Account{}
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&account.ID,
		&account.Username,
		&account.Email,
		&account.PasswordHash,
		&account.Status,
		&account.EmailVerified,
		&account.FailedLoginAttempts,
		&account.LockedUntil,
		&account.LastLoginAt,
		&account.LastLoginIP,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("account not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return account, nil
}

// GetByEmail 根據郵箱查詢帳號
func (r *AccountRepo) GetByEmail(ctx context.Context, email string) (*repository.Account, error) {
	query := `
		SELECT 
			id, username, email, password_hash, status, 
			email_verified, failed_login_attempts, locked_until,
			last_login_at, last_login_ip, created_at, updated_at
		FROM accounts
		WHERE email = $1
	`

	account := &repository.Account{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&account.ID,
		&account.Username,
		&account.Email,
		&account.PasswordHash,
		&account.Status,
		&account.EmailVerified,
		&account.FailedLoginAttempts,
		&account.LockedUntil,
		&account.LastLoginAt,
		&account.LastLoginIP,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("account not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return account, nil
}

// Update 更新帳號信息
func (r *AccountRepo) Update(ctx context.Context, account *repository.Account) error {
	query := `
		UPDATE accounts SET
			username = $2,
			email = $3,
			password_hash = $4,
			status = $5,
			email_verified = $6,
			updated_at = $7
		WHERE id = $1
	`

	account.UpdatedAt = time.Now()

	result, err := r.pool.Exec(ctx, query,
		account.ID,
		account.Username,
		account.Email,
		account.PasswordHash,
		account.Status,
		account.EmailVerified,
		account.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("account not found")
	}

	return nil
}

// UpdateLastLogin 更新最後登入時間和 IP
func (r *AccountRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	query := `
		UPDATE accounts SET
			last_login_at = $2,
			last_login_ip = $3,
			updated_at = $4
		WHERE id = $1
	`

	now := time.Now()
	_, err := r.pool.Exec(ctx, query, id, now, ip, now)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// IncrementFailedAttempts 增加失敗登入次數
func (r *AccountRepo) IncrementFailedAttempts(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE accounts SET
			failed_login_attempts = failed_login_attempts + 1,
			updated_at = $2
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to increment failed attempts: %w", err)
	}

	return nil
}

// ResetFailedAttempts 重置失敗登入次數
func (r *AccountRepo) ResetFailedAttempts(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE accounts SET
			failed_login_attempts = 0,
			locked_until = NULL,
			updated_at = $2
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to reset failed attempts: %w", err)
	}

	return nil
}

// LockAccount 鎖定帳號直到指定時間
func (r *AccountRepo) LockAccount(ctx context.Context, id uuid.UUID, until time.Time) error {
	query := `
		UPDATE accounts SET
			locked_until = $2,
			updated_at = $3
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, until, time.Now())
	if err != nil {
		return fmt.Errorf("failed to lock account: %w", err)
	}

	return nil
}
