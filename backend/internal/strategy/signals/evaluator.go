package signals

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/wonny/aegis/v14/internal/domain/signals"
	"github.com/wonny/aegis/v14/internal/domain/universe"
)

// Evaluator 6팩터 평가기
type Evaluator struct {
	factorRepo signals.FactorRepository
	criteria   *signals.SignalCriteria
}

// NewEvaluator 새 평가기 생성
func NewEvaluator(factorRepo signals.FactorRepository, criteria *signals.SignalCriteria) *Evaluator {
	return &Evaluator{
		factorRepo: factorRepo,
		criteria:   criteria,
	}
}

// EvaluateStock 종목 평가 (6팩터)
func (e *Evaluator) EvaluateStock(ctx context.Context, stock universe.UniverseStock) (*signals.Signal, error) {
	// 1. Load factors
	momentum, err := e.factorRepo.GetMomentumFactors(ctx, stock.Symbol)
	if err != nil {
		return nil, fmt.Errorf("get momentum factors: %w", err)
	}

	quality, err := e.factorRepo.GetQualityFactors(ctx, stock.Symbol)
	if err != nil {
		log.Warn().Err(err).Str("symbol", stock.Symbol).Msg("Quality factors missing, using default")
		quality = &signals.QualityFactors{Symbol: stock.Symbol}
	}

	value, err := e.factorRepo.GetValueFactors(ctx, stock.Symbol)
	if err != nil {
		log.Warn().Err(err).Str("symbol", stock.Symbol).Msg("Value factors missing, using default")
		value = &signals.ValueFactors{Symbol: stock.Symbol}
	}

	technical, err := e.factorRepo.GetTechnicalFactors(ctx, stock.Symbol)
	if err != nil {
		log.Warn().Err(err).Str("symbol", stock.Symbol).Msg("Technical factors missing, using default")
		technical = &signals.TechnicalFactors{Symbol: stock.Symbol}
	}

	// Flow 팩터 로드
	flow, err := e.factorRepo.GetFlowFactors(ctx, stock.Symbol)
	if err != nil {
		log.Warn().Err(err).Str("symbol", stock.Symbol).Msg("Flow factors missing, using default")
		flow = &signals.FlowFactors{Symbol: stock.Symbol}
	}

	// Event 팩터 로드
	event, err := e.factorRepo.GetEventFactors(ctx, stock.Symbol)
	if err != nil {
		log.Warn().Err(err).Str("symbol", stock.Symbol).Msg("Event factors missing, using default")
		event = &signals.EventFactors{Symbol: stock.Symbol}
	}

	// 2. Evaluate each factor (6팩터)
	momentumScore := e.evaluateMomentum(momentum)
	qualityScore := e.evaluateQuality(quality)
	valueScore := e.evaluateValue(value)
	technicalScore := e.evaluateTechnical(technical)
	flowScore := e.evaluateFlow(flow)
	eventScore := e.evaluateEvent(event)

	// 3. Calculate weighted total score (6팩터 가중치)
	totalScore := (momentumScore.Score * e.criteria.MomentumWeight) +
		(technicalScore.Score * e.criteria.TechnicalWeight) +
		(valueScore.Score * e.criteria.ValueWeight) +
		(qualityScore.Score * e.criteria.QualityWeight) +
		(flowScore.Score * e.criteria.FlowWeight) +
		(eventScore.Score * e.criteria.EventWeight)

	// 4. Determine signal type
	signalType := e.determineSignalType(totalScore)

	// 5. Calculate conviction (6팩터)
	conviction := e.calculateConviction6(momentumScore, technicalScore, valueScore, qualityScore, flowScore, eventScore)

	// 6. Generate reasons (6팩터)
	reasons := e.generateReasons6(signalType, momentumScore, technicalScore, valueScore, qualityScore, flowScore, eventScore)

	// 7. Create signal
	signal := &signals.Signal{
		SignalID:   uuid.New(),
		Symbol:     stock.Symbol,
		Name:       stock.Name,
		Market:     stock.Market,
		SignalType: signalType,
		Strength:   int(totalScore),
		Conviction: conviction,
		Factors: signals.SignalBreakdown{
			Momentum:  momentumScore,
			Quality:   qualityScore,
			Value:     valueScore,
			Technical: technicalScore,
			Flow:      flowScore,
			Event:     eventScore,
		},
		Reasons: reasons,
	}

	return signal, nil
}

