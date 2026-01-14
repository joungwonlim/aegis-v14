package kis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wonny/aegis/v14/internal/domain/price"
)

// WebSocketClient handles KIS WebSocket connections for real-time prices
type WebSocketClient struct {
	appKey       string
	appSecret    string
	wsURL        string
	approvalKey  string

	// Connection
	conn     *websocket.Conn
	connMu   sync.RWMutex
	isActive bool

	// Subscriptions (max 40)
	subscriptions map[string]bool // symbol -> subscribed
	subMu         sync.RWMutex
	maxSubs       int

	// Event handler
	onTick func(tick price.Tick)

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(appKey, appSecret, wsURL string) *WebSocketClient {
	return &WebSocketClient{
		appKey:        appKey,
		appSecret:     appSecret,
		wsURL:         wsURL,
		subscriptions: make(map[string]bool),
		maxSubs:       40,
	}
}

// SetTickHandler sets the tick event handler
func (c *WebSocketClient) SetTickHandler(handler func(tick price.Tick)) {
	c.onTick = handler
}

// Connect connects to KIS WebSocket
func (c *WebSocketClient) Connect(ctx context.Context) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.isActive {
		return fmt.Errorf("already connected")
	}

	// Get approval key
	approvalKey, err := c.getApprovalKey(ctx)
	if err != nil {
		return fmt.Errorf("get approval key: %w", err)
	}
	c.approvalKey = approvalKey

	// Connect to WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(c.wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	c.conn = conn
	c.isActive = true

	// Start context
	c.ctx, c.cancel = context.WithCancel(ctx)

	// Start message handler
	c.wg.Add(1)
	go c.handleMessages()

	return nil
}

// Disconnect closes WebSocket connection
func (c *WebSocketClient) Disconnect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if !c.isActive {
		return nil
	}

	// Cancel context
	if c.cancel != nil {
		c.cancel()
	}

	// Close connection
	if c.conn != nil {
		c.conn.Close()
	}

	c.isActive = false

	// Wait for handlers to finish
	c.wg.Wait()

	return nil
}

// Subscribe subscribes to real-time price updates for a symbol
func (c *WebSocketClient) Subscribe(symbol string) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	// Check if already subscribed
	if c.subscriptions[symbol] {
		return nil // Already subscribed
	}

	// Check max subscriptions
	if len(c.subscriptions) >= c.maxSubs {
		return fmt.Errorf("max subscriptions reached (%d/%d)", len(c.subscriptions), c.maxSubs)
	}

	// Send subscribe message
	msg := map[string]interface{}{
		"header": map[string]string{
			"approval_key": c.approvalKey,
			"custtype":     "P", // 개인
			"tr_type":      "1", // 등록
			"content-type": "utf-8",
		},
		"body": map[string]interface{}{
			"input": map[string]string{
				"tr_id":  "H0STCNT0", // 실시간 체결가
				"tr_key": symbol,
			},
		},
	}

	c.connMu.RLock()
	err := c.conn.WriteJSON(msg)
	c.connMu.RUnlock()

	if err != nil {
		return fmt.Errorf("send subscribe message: %w", err)
	}

	c.subscriptions[symbol] = true
	return nil
}

// Unsubscribe unsubscribes from real-time price updates for a symbol
func (c *WebSocketClient) Unsubscribe(symbol string) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	// Check if subscribed
	if !c.subscriptions[symbol] {
		return nil // Not subscribed
	}

	// Send unsubscribe message
	msg := map[string]interface{}{
		"header": map[string]string{
			"approval_key": c.approvalKey,
			"custtype":     "P",
			"tr_type":      "2", // 해제
			"content-type": "utf-8",
		},
		"body": map[string]interface{}{
			"input": map[string]string{
				"tr_id":  "H0STCNT0",
				"tr_key": symbol,
			},
		},
	}

	c.connMu.RLock()
	err := c.conn.WriteJSON(msg)
	c.connMu.RUnlock()

	if err != nil {
		return fmt.Errorf("send unsubscribe message: %w", err)
	}

	delete(c.subscriptions, symbol)
	return nil
}

