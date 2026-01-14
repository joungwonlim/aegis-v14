package universe

import "time"

// UniverseSnapshot represents a universe snapshot
type UniverseSnapshot struct {
	SnapshotID  string           `json:"snapshot_id"`  // YYYYMMDD-HHMM
	GeneratedAt time.Time        `json:"generated_at"` // 생성 시각
	TotalCount  int              `json:"total_count"`  // 전체 종목 수
	Holdings    []UniverseStock  `json:"holdings"`     // 보유종목
	Watchlist   []UniverseStock  `json:"watchlist"`    // 관심종목
	Rankings    RankingBreakdown `json:"rankings"`     // 랭킹 breakdown
	FilterStats FilterStats      `json:"filter_stats"` // 필터링 통계
}

// UniverseStock represents a stock in the universe
type UniverseStock struct {
	Symbol      string `json:"symbol"`        // 종목 코드
	Name        string `json:"name"`          // 종목명
	Market      string `json:"market"`        // KOSPI | KOSDAQ
	Sector      string `json:"sector"`        // 섹터
	Tier        string `json:"tier"`          // HOLDING | WATCHLIST | RANKING
	Source      string `json:"source"`        // 출처 (holding, watchlist, quantHigh, priceTop, ...)
	MarketCap   int64  `json:"market_cap"`    // 시가총액 (원)
	AvgVolume5D int64  `json:"avg_volume_5d"` // 5일 평균 거래량
	AvgValue5D  int64  `json:"avg_value_5d"`  // 5일 평균 거래대금 (원)
	IsActive    bool   `json:"is_active"`     // 활성 여부
}

// RankingBreakdown represents ranking data breakdown
type RankingBreakdown struct {
	QuantHigh      RankingData `json:"quant_high"`      // 거래량급증
	PriceTop       RankingData `json:"price_top"`       // 거래대금
	Upper          RankingData `json:"upper"`           // 상승
	Top            RankingData `json:"top"`             // 인기검색
	Capitalization RankingData `json:"capitalization"`  // 시가총액
}

// RankingData represents ranking data for a specific category
type RankingData struct {
	Kospi  []UniverseStock `json:"kospi"`  // KOSPI 종목
	Kosdaq []UniverseStock `json:"kosdaq"` // KOSDAQ 종목
}

// FilterStats represents filtering statistics
type FilterStats struct {
	TotalCandidates int `json:"total_candidates"` // 전체 후보
	AfterLiquidity  int `json:"after_liquidity"`  // 유동성 필터 후
	AfterMarketCap  int `json:"after_market_cap"` // 시총 필터 후
	AfterVolume     int `json:"after_volume"`     // 거래량 필터 후
	AfterExclusions int `json:"after_exclusions"` // 제외 대상 필터 후
	Final           int `json:"final"`            // 최종 (중복 제거)
}

// FilterCriteria represents universe filtering criteria
type FilterCriteria struct {
	MinMarketCap     int64 `json:"min_market_cap"`      // 최소 시가총액 (기본: 100억)
	MinAvgValue5D    int64 `json:"min_avg_value_5d"`    // 최소 5일 평균 거래대금 (기본: 10억)
	MinAvgVolume5D   int64 `json:"min_avg_volume_5d"`   // 최소 5일 평균 거래량 (기본: 10만주)
	ExcludeManaged   bool  `json:"exclude_managed"`     // 관리종목 제외 (기본: true)
	ExcludeSuspended bool  `json:"exclude_suspended"`   // 거래정지 제외 (기본: true)
	RankingLimit     int   `json:"ranking_limit"`       // 랭킹당 종목 수 (기본: 100)
}

// DefaultFilterCriteria returns default filtering criteria
func DefaultFilterCriteria() *FilterCriteria {
	return &FilterCriteria{
		MinMarketCap:     10_000_000_000, // 100억원
		MinAvgValue5D:    1_000_000_000,  // 10억원
		MinAvgVolume5D:   100_000,        // 10만주
		ExcludeManaged:   true,
		ExcludeSuspended: true,
		RankingLimit:     100,
	}
}

// Ranking Categories
const (
	CategoryQuantHigh      = "quantHigh"      // 거래량급증
	CategoryPriceTop       = "priceTop"       // 거래대금
	CategoryUpper          = "upper"          // 상승
	CategoryTop            = "top"            // 인기검색
	CategoryCapitalization = "capitalization" // 시가총액
)

// Tiers
const (
	TierHolding   = "HOLDING"   // 보유종목 (Tier 1)
	TierWatchlist = "WATCHLIST" // 관심종목 (Tier 2)
	TierRanking   = "RANKING"   // 랭킹 (Tier 3)
)