// evaluateMomentum 모멘텀 팩터 평가
func (e *Evaluator) evaluateMomentum(m *signals.MomentumFactors) signals.FactorScore {
	score := 0.0
	indicators := []string{}

	// 5일 수익률 (30점)
	if m.Return5D > 0.05 { // +5% 이상
		score += 30
		indicators = append(indicators, "5D_STRONG")
	} else if m.Return5D > 0.02 {
		score += 20
		indicators = append(indicators, "5D_GOOD")
	} else if m.Return5D > 0 {
		score += 10
		indicators = append(indicators, "5D_POSITIVE")
	}

	// 20일 수익률 (40점)
	if m.Return20D > 0.15 { // +15% 이상
		score += 40
		indicators = append(indicators, "20D_STRONG")
	} else if m.Return20D > 0.10 {
		score += 30
		indicators = append(indicators, "20D_GOOD")
	} else if m.Return20D > 0.05 {
		score += 20
		indicators = append(indicators, "20D_POSITIVE")
	} else if m.Return20D > 0 {
		score += 10
		indicators = append(indicators, "20D_FLAT")
	}

	// 60일 수익률 (30점)
	if m.Return60D > 0.30 { // +30% 이상
		score += 30
		indicators = append(indicators, "60D_STRONG")
	} else if m.Return60D > 0.20 {
		score += 20
		indicators = append(indicators, "60D_GOOD")
	} else if m.Return60D > 0.10 {
		score += 10
		indicators = append(indicators, "60D_POSITIVE")
	}

	// Normalize to 0-100
	score = math.Min(score, 100)

	return signals.FactorScore{
		Score:      score,
		Weight:     e.criteria.MomentumWeight,
		Triggered:  score >= 60,
		Indicators: indicators,
	}
}

// evaluateQuality 품질 팩터 평가
func (e *Evaluator) evaluateQuality(q *signals.QualityFactors) signals.FactorScore {
	score := 0.0
	indicators := []string{}

	// ROE (40점)
	if q.ROE > 0.20 { // 20% 이상
		score += 40
		indicators = append(indicators, "ROE_EXCELLENT")
	} else if q.ROE > 0.15 {
		score += 30
		indicators = append(indicators, "ROE_GOOD")
	} else if q.ROE > 0.10 {
		score += 20
		indicators = append(indicators, "ROE_OK")
	} else if q.ROE > 0.05 {
		score += 10
		indicators = append(indicators, "ROE_LOW")
	}

	// ROA (30점)
	if q.ROA > 0.10 { // 10% 이상
		score += 30
		indicators = append(indicators, "ROA_EXCELLENT")
	} else if q.ROA > 0.05 {
		score += 20
		indicators = append(indicators, "ROA_GOOD")
	} else if q.ROA > 0.02 {
		score += 10
		indicators = append(indicators, "ROA_OK")
	}

	// Debt Ratio (15점) - 낮을수록 좋음
	if q.DebtRatio < 0.30 { // 30% 미만
		score += 15
		indicators = append(indicators, "DEBT_LOW")
	} else if q.DebtRatio < 0.50 {
		score += 10
		indicators = append(indicators, "DEBT_OK")
	} else if q.DebtRatio < 1.00 {
		score += 5
		indicators = append(indicators, "DEBT_HIGH")
	}

	// Current Ratio (15점)
	if q.CurrentRatio > 2.0 { // 200% 이상
		score += 15
		indicators = append(indicators, "LIQUIDITY_EXCELLENT")
	} else if q.CurrentRatio > 1.5 {
		score += 10
		indicators = append(indicators, "LIQUIDITY_GOOD")
	} else if q.CurrentRatio > 1.0 {
		score += 5
		indicators = append(indicators, "LIQUIDITY_OK")
	}

	// Normalize to 0-100
	score = math.Min(score, 100)

	return signals.FactorScore{
		Score:      score,
		Weight:     e.criteria.QualityWeight,
		Triggered:  score >= 60,
		Indicators: indicators,
	}
}

