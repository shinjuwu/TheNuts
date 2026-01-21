package domain

import (
	"sort"
)

// HandCategory 代表牌型强度 (越小越強? No, let's use bigger is better)
// 為了比牌方便，我們讓數值越大代表牌型越強
type HandCategory int

const (
	HandHighCard HandCategory = iota
	HandPair
	HandTwoPair
	HandThreeOfAKind
	HandStraight
	HandFlush
	HandFullHouse
	HandFourOfAKind
	HandStraightFlush
	HandRoyalFlush
)

// Evaluate 計算 5-7 張牌的最大牌力分數
// 回傳值是一個 int32:
// Bits 24-27: HandCategory (0-9)
// Bits 0-23: Kickers (用於同牌型比大小)
//
// 為了實作簡單且正確，我們這裡先採用 "遍歷所有 5 張牌組合" 的方式找出最大牌型。
// 對於 7 選 5，總共有 C(7,5) = 21 種組合，運算量非常小。
func Evaluate(cards []Card) int32 {
	if len(cards) < 5 {
		return 0
	}

	var maxScore int32 = 0

	// 產生所有 5 張牌的組合
	combs := combinations(cards, 5)
	for _, comb := range combs {
		score := evaluate5(comb)
		if score > maxScore {
			maxScore = score
		}
	}
	return maxScore
}

// combinations 生成 n 選 k 的所有組合
func combinations(set []Card, k int) [][]Card {
	var result [][]Card
	var recurse func(start int, current []Card)
	recurse = func(start int, current []Card) {
		if len(current) == k {
			temp := make([]Card, k)
			copy(temp, current)
			result = append(result, temp)
			return
		}
		for i := start; i < len(set); i++ {
			recurse(i+1, append(current, set[i]))
		}
	}
	recurse(0, []Card{})
	return result
}

// evaluate5 計算 5 張牌的分數
func evaluate5(cards []Card) int32 {
	// 先排序，方便後續判斷順子與比大小
	// 注意: 這裡我們複製一份以免影響原 slice，但 evaluate5 每次收到的是 combination 的 copy 嗎？
	// combinations 函式裡 copy 了 temp，所以這裡是安全的。
	// 我們需要根據 Rank 排序 (大到小)
	sorted := make([]Card, 5)
	copy(sorted, cards)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Rank() > sorted[j].Rank() // Descending
	})

	isFlush := checkFlush(sorted)
	isStraight := checkStraight(sorted)

	// 1. Straight Flush & Royal Flush
	if isFlush && isStraight {
		if sorted[0].Rank() == RankA && sorted[1].Rank() == RankK {
			return makeScore(HandRoyalFlush, 0)
		}
		return makeScore(HandStraightFlush, sorted[0].Rank())
	}

	// 2. Count Ranks for Quads, FullHouse, Trips, TwoPair, Pair
	counts := make(map[int]int)
	for _, c := range sorted {
		counts[c.Rank()]++
	}

	var four, three, pair1, pair2 int = -1, -1, -1, -1
	for r, count := range counts {
		if count == 4 {
			four = r
		} else if count == 3 {
			if r > three { // 雖然 5 張牌不可能有兩組三條，但邏輯上保持一致
				three = r
			}
		} else if count == 2 {
			if r > pair1 {
				pair2 = pair1 // shift down
				pair1 = r
			} else if r > pair2 {
				pair2 = r
			}
		}
	}

	// 3. Four of a Kind
	if four != -1 {
		kicker := 0
		for _, c := range sorted {
			if c.Rank() != four {
				kicker = c.Rank()
				break
			}
		}
		return makeScore(HandFourOfAKind, (four<<4)|kicker)
	}

	// 4. Full House
	if three != -1 && pair1 != -1 {
		return makeScore(HandFullHouse, (three<<4)|pair1)
	}

	// 5. Flush
	if isFlush {
		// FlushKick: R1 R2 R3 R4 R5
		val := 0
		for _, c := range sorted {
			val = (val << 4) | c.Rank()
		}
		return makeScore(HandFlush, val)
	}

	// 6. Straight
	if isStraight {
		// A-5 Straight check is handled in checkStraight?
		// checkStraight returns true/false. If A-2-3-4-5, logic needs spec.
		// Standard straight high card logic.
		high := sorted[0].Rank()
		if sorted[0].Rank() == RankA && sorted[1].Rank() == Rank5 {
			high = Rank5 // A-5 Straight, high card is 5
		}
		return makeScore(HandStraight, high)
	}

	// 7. Three of a Kind
	if three != -1 {
		kickerVal := 0
		for _, c := range sorted {
			if c.Rank() != three {
				kickerVal = (kickerVal << 4) | c.Rank()
			}
		}
		return makeScore(HandThreeOfAKind, (three<<8)|kickerVal)
	}

	// 8. Two Pair
	if pair1 != -1 && pair2 != -1 {
		kicker := 0
		for _, c := range sorted {
			if c.Rank() != pair1 && c.Rank() != pair2 {
				kicker = c.Rank()
				break
			}
		}
		return makeScore(HandTwoPair, (pair1<<8)|(pair2<<4)|kicker)
	}

	// 9. Pair
	if pair1 != -1 {
		kickerVal := 0
		for _, c := range sorted {
			if c.Rank() != pair1 {
				kickerVal = (kickerVal << 4) | c.Rank()
			}
		}
		return makeScore(HandPair, (pair1<<12)|kickerVal)
	}

	// 10. High Card
	val := 0
	for _, c := range sorted {
		val = (val << 4) | c.Rank()
	}
	return makeScore(HandHighCard, val)
}

func checkFlush(sorted []Card) bool {
	s := sorted[0].Suit()
	for i := 1; i < 5; i++ {
		if sorted[i].Suit() != s {
			return false
		}
	}
	return true
}

func checkStraight(sorted []Card) bool {
	// Special Case: A-5-4-3-2 (Wheel)
	// sorted[0] is A, sorted[1] is 5? (Assuming sorted desc)
	// A(12), 5(3), 4(2), 3(1), 2(0)
	if sorted[0].Rank() == RankA && sorted[1].Rank() == Rank5 {
		// Check 5-4-3-2
		for i := 1; i < 4; i++ {
			if sorted[i].Rank() != sorted[i+1].Rank()+1 {
				return false
			}
		}
		// check last is 2
		return sorted[4].Rank() == Rank2
	}

	for i := 0; i < 4; i++ {
		if sorted[i].Rank() != sorted[i+1].Rank()+1 {
			return false
		}
	}
	return true
}

func makeScore(cat HandCategory, kickers int) int32 {
	return int32(cat)<<24 | int32(kickers)
}
