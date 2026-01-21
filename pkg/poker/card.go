package poker

// Card 點數與花色定義
const (
	Rank2 = iota
	Rank3
	Rank4
	Rank5
	Rank6
	Rank7
	Rank8
	Rank9
	RankT
	RankJ
	RankQ
	RankK
	RankA
)

const (
	SuitClub = iota
	SuitDiamond
	SuitHeart
	SuitSpade
)

// Card 代表一張撲克牌
type Card struct {
	Rank int
	Suit int
}
