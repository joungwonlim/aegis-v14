package kis

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
	ticks := make([]*price.Tick, 0, len(symbols))

	for _, symbol := range symbols {
		tick, err := c.GetCurrentPrice(ctx, symbol)
		if err != nil {
			// Log error but continue with other symbols
			// TODO: add logging
			continue
		}
		ticks = append(ticks, tick)

		// Rate limiting: KIS allows ~20 req/sec, so sleep 50ms between requests
		select {
		case <-ctx.Done():
			return ticks, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
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
