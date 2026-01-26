package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
		Host string `yaml:"host"`
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
		MinPlayers   int   `yaml:"min_players"`
		MaxPlayers   int   `yaml:"max_players"`
		DefaultChips int64 `yaml:"default_chips"`
	} `yaml:"game"`
}

// PostgresConfig 定義 PostgreSQL 連接配置
type PostgresConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	Database        string `yaml:"database"`
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

// RedisConfig 定義 Redis 連接配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
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
