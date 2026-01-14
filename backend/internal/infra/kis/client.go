package kis

import (
	"fmt"
	"os"
)

// Config holds KIS API configuration
type Config struct {
	AppKey    string
	AppSecret string
	BaseURL   string
	IsPaper   bool
}

// LoadConfigFromEnv loads KIS config from environment variables
func LoadConfigFromEnv() (*Config, error) {
	appKey := os.Getenv("KIS_APP_KEY")
	if appKey == "" {
		return nil, fmt.Errorf("KIS_APP_KEY not set")
	}

	appSecret := os.Getenv("KIS_APP_SECRET")
	if appSecret == "" {
		return nil, fmt.Errorf("KIS_APP_SECRET not set")
	}

	baseURL := os.Getenv("KIS_BASE_URL")
	if baseURL == "" {
		baseURL = "https://openapi.koreainvestment.com:9443" // Default production URL
	}

	isPaper := os.Getenv("KIS_IS_PAPER") == "true"

	return &Config{
		AppKey:    appKey,
		AppSecret: appSecret,
		BaseURL:   baseURL,
		IsPaper:   isPaper,
	}, nil
}

// Client wraps all KIS API clients
type Client struct {
	Auth *AuthClient
	REST *RESTClient
}

// NewClient creates a new KIS Client
func NewClient(config *Config) *Client {
	auth := NewAuthClient(config.AppKey, config.AppSecret, config.BaseURL)
	rest := NewRESTClient(auth, config.BaseURL)

	return &Client{
		Auth: auth,
		REST: rest,
	}
}

// NewClientFromEnv creates a new KIS Client from environment variables
func NewClientFromEnv() (*Client, error) {
	config, err := LoadConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return NewClient(config), nil
}
