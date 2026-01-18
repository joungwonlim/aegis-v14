package naver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
)

const (
	baseURL        = "https://finance.naver.com"
	fcChartURL     = "https://fchart.stock.naver.com"
	defaultTimeout = 30 * time.Second
)

// Client 네이버 금융 클라이언트
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// NewClient 클라이언트 생성
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
	}
}

// NewClientWithTimeout 타임아웃 지정 클라이언트 생성
func NewClientWithTimeout(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
	}
}

// =============================================================================
// Daily Prices
// =============================================================================

// FetchDailyPrices 일봉 데이터 수집
// days: 수집할 일수 (최대 약 100일)
func (c *Client) FetchDailyPrices(ctx context.Context, stockCode string, days int) ([]*fetcher.DailyPrice, error) {
	// sise_day.naver를 파싱하여 일봉 데이터 수집
	url := fmt.Sprintf("%s/item/sise_day.naver?code=%s&page=1", baseURL, stockCode)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var prices []*fetcher.DailyPrice

	// 테이블 파싱
	doc.Find("table.type2 tr").Each(func(i int, s *goquery.Selection) {
		// 헤더 행 건너뛰기
		if s.Find("th").Length() > 0 {
			return
		}

		tds := s.Find("td")
		if tds.Length() < 7 {
			return
		}

		// 날짜 파싱
		dateStr := strings.TrimSpace(tds.Eq(0).Text())
		if dateStr == "" {
			return
		}
		tradeDate, err := time.Parse("2006.01.02", dateStr)
		if err != nil {
			return
		}

		// 가격 데이터 파싱
		closePrice := parseNumber(tds.Eq(1).Text())
		openPrice := parseNumber(tds.Eq(3).Text())
		highPrice := parseNumber(tds.Eq(4).Text())
		lowPrice := parseNumber(tds.Eq(5).Text())
		volume := parseNumber(tds.Eq(6).Text())

		if closePrice == 0 {
			return
		}

		prices = append(prices, &fetcher.DailyPrice{
			StockCode:  stockCode,
			TradeDate:  tradeDate,
			OpenPrice:  float64(openPrice),
			HighPrice:  float64(highPrice),
			LowPrice:   float64(lowPrice),
			ClosePrice: float64(closePrice),
			Volume:     volume,
		})
	})

	// 요청한 일수만큼만 반환
	if len(prices) > days {
		prices = prices[:days]
	}

	log.Debug().
		Str("stock_code", stockCode).
		Int("count", len(prices)).
		Msg("Fetched daily prices from Naver")

	return prices, nil
}

// =============================================================================
// Investor Flow
// =============================================================================

// FetchInvestorFlow 투자자별 수급 수집
func (c *Client) FetchInvestorFlow(ctx context.Context, stockCode string, days int) ([]*fetcher.InvestorFlow, error) {
	// sise_trans.naver를 파싱하여 투자자별 수급 수집
	url := fmt.Sprintf("%s/item/frgn.naver?code=%s&page=1", baseURL, stockCode)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var flows []*fetcher.InvestorFlow

	// 테이블 파싱 (외국인/기관 순매매)
	doc.Find("table.type2 tr").Each(func(i int, s *goquery.Selection) {
		if s.Find("th").Length() > 0 {
			return
		}

		tds := s.Find("td")
		if tds.Length() < 9 {
			return
		}

		dateStr := strings.TrimSpace(tds.Eq(0).Text())
		if dateStr == "" {
			return
		}
		tradeDate, err := time.Parse("2006.01.02", dateStr)
		if err != nil {
			return
		}

		// 외국인, 기관 순매수량
		foreignNet := parseSignedNumber(tds.Eq(5).Text())
		instNet := parseSignedNumber(tds.Eq(6).Text())

		flows = append(flows, &fetcher.InvestorFlow{
			StockCode:     stockCode,
			TradeDate:     tradeDate,
			ForeignNetQty: foreignNet,
			InstNetQty:    instNet,
			IndivNetQty:   -(foreignNet + instNet), // 개인 = -(외국인+기관)
		})
	})

	if len(flows) > days {
		flows = flows[:days]
	}

	log.Debug().
		Str("stock_code", stockCode).
		Int("count", len(flows)).
		Msg("Fetched investor flow from Naver")

	return flows, nil
}

