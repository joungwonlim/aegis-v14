package signals

import (
	"context"
	"math"

	"github.com/rs/zerolog/log"
)

// ValueCalculator 가치 시그널 계산기
// SSOT: 가치 시그널 계산은 여기서만
type ValueCalculator struct{}

// NewValueCalculator 새 가치 계산기 생성
func NewValueCalculator() *ValueCalculator {
	return &ValueCalculator{}
}

// Calculate 가치 시그널 계산
// 입력: 밸류에이션 지표
// 출력: 점수 (-1.0 ~ 1.0), 상세 정보, 에러
func (c *ValueCalculator) Calculate(ctx context.Context, code string, metrics ValueMetrics) (float64, ValueDetails, error) {
	details := ValueDetails{
		PER: metrics.PER,
		PBR: metrics.PBR,
		PSR: metrics.PSR,
	}

	// 가치 점수 계산
	score := c.calculateScore(metrics.PER, metrics.PBR, metrics.PSR)

	log.Debug().
		Str("code", code).
		Float64("per", metrics.PER).
		Float64("pbr", metrics.PBR).
		Float64("psr", metrics.PSR).
		Float64("score", score).
		Msg("Calculated value signal")

	return score, details, nil
}

// calculateScore 가치 점수 계산 (-1.0 ~ 1.0)
// 가중치: PER 50%, PBR 30%, PSR 20%
// 낮을수록 저평가 = 높은 점수
func (c *ValueCalculator) calculateScore(per, pbr, psr float64) float64 {
	perScore := c.scorePER(per)
	pbrScore := c.scorePBR(pbr)
	psrScore := c.scorePSR(psr)

	// 가중 합산
	score := perScore*0.5 + pbrScore*0.3 + psrScore*0.2

	return score
}

// scorePER PER 점수화
// 기준: PER 10
// PER < 5: +1.0 (극도 저평가)
// PER 5~10: +0.5 ~ +1.0 (저평가)
// PER 10~20: 0 ~ +0.5 (적정)
// PER > 20: -0.5 ~ 0 (고평가)
// PER < 0 또는 비정상: 0 (적자/미계산)
func (c *ValueCalculator) scorePER(per float64) float64 {
	if per <= 0 || per > 100 {
		return 0.0 // 적자 또는 비정상
	}

	if per < 5 {
		return 1.0
	} else if per < 10 {
		// 5 ~ 10 → 0.5 ~ 1.0
		return 0.5 + (10-per)/10
	} else if per < 20 {
		// 10 ~ 20 → 0 ~ 0.5
		return (20 - per) / 20
	} else {
		// 20 ~ 40 → 0 ~ -0.5 (tanh로 점진적 감소)
		return math.Tanh(-(per - 20) / 40)
	}
}

// scorePBR PBR 점수화
// 기준: PBR 1.0
// PBR < 0.5: +1.0 (극도 저평가)
// PBR 0.5~1.0: +0.5 ~ +1.0 (저평가)
// PBR 1.0~2.0: 0 ~ +0.5 (적정)
// PBR > 2.0: -0.5 ~ 0 (고평가)
func (c *ValueCalculator) scorePBR(pbr float64) float64 {
	if pbr <= 0 || pbr > 10 {
		return 0.0 // 비정상
	}

	if pbr < 0.5 {
		return 1.0
	} else if pbr < 1.0 {
		// 0.5 ~ 1.0 → 0.5 ~ 1.0
		return 0.5 + (1.0-pbr)
	} else if pbr < 2.0 {
		// 1.0 ~ 2.0 → 0 ~ 0.5
		return (2.0 - pbr) / 2.0
	} else {
		// 2.0 ~ 4.0 → 0 ~ -0.5
		return math.Tanh(-(pbr - 2.0) / 4.0)
	}
}

// scorePSR PSR 점수화
// 기준: PSR 1.0
// PSR < 0.5: +1.0 (극도 저평가)
// PSR 0.5~1.0: +0.5 ~ +1.0 (저평가)
// PSR 1.0~3.0: 0 ~ +0.5 (적정)
// PSR > 3.0: -0.5 ~ 0 (고평가)
func (c *ValueCalculator) scorePSR(psr float64) float64 {
	if psr <= 0 || psr > 20 {
		return 0.0 // 비정상
	}

	if psr < 0.5 {
		return 1.0
	} else if psr < 1.0 {
		// 0.5 ~ 1.0 → 0.5 ~ 1.0
		return 0.5 + (1.0-psr)
	} else if psr < 3.0 {
		// 1.0 ~ 3.0 → 0 ~ 0.5
		return (3.0 - psr) / 4.0
	} else {
		// 3.0 ~ 6.0 → 0 ~ -0.5
		return math.Tanh(-(psr - 3.0) / 6.0)
	}
}
