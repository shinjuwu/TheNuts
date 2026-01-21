package domain

import (
	"crypto/rand"
	"math/big"
)

type Deck struct {
	Cards []Card
}

// NewDeck 建立一副全新的 52 張牌
func NewDeck() *Deck {
	d := &Deck{
		Cards: make([]Card, 0, 52),
	}
	for suit := 0; suit < 4; suit++ {
		for rank := 0; rank < 13; rank++ {
			d.Cards = append(d.Cards, NewCard(rank, suit))
		}
	}
	return d
}

// Shuffle 使用加密級隨機數洗牌 (Fisher-Yates Shuffle with crypto/rand)
func (d *Deck) Shuffle() {
	n := len(d.Cards)
	for i := n - 1; i > 0; i-- {
		// 生成 0 到 i 之間的隨機數
		// crypto/rand.Int 回傳的是 [0, max)，所以這裡傳入 i+1
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			// 在極端無法讀取系統隨機源的情況下，fallback 或 panic
			// 為了遊戲公平性，這裡選擇 panic 以防偽隨機被利用
			panic("failed to generate secure random number: " + err.Error())
		}
		j := int(jBig.Int64())

		// 交換
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	}
}

// Draw 從牌頂發 n 張牌
func (d *Deck) Draw(n int) []Card {
	if n > len(d.Cards) {
		return nil
	}
	drawn := d.Cards[:n]
	d.Cards = d.Cards[n:]
	return drawn
}
