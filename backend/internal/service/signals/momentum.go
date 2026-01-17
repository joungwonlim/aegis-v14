package signals

import (
	"context"
	"math"

	"github.com/rs/zerolog/log"
)

// MomentumCalculator 모멘텀 시그널 계산기
// SSOT: 모멘텀 시그널 계산은 여기서만
type MomentumCalculator struct{}

// NewMomentumCalculator 새 모멘텀 계산기 생성
func NewMomentumCalculator() *MomentumCalculator {
	return &MomentumCalculator{}
}

// Calculate 모멘텀 시그널 계산
// 입력: 가격 데이터 (prices[0]이 가장 최근)
// 출력: 점수 (-1.0 ~ 1.0), 상세 정보, 에러
func (c *MomentumCalculator) Calculate(ctx context.Context, code string, prices []PricePoint) (float64, MomentumDetails, error) {
	details := MomentumDetails{}

	if len(prices) == 0 {
		return 0.0, details, nil
	}

	// 수익률 계산 (거래일 기준)
	return1M := c.calculateReturn(prices, 20)  // ~1개월 (20 거래일)
	return3M := c.calculateReturn(prices, 60)  // ~3개월 (60 거래일)
	volumeRate := c.calculateVolumeGrowth(prices, 20)

	details.Return1M = return1M
	details.Return3M = return3M
	details.VolumeRate = volumeRate

	// 모멘텀 점수 계산
	score := c.calculateScore(return1M, return3M, volumeRate)

	log.Debug().
		Str("code", code).
		Float64("return_1m", return1M).
		Float64("return_3m", return3M).
		Float64("volume_rate", volumeRate).
		Float64("score", score).
		Msg("Calculated momentum signal")

	return score, details, nil
}

// calculateReturn 기간별 수익률 계산
func (c *MomentumCalculator) calculateReturn(prices []PricePoint, days int) float64 {
	if len(prices) < days+1 {
		return 0.0
	}

	// 현재 가격 (가장 최근)
	currentPrice := prices[0].Price
	if currentPrice == 0 {
		return 0.0
	}

	// N일 전 가격
	pastPrice := prices[days].Price
	if pastPrice == 0 {
		return 0.0
	}

	// 수익률 계산
	ret := (float64(currentPrice) - float64(pastPrice)) / float64(pastPrice)
	return ret
}

// calculateVolumeGrowth 거래량 성장률 계산
func (c *MomentumCalculator) calculateVolumeGrowth(prices []PricePoint, days int) float64 {
	if len(prices) < days*2 {
		return 0.0
	}

	// 최근 기간 평균 거래량
	recentVolume := c.averageVolume(prices[:days])

	// 이전 기간 평균 거래량
	pastVolume := c.averageVolume(prices[days : days*2])

	if pastVolume == 0 {
		return 0.0
	}

	// 성장률 계산
	growth := (recentVolume - pastVolume) / pastVolume
	return growth
}

// averageVolume 평균 거래량 계산
func (c *MomentumCalculator) averageVolume(prices []PricePoint) float64 {
	if len(prices) == 0 {
		return 0.0
	}

	var sum int64
	for _, p := range prices {
		sum += p.Volume
	}

	return float64(sum) / float64(len(prices))
}

// calculateScore 모멘텀 점수 계산 (-1.0 ~ 1.0)
// 가중치: Return1M 40%, Return3M 40%, VolumeRate 20%
func (c *MomentumCalculator) calculateScore(return1M, return3M, volumeRate float64) float64 {
	// 가중 합산
	score := return1M*0.4 + return3M*0.4 + volumeRate*0.2

	// tanh 정규화 (-1 ~ 1)
	// tanh는 (-inf, inf)를 (-1, 1)로 매핑
	// 일반적인 수익률 범위가 -50% ~ +50%이므로 스케일 조정
	normalizedScore := math.Tanh(score * 2)

	// 범위 제한 (tanh가 이미 처리하지만 안전장치)
	if normalizedScore > 1.0 {
		normalizedScore = 1.0
	} else if normalizedScore < -1.0 {
		normalizedScore = -1.0
	}

	return normalizedScore
}
