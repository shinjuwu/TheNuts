package ws

import "time"

// Request 代表前端發出的指令
type Request struct {
	Action     string    `json:"action"`
	TableID    string    `json:"table_id,omitempty"`
	PlayerID   string    `json:"player_id,omitempty"`
	Amount     int64     `json:"amount,omitempty"`
	SeatNo     int       `json:"seat_no,omitempty"`
	GameAction string    `json:"game_action,omitempty"` // FOLD, CHECK, CALL, BET, RAISE, ALL_IN
	Timestamp  time.Time `json:"timestamp"`
	TraceID    string    `json:"trace_id"`
}

// Response 代表伺服器回傳的訊息
type Response struct {
	Type      string      `json:"type"` // snapshot, update, error, notification
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	TraceID   string      `json:"trace_id"`
}

// ErrorPayload 錯誤訊息具體內容
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