// evaluateValue 가치 팩터 평가
func (e *Evaluator) evaluateValue(v *signals.ValueFactors) signals.FactorScore {
	score := 0.0
	indicators := []string{}

	// PER (40점) - 낮을수록 좋음
	if v.PER > 0 && v.PER < 10 {
		score += 40
		indicators = append(indicators, "PER_UNDERVALUED")
	} else if v.PER < 15 {
		score += 30
		indicators = append(indicators, "PER_FAIR")
	} else if v.PER < 20 {
		score += 20
		indicators = append(indicators, "PER_OK")
	} else if v.PER < 30 {
		score += 10
		indicators = append(indicators, "PER_HIGH")
	}

	// PBR (30점) - 낮을수록 좋음
	if v.PBR > 0 && v.PBR < 1.0 {
		score += 30
		indicators = append(indicators, "PBR_UNDERVALUED")
	} else if v.PBR < 1.5 {
		score += 20
		indicators = append(indicators, "PBR_FAIR")
	} else if v.PBR < 2.0 {
		score += 10
		indicators = append(indicators, "PBR_OK")
	}

	// Dividend Yield (30점)
	if v.DividendYield > 0.05 { // 5% 이상
		score += 30
		indicators = append(indicators, "DIV_HIGH")
	} else if v.DividendYield > 0.03 {
		score += 20
		indicators = append(indicators, "DIV_GOOD")
	} else if v.DividendYield > 0.01 {
		score += 10
		indicators = append(indicators, "DIV_OK")
	}

	// Normalize to 0-100
	score = math.Min(score, 100)

	return signals.FactorScore{
		Score:      score,
		Weight:     e.criteria.ValueWeight,
		Triggered:  score >= 60,
		Indicators: indicators,
	}
}

// evaluateTechnical 기술적 팩터 평가
func (e *Evaluator) evaluateTechnical(t *signals.TechnicalFactors) signals.FactorScore {
	score := 0.0
	indicators := []string{}

	// RSI (40점)
	if t.RSI > 30 && t.RSI < 70 { // 과매수/과매도 아님
		score += 40
		indicators = append(indicators, "RSI_NEUTRAL")
	} else if t.RSI <= 30 { // 과매도 (매수 신호)
		score += 50
		indicators = append(indicators, "RSI_OVERSOLD")
	} else if t.RSI >= 70 { // 과매수
		score += 20
		indicators = append(indicators, "RSI_OVERBOUGHT")
	}

	// MACD (40점)
	if t.MACD > t.MACDSignal { // 골든크로스
		score += 40
		indicators = append(indicators, "MACD_BULLISH")
	} else if t.MACD < t.MACDSignal { // 데드크로스
		score += 10
		indicators = append(indicators, "MACD_BEARISH")
	}

	// Bollinger Position (20점)
	if t.BollingerPos > 0.3 && t.BollingerPos < 0.7 { // 중간 대역
		score += 20
		indicators = append(indicators, "BB_MIDDLE")
	} else if t.BollingerPos <= 0.3 { // 하단 (매수 기회)
		score += 30
		indicators = append(indicators, "BB_LOWER")
	} else if t.BollingerPos >= 0.7 { // 상단
		score += 10
		indicators = append(indicators, "BB_UPPER")
	}

	// Normalize to 0-100
	score = math.Min(score, 100)

	return signals.FactorScore{
		Score:      score,
		Weight:     e.criteria.TechnicalWeight,
		Triggered:  score >= 60,
		Indicators: indicators,
	}
}

// evaluateFlow 수급 팩터 평가
func (e *Evaluator) evaluateFlow(f *signals.FlowFactors) signals.FactorScore {
	score := 0.0
	indicators := []string{}

	// 외국인 5일 순매수 (40점)
	if f.ForeignNet5D > 1000000 { // 100만주 이상
		score += 40
		indicators = append(indicators, "FOREIGN_5D_STRONG")
	} else if f.ForeignNet5D > 500000 {
		score += 30
		indicators = append(indicators, "FOREIGN_5D_GOOD")
	} else if f.ForeignNet5D > 0 {
		score += 20
		indicators = append(indicators, "FOREIGN_5D_BUY")
	} else if f.ForeignNet5D < -500000 {
		indicators = append(indicators, "FOREIGN_5D_SELL")
	}

	// 기관 5일 순매수 (30점)
	if f.InstNet5D > 500000 {
		score += 30
		indicators = append(indicators, "INST_5D_STRONG")
	} else if f.InstNet5D > 100000 {
		score += 20
		indicators = append(indicators, "INST_5D_GOOD")
	} else if f.InstNet5D > 0 {
		score += 10
		indicators = append(indicators, "INST_5D_BUY")
	}

	// 외국인 20일 추세 (30점)
	if f.ForeignNet20D > 5000000 { // 500만주 이상
		score += 30
		indicators = append(indicators, "FOREIGN_20D_STRONG")
	} else if f.ForeignNet20D > 1000000 {
		score += 20
		indicators = append(indicators, "FOREIGN_20D_GOOD")
	} else if f.ForeignNet20D > 0 {
		score += 10
		indicators = append(indicators, "FOREIGN_20D_BUY")
	}

	// Normalize to 0-100
	score = math.Min(score, 100)

	return signals.FactorScore{
		Score:      score,
		Weight:     e.criteria.FlowWeight,
		Triggered:  score >= 60,
		Indicators: indicators,
	}
}

