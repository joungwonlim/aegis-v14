package kis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// AuthClient handles KIS API authentication
type AuthClient struct {
	appKey    string
	appSecret string
	baseURL   string

	// Token cache
	mu          sync.RWMutex
	accessToken string
	expiresAt   time.Time

	httpClient *http.Client
}

// NewAuthClient creates a new AuthClient
func NewAuthClient(appKey, appSecret, baseURL string) *AuthClient {
	return &AuthClient{
		appKey:     appKey,
		appSecret:  appSecret,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// TokenResponse represents KIS token API response
type TokenResponse struct {
	AccessToken          string `json:"access_token"`
	AccessTokenExpired   string `json:"access_token_token_expired"`
	TokenType            string `json:"token_type"`
	ExpiresIn            int    `json:"expires_in"`
	AccessTokenExpiresAt string `json:"access_token_expires_at"` // YYYYMMDD format
}

// GetAccessToken returns valid access token (fetches new if expired)
func (c *AuthClient) GetAccessToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.accessToken != "" && time.Now().Before(c.expiresAt) {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	// Need to fetch new token
	return c.fetchNewToken(ctx)
}

// fetchNewToken fetches a new access token from KIS API
func (c *AuthClient) fetchNewToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring lock
	if c.accessToken != "" && time.Now().Before(c.expiresAt) {
		return c.accessToken, nil
	}

	// Request body
	reqBody := map[string]string{
		"grant_type": "client_credentials",
		"appkey":     c.appKey,
		"appsecret":  c.appSecret,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/oauth2/tokenP", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("KIS API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	// Cache token (valid for 24 hours, refresh after 23 hours)
	c.accessToken = tokenResp.AccessToken
	c.expiresAt = time.Now().Add(23 * time.Hour)

	return c.accessToken, nil
}

// ClearToken clears cached token (useful for testing or forced refresh)
func (c *AuthClient) ClearToken() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = ""
	c.expiresAt = time.Time{}
}
