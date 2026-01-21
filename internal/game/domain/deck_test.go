package domain

import (
	"fmt"
	"testing"
)

func TestCardString(t *testing.T) {
	// A Spades: Rank=12(A), Suit=3(Spade)
	c := NewCard(RankA, SuitSpade)
	if s := c.String(); s != "As" {
		t.Errorf("Expected As, got %s", s)
	}

	// 2 Clubs: Rank=0(2), Suit=0(Club)
	c2 := NewCard(Rank2, SuitClub)
	if s := c2.String(); s != "2c" {
		t.Errorf("Expected 2c, got %s", s)
	}
}

func TestNewDeck(t *testing.T) {
	d := NewDeck()
	if len(d.Cards) != 52 {
		t.Errorf("Expected 52 cards, got %d", len(d.Cards))
	}

	// 檢查是否有重複
	seen := make(map[Card]bool)
	for _, c := range d.Cards {
		if seen[c] {
			t.Errorf("Duplicate card found: %s", c.String())
		}
		seen[c] = true
	}
}

func TestShuffle(t *testing.T) {
	d := NewDeck()
	original := make([]Card, len(d.Cards))
	copy(original, d.Cards)

	d.Shuffle()

	// 檢查是否順序改變 (極低機率會一樣，連續測兩次幾乎不可能失敗)
	sameOrder := true
	for i, c := range d.Cards {
		if c != original[i] {
			sameOrder = false
			break
		}
	}

	if sameOrder {
		t.Log("Warning: Shuffle resulted in same order (extremely rare but possible)")
	}
}

func TestDraw(t *testing.T) {
	d := NewDeck()
	drawn := d.Draw(5)

	if len(drawn) != 5 {
		t.Errorf("Expected 5 cards drawn, got %d", len(drawn))
	}
	if len(d.Cards) != 47 {
		t.Errorf("Expected 47 cards remaining, got %d", len(d.Cards))
	}

	// 檢查 Draw 超出範圍
	tooMany := d.Draw(100)
	if tooMany != nil {
		t.Error("Expected nil when drawing more than remaining")
	}
}

func ExampleCard_String() {
	c := NewCard(RankK, SuitHeart)
	fmt.Println(c.String())
	// Output: Kh
}
