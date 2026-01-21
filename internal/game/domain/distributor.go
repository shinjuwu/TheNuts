package domain

// Distribute 負責將 Pots 中的籌碼分配給贏家
// 這是 Side Pot 邏輯的最後一步
func Distribute(pots []*Pot, players map[string]*Player, board []Card) map[string]int64 {
	payouts := make(map[string]int64)

	for _, pot := range pots {
		if pot.Amount == 0 {
			continue
		}

		// 1. 找出此 Pot 貢獻者中的最強牌力
		var winners []string
		var maxScore int32 = -1

		for pid := range pot.Contributors {
			p, exists := players[pid]
			if !exists || p.Status == StatusFolded {
				continue // 棄牌或不存在的玩家不能贏
			}

			// 結合手牌與公牌 (7張)
			allCards := append(p.HoleCards, board...)
			// 如果不夠 5 張 (例如只有 Preflop)，只比 HoldCards?
			// 根據規則，Preflop Allin 也要發完牌與公牌結合。
			// 但如果狀態是 Preflop，board 是空的。
			// 通常 Distribute 只在 Showdown 呼叫，那時 Board 應該是滿的。
			// 如果還沒滿(例如所有人都 Fold 只剩一人)，那是另一個邏輯 (Win by Default)。
			// 這裡假設是 Showdown 且 board 已發完，或者如果沒滿就比手牌 (不標準但避免 crash)。

			score := Evaluate(allCards)
			if score > maxScore {
				maxScore = score
				winners = []string{pid}
			} else if score == maxScore {
				winners = append(winners, pid)
			}
		}

		// 2. 如果沒有贏家 (都被 Fold)，這 Pot 該怎辦?
		// 通常最後一個 Fold 的人即便 Fold 了也會贏? 不，這在 FSM 層會處理 (剩一人直接贏)。
		// 這裡假設是 Showdown，所以一定有人沒 Fold。
		// 如果真的沒人 (e.g. 大家都 disconnect)，暫時忽略或還給 Dealer (誤)。
		if len(winners) == 0 {
			continue
		}

		// 3. 分錢
		share := pot.Amount / int64(len(winners))
		remainder := pot.Amount % int64(len(winners))

		for i, pid := range winners {
			amt := share
			if int64(i) < remainder {
				// TODO: 目前餘數分配是基於 Map 迭代順序 (隨機) 或者 Slice 順序。
				// 標準規則應分配給最靠近 Button 的玩家 (Position-based)。
				amt++ // 把餘數分給前幾位
			}
			payouts[pid] += amt
		}
	}

	return payouts
}
