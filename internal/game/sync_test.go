package game

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shinjuwu/TheNuts/internal/game/domain"
	"github.com/shinjuwu/TheNuts/internal/game/service"
	"github.com/shinjuwu/TheNuts/internal/infra/repository"
	"go.uber.org/zap"
)

// Mock Session Repo
type mockSessionRepo struct {
	mu               sync.RWMutex
	activeSessions   map[uuid.UUID]*repository.GameSession
	sessionsByPlayer map[uuid.UUID]*repository.GameSession
	updatedChips     map[uuid.UUID]int64
}

func newMockRepo() *mockSessionRepo {
	return &mockSessionRepo{
		activeSessions:   make(map[uuid.UUID]*repository.GameSession),
		sessionsByPlayer: make(map[uuid.UUID]*repository.GameSession),
		updatedChips:     make(map[uuid.UUID]int64),
	}
}

func (m *mockSessionRepo) Create(ctx context.Context, session *repository.GameSession) error {
	m.activeSessions[session.ID] = session
	m.sessionsByPlayer[session.PlayerID] = session
	return nil
}

func (m *mockSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*repository.GameSession, error) {
	if s, ok := m.activeSessions[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("session not found")
}

func (m *mockSessionRepo) GetActiveByPlayerID(ctx context.Context, playerID uuid.UUID) (*repository.GameSession, error) {
	if s, ok := m.sessionsByPlayer[playerID]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("session not found")
}

func (m *mockSessionRepo) Update(ctx context.Context, session *repository.GameSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updatedChips[session.ID] = session.CurrentChips
	return nil
}

func (m *mockSessionRepo) End(ctx context.Context, id uuid.UUID, finalChips int64) error {
	return nil
}

// Implement other interface methods to satisfy interface
// Since we don't trigger them in this test, empty imp is fine?
// But Go interface satisfaction requires method signature.
// Interface is in infra/repository/interfaces.go. GameSessionRepository has Create, GetByID, GetActiveByPlayerID, Update, End.
// We implemented all 5. So we are good.

func TestChipSync(t *testing.T) {
	// 1. Setup Mock Repo
	mockRepo := newMockRepo()

	// 2. Setup Data
	playerID := uuid.New()
	sessionID := uuid.New()
	session := &repository.GameSession{
		ID:           sessionID,
		PlayerID:     playerID,
		CurrentChips: 1000,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	mockRepo.Create(context.Background(), session)

	// 3. Create Service
	// GameService deps: PlayerRepo, WalletRepo, SessionRepo, UoW, Logger.
	// We only need SessionRepo for Sync.
	gs := service.NewGameService(nil, nil, mockRepo, nil, zap.NewNop())

	// 4. Create TableManager
	tm := NewTableManager(gs)

	// 5. Create Table
	tableName := "test-table-sync"
	table := tm.GetOrCreateTable(tableName)

	// 6. Add Player to Table (Simulate join)
	// Domain Player ID is string.
	pIDStr := playerID.String()
	domainPlayer := &domain.Player{
		ID:    pIDStr,
		Chips: 2500, // Chips changed from 1000 to 2500
	}
	table.Players[pIDStr] = domainPlayer

	// 7. Trigger Sync manually (as if hand ended)
	// We call the callback logic directly via the hook on the table instance
	if !table.HasOnHandCompleteCallbacks() {
		t.Fatal("OnHandComplete callbacks are empty")
	}

	table.FireOnHandComplete()

	// 8. Verify
	// onHandComplete 為異步執行，等待 goroutine 完成
	time.Sleep(100 * time.Millisecond)

	mockRepo.mu.RLock()
	chips, ok := mockRepo.updatedChips[sessionID]
	mockRepo.mu.RUnlock()
	if !ok {
		t.Fatal("Session Update was NOT called")
	}
	if chips != 2500 {
		t.Errorf("Expected 2500 chips, got %d", chips)
	}
}
