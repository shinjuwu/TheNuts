package domain

import (
	"fmt"
	"testing"
)

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name     string
		cards    []Card
		expected HandCategory
	}{
		{
			name: "Royal Flush",
			cards: []Card{
				NewCard(RankA, SuitSpade), NewCard(RankK, SuitSpade), NewCard(RankQ, SuitSpade),
				NewCard(RankJ, SuitSpade), NewCard(RankT, SuitSpade), NewCard(Rank2, SuitHeart), NewCard(Rank3, SuitDiamond),
			},
			expected: HandRoyalFlush,
		},
		{
			name: "Straight Flush",
			cards: []Card{
				NewCard(Rank9, SuitHeart), NewCard(Rank8, SuitHeart), NewCard(Rank7, SuitHeart),
				NewCard(Rank6, SuitHeart), NewCard(Rank5, SuitHeart), NewCard(RankA, SuitSpade), NewCard(RankK, SuitSpade),
			},
			expected: HandStraightFlush,
		},
		{
			name: "Four of a Kind",
			cards: []Card{
				NewCard(RankA, SuitSpade), NewCard(RankA, SuitHeart), NewCard(RankA, SuitDiamond),
				NewCard(RankA, SuitClub), NewCard(RankK, SuitSpade), NewCard(Rank2, SuitHeart),
			},
			expected: HandFourOfAKind,
		},
		{
			name: "Full House",
			cards: []Card{
				NewCard(RankK, SuitSpade), NewCard(RankK, SuitHeart), NewCard(RankK, SuitDiamond),
				NewCard(RankQ, SuitClub), NewCard(RankQ, SuitSpade), NewCard(Rank2, SuitHeart),
			},
			expected: HandFullHouse,
		},
		{
			name: "Flush",
			cards: []Card{
				NewCard(RankA, SuitSpade), NewCard(RankJ, SuitSpade), NewCard(Rank8, SuitSpade),
				NewCard(Rank6, SuitSpade), NewCard(Rank2, SuitSpade), NewCard(RankK, SuitHeart),
			},
			expected: HandFlush,
		},
		{
			name: "Straight",
			cards: []Card{
				NewCard(Rank9, SuitSpade), NewCard(Rank8, SuitHeart), NewCard(Rank7, SuitDiamond),
				NewCard(Rank6, SuitClub), NewCard(Rank5, SuitSpade), NewCard(RankA, SuitHeart),
			},
			expected: HandStraight,
		},
		{
			name: "Straight (Wheel A-5)",
			cards: []Card{
				NewCard(RankA, SuitSpade), NewCard(Rank5, SuitHeart), NewCard(Rank4, SuitDiamond),
				NewCard(Rank3, SuitClub), NewCard(Rank2, SuitSpade), NewCard(RankK, SuitHeart),
			},
			expected: HandStraight,
		},
		{
			name: "Three of a Kind",
			cards: []Card{
				NewCard(Rank8, SuitSpade), NewCard(Rank8, SuitHeart), NewCard(Rank8, SuitDiamond),
				NewCard(RankA, SuitClub), NewCard(RankK, SuitSpade),
			},
			expected: HandThreeOfAKind,
		},
		{
			name: "Two Pair",
			cards: []Card{
				NewCard(Rank8, SuitSpade), NewCard(Rank8, SuitHeart),
				NewCard(Rank4, SuitDiamond), NewCard(Rank4, SuitClub),
				NewCard(RankA, SuitSpade),
			},
			expected: HandTwoPair,
		},
		{
			name: "Pair",
			cards: []Card{
				NewCard(RankA, SuitSpade), NewCard(RankA, SuitHeart),
				NewCard(RankK, SuitDiamond), NewCard(RankQ, SuitClub),
				NewCard(RankJ, SuitSpade),
			},
			expected: HandPair,
		},
		{
			name: "High Card",
			cards: []Card{
				NewCard(RankA, SuitSpade), NewCard(RankK, SuitHeart),
				NewCard(RankQ, SuitDiamond), NewCard(RankJ, SuitClub),
				NewCard(Rank9, SuitSpade),
			},
			expected: HandHighCard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := Evaluate(tt.cards)
			category := HandCategory(score >> 24)
			if category != tt.expected {
				t.Errorf("Evaluate() category = %v, want %v", category, tt.expected)
			}
		})
	}
}

func TestEvaluateComparison(t *testing.T) {
	// 驗證同花順比四條大
	sf := []Card{
		NewCard(Rank9, SuitHeart), NewCard(Rank8, SuitHeart), NewCard(Rank7, SuitHeart),
		NewCard(Rank6, SuitHeart), NewCard(Rank5, SuitHeart),
	}
	quads := []Card{
		NewCard(RankA, SuitSpade), NewCard(RankA, SuitHeart), NewCard(RankA, SuitDiamond),
		NewCard(RankA, SuitClub), NewCard(RankK, SuitSpade),
	}

	scoreSF := Evaluate(sf)
	scoreQuads := Evaluate(quads)

	if scoreSF <= scoreQuads {
		t.Errorf("Straight Flush (%d) should beat Four of a Kind (%d)", scoreSF, scoreQuads)
	}

	// 驗證同花大於順子
	flush := []Card{
		NewCard(RankA, SuitSpade), NewCard(RankJ, SuitSpade), NewCard(Rank8, SuitSpade),
		NewCard(Rank6, SuitSpade), NewCard(Rank2, SuitSpade),
	}
	straight := []Card{
		NewCard(RankK, SuitSpade), NewCard(RankQ, SuitHeart), NewCard(RankJ, SuitDiamond),
		NewCard(RankT, SuitClub), NewCard(Rank9, SuitSpade),
	}
	if Evaluate(flush) <= Evaluate(straight) {
		t.Error("Flush should beat Straight")
	}
}

func ExampleEvaluate() {
	cards := []Card{
		NewCard(RankA, SuitSpade), NewCard(RankA, SuitHeart),
		NewCard(RankK, SuitClub), NewCard(RankK, SuitDiamond),
		NewCard(RankQ, SuitSpade),
	}
	score := Evaluate(cards)
	fmt.Printf("Category: %d\n", score>>24)
	// Output: Category: 2
}
