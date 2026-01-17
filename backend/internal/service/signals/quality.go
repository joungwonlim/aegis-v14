package signals

import (
	"context"
	"math"

	"github.com/rs/zerolog/log"
)

// QualityCalculator 품질 시그널 계산기
// SSOT: 품질 시그널 계산은 여기서만
type QualityCalculator struct{}

// NewQualityCalculator 새 품질 계산기 생성
func NewQualityCalculator() *QualityCalculator {
	return &QualityCalculator{}
}

// Calculate 품질 시그널 계산
// 입력: 품질 지표 (ROE, 부채비율)
// 출력: 점수 (-1.0 ~ 1.0), 상세 정보, 에러
func (c *QualityCalculator) Calculate(ctx context.Context, code string, metrics QualityMetrics) (float64, QualityDetails, error) {
	details := QualityDetails{
		ROE:       metrics.ROE,
		DebtRatio: metrics.DebtRatio,
	}

	// 품질 점수 계산
	score := c.calculateScore(metrics.ROE, metrics.DebtRatio)

	log.Debug().
		Str("code", code).
		Float64("roe", metrics.ROE).
		Float64("debt_ratio", metrics.DebtRatio).
		Float64("score", score).
		Msg("Calculated quality signal")

	return score, details, nil
}

// calculateScore 품질 점수 계산 (-1.0 ~ 1.0)
// 가중치: ROE 60%, DebtRatio 40%
func (c *QualityCalculator) calculateScore(roe, debtRatio float64) float64 {
	roeScore := c.scoreROE(roe)
	debtScore := c.scoreDebtRatio(debtRatio)

	// 가중 합산
	score := roeScore*0.6 + debtScore*0.4

	return score
}

// scoreROE ROE 점수화
// ROE(자기자본이익률): 높을수록 좋음
// 기준값:
// ROE > 20%: +1.0 (우량)
// ROE 15~20%: +0.5 ~ +1.0
// ROE 5~15%: 0 ~ +0.5 (중립)
// ROE < 5%: -0.5 ~ 0 (저품질)
// ROE < 0%: -1.0 (적자)
func (c *QualityCalculator) scoreROE(roe float64) float64 {
	// ROE는 일반적으로 백분율로 전달 (15% = 15.0)
	// 또는 비율로 전달 (15% = 0.15)
	// 여기서는 비율로 가정하고, 백분율도 처리

	// ROE가 1보다 크면 백분율로 간주
	if roe > 1.0 {
		roe = roe / 100.0
	}

	if roe < 0 {
		// 적자
		return math.Max(-1.0, math.Tanh(roe*5))
	}

	if roe > 0.20 {
		return 1.0
	} else if roe > 0.15 {
		// 15% ~ 20% → 0.5 ~ 1.0
		return 0.5 + (roe-0.15)*10
	} else if roe > 0.05 {
		// 5% ~ 15% → 0 ~ 0.5
		return (roe - 0.05) * 5
	} else {
		// 0% ~ 5% → -0.5 ~ 0
		return (roe - 0.05) * 10
	}
}

// scoreDebtRatio 부채비율 점수화
// 부채비율: 낮을수록 좋음
// 기준값:
// Debt < 50%: +1.0 (저위험)
// Debt 50~100%: +0.5 ~ +1.0
// Debt 100~150%: 0 ~ +0.5 (중립)
// Debt > 150%: -0.5 ~ 0 (고위험)
// Debt > 300%: -1.0 (극고위험)
func (c *QualityCalculator) scoreDebtRatio(debtRatio float64) float64 {
	// DebtRatio가 1보다 크면 백분율로 간주
	if debtRatio > 3.0 {
		debtRatio = debtRatio / 100.0
	}

	if debtRatio < 0 {
		return 0.0 // 비정상
	}

	if debtRatio < 0.50 {
		return 1.0
	} else if debtRatio < 1.00 {
		// 50% ~ 100% → 0.5 ~ 1.0
		return 1.0 - (debtRatio-0.50)
	} else if debtRatio < 1.50 {
		// 100% ~ 150% → 0 ~ 0.5
		return 0.5 - (debtRatio - 1.00)
	} else if debtRatio < 3.00 {
		// 150% ~ 300% → -0.5 ~ 0
		return math.Tanh(-(debtRatio - 1.50))
	} else {
		return -1.0
	}
}
