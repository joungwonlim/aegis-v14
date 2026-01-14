package price

import (
	"time"
)

// Source represents price data source
type Source string

const (
	SourceKISWebSocket Source = "KIS_WS"   // KIS WebSocket (실시간, 최우선)
	SourceKISREST      Source = "KIS_REST" // KIS REST API (폴링)
	SourceNaver        Source = "NAVER"    // Naver 증권 (fallback)
)

// IsValid checks if source is valid
func (s Source) IsValid() bool {
	switch s {
	case SourceKISWebSocket, SourceKISREST, SourceNaver:
		return true
	default:
		return false
	}
}

// Priority returns source priority (higher is better)
func (s Source) Priority() int {
	switch s {
	case SourceKISWebSocket:
		return 3
	case SourceKISREST:
		return 2
	case SourceNaver:
		return 1
	default:
		return 0
	}
}

// Tick represents a single price tick from a source
// Maps to market.prices_ticks table
type Tick struct {
	ID     int64  `json:"id" db:"id"`
	Symbol string `json:"symbol" db:"symbol"`
	Source Source `json:"source" db:"source"`

	// 가격 정보
	LastPrice   int64   `json:"last_price" db:"last_price"`     // 현재가 (원)
	ChangePrice *int64  `json:"change_price" db:"change_price"` // 전일대비
	ChangeRate  *float64 `json:"change_rate" db:"change_rate"`  // 등락률 (%)
	Volume      *int64  `json:"volume" db:"volume"`             // 거래량

	// 호가 정보 (선택)
	BidPrice  *int64 `json:"bid_price" db:"bid_price"`
	AskPrice  *int64 `json:"ask_price" db:"ask_price"`
	BidVolume *int64 `json:"bid_volume" db:"bid_volume"`
	AskVolume *int64 `json:"ask_volume" db:"ask_volume"`

	// 메타데이터
	TS        time.Time `json:"ts" db:"ts"`               // 수신 시각
	CreatedTS time.Time `json:"created_ts" db:"created_ts"` // 저장 시각
}

// BestPrice represents the best price for a symbol
// Maps to market.prices_best table
type BestPrice struct {
	Symbol string `json:"symbol" db:"symbol"`

	// Best Price (신선도 기준 최적 소스)
	BestPrice  int64     `json:"best_price" db:"best_price"`
	BestSource Source    `json:"best_source" db:"best_source"`
	BestTS     time.Time `json:"best_ts" db:"best_ts"`

	// 추가 정보
	ChangePrice *int64   `json:"change_price" db:"change_price"`
	ChangeRate  *float64 `json:"change_rate" db:"change_rate"`
	Volume      *int64   `json:"volume" db:"volume"`

	BidPrice *int64 `json:"bid_price" db:"bid_price"`
	AskPrice *int64 `json:"ask_price" db:"ask_price"`

	// 상태
	IsStale bool `json:"is_stale" db:"is_stale"` // 모든 소스가 stale 시 true

	// 메타데이터
	UpdatedTS time.Time `json:"updated_ts" db:"updated_ts"`
}

// Freshness represents freshness tracking per symbol per source
// Maps to market.freshness table
type Freshness struct {
	Symbol string `json:"symbol" db:"symbol"`
	Source Source `json:"source" db:"source"`

	// 최근 수신 정보
	LastTS    *time.Time `json:"last_ts" db:"last_ts"`       // 마지막 수신 시각
	LastPrice *int64     `json:"last_price" db:"last_price"` // 마지막 가격

	// 신선도 판정
	IsStale      bool   `json:"is_stale" db:"is_stale"`           // stale 여부
	StalenessMS  *int64 `json:"staleness_ms" db:"staleness_ms"`   // 현재 - last_ts (ms)
	QualityScore *int   `json:"quality_score" db:"quality_score"` // 0~100 점수

	// 메타데이터
	UpdatedTS time.Time `json:"updated_ts" db:"updated_ts"`
}