// evaluateEvent 이벤트 팩터 평가
func (e *Evaluator) evaluateEvent(ev *signals.EventFactors) signals.FactorScore {
	score := 50.0 // 중립 기준
	indicators := []string{}

	// 이벤트 점수 반영 (TotalScore는 -1.0 ~ 1.0)
	if ev.TotalScore > 0.5 {
		score = 80
		indicators = append(indicators, "EVENT_VERY_POSITIVE")
	} else if ev.TotalScore > 0.2 {
		score = 70
		indicators = append(indicators, "EVENT_POSITIVE")
	} else if ev.TotalScore > 0 {
		score = 60
		indicators = append(indicators, "EVENT_SLIGHTLY_POSITIVE")
	} else if ev.TotalScore < -0.5 {
		score = 20
		indicators = append(indicators, "EVENT_VERY_NEGATIVE")
	} else if ev.TotalScore < -0.2 {
		score = 30
		indicators = append(indicators, "EVENT_NEGATIVE")
	} else if ev.TotalScore < 0 {
		score = 40
		indicators = append(indicators, "EVENT_SLIGHTLY_NEGATIVE")
	}

	// 최근 이벤트 수 반영
	if ev.EventCount > 0 {
		indicators = append(indicators, fmt.Sprintf("EVENTS_%d", ev.EventCount))
	}

	return signals.FactorScore{
		Score:      score,
		Weight:     e.criteria.EventWeight,
		Triggered:  score >= 60,
		Indicators: indicators,
	}
}

// determineSignalType 신호 타입 결정
func (e *Evaluator) determineSignalType(totalScore float64) signals.SignalType {
	if totalScore >= float64(e.criteria.BuyThreshold) {
		return signals.SignalBuy
	} else if totalScore <= float64(e.criteria.SellThreshold) {
		return signals.SignalSell
	}
	return signals.SignalHold
}

// calculateConviction 신뢰도 계산 (4팩터 - 레거시)
func (e *Evaluator) calculateConviction(
	momentum, quality, value, technical signals.FactorScore,
) int {
	// 트리거된 팩터 개수
	triggeredCount := 0
	if momentum.Triggered {
		triggeredCount++
	}
	if quality.Triggered {
		triggeredCount++
	}
	if value.Triggered {
		triggeredCount++
	}
	if technical.Triggered {
		triggeredCount++
	}

	// 팩터 일관성 (표준편차 기반)
	scores := []float64{momentum.Score, quality.Score, value.Score, technical.Score}
	mean := (momentum.Score + quality.Score + value.Score + technical.Score) / 4.0
	variance := 0.0
	for _, s := range scores {
		variance += math.Pow(s-mean, 2)
	}
	stdDev := math.Sqrt(variance / 4.0)

	// 표준편차가 작을수록 일관성 높음 (높은 신뢰도)
	consistency := math.Max(0, 100-stdDev)

	// 트리거 비율 (0-100)
	triggerScore := float64(triggeredCount) * 25.0

	// 신뢰도 = 트리거 점수 70% + 일관성 30%
	conviction := (triggerScore * 0.7) + (consistency * 0.3)

	return int(math.Min(conviction, 100))
}

// calculateConviction6 신뢰도 계산 (6팩터)
func (e *Evaluator) calculateConviction6(
	momentum, technical, value, quality, flow, event signals.FactorScore,
) int {
	// 트리거된 팩터 개수
	triggeredCount := 0
	if momentum.Triggered {
		triggeredCount++
	}
	if technical.Triggered {
		triggeredCount++
	}
	if value.Triggered {
		triggeredCount++
	}
	if quality.Triggered {
		triggeredCount++
	}
	if flow.Triggered {
		triggeredCount++
	}
	if event.Triggered {
		triggeredCount++
	}

	// 팩터 일관성 (표준편차 기반)
	scores := []float64{momentum.Score, technical.Score, value.Score, quality.Score, flow.Score, event.Score}
	mean := (momentum.Score + technical.Score + value.Score + quality.Score + flow.Score + event.Score) / 6.0
	variance := 0.0
	for _, s := range scores {
		variance += math.Pow(s-mean, 2)
	}
	stdDev := math.Sqrt(variance / 6.0)

	// 표준편차가 작을수록 일관성 높음 (높은 신뢰도)
	consistency := math.Max(0, 100-stdDev)

	// 트리거 비율 (0-100) - 6팩터 기준
	triggerScore := float64(triggeredCount) * 100.0 / 6.0

	// 신뢰도 = 트리거 점수 70% + 일관성 30%
	conviction := (triggerScore * 0.7) + (consistency * 0.3)

	return int(math.Min(conviction, 100))
}

