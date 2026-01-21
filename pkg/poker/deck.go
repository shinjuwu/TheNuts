package poker

import (
	"crypto/rand"
	"math/big"
)

type Deck []int

func NewDeck() Deck {
	d := make(Deck, 52)
	for i := 0; i < 52; i++ {
		d[i] = i
	}
	return d
}

func (d Deck) Shuffle() {
	n := len(d)
	for i := n - 1; i > 0; i-- {
		jBig, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		j := int(jBig.Int64())
		d[i], d[j] = d[j], d[i]
	}
}
