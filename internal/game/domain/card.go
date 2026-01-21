package domain

import "fmt"

/****************************************************************************************
 * Card Encoding Scheme (inspired by Cactus Kev / 2+2)
 *
 * 我們使用 int32 來編碼一張牌，為了讓比牌演算法能達到極致效能。
 *
 * Bits:
 * Bits:
 * 0-3   (4 bits): Rank (0-12) where 2=0, ... A=12
 * 4-7   (4 bits): Suit (0-3) where Club=0, ..., Spade=3
 *
 * Note: This differs from the original Cactus Kev Prime-based implementation.
 * We are using a simplified (Suit << 4) | Rank encoding for now.
 * Structure: [Suit 4bit] [Rank 4bit]
 *
 * Example: Ace of Spades
 * Rank = 12 (A), Suit = Spade
 * Prime = 41
 * Format: xxx... [RankMask] [Suit] [Rank] [Prime]
 ****************************************************************************************/

type Card int32

// 我們先定義基礎的 Rank 和 Suit 常數
const (
	Rank2 = 0
	Rank3 = 1
	Rank4 = 2
	Rank5 = 3
	Rank6 = 4
	Rank7 = 5
	Rank8 = 6
	Rank9 = 7
	RankT = 8 // Ten
	RankJ = 9
	RankQ = 10
	RankK = 11
	RankA = 12
)

const (
	SuitClub    = 0 // 梅花
	SuitDiamond = 1 // 方塊
	SuitHeart   = 2 // 紅心
	SuitSpade   = 3 // 黑桃
)

// NewCard 建立一張牌
// 目前為了簡化顯示，我們暫時使用 (Suit << 8) | Rank 的簡單編碼。
// 當需要實作高效 Evaluator 時，我們會在 Evaluator 內部將此格式轉換為 Cactus Kev 格式，
// 或者直接在這裡實作複雜編碼。為了可讀性與測試，先採用簡單位元組合。
func NewCard(rank, suit int) Card {
	return Card((suit << 4) | rank)
}

func (c Card) Rank() int {
	return int(c & 0xF)
}

func (c Card) Suit() int {
	return int((c >> 4) & 0xF)
}

var rankChars = "23456789TJQKA"
var suitChars = "cdhs" // club, diamond, heart, spade

// String 回傳如 "Ah", "Ks" 的字串
func (c Card) String() string {
	r := c.Rank()
	s := c.Suit()
	if r < 0 || r > 12 || s < 0 || s > 3 {
		return "??"
	}
	return fmt.Sprintf("%c%c", rankChars[r], suitChars[s])
}
