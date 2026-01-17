package kis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
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
	nextRefresh time.Time // KIS 6-hour refresh rule

	// Rate limit protection (EGW00133: 1분당 1회)
	holdUntil time.Time

	// Singleflight to prevent stampede
	sf singleflight.Group

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
	now := time.Now()

	c.mu.RLock()
	// 1. Check if current token is still valid and not yet time to refresh
	if c.accessToken != "" && now.Before(c.expiresAt.Add(-30*time.Second)) && now.Before(c.nextRefresh) {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}

	// 2. Check if we're in hold period (EGW00133 rate limit)
	// Return existing token if still valid (even if expired soon)
	if now.Before(c.holdUntil) && c.accessToken != "" && now.Before(c.expiresAt) {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	// 3. Need to fetch new token - use singleflight to prevent stampede
	v, err, _ := c.sf.Do("refresh", func() (interface{}, error) {
		return c.fetchNewToken(ctx)
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

// fetchNewToken fetches a new access token from KIS API
func (c *AuthClient) fetchNewToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Double-check after acquiring lock
	if c.accessToken != "" && now.Before(c.expiresAt.Add(-30*time.Second)) && now.Before(c.nextRefresh) {
		return c.accessToken, nil
	}

	// Check hold period (EGW00133 rate limit)
	if now.Before(c.holdUntil) {
		if c.accessToken != "" && now.Before(c.expiresAt) {
			// Return existing token during hold period
			return c.accessToken, nil
		}
		return "", fmt.Errorf("token refresh on hold until %s (KIS rate limit)", c.holdUntil.Format(time.RFC3339))
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
		// Check for EGW00133 (rate limit)
		if isEGW00133(string(respBody)) {
			// Hold for 65 seconds (1 minute + 5 second buffer)
			c.holdUntil = time.Now().Add(65 * time.Second)
		}
		return "", fmt.Errorf("KIS API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	// Cache token
	// KIS: 24시간 유효, 6시간 내 재발급 시 기존 토큰 재사용
	c.accessToken = tokenResp.AccessToken
	c.expiresAt = time.Now().Add(24 * time.Hour)
	c.nextRefresh = time.Now().Add(6*time.Hour - 5*time.Minute) // 6시간 - 5분 버퍼

	return c.accessToken, nil
}

// isEGW00133 checks if the error is EGW00133 (rate limit)
func isEGW00133(body string) bool {
	return strings.Contains(body, "EGW00133") || strings.Contains(body, "1분당 1회")
}

// ClearToken clears cached token (useful for testing or forced refresh)
func (c *AuthClient) ClearToken() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = ""
	c.expiresAt = time.Time{}
	c.nextRefresh = time.Time{}
	c.holdUntil = time.Time{}
}
