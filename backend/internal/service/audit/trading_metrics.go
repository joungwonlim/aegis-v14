package audit

import (
	"math"

	"github.com/wonny/aegis/v14/internal/domain/audit"
)

// =============================================================================
// Trading Metrics
// =============================================================================

// CalculateWinRate 승률 계산
func CalculateWinRate(trades []audit.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}

	wins := 0
	for _, t := range trades {
		if t.PnL > 0 {
			wins++
		}
	}

	return float64(wins) / float64(len(trades))
}

// CalculateAvgWinLoss 평균 수익/손실 계산
func CalculateAvgWinLoss(trades []audit.Trade) (avgWin, avgLoss float64) {
	if len(trades) == 0 {
		return 0, 0
	}

	var sumWin, sumLoss float64
	var countWin, countLoss int

	for _, t := range trades {
		if t.PnL > 0 {
			sumWin += t.PnL
			countWin++
		} else if t.PnL < 0 {
			sumLoss += t.PnL
			countLoss++
		}
	}

	if countWin > 0 {
		avgWin = sumWin / float64(countWin)
	}
	if countLoss > 0 {
		avgLoss = sumLoss / float64(countLoss)
	}

	return avgWin, avgLoss
}

// CalculateProfitFactor 수익 팩터 계산
func CalculateProfitFactor(trades []audit.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}

	var totalWin, totalLoss float64

	for _, t := range trades {
		if t.PnL > 0 {
			totalWin += t.PnL
		} else if t.PnL < 0 {
			totalLoss += math.Abs(t.PnL)
		}
	}

	if totalLoss == 0 {
		if totalWin > 0 {
			return 999.99 // 손실 없음 (Inf 대신 최대값 반환)
		}
		return 0
	}

	return totalWin / totalLoss
}

// CalculateExpectancy 기대값 계산
func CalculateExpectancy(trades []audit.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}

	var totalPnL float64
	for _, t := range trades {
		totalPnL += t.PnL
	}

	return totalPnL / float64(len(trades))
}

// CalculateAvgHoldDays 평균 보유 기간 계산
func CalculateAvgHoldDays(trades []audit.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}

	var totalDays int
	for _, t := range trades {
		totalDays += t.HoldDays
	}

	return float64(totalDays) / float64(len(trades))
}

// CalculateMaxConsecutive 최대 연속 수익/손실 계산
func CalculateMaxConsecutive(trades []audit.Trade) (maxWins, maxLosses int) {
	if len(trades) == 0 {
		return 0, 0
	}

	currentWins, currentLosses := 0, 0

	for _, t := range trades {
		if t.PnL > 0 {
			currentWins++
			if currentWins > maxWins {
				maxWins = currentWins
			}
			currentLosses = 0
		} else if t.PnL < 0 {
			currentLosses++
			if currentLosses > maxLosses {
				maxLosses = currentLosses
			}
			currentWins = 0
		}
	}

	return maxWins, maxLosses
}

// CalculateLargestWinLoss 최대 수익/손실 거래 찾기
func CalculateLargestWinLoss(trades []audit.Trade) (largestWin, largestLoss float64) {
	if len(trades) == 0 {
		return 0, 0
	}

	for _, t := range trades {
		if t.PnL > largestWin {
			largestWin = t.PnL
		}
		if t.PnL < largestLoss {
			largestLoss = t.PnL
		}
	}

	return largestWin, largestLoss
}

// TradingMetrics 트레이딩 지표 집계
type TradingMetrics struct {
	TotalTrades       int     `json:"total_trades"`
	WinRate           float64 `json:"win_rate"`
	AvgWin            float64 `json:"avg_win"`
	AvgLoss           float64 `json:"avg_loss"`
	ProfitFactor      float64 `json:"profit_factor"`
	Expectancy        float64 `json:"expectancy"`
	AvgHoldDays       float64 `json:"avg_hold_days"`
	MaxConsecutiveWin int     `json:"max_consecutive_win"`
	MaxConsecutiveLoss int    `json:"max_consecutive_loss"`
	LargestWin        float64 `json:"largest_win"`
	LargestLoss       float64 `json:"largest_loss"`
}

// CalculateTradingMetrics 전체 트레이딩 지표 계산
func CalculateTradingMetrics(trades []audit.Trade) TradingMetrics {
	avgWin, avgLoss := CalculateAvgWinLoss(trades)
	maxWins, maxLosses := CalculateMaxConsecutive(trades)
	largestWin, largestLoss := CalculateLargestWinLoss(trades)

	return TradingMetrics{
		TotalTrades:       len(trades),
		WinRate:           CalculateWinRate(trades),
		AvgWin:            avgWin,
		AvgLoss:           avgLoss,
		ProfitFactor:      CalculateProfitFactor(trades),
		Expectancy:        CalculateExpectancy(trades),
		AvgHoldDays:       CalculateAvgHoldDays(trades),
		MaxConsecutiveWin: maxWins,
		MaxConsecutiveLoss: maxLosses,
		LargestWin:        largestWin,
		LargestLoss:       largestLoss,
	}
}
