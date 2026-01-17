package dart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
)

const (
	baseURL        = "https://opendart.fss.or.kr/api"
	defaultTimeout = 30 * time.Second
)

// Client DART OpenAPI 클라이언트
type Client struct {
	httpClient *http.Client
	apiKey     string
}

// NewClient 클라이언트 생성
func NewClient(apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		apiKey: apiKey,
	}
}

// NewClientWithTimeout 타임아웃 지정 클라이언트 생성
func NewClientWithTimeout(apiKey string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiKey: apiKey,
	}
}

// =============================================================================
// API Response Types
// =============================================================================

// ListResponse 공시 목록 API 응답
type ListResponse struct {
	Status       string          `json:"status"`
	Message      string          `json:"message"`
	PageNo       int             `json:"page_no"`
	PageCount    int             `json:"page_count"`
	TotalCount   int             `json:"total_count"`
	TotalPage    int             `json:"total_page"`
	Disclosures  []DisclosureDTO `json:"list"`
}

// DisclosureDTO 공시 항목 DTO
type DisclosureDTO struct {
	CorpCode    string `json:"corp_code"`    // 고유번호
	CorpName    string `json:"corp_name"`    // 회사명
	StockCode   string `json:"stock_code"`   // 종목코드
	CorpClass   string `json:"corp_cls"`     // 법인구분 (Y: 유가, K: 코스닥, N: 코넥스, E: 기타)
	ReportNm    string `json:"report_nm"`    // 보고서명
	RceptNo     string `json:"rcept_no"`     // 접수번호
	FlrNm       string `json:"flr_nm"`       // 공시제출인명
	RceptDt     string `json:"rcept_dt"`     // 접수일자 (YYYYMMDD)
	Rm          string `json:"rm"`           // 비고 (코/코스/유/유가/넥/코넥스)
}

// CorpCodeResponse 기업코드 API 응답
type CorpCodeResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	List    []CorpCodeDTO `json:"list"`
}

// CorpCodeDTO 기업코드 DTO
type CorpCodeDTO struct {
	CorpCode   string `json:"corp_code"`   // 고유번호
	CorpName   string `json:"corp_name"`   // 회사명
	StockCode  string `json:"stock_code"`  // 종목코드
	ModifyDate string `json:"modify_date"` // 최종변경일
}

// =============================================================================
// Disclosure Fetch
// =============================================================================

// FetchDisclosures 공시 목록 조회
// from, to: 공시 검색 기간
// corpCode: 회사 고유번호 (optional, 빈 문자열이면 전체)
func (c *Client) FetchDisclosures(ctx context.Context, from, to time.Time, corpCode string) ([]*fetcher.Disclosure, error) {
	params := url.Values{}
	params.Set("crtfc_key", c.apiKey)
	params.Set("bgn_de", from.Format("20060102"))
	params.Set("end_de", to.Format("20060102"))
	params.Set("page_no", "1")
	params.Set("page_count", "100")

	if corpCode != "" {
		params.Set("corp_code", corpCode)
	}

	url := fmt.Sprintf("%s/list.json?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// 상태 확인
	if listResp.Status != "000" && listResp.Status != "013" {
		// 000: 정상, 013: 조회 결과 없음
		return nil, fmt.Errorf("dart api error: %s - %s", listResp.Status, listResp.Message)
	}

	// DTO -> Domain 변환
	disclosures := make([]*fetcher.Disclosure, 0, len(listResp.Disclosures))
	for _, dto := range listResp.Disclosures {
		if dto.StockCode == "" {
			continue // 종목코드 없는 공시 스킵
		}

		disclosedAt, err := time.Parse("20060102", dto.RceptDt)
		if err != nil {
			continue
		}

		category := mapReportCategory(dto.ReportNm)
		disclosureURL := fmt.Sprintf("https://dart.fss.or.kr/dsaf001/main.do?rcpNo=%s", dto.RceptNo)

		disclosures = append(disclosures, &fetcher.Disclosure{
			StockCode:   dto.StockCode,
			DisclosedAt: disclosedAt,
			Title:       dto.ReportNm,
			Category:    &category,
			URL:         &disclosureURL,
			DartRceptNo: &dto.RceptNo,
		})
	}

	log.Debug().
		Int("total_count", listResp.TotalCount).
		Int("fetched", len(disclosures)).
		Time("from", from).
		Time("to", to).
		Msg("Fetched disclosures from DART")

	return disclosures, nil
}

// FetchAllDisclosures 전체 공시 조회 (페이징 처리)
func (c *Client) FetchAllDisclosures(ctx context.Context, from, to time.Time) ([]*fetcher.Disclosure, error) {
	var allDisclosures []*fetcher.Disclosure
	pageNo := 1
	pageCount := 100

	for {
		select {
		case <-ctx.Done():
			return allDisclosures, ctx.Err()
		default:
		}

		params := url.Values{}
		params.Set("crtfc_key", c.apiKey)
		params.Set("bgn_de", from.Format("20060102"))
		params.Set("end_de", to.Format("20060102"))
		params.Set("page_no", fmt.Sprintf("%d", pageNo))
		params.Set("page_count", fmt.Sprintf("%d", pageCount))

		url := fmt.Sprintf("%s/list.json?%s", baseURL, params.Encode())

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return allDisclosures, fmt.Errorf("create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return allDisclosures, fmt.Errorf("do request: %w", err)
		}

		var listResp ListResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			resp.Body.Close()
			return allDisclosures, fmt.Errorf("decode response: %w", err)
		}
		resp.Body.Close()

		if listResp.Status != "000" {
			if listResp.Status == "013" {
				// 조회 결과 없음
				break
			}
			return allDisclosures, fmt.Errorf("dart api error: %s - %s", listResp.Status, listResp.Message)
		}

		// DTO -> Domain 변환
		for _, dto := range listResp.Disclosures {
			if dto.StockCode == "" {
				continue
			}

			disclosedAt, err := time.Parse("20060102", dto.RceptDt)
			if err != nil {
				continue
			}

			category := mapReportCategory(dto.ReportNm)
			disclosureURL := fmt.Sprintf("https://dart.fss.or.kr/dsaf001/main.do?rcpNo=%s", dto.RceptNo)

			allDisclosures = append(allDisclosures, &fetcher.Disclosure{
				StockCode:   dto.StockCode,
				DisclosedAt: disclosedAt,
				Title:       dto.ReportNm,
				Category:    &category,
				URL:         &disclosureURL,
				DartRceptNo: &dto.RceptNo,
			})
		}

		// 다음 페이지 확인
		if pageNo >= listResp.TotalPage {
			break
		}
		pageNo++

		// Rate limiting (DART API는 분당 요청 제한 있음)
		time.Sleep(100 * time.Millisecond)
	}

	log.Info().
		Int("total", len(allDisclosures)).
		Time("from", from).
		Time("to", to).
		Msg("Fetched all disclosures from DART")

	return allDisclosures, nil
}