// =============================================================================
// Market Cap
// =============================================================================

// FetchMarketCap 시가총액 조회
func (c *Client) FetchMarketCap(ctx context.Context, stockCode string) (*fetcher.MarketCap, error) {
	url := fmt.Sprintf("%s/item/main.naver?code=%s", baseURL, stockCode)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	// 시가총액 파싱
	var marketCap int64
	var sharesOut int64

	doc.Find("table.no_info tr").Each(func(i int, s *goquery.Selection) {
		th := strings.TrimSpace(s.Find("th").Text())
		td := strings.TrimSpace(s.Find("td").Text())

		switch {
		case strings.Contains(th, "시가총액"):
			// "1,234,567억원" 형태
			marketCap = parseMarketCapValue(td)
		case strings.Contains(th, "상장주식수"):
			sharesOut = parseNumber(td)
		}
	})

	if marketCap == 0 {
		return nil, fetcher.ErrMarketCapNotFound
	}

	return &fetcher.MarketCap{
		StockCode: stockCode,
		TradeDate: time.Now().Truncate(24 * time.Hour),
		MarketCap: marketCap,
		SharesOut: &sharesOut,
	}, nil
}

// =============================================================================
// Stock Info (종목 마스터)
// =============================================================================

// FetchStockInfo 종목 기본 정보 수집
func (c *Client) FetchStockInfo(ctx context.Context, stockCode string) (*fetcher.Stock, error) {
	url := fmt.Sprintf("%s/item/main.naver?code=%s", baseURL, stockCode)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	// 종목명 파싱
	name := strings.TrimSpace(doc.Find("div.wrap_company h2 a").Text())
	if name == "" {
		name = strings.TrimSpace(doc.Find("div.h_company h2").Text())
	}

	// 시장 구분 파싱 (KOSPI, KOSDAQ)
	market := "KOSPI"
	img := doc.Find("img.kosdaq")
	if img.Length() > 0 {
		market = "KOSDAQ"
	}

	// 업종 파싱
	var sector string
	doc.Find("div.section.trade_compare a").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if strings.Contains(href, "upjong") {
			sector = strings.TrimSpace(s.Text())
		}
	})

	return &fetcher.Stock{
		Code:   stockCode,
		Name:   name,
		Market: market,
		Sector: &sector,
		Status: "active",
	}, nil
}

// =============================================================================
// Fundamentals
// =============================================================================

// FetchFundamentals 재무 지표 수집
func (c *Client) FetchFundamentals(ctx context.Context, stockCode string) (*fetcher.Fundamentals, error) {
	url := fmt.Sprintf("%s/item/main.naver?code=%s", baseURL, stockCode)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	fund := &fetcher.Fundamentals{
		StockCode:  stockCode,
		ReportDate: time.Now().Truncate(24 * time.Hour),
	}

	// PER, PBR 등 파싱
	doc.Find("table.per_table tr").Each(func(i int, s *goquery.Selection) {
		ths := s.Find("th")
		tds := s.Find("td")

		ths.Each(func(j int, th *goquery.Selection) {
			label := strings.TrimSpace(th.Text())
			if j < tds.Length() {
				valueStr := strings.TrimSpace(tds.Eq(j).Text())
				value := parseFloat(valueStr)

				switch {
				case strings.Contains(label, "PER"):
					fund.PER = &value
				case strings.Contains(label, "PBR"):
					fund.PBR = &value
				case strings.Contains(label, "ROE"):
					fund.ROE = &value
				}
			}
		})
	})

	return fund, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// parseNumber 숫자 문자열 파싱 (콤마 제거)
func parseNumber(s string) int64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")

	// 숫자만 추출
	re := regexp.MustCompile(`[\d]+`)
	matches := re.FindString(s)
	if matches == "" {
		return 0
	}

	n, _ := strconv.ParseInt(matches, 10, 64)
	return n
}

// parseSignedNumber 부호 있는 숫자 파싱
func parseSignedNumber(s string) int64 {
	s = strings.TrimSpace(s)
	negative := strings.Contains(s, "-") || strings.Contains(s, "−")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "−", "")
	s = strings.ReplaceAll(s, "+", "")

	re := regexp.MustCompile(`[\d]+`)
	matches := re.FindString(s)
	if matches == "" {
		return 0
	}

	n, _ := strconv.ParseInt(matches, 10, 64)
	if negative {
		return -n
	}
	return n
}

