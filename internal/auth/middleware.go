package auth

import (
	"context"
	"net/http"
	"strings"
)

// contextKey 用於在 context 中儲存玩家資訊
type contextKey string

const (
	// PlayerIDKey context 中的玩家 ID 鍵
	PlayerIDKey contextKey = "player_id"
	// UsernameKey context 中的使用者名稱鍵
	UsernameKey contextKey = "username"
)

// JWTMiddleware JWT 驗證中介層
func JWTMiddleware(jwtService *JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 從 Authorization Header 取得 Token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// 檢查格式：Bearer <token>
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// 驗證 Token
			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// 將玩家資訊存入 context
			ctx := context.WithValue(r.Context(), PlayerIDKey, claims.PlayerID)
			ctx = context.WithValue(ctx, UsernameKey, claims.Username)

			// 繼續處理請求
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetPlayerIDFromContext 從 context 取得玩家 ID
func GetPlayerIDFromContext(ctx context.Context) (string, bool) {
	playerID, ok := ctx.Value(PlayerIDKey).(string)
	return playerID, ok
}

// GetUsernameFromContext 從 context 取得使用者名稱
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(UsernameKey).(string)
	return username, ok
}
