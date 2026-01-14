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

	// Check KIS_IS_PAPER environment variable first (explicit override)
	// This allows using virtual APP_KEY with real trading API (like v10)
	isPaper := false
	if isPaperEnv := os.Getenv("KIS_IS_PAPER"); isPaperEnv != "" {
		isPaper = isPaperEnv == "true"
	} else {
		// Auto-detect if paper trading based on APP_KEY prefix
		isPaper = len(appKey) >= 2 && appKey[:2] == "PS"
	}

	// Always auto-select BASE_URL based on isPaper flag to avoid misconfiguration
	baseURL := ""
	if isPaper {
		baseURL = "https://openapivts.koreainvestment.com:29443" // Virtual trading
	} else {
		baseURL = "https://openapi.koreainvestment.com:9443" // Real trading
	}

	// Allow explicit override via KIS_BASE_URL_OVERRIDE if user knows what they're doing
	if envBaseURL := os.Getenv("KIS_BASE_URL_OVERRIDE"); envBaseURL != "" {
		baseURL = envBaseURL
	}

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
	WS   *WebSocketClient
}

// NewClient creates a new KIS Client
func NewClient(config *Config) *Client {
	auth := NewAuthClient(config.AppKey, config.AppSecret, config.BaseURL)
	rest := NewRESTClient(auth, config.BaseURL, config.IsPaper)

	// WebSocket URL from config (default if not set)
	wsURL := os.Getenv("KIS_WEBSOCKET_URL")
	if wsURL == "" {
		wsURL = "ws://ops.koreainvestment.com:21000"
	}
	ws := NewWebSocketClient(config.AppKey, config.AppSecret, wsURL)

	return &Client{
		Auth: auth,
		REST: rest,
		WS:   ws,
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
