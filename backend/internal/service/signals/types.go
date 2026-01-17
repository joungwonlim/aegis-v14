package signals

import "time"

// PricePoint 가격 데이터 포인트
type PricePoint struct {
	Date   time.Time
	Price  int64 // 종가 (원)
	Volume int64 // 거래량
}

// FlowData 수급 데이터 포인트
type FlowData struct {
	Date          string // YYYY-MM-DD
	ForeignNet    int64  // 외국인 순매수
	InstNet       int64  // 기관 순매수
	IndividualNet int64  // 개인 순매수
}

// ValueMetrics 가치 지표
type ValueMetrics struct {
	PER float64 // 주가수익비율
	PBR float64 // 주가순자산비율
	PSR float64 // 주가매출비율
}

// QualityMetrics 품질 지표
type QualityMetrics struct {
	ROE       float64 // 자기자본이익률 (%)
	DebtRatio float64 // 부채비율 (%)
}

// MomentumDetails 모멘텀 상세
type MomentumDetails struct {
	Return1M   float64 // 1개월 수익률
	Return3M   float64 // 3개월 수익률
	VolumeRate float64 // 거래량 성장률
}

// TechnicalDetails 기술적 지표 상세
type TechnicalDetails struct {
	RSI       float64 // 상대강도지수
	MACD      float64 // MACD
	MA20Cross int     // MA20 크로스 (-1, 0, 1)
}

// ValueDetails 가치 상세
type ValueDetails struct {
	PER float64
	PBR float64
	PSR float64
}

// QualityDetails 품질 상세
type QualityDetails struct {
	ROE       float64
	DebtRatio float64
}

// FlowDetails 수급 상세
type FlowDetails struct {
	ForeignNet5D  int64
	ForeignNet20D int64
	InstNet5D     int64
	InstNet20D    int64
}
