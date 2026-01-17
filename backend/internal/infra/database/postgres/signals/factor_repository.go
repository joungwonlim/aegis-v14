package signals

import (
	"context"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wonny/aegis/v14/internal/domain/signals"
)

// FactorRepository 6팩터 데이터 리포지토리
// Fetcher 모듈 데이터를 기반으로 팩터 데이터 조회
type FactorRepository struct {
	pool *pgxpool.Pool
}

// NewFactorRepository 새 리포지토리 생성
func NewFactorRepository(pool *pgxpool.Pool) *FactorRepository {
	return &FactorRepository{pool: pool}
}

// GetMomentumFactors 모멘텀 팩터 조회
// fetcher.daily_prices 데이터 기반
func (r *FactorRepository) GetMomentumFactors(ctx context.Context, symbol string) (*signals.MomentumFactors, error) {
	// 최근 60일 가격 데이터 조회
	query := `
		SELECT date, close_price, volume
		FROM fetcher.daily_prices
		WHERE stock_code = $1
		ORDER BY date DESC
		LIMIT 70
	`

	rows, err := r.pool.Query(ctx, query, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := make([]priceDataLocal, 0, 70)
	for rows.Next() {
		var p priceDataLocal
		if err := rows.Scan(&p.date, &p.close, &p.volume); err != nil {
			continue
		}
		prices = append(prices, p)
	}

	if len(prices) < 5 {
		return &signals.MomentumFactors{Symbol: symbol}, nil
	}

	factors := &signals.MomentumFactors{Symbol: symbol}

	// 수익률 계산
	if len(prices) >= 6 {
		factors.Return5D = calculateReturn(prices[0].close, prices[5].close)
	}
	if len(prices) >= 21 {
		factors.Return20D = calculateReturn(prices[0].close, prices[20].close)
	}
	if len(prices) >= 61 {
		factors.Return60D = calculateReturn(prices[0].close, prices[60].close)
	}

	// 거래량 성장률
	if len(prices) >= 40 {
		recentVol := avgVolumeFromPriceData(prices[:20])
		pastVol := avgVolumeFromPriceData(prices[20:40])
		if pastVol > 0 {
			factors.VolumeGrowth = (recentVol - pastVol) / pastVol
		}
	}

	return factors, nil
}

// GetQualityFactors 품질 팩터 조회
// fetcher.fundamentals 데이터 기반
func (r *FactorRepository) GetQualityFactors(ctx context.Context, symbol string) (*signals.QualityFactors, error) {
	query := `
		SELECT roe, debt_ratio, current_ratio
		FROM fetcher.fundamentals
		WHERE stock_code = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`

	factors := &signals.QualityFactors{Symbol: symbol}

	var roe, debtRatio, currentRatio *float64
	err := r.pool.QueryRow(ctx, query, symbol).Scan(&roe, &debtRatio, &currentRatio)
	if err != nil {
		if err == pgx.ErrNoRows {
			return factors, nil
		}
		return nil, err
	}

	if roe != nil {
		factors.ROE = *roe
	}
	if debtRatio != nil {
		factors.DebtRatio = *debtRatio
	}
	if currentRatio != nil {
		factors.CurrentRatio = *currentRatio
	}

	return factors, nil
}

// GetValueFactors 가치 팩터 조회
// fetcher.fundamentals 데이터 기반
func (r *FactorRepository) GetValueFactors(ctx context.Context, symbol string) (*signals.ValueFactors, error) {
	query := `
		SELECT per, pbr, dividend_yield
		FROM fetcher.fundamentals
		WHERE stock_code = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`

	factors := &signals.ValueFactors{Symbol: symbol}

	var per, pbr, dividendYield *float64
	err := r.pool.QueryRow(ctx, query, symbol).Scan(&per, &pbr, &dividendYield)
	if err != nil {
		if err == pgx.ErrNoRows {
			return factors, nil
		}
		return nil, err
	}

	if per != nil {
		factors.PER = *per
	}
	if pbr != nil {
		factors.PBR = *pbr
	}
	if dividendYield != nil {
		factors.DividendYield = *dividendYield
	}

	return factors, nil
}

// GetTechnicalFactors 기술적 팩터 조회
// fetcher.daily_prices 데이터 기반으로 계산
func (r *FactorRepository) GetTechnicalFactors(ctx context.Context, symbol string) (*signals.TechnicalFactors, error) {
	// 최근 120일 가격 데이터 조회
	query := `
		SELECT date, close_price
		FROM fetcher.daily_prices
		WHERE stock_code = $1
		ORDER BY date DESC
		LIMIT 130
	`

	rows, err := r.pool.Query(ctx, query, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := make([]int64, 0, 130)
	for rows.Next() {
		var date time.Time
		var close int64
		if err := rows.Scan(&date, &close); err != nil {
			continue
		}
		prices = append(prices, close)
	}

	factors := &signals.TechnicalFactors{Symbol: symbol}

	if len(prices) < 26 {
		return factors, nil
	}

	// RSI 계산 (14일)
	if len(prices) >= 15 {
		factors.RSI = calculateRSI(prices[:15])
	}

	// MACD 계산
	if len(prices) >= 26 {
		ema12 := calculateEMA(prices, 12)
		ema26 := calculateEMA(prices, 26)
		factors.MACD = ema12 - ema26
		factors.MACDSignal = factors.MACD // 단순화
	}

	// MA20 크로스
	if len(prices) >= 20 {
		ma20 := avgPrice(prices[:20])
		currentPrice := float64(prices[0])
		priceDiff := (currentPrice - ma20) / ma20
		if priceDiff > 0.02 {
			factors.MA20Cross = 1
		} else if priceDiff < -0.02 {
			factors.MA20Cross = -1
		}
	}

	// 볼린저밴드 위치
	if len(prices) >= 20 {
		ma20 := avgPrice(prices[:20])
		stdDev := calculateStdDev(prices[:20], ma20)
		upper := ma20 + 2*stdDev
		lower := ma20 - 2*stdDev
		currentPrice := float64(prices[0])
		if upper != lower {
			factors.BollingerPos = (currentPrice - lower) / (upper - lower)
		}
	}

	return factors, nil
}

// GetFlowFactors 수급 팩터 조회
// fetcher.investor_flows 데이터 기반
func (r *FactorRepository) GetFlowFactors(ctx context.Context, symbol string) (*signals.FlowFactors, error) {
	query := `
		SELECT date, foreign_net, institution_net, individual_net
		FROM fetcher.investor_flows
		WHERE stock_code = $1
		ORDER BY date DESC
		LIMIT 25
	`

	rows, err := r.pool.Query(ctx, query, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type flowData struct {
		foreign int64
		inst    int64
		indiv   int64
	}

	flows := make([]flowData, 0, 25)
	for rows.Next() {
		var date time.Time
		var f flowData
		if err := rows.Scan(&date, &f.foreign, &f.inst, &f.indiv); err != nil {
			continue
		}
		flows = append(flows, f)
	}

	factors := &signals.FlowFactors{Symbol: symbol}

	// 5일 순매수 합계
	for i := 0; i < min(5, len(flows)); i++ {
		factors.ForeignNet5D += flows[i].foreign
		factors.InstNet5D += flows[i].inst
		factors.IndivNet5D += flows[i].indiv
	}

	// 20일 순매수 합계
	for i := 0; i < min(20, len(flows)); i++ {
		factors.ForeignNet20D += flows[i].foreign
		factors.InstNet20D += flows[i].inst
		factors.IndivNet20D += flows[i].indiv
	}

	return factors, nil
}

// GetEventFactors 이벤트 팩터 조회
// fetcher.disclosures 데이터 기반
func (r *FactorRepository) GetEventFactors(ctx context.Context, symbol string) (*signals.EventFactors, error) {
	// 최근 90일 공시 조회
	query := `
		SELECT disclosure_date, title, category
		FROM fetcher.disclosures
		WHERE stock_code = $1
		  AND disclosure_date >= NOW() - INTERVAL '90 days'
		ORDER BY disclosure_date DESC
		LIMIT 20
	`

	rows, err := r.pool.Query(ctx, query, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	factors := &signals.EventFactors{Symbol: symbol}
	events := make([]signals.EventSignal, 0, 20)

	now := time.Now()
	var totalScore float64
	var totalWeight float64

	for rows.Next() {
		var date time.Time
		var title, category string
		if err := rows.Scan(&date, &title, &category); err != nil {
			continue
		}

		// 이벤트 타입 및 영향도 결정
		eventType := mapDisclosureToEventType(title)
		impact := signals.GetEventImpact(eventType)

		event := signals.EventSignal{
			Type:      eventType,
			Score:     impact,
			Title:     title,
			Source:    "DART",
			Timestamp: date,
		}
		events = append(events, event)

		// 시간 가중 점수 계산
		daysSince := now.Sub(date).Hours() / 24.0
		timeWeight := calculateTimeWeight(daysSince)
		totalScore += impact * timeWeight
		totalWeight += timeWeight
	}

	factors.Events = events
	factors.EventCount = len(events)

	if totalWeight > 0 {
		factors.TotalScore = totalScore / totalWeight
	}

	if len(events) > 0 {
		lastEvent := events[0].Timestamp
		factors.LastEventAt = &lastEvent
	}

	return factors, nil
}

// Helper functions

func calculateReturn(current, past int64) float64 {
	if past == 0 {
		return 0
	}
	return (float64(current) - float64(past)) / float64(past)
}

type priceDataLocal struct {
	date   time.Time
	close  int64
	volume int64
}

func avgVolumeFromPriceData(prices []priceDataLocal) float64 {
	if len(prices) == 0 {
		return 0
	}
	var sum int64
	for _, p := range prices {
		sum += p.volume
	}
	return float64(sum) / float64(len(prices))
}

func avgPrice(prices []int64) float64 {
	if len(prices) == 0 {
		return 0
	}
	var sum int64
	for _, p := range prices {
		sum += p
	}
	return float64(sum) / float64(len(prices))
}

func calculateRSI(prices []int64) float64 {
	if len(prices) < 2 {
		return 50
	}

	var gains, losses float64
	for i := 0; i < len(prices)-1; i++ {
		change := float64(prices[i] - prices[i+1])
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	if losses == 0 {
		return 100
	}

	period := float64(len(prices) - 1)
	avgGain := gains / period
	avgLoss := losses / period

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func calculateEMA(prices []int64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	// 초기 SMA
	var sum int64
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	ema := float64(sum) / float64(period)

	// EMA
	multiplier := 2.0 / (float64(period) + 1.0)
	for i := len(prices) - period - 1; i >= 0; i-- {
		ema = (float64(prices[i]) * multiplier) + (ema * (1 - multiplier))
	}

	return ema
}

func calculateStdDev(prices []int64, mean float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	var variance float64
	for _, p := range prices {
		variance += math.Pow(float64(p)-mean, 2)
	}
	return math.Sqrt(variance / float64(len(prices)))
}

func calculateTimeWeight(daysSince float64) float64 {
	const decayRate = 0.023
	weight := math.Exp(-decayRate * daysSince)
	if weight < 0.1 {
		weight = 0.1
	}
	return weight
}

// mapDisclosureToEventType 공시 제목을 이벤트 타입으로 변환
func mapDisclosureToEventType(title string) signals.EventType {
	// 간단한 키워드 매칭 (service/signals/event.go의 로직 재사용)
	// 긍정적 이벤트
	if containsAny(title, "자기주식", "자사주매입") {
		return signals.EventShareBuyback
	}
	if containsAny(title, "배당") {
		return signals.EventDividendIncrease
	}
	if containsAny(title, "신규사업", "신제품") {
		return signals.EventNewProduct
	}
	if containsAny(title, "MOU", "양해각서", "제휴") {
		return signals.EventPartnership
	}

	// 부정적 이벤트
	if containsAny(title, "소송", "피소") {
		return signals.EventLawsuit
	}
	if containsAny(title, "감사의견") {
		return signals.EventAuditOpinion
	}
	if containsAny(title, "행정처분", "과징금") {
		return signals.EventRegulatory
	}

	return signals.EventAnnouncement
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if len(s) >= len(kw) {
			for i := 0; i <= len(s)-len(kw); i++ {
				if s[i:i+len(kw)] == kw {
					return true
				}
			}
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
