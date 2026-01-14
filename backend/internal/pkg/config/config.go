package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config represents the application configuration
// SSOT: 모든 설정은 .env 파일에서 로드됨
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Logging  LoggingConfig
	KIS      KISConfig
	Naver    NaverConfig
}

type ServerConfig struct {
	Port         string
	Mode         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	URL             string // SSOT: DATABASE_URL
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

type RedisConfig struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	PoolTimeout  time.Duration
}

type LoggingConfig struct {
	Level         string
	Format        string
	FileEnabled   bool
	FilePath      string
	RotationSize  int
	RetentionDays int
}

type KISConfig struct {
	AppKey       string
	SecretKey    string
	BaseURL      string
	WebSocketURL string
}

type NaverConfig struct {
	BaseURL string
}

// Load loads configuration from .env file
// SSOT: .env 파일이 모든 설정의 유일한 진실 소스
func Load() (*Config, error) {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		// .env 파일이 없어도 계속 진행 (환경 변수에서 로드 시도)
		fmt.Println("Warning: .env file not found, using environment variables")
	}

	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8099"),
			Mode:         getEnv("GIN_MODE", "debug"),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			Name:            getEnv("DB_NAME", "aegis_v14"),
			User:            getEnv("DB_USER", "aegis_v14"),
			Password:        getEnv("DB_PASSWORD", "aegis_v14_won"),
			URL:             getEnv("DATABASE_URL", "postgresql://aegis_v14:aegis_v14_won@localhost:5432/aegis_v14?sslmode=disable"),
			MaxConns:        25,
			MinConns:        5,
			MaxConnLifetime: 1 * time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnv("REDIS_PORT", "6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           0,
			PoolSize:     10,
			MinIdleConns: 5,
			PoolTimeout:  30 * time.Second,
		},
		Logging: LoggingConfig{
			Level:         getEnv("LOG_LEVEL", "debug"),
			Format:        getEnv("LOG_FORMAT", "pretty"),
			FileEnabled:   getBoolEnv("LOG_FILE_ENABLED", true),
			FilePath:      getEnv("LOG_FILE_PATH", "./logs"),
			RotationSize:  getIntEnv("LOG_ROTATION_SIZE", 100),
			RetentionDays: getIntEnv("LOG_RETENTION_DAYS", 30),
		},
		KIS: KISConfig{
			AppKey:       getEnv("KIS_APP_KEY", ""),
			SecretKey:    getEnv("KIS_SECRET_KEY", ""),
			BaseURL:      getEnv("KIS_BASE_URL", "https://openapi.koreainvestment.com:9443"),
			WebSocketURL: getEnv("KIS_WEBSOCKET_URL", "ws://ops.koreainvestment.com:21000"),
		},
		Naver: NaverConfig{
			BaseURL: getEnv("NAVER_BASE_URL", "https://finance.naver.com"),
		},
	}

	return config, nil
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getBoolEnv gets boolean environment variable with fallback
func getBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value == "true" || value == "1" || value == "yes"
}

// getIntEnv gets integer environment variable with fallback
func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return fallback
	}
	return result
}
