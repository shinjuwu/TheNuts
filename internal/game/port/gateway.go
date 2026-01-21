package port

type Gateway interface {
	Broadcast(message interface{})
	SendToPlayer(playerID string, message interface{})
}
