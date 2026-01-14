package stock

import (
	"time"
)

// Stock represents a stock in the market
// Maps to market.stocks table
type Stock struct {
	Symbol          string     `json:"symbol" db:"symbol"`                       // 종목 코드 (6자리 숫자)
	Name            string     `json:"name" db:"name"`                           // 종목명
	Market          string     `json:"market" db:"market"`                       // KOSPI | KOSDAQ | KONEX
	Status          string     `json:"status" db:"status"`                       // ACTIVE | SUSPENDED | DELISTED
	ListingDate     *time.Time `json:"listing_date" db:"listing_date"`           // 상장일
	DelistingDate   *time.Time `json:"delisting_date" db:"delisting_date"`       // 상장폐지일
	Sector          *string    `json:"sector" db:"sector"`                       // 섹터
	Industry        *string    `json:"industry" db:"industry"`                   // 업종
	MarketCap       *int64     `json:"market_cap" db:"market_cap"`               // 시가총액 (원)
	IsTradable      bool       `json:"is_tradable" db:"is_tradable"`             // 거래 가능 여부
	TradeHaltReason *string    `json:"trade_halt_reason" db:"trade_halt_reason"` // 거래정지 사유
	CreatedTS       time.Time  `json:"created_ts" db:"created_ts"`               // 생성 일시
	UpdatedTS       time.Time  `json:"updated_ts" db:"updated_ts"`               // 수정 일시
}

// ListFilter represents filter options for listing stocks
type ListFilter struct {
	Market     *string // KOSPI, KOSDAQ, KONEX
	Status     string  // ACTIVE, SUSPENDED, DELISTED, ALL (default: ACTIVE)
	IsTradable *bool   // true, false, nil (all)
	Search     string  // 종목명 또는 코드 검색 (부분 일치)
	Sort       string  // symbol, name, market_cap (default: symbol)
	Order      string  // asc, desc (default: asc)
	Page       int     // 페이지 번호 (1부터 시작)
	Limit      int     // 페이지 크기 (기본: 20, 최대: 100)
}

// ListResult represents paginated list result
type ListResult struct {
	Stocks     []Stock
	TotalCount int
	Page       int
	Limit      int
}

// ValidateSymbol validates stock symbol format (6-digit number)
func ValidateSymbol(symbol string) bool {
	if len(symbol) != 6 {
		return false
	}
	for _, c := range symbol {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ValidateMarket validates market value
func ValidateMarket(market string) bool {
	return market == "KOSPI" || market == "KOSDAQ" || market == "KONEX" || market == "ETF"
}

// ValidateStatus validates status value
func ValidateStatus(status string) bool {
	return status == "ACTIVE" || status == "SUSPENDED" || status == "DELISTED" || status == "ALL"
}

// ValidateSort validates sort field
func ValidateSort(sort string) bool {
	return sort == "symbol" || sort == "name" || sort == "market_cap"
}

// ValidateOrder validates order direction
func ValidateOrder(order string) bool {
	return order == "asc" || order == "desc"
}

// Normalize normalizes and validates ListFilter
func (f *ListFilter) Normalize() error {
	// Default status
	if f.Status == "" {
		f.Status = "ACTIVE"
	}

	// Validate status
	if !ValidateStatus(f.Status) {
		return ErrInvalidStatus
	}

	// Validate market if provided
	if f.Market != nil && !ValidateMarket(*f.Market) {
		return ErrInvalidMarket
	}

	// Default sort
	if f.Sort == "" {
		f.Sort = "symbol"
	}
	if !ValidateSort(f.Sort) {
		return ErrInvalidSort
	}

	// Default order
	if f.Order == "" {
		f.Order = "asc"
	}
	if !ValidateOrder(f.Order) {
		return ErrInvalidOrder
	}

	// Normalize page (minimum 1)
	if f.Page < 1 {
		f.Page = 1
	}

	// Normalize limit (default 20, min 1, max 100)
	if f.Limit < 1 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}

	return nil
}
