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

// SignalBreakdown 팩터별 점수 분해 (6팩터)
type SignalBreakdown struct {
	Momentum  FactorScore `json:"momentum"`  // 모멘텀 팩터
	Quality   FactorScore `json:"quality"`   // 품질 팩터
	Value     FactorScore `json:"value"`     // 가치 팩터
	Technical FactorScore `json:"technical"` // 기술적 팩터
	Flow      FactorScore `json:"flow"`      // 수급 팩터
	Event     FactorScore `json:"event"`     // 이벤트 팩터
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

// SignalCriteria 신호 생성 기준 (6팩터)
type SignalCriteria struct {
	// 팩터 가중치 (합계 1.0) - v13 설계 기준
	MomentumWeight  float64 `json:"momentum_weight"`  // 0.20
	TechnicalWeight float64 `json:"technical_weight"` // 0.15
	ValueWeight     float64 `json:"value_weight"`     // 0.20
	QualityWeight   float64 `json:"quality_weight"`   // 0.15
	FlowWeight      float64 `json:"flow_weight"`      // 0.20
	EventWeight     float64 `json:"event_weight"`     // 0.10

	// 신호 임계값
	BuyThreshold  int `json:"buy_threshold"`  // 65 (65점 이상 BUY)
	SellThreshold int `json:"sell_threshold"` // 35 (35점 이하 SELL)

	// 필터링
	MinConviction int `json:"min_conviction"` // 50 (최소 신뢰도)
	MaxSignals    int `json:"max_signals"`    // 30 (최대 신호 수)
}

// DefaultSignalCriteria 기본 기준 (6팩터 가중치)
func DefaultSignalCriteria() *SignalCriteria {
	return &SignalCriteria{
		MomentumWeight:  0.20, // 추세 추종
		TechnicalWeight: 0.15, // 기술적 분석
		ValueWeight:     0.20, // 가치 평가
		QualityWeight:   0.15, // 기업 품질
		FlowWeight:      0.20, // 수급 분석
		EventWeight:     0.10, // 이벤트 영향
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
	MA20Cross    int     `json:"ma20_cross"`    // MA20 크로스 (-1: 하락, 0: 중립, 1: 상승)
}

// FlowFactors 수급 팩터
type FlowFactors struct {
	Symbol        string `json:"symbol"`
	ForeignNet5D  int64  `json:"foreign_net_5d"`  // 외국인 5일 순매수
	ForeignNet20D int64  `json:"foreign_net_20d"` // 외국인 20일 순매수
	InstNet5D     int64  `json:"inst_net_5d"`     // 기관 5일 순매수
	InstNet20D    int64  `json:"inst_net_20d"`    // 기관 20일 순매수
	IndivNet5D    int64  `json:"indiv_net_5d"`    // 개인 5일 순매수 (역지표)
	IndivNet20D   int64  `json:"indiv_net_20d"`   // 개인 20일 순매수 (역지표)
}

// EventFactors 이벤트 팩터
type EventFactors struct {
	Symbol       string        `json:"symbol"`
	Events       []EventSignal `json:"events"`        // 최근 이벤트 목록
	TotalScore   float64       `json:"total_score"`   // 시간 가중 합산 점수
	EventCount   int           `json:"event_count"`   // 이벤트 개수
	LastEventAt  *time.Time    `json:"last_event_at"` // 마지막 이벤트 시간
}

// EventSignal 개별 이벤트 신호
type EventSignal struct {
	Type      EventType `json:"type"`       // 이벤트 유형
	Score     float64   `json:"score"`      // 영향도 (-1.0 ~ 1.0)
	Title     string    `json:"title"`      // 이벤트 제목
	Source    string    `json:"source"`     // 출처 (DART, KIS, Naver 등)
	Timestamp time.Time `json:"timestamp"`  // 발생 시간
}

// EventType 이벤트 유형
type EventType string

const (
	// 긍정적 이벤트
	EventEarningsPositive EventType = "earnings_positive"  // 실적 개선
	EventMergerPositive   EventType = "merger_positive"    // 인수합병 긍정
	EventShareBuyback     EventType = "share_buyback"      // 자사주 매입
	EventNewProduct       EventType = "new_product"        // 신제품 출시
	EventDividendIncrease EventType = "dividend_increase"  // 배당 증가
	EventPartnership      EventType = "partnership"        // 파트너십 체결
	EventCapexIncrease    EventType = "capex_increase"     // 설비 투자
	EventPatent           EventType = "patent"             // 특허 취득

	// 부정적 이벤트
	EventEarningsNegative EventType = "earnings_negative"  // 실적 악화
	EventAuditOpinion     EventType = "audit_opinion"      // 감사 의견
	EventMergerNegative   EventType = "merger_negative"    // 인수합병 부정
	EventRecall           EventType = "recall"             // 제품 리콜
	EventLawsuit          EventType = "lawsuit"            // 소송
	EventRegulatory       EventType = "regulatory"         // 규제 이슈
	EventDividendDecrease EventType = "dividend_decrease"  // 배당 감소
	EventManagementChange EventType = "management_change"  // 경영진 교체

	// 중립 이벤트
	EventGeneralNews   EventType = "general_news"   // 일반 뉴스
	EventAnnouncement  EventType = "announcement"   // 일반 공시
)

// GetEventImpact 이벤트 유형별 영향도 반환
func GetEventImpact(eventType EventType) float64 {
	impacts := map[EventType]float64{
		// 긍정적 이벤트
		EventEarningsPositive: 1.0,
		EventMergerPositive:   0.9,
		EventShareBuyback:     0.8,
		EventNewProduct:       0.7,
		EventDividendIncrease: 0.6,
		EventPartnership:      0.6,
		EventCapexIncrease:    0.5,
		EventPatent:           0.5,

		// 부정적 이벤트
		EventEarningsNegative: -1.0,
		EventAuditOpinion:     -0.9,
		EventMergerNegative:   -0.8,
		EventRecall:           -0.8,
		EventLawsuit:          -0.7,
		EventRegulatory:       -0.7,
		EventDividendDecrease: -0.6,
		EventManagementChange: -0.5,

		// 중립 이벤트
		EventGeneralNews:  0.0,
		EventAnnouncement: 0.0,
	}

	if impact, ok := impacts[eventType]; ok {
		return impact
	}
	return 0.0
}
