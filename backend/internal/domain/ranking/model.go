package ranking

import (
	"time"

	"github.com/google/uuid"
)

// RankedStock 순위가 매겨진 종목
type RankedStock struct {
	RankID      uuid.UUID `json:"rank_id"`
	SnapshotID  string    `json:"snapshot_id"` // Ranking Snapshot ID
	SignalID    uuid.UUID `json:"signal_id"`   // 연관 Signal ID
	GeneratedAt time.Time `json:"generated_at"`

	// 종목 정보
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Market string `json:"market"`
	Sector string `json:"sector"`

	// 순위
	Rank     int  `json:"rank"`     // 최종 순위 (1-based)
	Selected bool `json:"selected"` // 선정 여부

	// 종합 점수
	TotalScore float64 `json:"total_score"` // 0-100 (최종 점수)
	AlphaScore float64 `json:"alpha_score"` // Signals에서 받은 점수
	RiskScore  float64 `json:"risk_score"`  // 리스크 조정 점수
	Adjustment float64 `json:"adjustment"`  // 다양성 조정값

	// 점수 분해
	Breakdown RankingBreakdown `json:"breakdown"`

	// 선정/탈락 사유
	Reason string `json:"reason"`
}

// RankingBreakdown 점수 분해
type RankingBreakdown struct {
	// Signals 팩터 (from Signal)
	SignalStrength   float64 `json:"signal_strength"`   // 신호 강도
	SignalConviction float64 `json:"signal_conviction"` // 신호 신뢰도

	// 리스크 조정
	VolatilityRisk    float64 `json:"volatility_risk"`    // 변동성 리스크
	LiquidityRisk     float64 `json:"liquidity_risk"`     // 유동성 리스크
	ConcentrationRisk float64 `json:"concentration_risk"` // 집중도 리스크

	// 다양성 조정
	SectorPenalty      float64 `json:"sector_penalty"`      // 섹터 집중 페널티
	CorrelationPenalty float64 `json:"correlation_penalty"` // 상관관계 페널티
}

// RankingSnapshot 특정 시점의 전체 순위 스냅샷
type RankingSnapshot struct {
	SnapshotID    string    `json:"snapshot_id"`    // YYYYMMDD-HHMM
	SignalID      string    `json:"signal_id"`      // 연관 Signal Snapshot
	GeneratedAt   time.Time `json:"generated_at"`
	TotalCount    int       `json:"total_count"`    // 평가된 총 종목 수
	SelectedCount int       `json:"selected_count"` // 선정된 종목 수

	Rankings []RankedStock `json:"rankings"` // 순위 목록

	// 통계
	Stats RankingStats `json:"stats"`
}

// RankingStats 순위 통계
type RankingStats struct {
	AvgTotalScore      float64        `json:"avg_total_score"`
	AvgRiskScore       float64        `json:"avg_risk_score"`
	SectorDistribution map[string]int `json:"sector_distribution"` // 섹터별 선정 수
	MarketDistribution map[string]int `json:"market_distribution"` // 시장별 선정 수
}

// RankingCriteria 순위 산정 기준
type RankingCriteria struct {
	// 점수 계산
	AlphaWeight float64 `json:"alpha_weight"` // 0.70 (신호 강도)
	RiskWeight  float64 `json:"risk_weight"`  // 0.30 (리스크)

	// 리스크 임계값
	MaxVolatility float64 `json:"max_volatility"` // 60% (최대 변동성)
	MinLiquidity  float64 `json:"min_liquidity"`  // 10억 (최소 일평균 거래대금)

	// 다양성 제약
	MaxPerSector  int     `json:"max_per_sector"`  // 5 (섹터당 최대)
	MaxPerMarket  int     `json:"max_per_market"`  // 15 (시장당 최대)
	SectorPenalty float64 `json:"sector_penalty"`  // 5.0 (섹터 중복 페널티)

	// 선정 기준
	MinTotalScore float64 `json:"min_total_score"` // 60.0 (최소 종합 점수)
	MaxSelections int     `json:"max_selections"`  // 20 (최대 선정 수)
}

// DefaultRankingCriteria 기본 기준
func DefaultRankingCriteria() *RankingCriteria {
	return &RankingCriteria{
		AlphaWeight:   0.70,
		RiskWeight:    0.30,
		MaxVolatility: 60.0,
		MinLiquidity:  1_000_000_000, // 10억
		MaxPerSector:  5,
		MaxPerMarket:  15,
		SectorPenalty: 5.0,
		MinTotalScore: 60.0,
		MaxSelections: 20,
	}
}

// RiskData 리스크 데이터
type RiskData struct {
	Symbol string `json:"symbol"`

	// 변동성
	Volatility60D float64 `json:"volatility_60d"` // 60일 변동성 (%)

	// 유동성
	AvgDailyValue float64 `json:"avg_daily_value"` // 일평균 거래대금 (원)

	// 집중도
	MarketCap       float64 `json:"market_cap"`        // 시가총액
	FreeFloatRatio  float64 `json:"free_float_ratio"`  // 유동주식비율
	ForeignOwnership float64 `json:"foreign_ownership"` // 외국인 보유비율
}
