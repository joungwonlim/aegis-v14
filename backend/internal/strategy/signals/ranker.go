package signals

import (
	"sort"

	"github.com/wonny/aegis/v14/internal/domain/signals"
)

// Ranker 신호 순위 매기기
type Ranker struct {
	criteria *signals.SignalCriteria
}

// NewRanker 새 Ranker 생성
func NewRanker(criteria *signals.SignalCriteria) *Ranker {
	return &Ranker{
		criteria: criteria,
	}
}

// RankSignals 신호 순위 매기기
func (r *Ranker) RankSignals(allSignals []signals.Signal) []signals.Signal {
	if len(allSignals) == 0 {
		return allSignals
	}

	// 1. Sort by composite score
	sort.Slice(allSignals, func(i, j int) bool {
		// 복합 점수 = Strength * 0.7 + Conviction * 0.3
		scoreI := float64(allSignals[i].Strength)*0.7 + float64(allSignals[i].Conviction)*0.3
		scoreJ := float64(allSignals[j].Strength)*0.7 + float64(allSignals[j].Conviction)*0.3

		if scoreI != scoreJ {
			return scoreI > scoreJ
		}

		// Tie-breaker: Strength
		if allSignals[i].Strength != allSignals[j].Strength {
			return allSignals[i].Strength > allSignals[j].Strength
		}

		// Tie-breaker: Conviction
		return allSignals[i].Conviction > allSignals[j].Conviction
	})

	// 2. Assign rank
	for i := range allSignals {
		allSignals[i].Rank = i + 1
	}

	return allSignals
}
