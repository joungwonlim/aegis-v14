package signals

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/signals"
)

// EventCalculator 이벤트 시그널 계산기
// SSOT: 이벤트 시그널 계산은 여기서만
type EventCalculator struct{}

// NewEventCalculator 새 이벤트 계산기 생성
func NewEventCalculator() *EventCalculator {
	return &EventCalculator{}
}

// Calculate 이벤트 시그널 계산
// 입력: 이벤트 목록, 기준 날짜
// 출력: 점수 (-1.0 ~ 1.0), 이벤트 목록, 에러
func (c *EventCalculator) Calculate(ctx context.Context, code string, events []signals.EventSignal, date time.Time) (float64, []signals.EventSignal, error) {
	if len(events) == 0 {
		return 0.0, nil, nil
	}

	// 시간 가중 점수 합산
	var totalScore float64
	var totalWeight float64

	for _, event := range events {
		// 이벤트 발생 이후 경과 일수
		daysSince := date.Sub(event.Timestamp).Hours() / 24.0
		if daysSince < 0 {
			continue // 미래 이벤트 무시
		}

		// 시간 가중치 계산 (Exponential Decay)
		timeWeight := c.calculateTimeWeight(daysSince)

		// 이벤트 영향도
		impact := event.Score
		if impact == 0 {
			impact = signals.GetEventImpact(event.Type)
		}

		totalScore += impact * timeWeight
		totalWeight += timeWeight
	}

	// 가중 평균 계산
	var score float64
	if totalWeight > 0 {
		score = totalScore / totalWeight
	}

	// tanh 정규화 (이벤트가 많을수록 영향 증가)
	score = math.Tanh(score * 1.5)

	log.Debug().
		Str("code", code).
		Int("event_count", len(events)).
		Float64("score", score).
		Msg("Calculated event signal")

	return score, events, nil
}

// calculateTimeWeight 시간 가중치 계산 (Exponential Decay)
// 감쇠율 k = 0.023
// 7일 이내: ~100%
// 30일 이내: ~50%
// 90일 이내: ~25%
// 90일 초과: 10% (floor)
func (c *EventCalculator) calculateTimeWeight(daysSince float64) float64 {
	const decayRate = 0.023
	weight := math.Exp(-decayRate * daysSince)

	if weight < 0.1 {
		weight = 0.1 // 최소 가중치
	}

	return weight
}

// MapDisclosureToEventType DART 공시 제목을 이벤트 타입으로 변환
func MapDisclosureToEventType(title string) signals.EventType {
	// 긍정적 이벤트
	if containsAny(title, "자기주식취득", "자사주매입", "자기주식매입", "자기주식신탁") {
		return signals.EventShareBuyback
	}

	if containsAny(title, "사채취득", "조기상환", "사채상환") {
		return signals.EventShareBuyback // 희석 위험 감소
	}

	if containsAny(title, "대량보유상황보고", "주식등의대량보유") {
		return signals.EventPartnership // 기관 관심
	}

	if containsAny(title, "배당", "배당금") {
		return signals.EventDividendIncrease
	}

	if containsAny(title, "신규사업", "신제품", "신규계약") {
		return signals.EventNewProduct
	}

	if containsAny(title, "설비투자", "공장증설", "투자결정") {
		return signals.EventCapexIncrease
	}

	if containsAny(title, "인수", "합병", "경영권") {
		return signals.EventMergerPositive
	}

	if containsAny(title, "MOU", "양해각서", "업무협약", "파트너십", "제휴") {
		return signals.EventPartnership
	}

	if containsAny(title, "특허", "기술이전") {
		return signals.EventPatent
	}

	// 부정적 이벤트
	if containsAny(title, "소송", "소제기", "피소", "손해배상") {
		return signals.EventLawsuit
	}

	if containsAny(title, "감사의견", "감사보고서", "한정의견", "부적정의견") {
		return signals.EventAuditOpinion
	}

	if containsAny(title, "행정처분", "과징금", "제재", "시정명령") {
		return signals.EventRegulatory
	}

	if containsAny(title, "사임", "해임", "퇴임") {
		return signals.EventManagementChange
	}

	if containsAny(title, "리콜", "자진회수") {
		return signals.EventRecall
	}

	// 중립 이벤트
	if containsAny(title, "대표이사", "임원", "선임", "이사회") {
		return signals.EventGeneralNews
	}

	if containsAny(title, "실적", "매출", "영업이익", "순이익", "사업보고서", "반기보고서", "분기보고서") {
		return signals.EventGeneralNews
	}

	if containsAny(title, "유상증자", "무상증자", "증자결정") {
		return signals.EventGeneralNews
	}

	if containsAny(title, "전환사채", "CB발행", "CB)", "신주인수권", "사채권발행") {
		return signals.EventGeneralNews
	}

	if containsAny(title, "주주총회", "임시주총", "정기주총") {
		return signals.EventGeneralNews
	}

	// 기타 일반 공시
	return signals.EventAnnouncement
}

// containsAny 문자열에 키워드 포함 여부 확인
func containsAny(s string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}
