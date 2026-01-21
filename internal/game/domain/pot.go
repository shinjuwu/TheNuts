package domain

// Pot 代表一個底池
type Pot struct {
	Amount int64
	// Contributors 包含所有對此 Pot 有貢獻的玩家 ID (含已 Fold)。
	// 注意: 派彩時 (Distributor) 必須再次檢查玩家是否 StatusFolded，確定是否有資格贏取。
	Contributors map[string]bool
}

func NewPot() *Pot {
	return &Pot{
		Amount:       0,
		Contributors: make(map[string]bool),
	}
}

// CanWin 檢查玩家是否有資格贏取此底池
func (p *Pot) CanWin(playerID string) bool {
	return p.Contributors[playerID]
}

// PotManager 管理所有底池
type PotManager struct {
	Pots []*Pot
}

func NewPotManager() *PotManager {
	// 初始化時至少有一個空的 Main Pot
	return &PotManager{
		Pots: []*Pot{NewPot()},
	}
}

// Accumulate 將本輪下注額分配到各個邊池中
// 這是 Side Pot 的核心演算法
func (pm *PotManager) Accumulate(bets map[string]int64) {
	// 1. 移除 0 的下注 (沒下注的人不影響切分)
	validBets := make(map[string]int64)
	for pid, amt := range bets {
		if amt > 0 {
			validBets[pid] = amt
		}
	}

	if len(validBets) == 0 {
		return
	}

	// 2. 切分下注額 (Slicing) 為多個臨時 Pot
	// 例如: P1:100, P2:200, P3:500
	// Slice 1 (100): P1, P2, P3 (Amt: 300)
	// Slice 2 (100): P2, P3     (Amt: 200)
	// Slice 3 (300): P3         (Amt: 300)
	tempPots := sliceBets(validBets)

	// 3. 將臨時 Pots 合併到現有的 Pots 中
	// 邏輯: 依序檢查臨時 Pot 的 Contributors 是否與最後一個現有 Pot 相同
	// 若相同 -> 合併金額
	// 若不同 -> 新增為新的 Side Pot
	for _, tp := range tempPots {
		lastPot := pm.Pots[len(pm.Pots)-1]

		// 如果最後一個 Pot 是空的 (剛初始化的 Main Pot)，直接使用 tp 的 contributors
		if len(lastPot.Contributors) == 0 && lastPot.Amount == 0 {
			lastPot.Amount = tp.Amount
			lastPot.Contributors = tp.Contributors
			continue
		}

		if areContributorsSame(lastPot.Contributors, tp.Contributors) {
			lastPot.Amount += tp.Amount
		} else {
			pm.Pots = append(pm.Pots, tp)
		}
	}
}

// Total 取得目前所有底池總金額
func (pm *PotManager) Total() int64 {
	var total int64
	for _, p := range pm.Pots {
		total += p.Amount
	}
	return total
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

func sliceBets(bets map[string]int64) []*Pot {
	var slices []*Pot

	for len(bets) > 0 {
		// 找出目前最小的非零下注額
		var minBet int64 = -1
		for _, amt := range bets {
			if minBet == -1 || amt < minBet {
				minBet = amt
			}
		}

		if minBet <= 0 {
			break // Should not happen if filtered correctly
		}

		// 建立這一層的 Pot
		pot := NewPot()
		contributors := make([]string, 0)

		// 收集貢獻者並扣除金額
		for pid, amt := range bets {
			contribution := minBet
			if amt < minBet {
				contribution = amt // 理論上不會發生，因為 minBet 是最小的
			}

			pot.Amount += contribution
			pot.Contributors[pid] = true
			contributors = append(contributors, pid)

			// 扣除已處理金額
			remain := amt - contribution
			if remain == 0 {
				delete(bets, pid)
			} else {
				bets[pid] = remain
			}
		}
		slices = append(slices, pot)
	}
	return slices
}

func areContributorsSame(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}
