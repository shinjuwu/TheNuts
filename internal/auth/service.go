package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shinjuwu/TheNuts/internal/infra/repository"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials 无效的凭证
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrAccountLocked 账号被锁定
	ErrAccountLocked = errors.New("account is locked")
	// ErrAccountSuspended 账号被暂停
	ErrAccountSuspended = errors.New("account is suspended")
	// ErrAccountBanned 账号被封禁
	ErrAccountBanned = errors.New("account is banned")
	// ErrUsernameExists 用户名已存在
	ErrUsernameExists = errors.New("username already exists")
	// ErrEmailExists 邮箱已存在
	ErrEmailExists = errors.New("email already exists")
)

const (
	// MaxFailedAttempts 最大失败次数
	MaxFailedAttempts = 5
	// LockDuration 锁定时长
	LockDuration = 30 * time.Minute
	// BcryptCost bcrypt 加密强度
	BcryptCost = 12
)

// AuthService 认证服务，处理用户认证逻辑
type AuthService struct {
	accountRepo repository.AccountRepository
	playerRepo  repository.PlayerRepository
	logger      *zap.Logger
}

// NewAuthService 创建认证服务
func NewAuthService(
	accountRepo repository.AccountRepository,
	playerRepo repository.PlayerRepository,
	logger *zap.Logger,
) *AuthService {
	return &AuthService{
		accountRepo: accountRepo,
		playerRepo:  playerRepo,
		logger:      logger,
	}
}

// Register 注册新用户
func (s *AuthService) Register(ctx context.Context, username, email, password string) (*repository.Account, *repository.Player, error) {
	// 1. 验证输入
	if username == "" || email == "" || password == "" {
		return nil, nil, errors.New("username, email and password are required")
	}

	// 2. 检查用户名是否已存在
	existingAccount, err := s.accountRepo.GetByUsername(ctx, username)
	if err == nil && existingAccount != nil {
		return nil, nil, ErrUsernameExists
	}

	// 3. 检查邮箱是否已存在
	existingAccount, err = s.accountRepo.GetByEmail(ctx, email)
	if err == nil && existingAccount != nil {
		return nil, nil, ErrEmailExists
	}

	// 4. 哈希密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 5. 创建账号
	account := &repository.Account{
		ID:            uuid.New(),
		Username:      username,
		Email:         email,
		PasswordHash:  string(passwordHash),
		Status:        "active",
		EmailVerified: false, // 实际应用中需要邮箱验证
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		s.logger.Error("failed to create account",
			zap.String("username", username),
			zap.Error(err),
		)
		return nil, nil, fmt.Errorf("failed to create account: %w", err)
	}

	// 6. 创建玩家档案
	player := &repository.Player{
		ID:          uuid.New(),
		AccountID:   account.ID,
		DisplayName: username, // 默认使用用户名作为显示名称
		Level:       1,
		VipLevel:    0,
	}

	if err := s.playerRepo.Create(ctx, player); err != nil {
		s.logger.Error("failed to create player",
			zap.String("account_id", account.ID.String()),
			zap.Error(err),
		)
		return nil, nil, fmt.Errorf("failed to create player: %w", err)
	}

	s.logger.Info("user registered successfully",
		zap.String("username", username),
		zap.String("account_id", account.ID.String()),
		zap.String("player_id", player.ID.String()),
	)

	return account, player, nil
}

// Authenticate 验证用户凭证
func (s *AuthService) Authenticate(ctx context.Context, username, password, ipAddress string) (*repository.Account, *repository.Player, error) {
	// 1. 查询账号
	account, err := s.accountRepo.GetByUsername(ctx, username)
	if err != nil {
		s.logger.Warn("authentication failed: account not found",
			zap.String("username", username),
			zap.String("ip", ipAddress),
		)
		return nil, nil, ErrInvalidCredentials
	}

	// 2. 检查账号状态
	if account.Status == "suspended" {
		s.logger.Warn("authentication failed: account suspended",
			zap.String("username", username),
			zap.String("account_id", account.ID.String()),
		)
		return nil, nil, ErrAccountSuspended
	}

	if account.Status == "banned" {
		s.logger.Warn("authentication failed: account banned",
			zap.String("username", username),
			zap.String("account_id", account.ID.String()),
		)
		return nil, nil, ErrAccountBanned
	}

	// 3. 检查是否被锁定
	if account.LockedUntil != nil && account.LockedUntil.After(time.Now()) {
		s.logger.Warn("authentication failed: account locked",
			zap.String("username", username),
			zap.String("account_id", account.ID.String()),
			zap.Time("locked_until", *account.LockedUntil),
		)
		return nil, nil, ErrAccountLocked
	}

	// 4. 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		s.logger.Warn("authentication failed: invalid password",
			zap.String("username", username),
			zap.String("account_id", account.ID.String()),
			zap.String("ip", ipAddress),
		)

		// 增加失败次数
		if err := s.accountRepo.IncrementFailedAttempts(ctx, account.ID); err != nil {
			s.logger.Error("failed to increment failed attempts", zap.Error(err))
		}

		// 检查是否需要锁定账号
		account.FailedLoginAttempts++
		if account.FailedLoginAttempts >= MaxFailedAttempts {
			lockUntil := time.Now().Add(LockDuration)
			if err := s.accountRepo.LockAccount(ctx, account.ID, lockUntil); err != nil {
				s.logger.Error("failed to lock account", zap.Error(err))
			} else {
				s.logger.Warn("account locked due to too many failed attempts",
					zap.String("account_id", account.ID.String()),
					zap.Time("locked_until", lockUntil),
				)
			}
		}

		return nil, nil, ErrInvalidCredentials
	}

	// 5. 密码正确，重置失败次数
	if account.FailedLoginAttempts > 0 {
		if err := s.accountRepo.ResetFailedAttempts(ctx, account.ID); err != nil {
			s.logger.Error("failed to reset failed attempts", zap.Error(err))
		}
	}

	// 6. 更新最后登录时间和 IP
	if err := s.accountRepo.UpdateLastLogin(ctx, account.ID, ipAddress); err != nil {
		s.logger.Error("failed to update last login", zap.Error(err))
		// 不影响登录流程，继续
	}

	// 7. 查询玩家信息
	player, err := s.playerRepo.GetByAccountID(ctx, account.ID)
	if err != nil {
		s.logger.Error("failed to get player by account ID",
			zap.String("account_id", account.ID.String()),
			zap.Error(err),
		)
		return nil, nil, fmt.Errorf("failed to get player: %w", err)
	}

	s.logger.Info("user authenticated successfully",
		zap.String("username", username),
		zap.String("account_id", account.ID.String()),
		zap.String("player_id", player.ID.String()),
		zap.String("ip", ipAddress),
	)

	return account, player, nil
}

// GetPlayerByAccountID 根据账号 ID 获取玩家信息
func (s *AuthService) GetPlayerByAccountID(ctx context.Context, accountID uuid.UUID) (*repository.Player, error) {
	return s.playerRepo.GetByAccountID(ctx, accountID)
}

// HashPassword 哈希密码（辅助函数，可用于测试或管理命令）
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ComparePassword 比较密码（辅助函数）
func ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
