package kis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/price"
)

// ExecutionNotification represents a real-time execution notification from KIS
type ExecutionNotification struct {
	AccountNo    string    // 계좌번호
	OrderNo      string    // 주문번호
	OrigOrderNo  string    // 원주문번호
	Symbol       string    // 종목코드
	SymbolName   string    // 종목명
	Side         string    // 매도매수구분 (01=매도, 02=매수)
	OrderType    string    // 정정취소구분
	OrderQty     int64     // 주문수량
	OrderPrice   int64     // 주문가격
	FilledQty    int64     // 체결수량
	FilledPrice  int64     // 체결단가
	FilledAmount int64     // 체결금액
	RemainingQty int64     // 잔여수량
	TotalFilledQty int64   // 총체결수량
	Timestamp    time.Time // 체결시간
}

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

	// Execution subscription
	execSubscribed bool      // 체결통보 구독 여부
	execAccountNo  string    // 체결통보 구독 계좌번호

	// Event handlers
	onTick      func(tick price.Tick)
	onExecution func(exec ExecutionNotification)

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

// SetExecutionHandler sets the execution notification handler
func (c *WebSocketClient) SetExecutionHandler(handler func(exec ExecutionNotification)) {
	c.onExecution = handler
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

// SubscribeExecution subscribes to real-time execution notifications for an account
// TR_ID: H0STCNI0 (실시간 체결통보)
func (c *WebSocketClient) SubscribeExecution(accountNo string) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	// Check if already subscribed
	if c.execSubscribed && c.execAccountNo == accountNo {
		return nil // Already subscribed
	}

	// Send subscribe message
	// 체결통보는 HTS ID를 tr_key로 사용하지만, 실전투자에서는 계좌번호 앞 8자리 사용
	trKey := accountNo
	if len(trKey) > 8 {
		trKey = trKey[:8]
	}

	msg := map[string]interface{}{
		"header": map[string]string{
			"approval_key": c.approvalKey,
			"custtype":     "P", // 개인
			"tr_type":      "1", // 등록
			"content-type": "utf-8",
		},
		"body": map[string]interface{}{
			"input": map[string]string{
				"tr_id":  "H0STCNI0", // 실시간 체결통보
				"tr_key": trKey,
			},
		},
	}

	c.connMu.RLock()
	err := c.conn.WriteJSON(msg)
	c.connMu.RUnlock()

	if err != nil {
		return fmt.Errorf("send execution subscribe message: %w", err)
	}

	c.execSubscribed = true
	c.execAccountNo = accountNo

	log.Info().
		Str("account_no", trKey).
		Msg("[WS] Subscribed to execution notifications")

	return nil
}

// UnsubscribeExecution unsubscribes from execution notifications
func (c *WebSocketClient) UnsubscribeExecution() error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	if !c.execSubscribed {
		return nil
	}

	trKey := c.execAccountNo
	if len(trKey) > 8 {
		trKey = trKey[:8]
	}

	msg := map[string]interface{}{
		"header": map[string]string{
			"approval_key": c.approvalKey,
			"custtype":     "P",
			"tr_type":      "2", // 해제
			"content-type": "utf-8",
		},
		"body": map[string]interface{}{
			"input": map[string]string{
				"tr_id":  "H0STCNI0",
				"tr_key": trKey,
			},
		},
	}

	c.connMu.RLock()
	err := c.conn.WriteJSON(msg)
	c.connMu.RUnlock()

	if err != nil {
		return fmt.Errorf("send execution unsubscribe message: %w", err)
	}

	c.execSubscribed = false
	c.execAccountNo = ""

	log.Info().Msg("[WS] Unsubscribed from execution notifications")

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

// CanSubscribe returns whether there's room for more subscriptions
func (c *WebSocketClient) CanSubscribe() bool {
	c.subMu.RLock()
	defer c.subMu.RUnlock()
	return len(c.subscriptions) < c.maxSubs
}

// Start starts the WebSocket client (alias for Connect)
func (c *WebSocketClient) Start(ctx context.Context) error {
	return c.Connect(ctx)
}

// Stop stops the WebSocket client (alias for Disconnect)
func (c *WebSocketClient) Stop() error {
	return c.Disconnect()
}