// FreshnessThreshold represents staleness threshold in milliseconds
type FreshnessThreshold struct {
	Source            Source
	InTradingMS       int64 // 장중 임계값 (ms)
	OutOfTradingMS    int64 // 장전/장후 임계값 (ms)
}

// DefaultFreshnessThresholds returns default freshness thresholds
func DefaultFreshnessThresholds() []FreshnessThreshold {
	return []FreshnessThreshold{
		{
			Source:         SourceKISWebSocket,
			InTradingMS:    2000,  // 2초
			OutOfTradingMS: 10000, // 10초
		},
		{
			Source:         SourceKISREST,
			InTradingMS:    10000, // 10초
			OutOfTradingMS: 30000, // 30초
		},
		{
			Source:         SourceNaver,
			InTradingMS:    30000, // 30초
			OutOfTradingMS: 60000, // 60초
		},
	}
}

// GetThreshold returns threshold for given source and trading state
func GetThreshold(source Source, isTrading bool) int64 {
	thresholds := DefaultFreshnessThresholds()
	for _, t := range thresholds {
		if t.Source == source {
			if isTrading {
				return t.InTradingMS
			}
			return t.OutOfTradingMS
		}
	}
	// Default fallback
	if isTrading {
		return 10000
	}
	return 30000
}

// CalculateStaleness calculates staleness in milliseconds
func CalculateStaleness(lastTS time.Time, now time.Time) int64 {
	return now.Sub(lastTS).Milliseconds()
}

// IsStale checks if data is stale based on threshold
func IsStale(lastTS time.Time, now time.Time, thresholdMS int64) bool {
	staleness := CalculateStaleness(lastTS, now)
	return staleness > thresholdMS
}

// CalculateQualityScore calculates quality score (0~100)
// Higher is better
func CalculateQualityScore(source Source, staleness int64, thresholdMS int64) int {
	// Base score by source priority
	baseScore := source.Priority() * 30 // WS=90, REST=60, NAVER=30

	// Penalty for staleness
	if staleness >= thresholdMS {
		// Stale: 0 score
		return 0
	}

	// Fresh: reduce score based on staleness ratio
	stalenessRatio := float64(staleness) / float64(thresholdMS)
	penalty := int(stalenessRatio * float64(baseScore))

	score := baseScore - penalty
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

// SelectBestSource selects best source based on freshness
// Returns the source with highest quality score
func SelectBestSource(freshnesses []Freshness) (Source, bool) {
	var bestSource Source
	bestScore := -1

	for _, f := range freshnesses {
		if f.QualityScore == nil {
			continue
		}
		if *f.QualityScore > bestScore {
			bestScore = *f.QualityScore
			bestSource = f.Source
		}
	}

	if bestScore < 0 {
		return "", false
	}

	return bestSource, true
}

// CreateTickFromBestPrice creates a Tick from BestPrice
func CreateTickFromBestPrice(bp BestPrice) Tick {
	return Tick{
		Symbol:      bp.Symbol,
		Source:      bp.BestSource,
		LastPrice:   bp.BestPrice,
		ChangePrice: bp.ChangePrice,
		ChangeRate:  bp.ChangeRate,
		Volume:      bp.Volume,
		BidPrice:    bp.BidPrice,
		AskPrice:    bp.AskPrice,
		TS:          bp.BestTS,
		CreatedTS:   bp.UpdatedTS,
	}
}

// UpsertBestPriceInput represents input for upserting best price
type UpsertBestPriceInput struct {
	Symbol      string
	BestPrice   int64
	BestSource  Source
	BestTS      time.Time
	ChangePrice *int64
	ChangeRate  *float64
	Volume      *int64
	BidPrice    *int64
	AskPrice    *int64
	IsStale     bool
}

// UpsertFreshnessInput represents input for upserting freshness
type UpsertFreshnessInput struct {
	Symbol       string
	Source       Source
	LastTS       time.Time
	LastPrice    int64
	IsStale      bool
	StalenessMS  int64
	QualityScore int
}