// GetSubscriptions returns currently subscribed symbols
func (c *WebSocketClient) GetSubscriptions() []string {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	symbols := make([]string, 0, len(c.subscriptions))
	for symbol := range c.subscriptions {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// GetSubscriptionCount returns number of active subscriptions
func (c *WebSocketClient) GetSubscriptionCount() int {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	return len(c.subscriptions)
}

// handleMessages handles incoming WebSocket messages
func (c *WebSocketClient) handleMessages() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.connMu.RLock()
		_, message, err := c.conn.ReadMessage()
		c.connMu.RUnlock()

		if err != nil {
			// Connection closed or error
			return
		}

		// Parse message
		tick, err := c.parseMessage(message)
		if err != nil {
			// Skip invalid messages
			continue
		}

		// Call handler
		if c.onTick != nil && tick != nil {
			c.onTick(*tick)
		}
	}
}

// WSMessage represents WebSocket message structure
type WSMessage struct {
	Header WSHeader `json:"header"`
	Body   WSBody   `json:"body"`
}

type WSHeader struct {
	TrID   string `json:"tr_id"`
	TrKey  string `json:"tr_key"`
	Status string `json:"status"`
}

type WSBody struct {
	RTCd   string `json:"rt_cd"`   // 응답코드
	MsgCd  string `json:"msg_cd"`  // 메시지코드
	Msg1   string `json:"msg1"`    // 메시지
	Output string `json:"output"`  // 실시간 데이터 (구분자로 연결된 문자열)
}

// parseMessage parses WebSocket message into Tick
func (c *WebSocketClient) parseMessage(data []byte) (*price.Tick, error) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	// Check if it's a price update
	if msg.Header.TrID != "H0STCNT0" {
		return nil, fmt.Errorf("not a price update: tr_id=%s", msg.Header.TrID)
	}

	// Check status
	if msg.Body.RTCd != "0" {
		return nil, fmt.Errorf("error response: rt_cd=%s msg=%s", msg.Body.RTCd, msg.Body.Msg1)
	}

	// Parse output (구분자: ^)
	// Format: 유가증권단축종목코드^주식체결시간^주식현재가^전일대비부호^전일대비^...
	fields := parseDelimited(msg.Body.Output, "^")
	if len(fields) < 5 {
		return nil, fmt.Errorf("invalid output format: %s", msg.Body.Output)
	}

	symbol := fields[0]

	// 현재가 (index 2)
	lastPrice, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse last price: %w", err)
	}

	tick := &price.Tick{
		Symbol:    symbol,
		Source:    price.SourceKISWebSocket,
		LastPrice: lastPrice,
		TS:        time.Now(),
		CreatedTS: time.Now(),
	}

	// 전일대비 (index 4)
	if len(fields) > 4 && fields[4] != "" {
		if vrss, err := strconv.ParseInt(fields[4], 10, 64); err == nil {
			// 전일대비부호 (index 3): 1=상한, 2=상승, 3=보합, 4=하한, 5=하락
			sign := fields[3]
			switch sign {
			case "2": // 상승
				tick.ChangePrice = &vrss
			case "5": // 하락
				neg := -vrss
				tick.ChangePrice = &neg
			case "3": // 보합
				zero := int64(0)
				tick.ChangePrice = &zero
			default:
				tick.ChangePrice = &vrss
			}
		}
	}

	// 등락률 (index 5)
	if len(fields) > 5 && fields[5] != "" {
		if rate, err := strconv.ParseFloat(fields[5], 64); err == nil {
			tick.ChangeRate = &rate
		}
	}

	// 거래량 (index 9)
	if len(fields) > 9 && fields[9] != "" {
		if vol, err := strconv.ParseInt(fields[9], 10, 64); err == nil {
			tick.Volume = &vol
		}
	}

	return tick, nil
}

// parseDelimited splits string by delimiter
func parseDelimited(s, delim string) []string {
	result := []string{}
	current := ""
	for i := 0; i < len(s); i++ {
		if i < len(s)-len(delim) && s[i:i+len(delim)] == delim {
			result = append(result, current)
			current = ""
			i += len(delim) - 1
		} else {
			current += string(s[i])
		}
	}
	result = append(result, current)
	return result
}

// getApprovalKey gets WebSocket approval key
func (c *WebSocketClient) getApprovalKey(ctx context.Context) (string, error) {
	// KIS WebSocket approval key API
	// POST /oauth2/Approval
	url := "https://openapi.koreainvestment.com:9443/oauth2/Approval"

	reqBody := map[string]string{
		"grant_type": "client_credentials",
		"appkey":     c.appKey,
		"secretkey":  c.appSecret,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
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
	var approvalResp struct {
		ApprovalKey string `json:"approval_key"`
	}
	if err := json.Unmarshal(respBody, &approvalResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return approvalResp.ApprovalKey, nil
}