// handleMessages handles incoming WebSocket messages
func (c *WebSocketClient) handleMessages() {
	defer c.wg.Done()

	log.Debug().Msg("[WS] Message handler started")

	for {
		select {
		case <-c.ctx.Done():
			log.Debug().Msg("[WS] Context cancelled, stopping message handler")
			return
		default:
		}

		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()

		if conn == nil {
			log.Error().Msg("[WS] Connection is nil")
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			// Connection closed or error - attempt reconnect
			log.Error().Err(err).Msg("[WS] Connection error, attempting reconnect...")

			if reconnectErr := c.reconnect(); reconnectErr != nil {
				log.Error().Err(reconnectErr).Msg("[WS] Reconnect failed, stopping message handler")
				return
			}
			continue
		}

		// Handle PINGPONG (KIS custom heartbeat)
		// KIS sends "PINGPONG" string, client must echo it back
		if strings.Contains(string(message), "PINGPONG") {
			c.connMu.RLock()
			if c.conn != nil {
				if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
					log.Warn().Err(err).Msg("[WS] Failed to send PINGPONG response")
				} else {
					log.Debug().Msg("[WS] PINGPONG responded")
				}
			}
			c.connMu.RUnlock()
			continue
		}

		log.Debug().
			Str("message", string(message)).
			Int("length", len(message)).
			Msg("[WS] Received message")

		// Try to parse as execution notification first
		exec, execErr := c.parseExecutionMessage(message)
		if execErr == nil && exec != nil {
			log.Info().
				Str("symbol", exec.Symbol).
				Str("order_no", exec.OrderNo).
				Int64("filled_qty", exec.FilledQty).
				Int64("filled_price", exec.FilledPrice).
				Msg("[WS] 📣 Execution notification received")

			if c.onExecution != nil {
				c.onExecution(*exec)
			}
			continue
		}

		// Parse as tick message
		tick, err := c.parseMessage(message)
		if err != nil {
			// Skip invalid messages (control messages, etc.)
			log.Debug().Err(err).Msg("[WS] Skipping non-tick message")
			continue
		}

		// Call handler
		if c.onTick != nil && tick != nil {
			log.Debug().
				Str("symbol", tick.Symbol).
				Int64("price", tick.LastPrice).
				Msg("[WS] Calling tick handler")
			c.onTick(*tick)
		}
	}
}

// reconnect attempts to reconnect to WebSocket with exponential backoff
func (c *WebSocketClient) reconnect() error {
	c.connMu.Lock()
	// Close existing connection
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.isActive = false
	c.connMu.Unlock()

	// Save current subscriptions to restore after reconnect
	c.subMu.RLock()
	symbols := make([]string, 0, len(c.subscriptions))
	for symbol := range c.subscriptions {
		symbols = append(symbols, symbol)
	}
	execSubscribed := c.execSubscribed
	execAccountNo := c.execAccountNo
	c.subMu.RUnlock()

	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second
	maxAttempts := 10

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		log.Info().
			Int("attempt", attempt).
			Dur("backoff", backoff).
			Msg("[WS] Attempting reconnect...")

		// Get new approval key
		approvalKey, err := c.getApprovalKey(c.ctx)
		if err != nil {
			log.Warn().Err(err).Msg("[WS] Failed to get approval key")
			time.Sleep(backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}
		c.approvalKey = approvalKey

		// Connect to WebSocket
		dialer := websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		}

		conn, _, err := dialer.Dial(c.wsURL, nil)
		if err != nil {
			log.Warn().Err(err).Msg("[WS] Failed to dial")
			time.Sleep(backoff)
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		c.connMu.Lock()
		c.conn = conn
		c.isActive = true
		c.connMu.Unlock()

		// Wait for connection to stabilize before subscribing
		time.Sleep(500 * time.Millisecond)

		// Restore subscriptions
		restoredCount := 0
		for _, symbol := range symbols {
			// Send subscribe message directly (don't update subscriptions map)
			msg := map[string]interface{}{
				"header": map[string]string{
					"approval_key": c.approvalKey,
					"custtype":     "P",
					"tr_type":      "1",
					"content-type": "utf-8",
				},
				"body": map[string]interface{}{
					"input": map[string]string{
						"tr_id":  "H0STCNT0",
						"tr_key": symbol,
					},
				},
			}

			if err := conn.WriteJSON(msg); err != nil {
				log.Warn().Err(err).Str("symbol", symbol).Msg("[WS] Failed to restore subscription")
			} else {
				restoredCount++
			}

			// Small delay between subscriptions
			time.Sleep(50 * time.Millisecond)
		}

		// Restore execution subscription if it was active
		if execSubscribed && execAccountNo != "" {
			trKey := execAccountNo
			if len(trKey) > 8 {
				trKey = trKey[:8]
			}

			execMsg := map[string]interface{}{
				"header": map[string]string{
					"approval_key": c.approvalKey,
					"custtype":     "P",
					"tr_type":      "1",
					"content-type": "utf-8",
				},
				"body": map[string]interface{}{
					"input": map[string]string{
						"tr_id":  "H0STCNI0",
						"tr_key": trKey,
					},
				},
			}

			if err := conn.WriteJSON(execMsg); err != nil {
				log.Warn().Err(err).Msg("[WS] Failed to restore execution subscription")
			} else {
				log.Info().Str("account_no", trKey).Msg("[WS] Restored execution subscription")
			}
		}

		log.Info().
			Int("restored", restoredCount).
			Int("total", len(symbols)).
			Bool("exec_subscribed", execSubscribed).
			Msg("[WS] ✅ Reconnected and subscriptions restored")

		return nil
	}

	return fmt.Errorf("failed to reconnect after %d attempts", maxAttempts)
}

