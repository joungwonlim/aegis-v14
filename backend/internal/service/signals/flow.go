package signals

import (
	"context"
	"math"

	"github.com/rs/zerolog/log"
)

// FlowCalculator 수급 시그널 계산기
// SSOT: 수급 시그널 계산은 여기서만
type FlowCalculator struct{}

// NewFlowCalculator 새 수급 계산기 생성
func NewFlowCalculator() *FlowCalculator {
	return &FlowCalculator{}
}

// Calculate 수급 시그널 계산
// 입력: 투자자별 수급 데이터 (flowData[0]이 가장 최근, 최소 20일 필요)
// 출력: 점수 (-1.0 ~ 1.0), 상세 정보, 에러
func (c *FlowCalculator) Calculate(ctx context.Context, code string, flowData []FlowData) (float64, FlowDetails, error) {
	details := FlowDetails{}

	if len(flowData) < 20 {
		return 0.0, details, nil
	}

	// 5일/20일 누적 순매수 계산
	foreignNet5D := c.sumNet(flowData[:5], "foreign")
	foreignNet20D := c.sumNet(flowData[:20], "foreign")
	instNet5D := c.sumNet(flowData[:5], "inst")
	instNet20D := c.sumNet(flowData[:20], "inst")

	details.ForeignNet5D = foreignNet5D
	details.ForeignNet20D = foreignNet20D
	details.InstNet5D = instNet5D
	details.InstNet20D = instNet20D

	// 수급 점수 계산
	score := c.calculateScore(foreignNet5D, foreignNet20D, instNet5D, instNet20D)

	log.Debug().
		Str("code", code).
		Int64("foreign_net_5d", foreignNet5D).
		Int64("foreign_net_20d", foreignNet20D).
		Int64("inst_net_5d", instNet5D).
		Int64("inst_net_20d", instNet20D).
		Float64("score", score).
		Msg("Calculated flow signal")

	return score, details, nil
}

// sumNet 투자자별 순매수 합계
func (c *FlowCalculator) sumNet(data []FlowData, investor string) int64 {
	var sum int64
	for _, d := range data {
		switch investor {
		case "foreign":
			sum += d.ForeignNet
		case "inst":
			sum += d.InstNet
		case "individual":
			sum += d.IndividualNet
		}
	}
	return sum
}

// calculateScore 수급 점수 계산 (-1.0 ~ 1.0)
// 가중치:
// - 외국인 60% (5D 70%, 20D 30%)
// - 기관 40% (5D 70%, 20D 30%)
// 스마트머니 원칙: 외국인/기관 = Smart Money
func (c *FlowCalculator) calculateScore(foreignNet5D, foreignNet20D, instNet5D, instNet20D int64) float64 {
	// tanh 정규화 기준값
	// 5D 기준: 50만주
	// 20D 기준: 200만주
	const base5D = 500_000.0
	const base20D = 2_000_000.0

	// 외국인 점수
	foreignScore5D := math.Tanh(float64(foreignNet5D) / base5D)
	foreignScore20D := math.Tanh(float64(foreignNet20D) / base20D)
	foreignScore := foreignScore5D*0.7 + foreignScore20D*0.3

	// 기관 점수
	instScore5D := math.Tanh(float64(instNet5D) / base5D)
	instScore20D := math.Tanh(float64(instNet20D) / base20D)
	instScore := instScore5D*0.7 + instScore20D*0.3

	// 가중 합산 (외국인 60%, 기관 40%)
	score := foreignScore*0.6 + instScore*0.4

	// 범위 제한
	if score > 1.0 {
		score = 1.0
	} else if score < -1.0 {
		score = -1.0
	}

	return score
}
