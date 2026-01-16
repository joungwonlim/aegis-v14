package naver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
		baseURL:    "https://polling.finance.naver.com/api/realtime/domestic/stock",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// NaverRealtimeResponse represents Naver realtime API response
type NaverRealtimeResponse struct {
	Datas []NaverPriceData `json:"datas"`
}

// NaverPriceData represents individual stock data
type NaverPriceData struct {
	ClosePrice               string `json:"closePrice"`               // 종가 (전일)
	CompareToPreviousClosePrice string `json:"compareToPreviousClosePrice"` // 전일대비
	FluctuationsRatio        string `json:"fluctuationsRatio"`        // 등락률
	AccumulatedTradingVolume string `json:"accumulatedTradingVolume"` // 거래량
	MarketStatus             string `json:"marketStatus"`             // 시장상태
	OverMarketPriceInfo      *struct {
		OverPrice                    string `json:"overPrice"`                    // 시간외 현재가
		CompareToPreviousClosePrice  string `json:"compareToPreviousClosePrice"`  // 전일대비
		FluctuationsRatio            string `json:"fluctuationsRatio"`            // 등락률
		AccumulatedTradingVolume     string `json:"accumulatedTradingVolume"`     // 거래량
	} `json:"overMarketPriceInfo"` // 시간외 정보
}

// GetCurrentPrice fetches current price from Naver Finance
func (c *Client) GetCurrentPrice(ctx context.Context, symbol string) (*price.Tick, error) {
	// Naver API URL
	url := fmt.Sprintf("%s/%s", c.baseURL, symbol)

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
	var realtimeResp NaverRealtimeResponse
	if err := json.Unmarshal(respBody, &realtimeResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Check if data exists
	if len(realtimeResp.Datas) == 0 {
		return nil, fmt.Errorf("no data for symbol %s", symbol)
	}

	// Convert to Tick
	tick, err := convertToTick(symbol, realtimeResp.Datas[0])
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
func convertToTick(symbol string, data NaverPriceData) (*price.Tick, error) {
	// Determine which price to use (prioritize over-market if available)
	var priceStr, changeStr, ratioStr, volumeStr string

	if data.OverMarketPriceInfo != nil && data.OverMarketPriceInfo.OverPrice != "" && data.OverMarketPriceInfo.OverPrice != "-" {
		// Use over-market (pre-market or after-hours) price
		priceStr = data.OverMarketPriceInfo.OverPrice
		changeStr = data.OverMarketPriceInfo.CompareToPreviousClosePrice
		ratioStr = data.OverMarketPriceInfo.FluctuationsRatio
		volumeStr = data.OverMarketPriceInfo.AccumulatedTradingVolume
	} else {
		// Use regular market price
		priceStr = data.ClosePrice
		changeStr = data.CompareToPreviousClosePrice
		ratioStr = data.FluctuationsRatio
		volumeStr = data.AccumulatedTradingVolume
	}

	// Remove commas and parse price
	priceStr = removeCommas(priceStr)
	if priceStr == "" || priceStr == "-" {
		return nil, fmt.Errorf("no valid price for symbol %s", symbol)
	}

	lastPrice, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse price %s: %w", priceStr, err)
	}

	tick := &price.Tick{
		Symbol:    symbol,
		Source:    price.SourceNaver,
		LastPrice: lastPrice,
		TS:        time.Now(),
		CreatedTS: time.Now(),
	}

	// Parse optional fields
	if changeStr != "" && changeStr != "-" {
		changeStr = removeCommas(changeStr)
		if change, err := strconv.ParseInt(changeStr, 10, 64); err == nil {
			tick.ChangePrice = &change
		}
	}

	if ratioStr != "" && ratioStr != "-" {
		if rate, err := strconv.ParseFloat(ratioStr, 64); err == nil {
			tick.ChangeRate = &rate
		}
	}

	if volumeStr != "" && volumeStr != "-" {
		volumeStr = removeCommas(volumeStr)
		if vol, err := strconv.ParseInt(volumeStr, 10, 64); err == nil {
			tick.Volume = &vol
		}
	}

	return tick, nil
}

// removeCommas removes commas from number strings
func removeCommas(s string) string {
	return strings.ReplaceAll(s, ",", "")
}