// parseFloat 실수 파싱
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "배", "")
	s = strings.ReplaceAll(s, "%", "")

	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parseMarketCapValue 시가총액 파싱 (억원 단위)
func parseMarketCapValue(s string) int64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "억원", "")
	s = strings.ReplaceAll(s, "억", "")
	s = strings.TrimSpace(s)

	n, _ := strconv.ParseInt(s, 10, 64)
	return n * 100_000_000 // 억 -> 원
}

// =============================================================================
// JSON API (Alternative)
// =============================================================================

// ChartData fchart API 응답
type ChartData struct {
	Symbol string      `json:"symbol"`
	Name   string      `json:"name"`
	Items  []ChartItem `json:"item"`
}

// ChartItem 차트 데이터 항목
type ChartItem struct {
	Date   string `json:"0"` // YYYYMMDD
	Open   string `json:"1"`
	High   string `json:"2"`
	Low    string `json:"3"`
	Close  string `json:"4"`
	Volume string `json:"5"`
}

// FetchDailyPricesJSON JSON API로 일봉 조회 (대안)
func (c *Client) FetchDailyPricesJSON(ctx context.Context, stockCode string, days int) ([]*fetcher.DailyPrice, error) {
	// fchart.stock.naver.com API 사용
	endDate := time.Now().Format("20060102")
	startDate := time.Now().AddDate(0, 0, -days-10).Format("20060102")

	url := fmt.Sprintf("%s/sise.nhn?symbol=%s&timeframe=day&count=%d&requestType=0&startTime=%s&endTime=%s",
		fcChartURL, stockCode, days+10, startDate, endDate)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// XML 파싱 (네이버 차트 API는 XML 반환)
	// 간단한 정규식으로 파싱
	var prices []*fetcher.DailyPrice
	re := regexp.MustCompile(`<item data="(\d+)\|(\d+)\|(\d+)\|(\d+)\|(\d+)\|(\d+)"`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	for _, m := range matches {
		if len(m) < 7 {
			continue
		}

		dateStr := m[1]
		tradeDate, err := time.Parse("20060102", dateStr)
		if err != nil {
			continue
		}

		open, _ := strconv.ParseFloat(m[2], 64)
		high, _ := strconv.ParseFloat(m[3], 64)
		low, _ := strconv.ParseFloat(m[4], 64)
		close, _ := strconv.ParseFloat(m[5], 64)
		volume, _ := strconv.ParseInt(m[6], 10, 64)

		prices = append(prices, &fetcher.DailyPrice{
			StockCode:  stockCode,
			TradeDate:  tradeDate,
			OpenPrice:  open,
			HighPrice:  high,
			LowPrice:   low,
			ClosePrice: close,
			Volume:     volume,
		})
	}

	// 최신순 정렬 (역순)
	for i, j := 0, len(prices)-1; i < j; i, j = i+1, j-1 {
		prices[i], prices[j] = prices[j], prices[i]
	}

	if len(prices) > days {
		prices = prices[:days]
	}

	return prices, nil
}

// HealthCheck API 상태 확인
func (c *Client) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/item/main.naver?code=005930", baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %d", resp.StatusCode)
	}

	return nil
}

// FetchMarketCapRanking 시가총액 순위 조회 (KOSPI/KOSDAQ)
func (c *Client) FetchMarketCapRanking(ctx context.Context, market string, limit int) ([]*fetcher.Stock, error) {
	page := 1
	marketCode := "0" // KOSPI
	if market == "KOSDAQ" {
		marketCode = "1"
	}

	url := fmt.Sprintf("%s/sise/sise_market_sum.naver?sosok=%s&page=%d", baseURL, marketCode, page)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var stocks []*fetcher.Stock

	doc.Find("table.type_2 tr").Each(func(i int, s *goquery.Selection) {
		if len(stocks) >= limit {
			return
		}

		tds := s.Find("td")
		if tds.Length() < 7 {
			return
		}

		// 종목 코드 추출
		link := tds.Eq(1).Find("a")
		href, exists := link.Attr("href")
		if !exists {
			return
		}

		re := regexp.MustCompile(`code=(\d{6})`)
		matches := re.FindStringSubmatch(href)
		if len(matches) < 2 {
			return
		}
		code := matches[1]
		name := strings.TrimSpace(link.Text())

		stocks = append(stocks, &fetcher.Stock{
			Code:   code,
			Name:   name,
			Market: market,
			Status: "active",
		})
	})

	return stocks, nil
}

