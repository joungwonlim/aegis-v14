package signals

import (
	"context"
	"math"

	"github.com/rs/zerolog/log"
)

// TechnicalCalculator 기술적 시그널 계산기
// SSOT: 기술적 지표 계산은 여기서만
type TechnicalCalculator struct{}

// NewTechnicalCalculator 새 기술적 계산기 생성
func NewTechnicalCalculator() *TechnicalCalculator {
	return &TechnicalCalculator{}
}

// Calculate 기술적 시그널 계산
// 입력: 가격 데이터 (prices[0]이 가장 최근, 최소 120일 필요)
// 출력: 점수 (-1.0 ~ 1.0), 상세 정보, 에러
func (c *TechnicalCalculator) Calculate(ctx context.Context, code string, prices []PricePoint) (float64, TechnicalDetails, error) {
	details := TechnicalDetails{}

	if len(prices) < 120 {
		// MA120 계산을 위해 최소 120일 필요
		return 0.0, details, nil
	}

	// RSI 계산 (14일)
	rsi := c.calculateRSI(prices, 14)
	details.RSI = rsi

	// MACD 계산 (12, 26, 9)
	macd, _ := c.calculateMACD(prices)
	details.MACD = macd

	// MA20 크로스 계산
	ma20Cross := c.calculateMA20Cross(prices)
	details.MA20Cross = ma20Cross

	// 기술적 점수 계산
	score := c.calculateScore(rsi, macd, ma20Cross)

	log.Debug().
		Str("code", code).
		Float64("rsi", rsi).
		Float64("macd", macd).
		Int("ma20_cross", ma20Cross).
		Float64("score", score).
		Msg("Calculated technical signal")

	return score, details, nil
}

// calculateRSI RSI(Relative Strength Index) 계산
// RSI = 100 - (100 / (1 + RS))
// RS = 평균 상승폭 / 평균 하락폭
func (c *TechnicalCalculator) calculateRSI(prices []PricePoint, period int) float64 {
	if len(prices) < period+1 {
		return 50.0 // 중립
	}

	var gains, losses float64

	for i := 0; i < period; i++ {
		change := float64(prices[i].Price - prices[i+1].Price)
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	if losses == 0 {
		return 100.0
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateMACD MACD 계산
// MACD = EMA12 - EMA26
// Signal = EMA9 of MACD
func (c *TechnicalCalculator) calculateMACD(prices []PricePoint) (float64, float64) {
	if len(prices) < 26 {
		return 0.0, 0.0
	}

	// EMA12, EMA26 계산
	ema12 := c.calculateEMA(prices, 12)
	ema26 := c.calculateEMA(prices, 26)

	// MACD = EMA12 - EMA26
	macd := ema12 - ema26

	// 시그널 라인 (단순화: 현재 MACD 값 사용)
	signal := macd

	return macd, signal
}

// calculateEMA EMA(Exponential Moving Average) 계산
func (c *TechnicalCalculator) calculateEMA(prices []PricePoint, period int) float64 {
	if len(prices) < period {
		return 0.0
	}

	// 초기 SMA 계산
	var sum float64
	for i := 0; i < period; i++ {
		sum += float64(prices[len(prices)-period+i].Price)
	}
	sma := sum / float64(period)

	// EMA 계산
	multiplier := 2.0 / (float64(period) + 1.0)
	ema := sma

	for i := len(prices) - period - 1; i >= 0; i-- {
		ema = (float64(prices[i].Price) * multiplier) + (ema * (1 - multiplier))
	}

	return ema
}

// calculateMA20Cross MA20 크로스 시그널 계산
// 반환: -1 (데드크로스), 0 (중립), 1 (골든크로스)
func (c *TechnicalCalculator) calculateMA20Cross(prices []PricePoint) int {
	if len(prices) < 20 {
		return 0
	}

	// MA20 계산
	var sum int64
	for i := 0; i < 20; i++ {
		sum += prices[i].Price
	}
	ma20 := float64(sum) / 20.0

	currentPrice := float64(prices[0].Price)

	// 가격과 MA20 차이 비율
	priceDiff := (currentPrice - ma20) / ma20

	if priceDiff > 0.02 { // 가격 > MA20 2% 이상
		return 1 // 골든크로스
	} else if priceDiff < -0.02 { // 가격 < MA20 2% 이상
		return -1 // 데드크로스
	}

	return 0 // 중립
}

// calculateScore 기술적 점수 계산 (-1.0 ~ 1.0)
// 가중치: RSI 40%, MACD 40%, MA20 Cross 20%
func (c *TechnicalCalculator) calculateScore(rsi, macd float64, ma20Cross int) float64 {
	// RSI 점수 변환 (0-100 → -1 ~ 1)
	// RSI < 30: 과매도 (긍정적)
	// RSI > 70: 과매수 (부정적)
	// RSI = 50: 중립
	rsiScore := 0.0
	if rsi < 30 {
		rsiScore = (30 - rsi) / 30 // 0 ~ 1
	} else if rsi > 70 {
		rsiScore = (70 - rsi) / 30 // 0 ~ -1
	} else {
		rsiScore = (50 - rsi) / 20 // -1 ~ 1
	}

	// MACD 점수 변환 (tanh 정규화)
	// 일반적인 MACD 범위: -1000 ~ 1000
	macdScore := math.Tanh(macd / 500)

	// MA20 크로스 점수 (-1, 0, 1)
	ma20Score := float64(ma20Cross)

	// 가중 합산
	// RSI: 40%, MACD: 40%, MA20: 20%
	score := rsiScore*0.4 + macdScore*0.4 + ma20Score*0.2

	// 범위 제한
	if score > 1.0 {
		score = 1.0
	} else if score < -1.0 {
		score = -1.0
	}

	return score
}
