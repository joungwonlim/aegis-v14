package audit

import (
	"math"
	"sort"
)

// =============================================================================
// Constants
// =============================================================================

const (
	// TradingDays 연간 거래일 수 (한국)
	TradingDays = 252

	// RiskFreeRate 무위험 수익률 (한국 국채 금리 참고)
	RiskFreeRate = 0.03
)

// =============================================================================
// Return Calculations
// =============================================================================

// CalculateTotalReturn 누적 수익률 계산
func CalculateTotalReturn(dailyReturns []float64) float64 {
	if len(dailyReturns) == 0 {
		return 0
	}

	cumReturn := 1.0
	for _, r := range dailyReturns {
		cumReturn *= (1.0 + r)
	}
	return cumReturn - 1.0
}

// CalculateAnnualizedReturn 연환산 수익률 계산
func CalculateAnnualizedReturn(totalReturn float64, days int) float64 {
	if days == 0 {
		return 0
	}
	return math.Pow(1.0+totalReturn, float64(TradingDays)/float64(days)) - 1.0
}

// =============================================================================
// Risk Calculations
// =============================================================================

// CalculateVolatility 연환산 변동성 계산
func CalculateVolatility(dailyReturns []float64) float64 {
	if len(dailyReturns) < 2 {
		return 0
	}

	mean := CalculateMean(dailyReturns)
	variance := CalculateVariance(dailyReturns, mean)

	// 연환산 변동성 = 일간 표준편차 × √252
	return math.Sqrt(variance) * math.Sqrt(float64(TradingDays))
}

// CalculateSharpe 샤프 비율 계산
func CalculateSharpe(annualReturn, volatility float64) float64 {
	if volatility == 0 {
		return 0
	}
	return (annualReturn - RiskFreeRate) / volatility
}

// CalculateSortino 소르티노 비율 계산
func CalculateSortino(dailyReturns []float64) float64 {
	if len(dailyReturns) < 2 {
		return 0
	}

	// 연환산 수익률
	totalReturn := CalculateTotalReturn(dailyReturns)
	annualReturn := CalculateAnnualizedReturn(totalReturn, len(dailyReturns))

	// Downside deviation (음수 수익률만 사용)
	var sumSquaredNegative float64
	var countNegative int

	for _, r := range dailyReturns {
		if r < 0 {
			sumSquaredNegative += r * r
			countNegative++
		}
	}

	if countNegative == 0 {
		return 0
	}

	downsideVol := math.Sqrt(sumSquaredNegative/float64(countNegative)) * math.Sqrt(float64(TradingDays))

	if downsideVol == 0 {
		return 0
	}

	return (annualReturn - RiskFreeRate) / downsideVol
}

// CalculateMaxDrawdown 최대 낙폭 (MDD) 계산
func CalculateMaxDrawdown(dailyReturns []float64) float64 {
	if len(dailyReturns) == 0 {
		return 0
	}

	cumValue := 1.0
	peak := 1.0
	maxDD := 0.0

	for _, r := range dailyReturns {
		cumValue *= (1.0 + r)
		if cumValue > peak {
			peak = cumValue
		}
		dd := (cumValue - peak) / peak
		if dd < maxDD {
			maxDD = dd
		}
	}

	return maxDD // 음수로 반환
}

// CalculateVaR VaR (Value at Risk) 계산 - Historical method
func CalculateVaR(dailyReturns []float64, confidence float64) float64 {
	if len(dailyReturns) == 0 {
		return 0
	}

	// 오름차순 정렬
	sorted := make([]float64, len(dailyReturns))
	copy(sorted, dailyReturns)
	sort.Float64s(sorted)

	// 하위 (1-confidence)% 위치
	idx := int(float64(len(sorted)) * (1.0 - confidence))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	if idx < 0 {
		idx = 0
	}

	return -sorted[idx] // 양수로 반환 (손실 크기)
}

// CalculateCVaR CVaR (Conditional VaR) 계산
func CalculateCVaR(dailyReturns []float64, confidence float64) float64 {
	if len(dailyReturns) == 0 {
		return 0
	}

	sorted := make([]float64, len(dailyReturns))
	copy(sorted, dailyReturns)
	sort.Float64s(sorted)

	// 하위 (1-confidence)% 값들의 평균
	cutoffIdx := int(float64(len(sorted)) * (1.0 - confidence))
	if cutoffIdx <= 0 {
		cutoffIdx = 1
	}

	var sum float64
	for i := 0; i < cutoffIdx; i++ {
		sum += sorted[i]
	}

	return -sum / float64(cutoffIdx) // 양수로 반환
}

// =============================================================================
// Benchmark Calculations
// =============================================================================

// CalculateAlpha 알파 (초과 수익률) 계산
func CalculateAlpha(portfolioReturn, benchmarkReturn float64) float64 {
	return portfolioReturn - benchmarkReturn
}

// CalculateBeta 베타 (시장 민감도) 계산
func CalculateBeta(portfolioReturns, benchmarkReturns []float64) float64 {
	if len(portfolioReturns) != len(benchmarkReturns) || len(portfolioReturns) < 2 {
		return 0
	}

	covariance := CalculateCovariance(portfolioReturns, benchmarkReturns)
	benchmarkVariance := CalculateVariance(benchmarkReturns, CalculateMean(benchmarkReturns))

	if benchmarkVariance == 0 {
		return 0
	}

	return covariance / benchmarkVariance
}

// =============================================================================
// Statistical Helpers
// =============================================================================

// CalculateMean 평균 계산
func CalculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// CalculateVariance 분산 계산 (표본 분산)
func CalculateVariance(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}

	var sumSquaredDiff float64
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	return sumSquaredDiff / float64(len(values)-1)
}

// CalculateCovariance 공분산 계산
func CalculateCovariance(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	meanX := CalculateMean(x)
	meanY := CalculateMean(y)

	var sum float64
	for i := 0; i < len(x); i++ {
		sum += (x[i] - meanX) * (y[i] - meanY)
	}

	return sum / float64(len(x)-1)
}