// minDuration returns the minimum of two durations
func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
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
	RTCd   string      `json:"rt_cd"`   // 응답코드
	MsgCd  string      `json:"msg_cd"`  // 메시지코드
	Msg1   string      `json:"msg1"`    // 메시지
	Output interface{} `json:"output"`  // 실시간 데이터 (문자열) 또는 구독 응답 (객체)
}

// parseMessage parses WebSocket message into Tick
// Supports two formats:
// 1. Pipe-delimited: status|tr_id|tr_count|data (real-time data)
//    Example: 0|H0STCNT0|002|000660^090607^750000^2^1000^0.13^...
// 2. JSON: {"header":{...},"body":{...}} (control messages, errors)
func (c *WebSocketClient) parseMessage(data []byte) (*price.Tick, error) {
	dataStr := string(data)

	// Check if it's JSON format (control messages)
	if len(dataStr) > 0 && dataStr[0] == '{' {
		// Try to parse as JSON
		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err == nil {
			// JSON message - check if it's an error
			if msg.Body.RTCd != "0" {
				return nil, fmt.Errorf("KIS error: code=%s msg=%s", msg.Body.RTCd, msg.Body.Msg1)
			}
			// Subscription confirmation or other control message - not a price update
			// Return nil to indicate this is not a tick (but not an error either)
			return nil, nil
		}
	}

	// Parse as pipe-delimited format
	parts := parseDelimited(dataStr, "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid message format: expected 4 parts, got %d", len(parts))
	}

	status := parts[0]
	trID := parts[1]
	outputStr := parts[3]

	// Check if it's a price update
	if trID != "H0STCNT0" {
		return nil, fmt.Errorf("not a price update: tr_id=%s", trID)
	}

	// Check status
	if status != "0" {
		return nil, fmt.Errorf("error response: status=%s", status)
	}

	// Parse output (구분자: ^)
	// Format: 유가증권단축종목코드^주식체결시간^주식현재가^전일대비부호^전일대비^...
	fields := parseDelimited(outputStr, "^")
	if len(fields) < 5 {
		return nil, fmt.Errorf("invalid output format: %s", outputStr)
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

// parseExecutionMessage parses execution notification message (H0STCNI0)
// Format: 0|H0STCNI0|001|고객ID^계좌번호^주문번호^원주문번호^매도매수구분^정정취소구분^...
//
// Field indices for H0STCNI0:
// 0: 고객ID
// 1: 계좌번호
// 2: 주문번호
// 3: 원주문번호
// 4: 매도매수구분 (01=매도, 02=매수)
// 5: 정정취소구분
// 6: 주문수량
// 7: 주문단가
// 8: 체결수량
// 9: 체결단가
// 10: 체결금액
// 11: 주식잔고수량
// 12: 총체결수량
// 13: 종목코드
// ...
func (c *WebSocketClient) parseExecutionMessage(data []byte) (*ExecutionNotification, error) {
	dataStr := string(data)

	// Check if it's JSON format (control messages) - skip
	if len(dataStr) > 0 && dataStr[0] == '{' {
		return nil, fmt.Errorf("JSON control message, not execution notification")
	}

	// Parse as pipe-delimited format
	parts := parseDelimited(dataStr, "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid message format: expected 4 parts, got %d", len(parts))
	}

	status := parts[0]
	trID := parts[1]
	outputStr := parts[3]

	// Check if it's an execution notification
	if trID != "H0STCNI0" {
		return nil, fmt.Errorf("not an execution notification: tr_id=%s", trID)
	}

	// Check status
	if status != "0" {
		return nil, fmt.Errorf("error response: status=%s", status)
	}

	// Parse output (구분자: ^)
	fields := parseDelimited(outputStr, "^")
	if len(fields) < 14 {
		return nil, fmt.Errorf("invalid execution output format: expected at least 14 fields, got %d", len(fields))
	}

	exec := &ExecutionNotification{
		AccountNo:   fields[1],
		OrderNo:     fields[2],
		OrigOrderNo: fields[3],
		Side:        fields[4],
		OrderType:   fields[5],
		Timestamp:   time.Now(),
	}

	// Parse numeric fields
	if qty, err := strconv.ParseInt(fields[6], 10, 64); err == nil {
		exec.OrderQty = qty
	}
	if price, err := strconv.ParseInt(fields[7], 10, 64); err == nil {
		exec.OrderPrice = price
	}
	if qty, err := strconv.ParseInt(fields[8], 10, 64); err == nil {
		exec.FilledQty = qty
	}
	if price, err := strconv.ParseInt(fields[9], 10, 64); err == nil {
		exec.FilledPrice = price
	}
	if amount, err := strconv.ParseInt(fields[10], 10, 64); err == nil {
		exec.FilledAmount = amount
	}
	if qty, err := strconv.ParseInt(fields[11], 10, 64); err == nil {
		exec.RemainingQty = qty
	}
	if qty, err := strconv.ParseInt(fields[12], 10, 64); err == nil {
		exec.TotalFilledQty = qty
	}

	// 종목코드 (index 13)
	exec.Symbol = fields[13]

	// 종목명 (index 14 if available)
	if len(fields) > 14 {
		exec.SymbolName = fields[14]
	}

	return exec, nil
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
