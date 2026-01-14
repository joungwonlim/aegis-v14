package naver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/wonny/aegis/v14/internal/domain/price"
)

// Client handles Naver Finance API requests
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Naver client
func NewClient() *Client {
	return &Client{
		baseURL:    "https://api.stock.naver.com/stock",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// NaverPriceResponse represents Naver API response
type NaverPriceResponse struct {
	ClosePrice         string `json:"closePrice"`         // 현재가
	CompareToPreviousPrice string `json:"compareToPreviousPrice"` // 전일대비
	FluctuationsRatio  string `json:"fluctuationsRatio"`  // 등락률
	AccumulatedTradingVolume string `json:"accumulatedTradingVolume"` // 누적거래량
}

// GetCurrentPrice fetches current price from Naver Finance
func (c *Client) GetCurrentPrice(ctx context.Context, symbol string) (*price.Tick, error) {
	// Naver API URL
	url := fmt.Sprintf("%s/%s/basic", c.baseURL, symbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add headers
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Naver API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var priceResp NaverPriceResponse
	if err := json.Unmarshal(respBody, &priceResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Convert to Tick
	tick, err := convertToTick(symbol, priceResp)
	if err != nil {
		return nil, fmt.Errorf("convert to tick: %w", err)
	}

	return tick, nil
}

// GetCurrentPrices fetches current prices for multiple symbols
func (c *Client) GetCurrentPrices(ctx context.Context, symbols []string) ([]*price.Tick, error) {
	ticks := make([]*price.Tick, 0, len(symbols))

	for _, symbol := range symbols {
		tick, err := c.GetCurrentPrice(ctx, symbol)
		if err != nil {
			// Log error but continue with other symbols
			continue
		}
		ticks = append(ticks, tick)

		// Rate limiting: sleep between requests
		select {
		case <-ctx.Done():
			return ticks, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return ticks, nil
}

// convertToTick converts Naver API response to price.Tick
func convertToTick(symbol string, resp NaverPriceResponse) (*price.Tick, error) {
	// Parse current price (required)
	lastPrice, err := strconv.ParseInt(resp.ClosePrice, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse close price: %w", err)
	}

	tick := &price.Tick{
		Symbol:    symbol,
		Source:    price.SourceNaver,
		LastPrice: lastPrice,
		TS:        time.Now(),
		CreatedTS: time.Now(),
	}

	// Parse optional fields
	if resp.CompareToPreviousPrice != "" {
		if vrss, err := strconv.ParseInt(resp.CompareToPreviousPrice, 10, 64); err == nil {
			tick.ChangePrice = &vrss
		}
	}

	if resp.FluctuationsRatio != "" {
		if rate, err := strconv.ParseFloat(resp.FluctuationsRatio, 64); err == nil {
			tick.ChangeRate = &rate
		}
	}

	if resp.AccumulatedTradingVolume != "" {
		if vol, err := strconv.ParseInt(resp.AccumulatedTradingVolume, 10, 64); err == nil {
			tick.Volume = &vol
		}
	}

	return tick, nil
}
