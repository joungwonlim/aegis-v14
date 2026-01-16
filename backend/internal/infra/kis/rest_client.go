package kis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/wonny/aegis/v14/internal/domain/price"
)

// RESTClient handles KIS REST API requests
type RESTClient struct {
	auth       *AuthClient
	baseURL    string
	isPaper    bool
	httpClient *http.Client
}

// NewRESTClient creates a new RESTClient
func NewRESTClient(auth *AuthClient, baseURL string, isPaper bool) *RESTClient {
	return &RESTClient{
		auth:       auth,
		baseURL:    baseURL,
		isPaper:    isPaper,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// CurrentPriceResponse represents KIS current price API response
type CurrentPriceResponse struct {
	RetCode    string `json:"rt_cd"`    // "0" = success
	MsgCode    string `json:"msg_cd"`   // Message code
	Msg1       string `json:"msg1"`     // Message
	Output     CurrentPriceOutput `json:"output"`
}

// CurrentPriceOutput represents price data
type CurrentPriceOutput struct {
	StockCode      string `json:"stck_shrn_iscd"` // 종목코드
	StockPrice     string `json:"stck_prpr"`      // 현재가
	PrdyVrss       string `json:"prdy_vrss"`      // 전일대비
	PrdyVrssSign   string `json:"prdy_vrss_sign"` // 전일대비부호 (1:상한, 2:상승, 3:보합, 4:하한, 5:하락)
	PrdyCtrt       string `json:"prdy_ctrt"`      // 전일대비율
	AccuVol        string `json:"acml_vol"`       // 누적거래량
	AccuTrPbmn     string `json:"acml_tr_pbmn"`   // 누적거래대금
	SelnAskrspr1   string `json:"seln_askrspr1"`  // 매도호가1
	ShtnAskrspr1   string `json:"shtn_askrspr1"`  // 매수호가1
	BiddVolume1    string `json:"bidp_volume1"`   // 매수호가잔량1
	AskoVolume1    string `json:"askp_volume1"`   // 매도호가잔량1
	StckHgpr       string `json:"stck_hgpr"`      // 최고가
	StckLwpr       string `json:"stck_lwpr"`      // 최저가
	StckOprc       string `json:"stck_oprc"`      // 시가
	StckSdpr       string `json:"stck_sdpr"`      // 기준가
}

// GetCurrentPrice fetches current price for a symbol
func (c *RESTClient) GetCurrentPrice(ctx context.Context, symbol string) (*price.Tick, error) {
	// Get access token
	token, err := c.auth.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	// Build request URL
	// 국내주식시세 > 주식현재가 시세
	url := fmt.Sprintf("%s/uapi/domestic-stock/v1/quotations/inquire-price", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("FID_COND_MRKT_DIV_CODE", "J") // J: 주식, ETF, ETN
	q.Add("FID_INPUT_ISCD", symbol)       // 종목코드
	req.URL.RawQuery = q.Encode()

	// Add headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("appkey", c.auth.appKey)
	req.Header.Set("appsecret", c.auth.appSecret)
	req.Header.Set("tr_id", "FHKST01010100") // 국내주식 현재가 시세 조회

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
		return nil, fmt.Errorf("KIS API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var priceResp CurrentPriceResponse
	if err := json.Unmarshal(respBody, &priceResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if priceResp.RetCode != "0" {
		return nil, fmt.Errorf("KIS API error: code=%s msg=%s", priceResp.MsgCode, priceResp.Msg1)
	}

	// Convert to Tick
	tick, err := convertToTick(symbol, priceResp.Output)
	if err != nil {
		return nil, fmt.Errorf("convert to tick: %w", err)
	}

	return tick, nil
}

// convertToTick converts KIS API output to price.Tick
func convertToTick(symbol string, output CurrentPriceOutput) (*price.Tick, error) {
	// Parse current price (required)
	lastPrice, err := strconv.ParseInt(output.StockPrice, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse current price: %w", err)
	}

	tick := &price.Tick{
		Symbol:    symbol,
		Source:    price.SourceKISREST,
		LastPrice: lastPrice,
		TS:        time.Now(),
		CreatedTS: time.Now(),
	}

	// Parse optional fields
	if output.PrdyVrss != "" {
		if vrss, err := strconv.ParseInt(output.PrdyVrss, 10, 64); err == nil {
			// Apply sign
			switch output.PrdyVrssSign {
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

	if output.PrdyCtrt != "" {
		if ctrt, err := strconv.ParseFloat(output.PrdyCtrt, 64); err == nil {
			tick.ChangeRate = &ctrt
		}
	}

	if output.AccuVol != "" {
		if vol, err := strconv.ParseInt(output.AccuVol, 10, 64); err == nil {
			tick.Volume = &vol
		}
	}

	// Bid/Ask prices
	if output.ShtnAskrspr1 != "" {
		if bid, err := strconv.ParseInt(output.ShtnAskrspr1, 10, 64); err == nil {
			tick.BidPrice = &bid
		}
	}

	if output.SelnAskrspr1 != "" {
		if ask, err := strconv.ParseInt(output.SelnAskrspr1, 10, 64); err == nil {
			tick.AskPrice = &ask
		}
	}

	// Bid/Ask volumes
	if output.BiddVolume1 != "" {
		if bidVol, err := strconv.ParseInt(output.BiddVolume1, 10, 64); err == nil {
			tick.BidVolume = &bidVol
		}
	}

	if output.AskoVolume1 != "" {
		if askVol, err := strconv.ParseInt(output.AskoVolume1, 10, 64); err == nil {
			tick.AskVolume = &askVol
		}
	}

	return tick, nil
}

// GetCurrentPrices fetches current prices for multiple symbols
func (c *RESTClient) GetCurrentPrices(ctx context.Context, symbols []string) ([]*price.Tick, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	ticks := make([]*price.Tick, 0, len(symbols))
	var lastErr error
	failCount := 0

	for i, symbol := range symbols {
		// Rate limiting FIRST (except for first request)
		// KIS allows ~20 req/sec, so sleep 50ms between requests
		// This prevents burst on failures
		if i > 0 {
			select {
			case <-ctx.Done():
				return ticks, ctx.Err()
			case <-time.After(50 * time.Millisecond):
			}
		}

		tick, err := c.GetCurrentPrice(ctx, symbol)
		if err != nil {
			failCount++
			lastErr = err
			// Continue to next symbol, but error is tracked
			continue
		}
		ticks = append(ticks, tick)
	}

	// Return error if ALL calls failed
	if len(ticks) == 0 && failCount > 0 {
		return nil, fmt.Errorf("all %d price fetches failed, last error: %w", failCount, lastErr)
	}

	return ticks, nil
}

// HoldingResponse represents KIS holdings inquiry API response
type HoldingResponse struct {
	RetCode string          `json:"rt_cd"` // "0" = success
	MsgCode string          `json:"msg_cd"`
	Msg1    string          `json:"msg1"`
	Output1 []HoldingOutput `json:"output1"`
	Output2 []struct {
		TotalPurchaseAmount  string `json:"pchs_amt_smtl_amt"`  // 매입금액합계금액
		TotalEvaluateAmount  string `json:"evlu_amt_smtl_amt"`  // 평가금액합계금액
		TotalEvaluateProfitLoss string `json:"evlu_pfls_smtl_amt"` // 평가손익합계금액
	} `json:"output2"`
}

// HoldingOutput represents holding data
type HoldingOutput struct {
	Symbol              string `json:"pdno"`           // 종목코드
	SymbolName          string `json:"prdt_name"`      // 종목명
	HoldingQty          string `json:"hldg_qty"`       // 보유수량
	AvgPurchasePrice    string `json:"pchs_avg_pric"`  // 매입평균가격
	CurrentPrice        string `json:"prpr"`           // 현재가
	EvaluateAmount      string `json:"evlu_amt"`       // 평가금액
	EvaluateProfitLoss  string `json:"evlu_pfls_amt"`  // 평가손익금액
	EvaluateProfitLossRate string `json:"evlu_pfls_rt"` // 평가손익율
	PurchaseAmount      string `json:"pchs_amt"`       // 매입금액
}

// GetMarket infers market type from symbol code
// KOSPI: 000000~099999 (generally starting with 0)
// KOSDAQ: 100000~399999 (generally starting with 1-3)
func (h *HoldingOutput) GetMarket() string {
	if len(h.Symbol) != 6 {
		return "UNKNOWN"
	}

	firstChar := h.Symbol[0]
	switch {
	case firstChar == '0':
		return "KOSPI"
	case firstChar >= '1' && firstChar <= '3':
		return "KOSDAQ"
	default:
		return "UNKNOWN"
	}
}

// ============================================================================
// 주문 조회 API (TTTC8001R)
// ============================================================================

// OrdersResponse represents KIS orders API response
type OrdersResponse struct {
	RetCode      string        `json:"rt_cd"`
	MsgCode      string        `json:"msg_cd"`
	Msg1         string        `json:"msg1"`
	CtxAreaFK100 string        `json:"ctx_area_fk100"`
	CtxAreaNK100 string        `json:"ctx_area_nk100"`
	Output1      []OrderOutput `json:"output1"`
}

// OrderOutput represents order data from KIS
type OrderOutput struct {
	OrderDate       string `json:"ord_dt"`               // 주문일자
	OrderNo         string `json:"odno"`                 // 주문번호
	OrigOrderNo     string `json:"orgn_odno"`            // 원주문번호
	OrderSide       string `json:"sll_buy_dvsn_cd"`      // 매도매수구분 (01:매도, 02:매수)
	OrderSideName   string `json:"sll_buy_dvsn_cd_name"` // 매도매수구분명
	StockCode       string `json:"pdno"`                 // 종목코드
	StockName       string `json:"prdt_name"`            // 종목명
	OrderQty        string `json:"ord_qty"`              // 주문수량
	OrderPrice      string `json:"ord_unpr"`             // 주문단가
	OrderTime       string `json:"ord_tmd"`              // 주문시간 (HHMMSS)
	TotalExecQty    string `json:"tot_ccld_qty"`         // 총체결수량
	AvgExecPrice    string `json:"avg_prvs"`             // 체결평균가
	TotalExecAmount string `json:"tot_ccld_amt"`         // 총체결금액
	RemainingQty    string `json:"rmn_qty"`              // 잔여수량
	OrderTypeName   string `json:"ord_dvsn_name"`        // 주문구분명
	CancelYN        string `json:"cncl_yn"`              // 취소여부
}

// GetUnfilledOrders fetches unfilled orders (미체결 주문 조회)
func (c *RESTClient) GetUnfilledOrders(ctx context.Context, accountNo string, accountProductCode string) ([]OrderOutput, error) {
	return c.getOrders(ctx, accountNo, accountProductCode, "02") // 02: 미체결
}

// GetFilledOrders fetches filled orders (체결 주문 조회)
func (c *RESTClient) GetFilledOrders(ctx context.Context, accountNo string, accountProductCode string) ([]OrderOutput, error) {
	return c.getOrders(ctx, accountNo, accountProductCode, "01") // 01: 체결
}

// getOrders fetches orders with filter (ccldDvsn: 00-전체, 01-체결, 02-미체결)
func (c *RESTClient) getOrders(ctx context.Context, accountNo string, accountProductCode string, ccldDvsn string) ([]OrderOutput, error) {
	// Get access token
	token, err := c.auth.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	// 오늘 날짜
	today := time.Now().Format("20060102")

	// Build request URL (주문체결내역 조회)
	url := fmt.Sprintf("%s/uapi/domestic-stock/v1/trading/inquire-daily-ccld", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("CANO", accountNo)
	q.Add("ACNT_PRDT_CD", accountProductCode)
	q.Add("INQR_STRT_DT", today)
	q.Add("INQR_END_DT", today)
	q.Add("SLL_BUY_DVSN_CD", "00")  // 00: 전체
	q.Add("INQR_DVSN", "00")        // 00: 역순
	q.Add("PDNO", "")               // 종목코드 (빈값: 전체)
	q.Add("CCLD_DVSN", ccldDvsn)    // 체결구분
	q.Add("ORD_GNO_BRNO", "")
	q.Add("ODNO", "")
	q.Add("INQR_DVSN_3", "00")
	q.Add("INQR_DVSN_1", "")
	q.Add("CTX_AREA_FK100", "")
	q.Add("CTX_AREA_NK100", "")
	req.URL.RawQuery = q.Encode()

	// TR_ID (실전투자만 지원)
	trID := "TTTC8001R"
	if c.isPaper {
		return nil, fmt.Errorf("order inquiry not supported in paper trading mode")
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("appkey", c.auth.appKey)
	req.Header.Set("appsecret", c.auth.appSecret)
	req.Header.Set("tr_id", trID)

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
		return nil, fmt.Errorf("KIS API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var ordersResp OrdersResponse
	if err := json.Unmarshal(respBody, &ordersResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if ordersResp.RetCode != "0" {
		return nil, fmt.Errorf("KIS API error: code=%s msg=%s", ordersResp.MsgCode, ordersResp.Msg1)
	}

	return ordersResp.Output1, nil
}

// ============================================================================
// 주문 취소 API (TTTC0803U)
// ============================================================================

// CancelOrderResponse represents cancel order response
type CancelOrderResponse struct {
	RetCode string `json:"rt_cd"`
	MsgCode string `json:"msg_cd"`
	Msg1    string `json:"msg1"`
	Output  struct {
		OrderNo   string `json:"ODNO"`     // 주문번호
		OrderTime string `json:"ORD_TMD"`  // 주문시간
	} `json:"output"`
}

// CancelOrderResult represents cancel order result
type CancelOrderResult struct {
	Success   bool
	OrderNo   string
	OrderTime string
	Message   string
}

// CancelOrder cancels an order (주문 취소)
func (c *RESTClient) CancelOrder(ctx context.Context, accountNo string, accountProductCode string, orderNo string) (*CancelOrderResult, error) {
	// Get access token
	token, err := c.auth.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	// 모의투자는 지원하지 않음
	if c.isPaper {
		return nil, fmt.Errorf("order cancel not supported in paper trading mode")
	}

	// Build request URL (주문정정취소)
	url := fmt.Sprintf("%s/uapi/domestic-stock/v1/trading/order-rvsecncl", c.baseURL)

	// Request body
	body := map[string]string{
		"CANO":           accountNo,
		"ACNT_PRDT_CD":   accountProductCode,
		"KRX_FWDG_ORD_ORGNO": "",       // 공백
		"ORGN_ODNO":      orderNo,       // 원주문번호
		"ORD_DVSN":       "00",          // 주문구분 (00: 지정가)
		"RVSE_CNCL_DVSN_CD": "02",       // 02: 취소
		"ORD_QTY":        "0",           // 취소시 0
		"ORD_UNPR":       "0",           // 취소시 0
		"QTY_ALL_ORD_YN": "Y",           // Y: 전량취소
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// TR_ID
	trID := "TTTC0803U"

	// Add headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("appkey", c.auth.appKey)
	req.Header.Set("appsecret", c.auth.appSecret)
	req.Header.Set("tr_id", trID)

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
		return &CancelOrderResult{
			Success: false,
			Message: fmt.Sprintf("cancel failed: status=%d body=%s", resp.StatusCode, string(respBody)),
		}, nil
	}

	// Parse response
	var cancelResp CancelOrderResponse
	if err := json.Unmarshal(respBody, &cancelResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if cancelResp.RetCode != "0" {
		return &CancelOrderResult{
			Success: false,
			Message: fmt.Sprintf("KIS API error: code=%s msg=%s", cancelResp.MsgCode, cancelResp.Msg1),
		}, nil
	}

	return &CancelOrderResult{
		Success:   true,
		OrderNo:   cancelResp.Output.OrderNo,
		OrderTime: cancelResp.Output.OrderTime,
		Message:   cancelResp.Msg1,
	}, nil
}

// PlaceOrderResult represents place order result
type PlaceOrderResult struct {
	Success   bool
	OrderNo   string
	OrderTime string
	Message   string
}

// PlaceOrderResponse represents KIS place order API response
type PlaceOrderResponse struct {
	RetCode string `json:"rt_cd"`
	MsgCode string `json:"msg_cd"`
	Msg1    string `json:"msg1"`
	Output  struct {
		OrderNo   string `json:"ODNO"`
		OrderTime string `json:"ORD_TMD"`
	} `json:"output"`
}

// PlaceOrder submits an order to KIS (현금 매수/매도)
// side: "buy" or "sell"
// orderType: "limit" or "market"
func (c *RESTClient) PlaceOrder(ctx context.Context, accountNo string, accountProductCode string, symbol string, side string, orderType string, qty int64, price int64) (*PlaceOrderResult, error) {
	// Get access token
	token, err := c.auth.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	// 모의투자는 지원하지 않음
	if c.isPaper {
		return nil, fmt.Errorf("order placement not supported in paper trading mode")
	}

	// Build request URL
	url := fmt.Sprintf("%s/uapi/domestic-stock/v1/trading/order-cash", c.baseURL)

	// TR_ID: TTTC0802U (매수), TTTC0801U (매도)
	var trID string
	if side == "buy" {
		trID = "TTTC0802U"
	} else {
		trID = "TTTC0801U"
	}

	// 주문구분: 00(지정가), 01(시장가)
	var ordDvsn string
	if orderType == "market" {
		ordDvsn = "01"
		price = 0 // 시장가는 가격 0
	} else {
		ordDvsn = "00"
	}

	// Request body
	body := map[string]string{
		"CANO":         accountNo,
		"ACNT_PRDT_CD": accountProductCode,
		"PDNO":         symbol,
		"ORD_DVSN":     ordDvsn,
		"ORD_QTY":      fmt.Sprintf("%d", qty),
		"ORD_UNPR":     fmt.Sprintf("%d", price),
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("appkey", c.auth.appKey)
	req.Header.Set("appsecret", c.auth.appSecret)
	req.Header.Set("tr_id", trID)

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
		return &PlaceOrderResult{
			Success: false,
			Message: fmt.Sprintf("order failed: status=%d body=%s", resp.StatusCode, string(respBody)),
		}, nil
	}

	// Parse response
	var orderResp PlaceOrderResponse
	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if orderResp.RetCode != "0" {
		return &PlaceOrderResult{
			Success: false,
			Message: fmt.Sprintf("KIS API error: code=%s msg=%s", orderResp.MsgCode, orderResp.Msg1),
		}, nil
	}

	return &PlaceOrderResult{
		Success:   true,
		OrderNo:   orderResp.Output.OrderNo,
		OrderTime: orderResp.Output.OrderTime,
		Message:   orderResp.Msg1,
	}, nil
}

// GetHoldings fetches current holdings (보유종목 조회)
func (c *RESTClient) GetHoldings(ctx context.Context, accountNo string, accountProductCode string) ([]HoldingOutput, error) {
	// Get access token
	token, err := c.auth.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	// Build request URL
	// 국내주식주문 > 주식잔고조회
	url := fmt.Sprintf("%s/uapi/domestic-stock/v1/trading/inquire-balance", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("CANO", accountNo)                // 계좌번호
	q.Add("ACNT_PRDT_CD", accountProductCode) // 계좌상품코드
	q.Add("AFHR_FLPR_YN", "N")              // 시간외단일가여부 (N: 기본값)
	q.Add("OFL_YN", "")                     // 오프라인여부
	q.Add("INQR_DVSN", "02")                // 조회구분 (01: 대출일별, 02: 종목별)
	q.Add("UNPR_DVSN", "01")                // 단가구분 (01: 기본값)
	q.Add("FUND_STTL_ICLD_YN", "N")         // 펀드결제분포함여부
	q.Add("FNCG_AMT_AUTO_RDPT_YN", "N")     // 융자금액자동상환여부
	q.Add("PRCS_DVSN", "01")                // 처리구분 (01: 전일매매포함)
	q.Add("CTX_AREA_FK100", "")             // 연속조회검색조건100
	q.Add("CTX_AREA_NK100", "")             // 연속조회키100
	req.URL.RawQuery = q.Encode()

	// Determine TR_ID based on isPaper flag (실전/모의투자)
	trID := "TTTC8434R" // 실전투자 (default)
	if c.isPaper {
		trID = "VTTC8434R" // 모의투자
	}

	// Debug log
	fmt.Printf("[DEBUG] isPaper: %v, TR_ID: %s\n", c.isPaper, trID)

	// Add headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("appkey", c.auth.appKey)
	req.Header.Set("appsecret", c.auth.appSecret)
	req.Header.Set("tr_id", trID)

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
		return nil, fmt.Errorf("KIS API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var holdingResp HoldingResponse
	if err := json.Unmarshal(respBody, &holdingResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if holdingResp.RetCode != "0" {
		return nil, fmt.Errorf("KIS API error: code=%s msg=%s", holdingResp.MsgCode, holdingResp.Msg1)
	}

	return holdingResp.Output1, nil
}
