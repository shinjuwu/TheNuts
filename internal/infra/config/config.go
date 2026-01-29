package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port           string   `yaml:"port"`
		Host           string   `yaml:"host"`
		AllowedOrigins []string `yaml:"allowed_origins"` // WebSocket Origin 白名單，空表示允許所有（開發模式）
	} `yaml:"server"`
	Auth struct {
		JWTSecret        string `yaml:"jwt_secret"`
		TicketTTLSeconds int    `yaml:"ticket_ttl_seconds"`
	} `yaml:"auth"`
	Database struct {
		Postgres PostgresConfig `yaml:"postgres"`
		Redis    RedisConfig    `yaml:"redis"`
	} `yaml:"database"`
	Game struct {
		MinPlayers      int    `yaml:"min_players"`
		MaxPlayers      int    `yaml:"max_players"`
		DefaultChips    int64  `yaml:"default_chips"`
		DefaultCurrency string `yaml:"default_currency"` // Default wallet currency (e.g., USD, CNY)
	} `yaml:"game"`
}

// PostgresConfig 定義 PostgreSQL 連接配置
type PostgresConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	Database        string `yaml:"database"`
	SSLMode         string `yaml:"ssl_mode"` // SSL mode (disable, require, verify-ca, verify-full)
	MaxConns        int32  `yaml:"max_conns"`
	MinConns        int32  `yaml:"min_conns"`
	MaxConnLifetime string `yaml:"max_conn_lifetime"`
}

// GetMaxConnLifetime 將字串轉換為 time.Duration
func (p *PostgresConfig) GetMaxConnLifetime() time.Duration {
	d, err := time.ParseDuration(p.MaxConnLifetime)
	if err != nil {
		return 5 * time.Minute // 默認值
	}
	return d
}

// GetSSLMode 取得 SSL 模式，如果未設定則返回 "disable"
func (p *PostgresConfig) GetSSLMode() string {
	if p.SSLMode == "" {
		return "disable" // 開發環境默認值
	}
	return p.SSLMode
}

// RedisConfig 定義 Redis 連接配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// GetDefaultCurrency 取得默認貨幣，如果未設定則返回 "USD"
func (c *Config) GetDefaultCurrency() string {
	if c.Game.DefaultCurrency == "" {
		return "USD" // 默認貨幣
	}
	return c.Game.DefaultCurrency
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
