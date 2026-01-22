package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTClaims JWT 聲明
type JWTClaims struct {
	PlayerID string `json:"player_id"`
	Username string `json:"username,omitempty"`
	Exp      int64  `json:"exp"` // 過期時間（Unix timestamp）
	Iat      int64  `json:"iat"` // 簽發時間（Unix timestamp）
}

// JWTService JWT 服務
type JWTService struct {
	secretKey []byte
}

// NewJWTService 創建 JWT 服務
func NewJWTService(secretKey string) *JWTService {
	return &JWTService{
		secretKey: []byte(secretKey),
	}
}

// GenerateToken 生成 JWT Token
func (s *JWTService) GenerateToken(playerID, username string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		PlayerID: playerID,
		Username: username,
		Iat:      now.Unix(),
		Exp:      now.Add(ttl).Unix(),
	}

	// 構建 Header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	// 構建 Payload
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	// Base64 編碼
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// 簽名
	message := headerB64 + "." + payloadB64
	signature := s.sign(message)
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	// 組合 Token
	token := message + "." + signatureB64

	return token, nil
}

// ValidateToken 驗證 JWT Token
func (s *JWTService) ValidateToken(token string) (*JWTClaims, error) {
	// 分割 Token
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	headerB64 := parts[0]
	payloadB64 := parts[1]
	signatureB64 := parts[2]

	// 驗證簽名
	message := headerB64 + "." + payloadB64
	expectedSignature := s.sign(message)
	expectedSignatureB64 := base64.RawURLEncoding.EncodeToString(expectedSignature)

	if signatureB64 != expectedSignatureB64 {
		return nil, fmt.Errorf("invalid signature")
	}

	// 解碼 Payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("invalid payload format: %w", err)
	}

	// 檢查過期時間
	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// sign 使用 HMAC-SHA256 簽名
func (s *JWTService) sign(message string) []byte {
	h := hmac.New(sha256.New, s.secretKey)
	h.Write([]byte(message))
	return h.Sum(nil)
}
