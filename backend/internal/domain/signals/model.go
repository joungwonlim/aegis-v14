package signals

import (
	"time"

	"github.com/google/uuid"
)

// Signal 매매 신호
type Signal struct {
	SignalID    uuid.UUID       `json:"signal_id"`
	SnapshotID  string          `json:"snapshot_id"` // Universe Snapshot ID
	GeneratedAt time.Time       `json:"generated_at"`

	// 종목 정보
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Market string `json:"market"` // KOSPI, KOSDAQ

	// 신호
	SignalType SignalType `json:"signal_type"` // BUY, SELL, HOLD
	Strength   int        `json:"strength"`    // 0-100 (신호 강도)
	Conviction int        `json:"conviction"`  // 0-100 (신뢰도)

	// 팩터 점수
	Factors SignalBreakdown `json:"factors"`

	// 순위
	Rank int `json:"rank"`

	// 근거
	Reasons []string `json:"reasons"` // 신호 생성 근거
}

// SignalType 신호 타입
type SignalType string

const (
	SignalBuy  SignalType = "BUY"
	SignalSell SignalType = "SELL"
	SignalHold SignalType = "HOLD"
)

// SignalBreakdown 팩터별 점수 분해
type SignalBreakdown struct {
	Momentum  FactorScore `json:"momentum"`  // 모멘텀 팩터
	Quality   FactorScore `json:"quality"`   // 품질 팩터
	Value     FactorScore `json:"value"`     // 가치 팩터
	Technical FactorScore `json:"technical"` // 기술적 팩터
}

// FactorScore 개별 팩터 점수
type FactorScore struct {
	Score      float64  `json:"score"`      // 0-100
	Weight     float64  `json:"weight"`     // 가중치
	Triggered  bool     `json:"triggered"`  // 트리거 여부
	Indicators []string `json:"indicators"` // 사용된 지표
}

// SignalSnapshot 특정 시점의 전체 신호 스냅샷
type SignalSnapshot struct {
	SnapshotID  string    `json:"snapshot_id"`  // YYYYMMDD-HHMM
	UniverseID  string    `json:"universe_id"`  // 연관 Universe Snapshot
	GeneratedAt time.Time `json:"generated_at"`

	TotalCount  int      `json:"total_count"`
	BuySignals  []Signal `json:"buy_signals"`
	SellSignals []Signal `json:"sell_signals"`

	// 통계
	Stats SignalStats `json:"stats"`
}

// SignalStats 신호 통계
type SignalStats struct {
	AvgStrength   float64 `json:"avg_strength"`
	AvgConviction float64 `json:"avg_conviction"`
	BuyCount      int     `json:"buy_count"`
	SellCount     int     `json:"sell_count"`
	HoldCount     int     `json:"hold_count"`
}

// SignalCriteria 신호 생성 기준
type SignalCriteria struct {
	// 팩터 가중치 (합계 1.0)
	MomentumWeight  float64 `json:"momentum_weight"`  // 0.35
	QualityWeight   float64 `json:"quality_weight"`   // 0.25
	ValueWeight     float64 `json:"value_weight"`     // 0.20
	TechnicalWeight float64 `json:"technical_weight"` // 0.20

	// 신호 임계값
	BuyThreshold  int `json:"buy_threshold"`  // 65 (65점 이상 BUY)
	SellThreshold int `json:"sell_threshold"` // 35 (35점 이하 SELL)

	// 필터링
	MinConviction int `json:"min_conviction"` // 50 (최소 신뢰도)
	MaxSignals    int `json:"max_signals"`    // 30 (최대 신호 수)
}

// DefaultSignalCriteria 기본 기준
func DefaultSignalCriteria() *SignalCriteria {
	return &SignalCriteria{
		MomentumWeight:  0.35,
		QualityWeight:   0.25,
		ValueWeight:     0.20,
		TechnicalWeight: 0.20,
		BuyThreshold:    65,
		SellThreshold:   35,
		MinConviction:   50,
		MaxSignals:      30,
	}
}

// MomentumFactors 모멘텀 팩터
type MomentumFactors struct {
	Symbol           string  `json:"symbol"`
	Return5D         float64 `json:"return_5d"`         // 5일 수익률
	Return20D        float64 `json:"return_20d"`        // 20일 수익률
	Return60D        float64 `json:"return_60d"`        // 60일 수익률
	RelativeStrength float64 `json:"relative_strength"` // 상대강도
	VolumeGrowth     float64 `json:"volume_growth"`     // 거래량 증가율
}

// QualityFactors 품질 팩터
type QualityFactors struct {
	Symbol       string  `json:"symbol"`
	ROE          float64 `json:"roe"`           // 자기자본이익률
	ROA          float64 `json:"roa"`           // 총자산이익률
	DebtRatio    float64 `json:"debt_ratio"`    // 부채비율
	CurrentRatio float64 `json:"current_ratio"` // 유동비율
}

// ValueFactors 가치 팩터
type ValueFactors struct {
	Symbol        string  `json:"symbol"`
	PER           float64 `json:"per"`            // 주가수익비율
	PBR           float64 `json:"pbr"`            // 주가순자산비율
	PSR           float64 `json:"psr"`            // 주가매출비율
	DividendYield float64 `json:"dividend_yield"` // 배당수익률
}

// TechnicalFactors 기술적 팩터
type TechnicalFactors struct {
	Symbol       string  `json:"symbol"`
	RSI          float64 `json:"rsi"`           // 상대강도지수
	MACD         float64 `json:"macd"`          // MACD
	MACDSignal   float64 `json:"macd_signal"`   // MACD 시그널
	BollingerPos float64 `json:"bollinger_pos"` // 볼린저밴드 위치 (0-1)
}