// =============================================================================
// Company Overview (기업 개요)
// =============================================================================

const stockNaverURL = "https://stock.naver.com"

// CompanyOverview 기업 개요 정보
type CompanyOverview struct {
	Symbol      string `json:"symbol"`
	SymbolName  string `json:"symbol_name"`
	Overview    string `json:"overview"`
	FetchedFrom string `json:"fetched_from"`
}

// FetchCompanyOverview 기업 개요 수집 (stock.naver.com API)
// URL: https://stock.naver.com/domestic/stock/{symbol}/info/company
func (c *Client) FetchCompanyOverview(ctx context.Context, stockCode string) (*CompanyOverview, error) {
	// stock.naver.com은 내부적으로 API를 호출함
	// API 엔드포인트: https://api.stock.naver.com/stock/{code}/basic
	apiURL := fmt.Sprintf("https://api.stock.naver.com/stock/%s/basic", stockCode)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// API 실패 시 HTML 파싱 시도
		return c.fetchCompanyOverviewFromHTML(ctx, stockCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// JSON에서 기업 개요 추출
	// 응답 구조: {"stockName": "...", "corporateSummary": "..."}
	type BasicResponse struct {
		StockName        string `json:"stockName"`
		CorporateSummary string `json:"corporateSummary"`
	}

	var basicResp BasicResponse
	if err := json.Unmarshal(body, &basicResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if basicResp.CorporateSummary == "" {
		// API에 데이터 없으면 HTML 파싱 시도
		return c.fetchCompanyOverviewFromHTML(ctx, stockCode)
	}

	return &CompanyOverview{
		Symbol:      stockCode,
		SymbolName:  basicResp.StockName,
		Overview:    basicResp.CorporateSummary,
		FetchedFrom: "naver_api",
	}, nil
}

// fetchCompanyOverviewFromHTML HTML 파싱으로 기업 개요 수집 (fallback)
func (c *Client) fetchCompanyOverviewFromHTML(ctx context.Context, stockCode string) (*CompanyOverview, error) {
	// finance.naver.com에서 기업 개요 파싱
	url := fmt.Sprintf("%s/item/coinfo.naver?code=%s", baseURL, stockCode)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	// 종목명 파싱
	name := strings.TrimSpace(doc.Find("div.wrap_company h2 a").Text())
	if name == "" {
		name = strings.TrimSpace(doc.Find("div.h_company h2").Text())
	}

	// 기업 개요 파싱 (summary 영역)
	var overview string
	doc.Find("div.summary").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			overview = text
		}
	})

	if overview == "" {
		// 다른 위치에서 시도
		doc.Find("p.content").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" && len(text) > 50 {
				overview = text
			}
		})
	}

	if overview == "" {
		return nil, fmt.Errorf("company overview not found for %s", stockCode)
	}

	return &CompanyOverview{
		Symbol:      stockCode,
		SymbolName:  name,
		Overview:    overview,
		FetchedFrom: "naver_html",
	}, nil
}

// =============================================================================
// Batch Operations
// =============================================================================

// FetchAllPricesResult 배치 가격 수집 결과
type FetchAllPricesResult struct {
	Success int
	Failed  int
	Errors  []string
}

// FetchAllPrices 복수 종목 가격 수집
func (c *Client) FetchAllPrices(
	ctx context.Context,
	stockCodes []string,
	days int,
	rateLimitMs int,
	onPrice func(prices []*fetcher.DailyPrice),
) *FetchAllPricesResult {
	result := &FetchAllPricesResult{}

	for _, code := range stockCodes {
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, "context cancelled")
			return result
		default:
		}

		prices, err := c.FetchDailyPrices(ctx, code, days)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", code, err))
			continue
		}

		if onPrice != nil && len(prices) > 0 {
			onPrice(prices)
		}

		result.Success++

		// Rate limiting
		if rateLimitMs > 0 {
			time.Sleep(time.Duration(rateLimitMs) * time.Millisecond)
		}
	}

	return result
}