// generateReasons 신호 생성 근거 (4팩터 - 레거시)
func (e *Evaluator) generateReasons(
	signalType signals.SignalType,
	momentum, quality, value, technical signals.FactorScore,
) []string {
	reasons := []string{}

	// 신호 타입별 주요 근거
	switch signalType {
	case signals.SignalBuy:
		if momentum.Triggered {
			reasons = append(reasons, fmt.Sprintf("강한 모멘텀 (%.0f점)", momentum.Score))
		}
		if quality.Triggered {
			reasons = append(reasons, fmt.Sprintf("우수한 품질 (%.0f점)", quality.Score))
		}
		if value.Triggered {
			reasons = append(reasons, fmt.Sprintf("저평가 (%.0f점)", value.Score))
		}
		if technical.Triggered {
			reasons = append(reasons, fmt.Sprintf("긍정적 기술 지표 (%.0f점)", technical.Score))
		}

	case signals.SignalSell:
		if !momentum.Triggered {
			reasons = append(reasons, fmt.Sprintf("약한 모멘텀 (%.0f점)", momentum.Score))
		}
		if !quality.Triggered {
			reasons = append(reasons, fmt.Sprintf("낮은 품질 (%.0f점)", quality.Score))
		}
		if !value.Triggered {
			reasons = append(reasons, fmt.Sprintf("고평가 (%.0f점)", value.Score))
		}
		if !technical.Triggered {
			reasons = append(reasons, fmt.Sprintf("부정적 기술 지표 (%.0f점)", technical.Score))
		}
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "중립적 신호")
	}

	return reasons
}

// generateReasons6 신호 생성 근거 (6팩터)
func (e *Evaluator) generateReasons6(
	signalType signals.SignalType,
	momentum, technical, value, quality, flow, event signals.FactorScore,
) []string {
	reasons := []string{}

	// 신호 타입별 주요 근거
	switch signalType {
	case signals.SignalBuy:
		if momentum.Triggered {
			reasons = append(reasons, fmt.Sprintf("강한 모멘텀 (%.0f점)", momentum.Score))
		}
		if technical.Triggered {
			reasons = append(reasons, fmt.Sprintf("긍정적 기술 지표 (%.0f점)", technical.Score))
		}
		if value.Triggered {
			reasons = append(reasons, fmt.Sprintf("저평가 (%.0f점)", value.Score))
		}
		if quality.Triggered {
			reasons = append(reasons, fmt.Sprintf("우수한 품질 (%.0f점)", quality.Score))
		}
		if flow.Triggered {
			reasons = append(reasons, fmt.Sprintf("스마트머니 유입 (%.0f점)", flow.Score))
		}
		if event.Triggered {
			reasons = append(reasons, fmt.Sprintf("긍정적 이벤트 (%.0f점)", event.Score))
		}

	case signals.SignalSell:
		if !momentum.Triggered {
			reasons = append(reasons, fmt.Sprintf("약한 모멘텀 (%.0f점)", momentum.Score))
		}
		if !technical.Triggered {
			reasons = append(reasons, fmt.Sprintf("부정적 기술 지표 (%.0f점)", technical.Score))
		}
		if !value.Triggered {
			reasons = append(reasons, fmt.Sprintf("고평가 (%.0f점)", value.Score))
		}
		if !quality.Triggered {
			reasons = append(reasons, fmt.Sprintf("낮은 품질 (%.0f점)", quality.Score))
		}
		if !flow.Triggered {
			reasons = append(reasons, fmt.Sprintf("스마트머니 이탈 (%.0f점)", flow.Score))
		}
		if !event.Triggered {
			reasons = append(reasons, fmt.Sprintf("부정적 이벤트 (%.0f점)", event.Score))
		}
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "중립적 신호")
	}

	return reasons
}