// FetchDisclosuresByStock 종목별 공시 조회
func (c *Client) FetchDisclosuresByStock(ctx context.Context, stockCode string, from, to time.Time) ([]*fetcher.Disclosure, error) {
	// 종목코드로 회사 고유번호 조회 필요
	// DART API는 종목코드가 아닌 회사 고유번호(corp_code)로 검색
	// 여기서는 일단 전체 공시에서 필터링

	allDisc, err := c.FetchDisclosures(ctx, from, to, "")
	if err != nil {
		return nil, err
	}

	var filtered []*fetcher.Disclosure
	for _, d := range allDisc {
		if d.StockCode == stockCode {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}

// =============================================================================
// Fundamentals Fetch
// =============================================================================

// FinancialResponse 재무 정보 API 응답
type FinancialResponse struct {
	Status  string         `json:"status"`
	Message string         `json:"message"`
	List    []FinancialDTO `json:"list"`
}

// FinancialDTO 재무 정보 DTO
type FinancialDTO struct {
	RceptNo      string `json:"rcept_no"`      // 접수번호
	CorpCode     string `json:"corp_code"`     // 회사코드
	BsnsYear     string `json:"bsns_year"`     // 사업연도
	StockCode    string `json:"stock_code"`    // 종목코드
	ReprtCode    string `json:"reprt_code"`    // 보고서코드 (11011: 사업보고서)
	AccountNm    string `json:"account_nm"`    // 계정명
	ThstrmAmount string `json:"thstrm_amount"` // 당기금액
}

// FetchFinancials 재무 정보 조회
func (c *Client) FetchFinancials(ctx context.Context, corpCode string, year int, reportCode string) (*fetcher.Fundamentals, error) {
	params := url.Values{}
	params.Set("crtfc_key", c.apiKey)
	params.Set("corp_code", corpCode)
	params.Set("bsns_year", fmt.Sprintf("%d", year))
	params.Set("reprt_code", reportCode) // 11011: 사업보고서, 11012: 반기보고서, 11013: 1분기보고서, 11014: 3분기보고서

	url := fmt.Sprintf("%s/fnlttSinglAcnt.json?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	var finResp FinancialResponse
	if err := json.NewDecoder(resp.Body).Decode(&finResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if finResp.Status != "000" {
		if finResp.Status == "013" {
			return nil, fetcher.ErrFundamentalsNotFound
		}
		return nil, fmt.Errorf("dart api error: %s - %s", finResp.Status, finResp.Message)
	}

	// 재무 데이터 파싱
	fund := &fetcher.Fundamentals{
		ReportDate: time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC),
	}

	for _, dto := range finResp.List {
		if fund.StockCode == "" && dto.StockCode != "" {
			fund.StockCode = dto.StockCode
		}

		// 계정명에 따라 값 할당
		// (실제 구현 시 계정 코드 매핑 필요)
		// 여기서는 간단히 계정명으로 구분
		switch dto.AccountNm {
		case "매출액":
			val := parseAmount(dto.ThstrmAmount)
			fund.Revenue = &val
		case "영업이익":
			val := parseAmount(dto.ThstrmAmount)
			fund.OperatingProfit = &val
		case "당기순이익":
			val := parseAmount(dto.ThstrmAmount)
			fund.NetProfit = &val
		}
	}

	return fund, nil
}

// =============================================================================
// Health Check
// =============================================================================

// HealthCheck API 상태 확인
func (c *Client) HealthCheck(ctx context.Context) error {
	// 간단한 API 호출로 상태 확인
	params := url.Values{}
	params.Set("crtfc_key", c.apiKey)
	params.Set("bgn_de", time.Now().AddDate(0, 0, -1).Format("20060102"))
	params.Set("end_de", time.Now().Format("20060102"))
	params.Set("page_no", "1")
	params.Set("page_count", "1")

	url := fmt.Sprintf("%s/list.json?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %d", resp.StatusCode)
	}

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	// 000: 정상, 013: 조회 결과 없음 (둘 다 정상)
	if listResp.Status != "000" && listResp.Status != "013" {
		return fmt.Errorf("dart api error: %s - %s", listResp.Status, listResp.Message)
	}

	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// mapReportCategory 보고서명을 카테고리로 매핑
func mapReportCategory(reportNm string) string {
	// 간단한 키워드 기반 분류
	categories := map[string][]string{
		"정기공시":   {"사업보고서", "반기보고서", "분기보고서"},
		"주요사항":   {"주요사항보고서", "공정공시"},
		"지분공시":   {"임원ㆍ주요주주", "최대주주", "주식등의대량보유"},
		"합병/분할":  {"합병", "분할"},
		"유상증자":   {"유상증자", "주주배정"},
		"무상증자":   {"무상증자", "주식배당"},
		"전환사채":   {"전환사채", "CB"},
		"자사주":    {"자기주식", "자사주"},
		"실적공시":   {"영업(잠정)실적", "매출액"},
		"기타공시":   {},
	}

	for category, keywords := range categories {
		for _, keyword := range keywords {
			if containsKeyword(reportNm, keyword) {
				return category
			}
		}
	}

	return "기타공시"
}

// containsKeyword 키워드 포함 여부 확인
func containsKeyword(s, keyword string) bool {
	return len(keyword) > 0 && (len(s) >= len(keyword)) &&
		(s == keyword || contains(s, keyword))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// parseAmount 금액 문자열 파싱
func parseAmount(s string) int64 {
	if s == "" {
		return 0
	}

	// 콤마 제거
	clean := ""
	negative := false
	for _, r := range s {
		if r == '-' {
			negative = true
			continue
		}
		if r >= '0' && r <= '9' {
			clean += string(r)
		}
	}

	if clean == "" {
		return 0
	}

	var val int64
	for _, r := range clean {
		val = val*10 + int64(r-'0')
	}

	if negative {
		return -val
	}
	return val
}

// =============================================================================
// DART Error Codes
// =============================================================================

// DART API 상태 코드
const (
	StatusOK              = "000" // 정상
	StatusNoData          = "013" // 조회 결과 없음
	StatusInvalidAPIKey   = "010" // API 키 오류
	StatusNoCorpCode      = "011" // 고유번호 오류
	StatusRateLimitExceed = "020" // 요청 제한 초과
	StatusInternalError   = "800" // 시스템 점검 중
)

// IsDARTError DART API 오류 여부 확인
func IsDARTError(status string) bool {
	return status != StatusOK && status != StatusNoData
}

// GetDARTErrorMessage DART 에러 메시지 반환
func GetDARTErrorMessage(status string) string {
	messages := map[string]string{
		"000": "정상",
		"010": "등록되지 않은 키입니다",
		"011": "사용할 수 없는 키입니다",
		"013": "조회된 데이터가 없습니다",
		"020": "요청 제한을 초과하였습니다",
		"100": "필수 파라미터가 누락되었습니다",
		"800": "시스템 점검 중입니다",
		"900": "알 수 없는 오류",
	}

	if msg, ok := messages[status]; ok {
		return msg
	}
	return "알 수 없는 오류"
}
