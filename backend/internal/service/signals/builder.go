package signals

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/signals"
)

// Builder 6팩터 시그널 빌더 (오케스트레이터)
// SSOT: 시그널 생성 오케스트레이션은 여기서만
type Builder struct {
	// Signal calculators
	momentum  *MomentumCalculator
	technical *TechnicalCalculator
	value     *ValueCalculator
	quality   *QualityCalculator
	flow      *FlowCalculator
	event     *EventCalculator

	// Data readers (외부 데이터 소스 인터페이스)
	priceReader      PriceReader
	flowReader       FlowReader
	financialReader  FinancialReader
	disclosureReader DisclosureReader
}

// PriceReader 가격 데이터 리더
type PriceReader interface {
	GetPriceHistory(ctx context.Context, stockCode string, from, to time.Time) ([]PricePoint, error)
}

// FlowReader 수급 데이터 리더
type FlowReader interface {
	GetFlowHistory(ctx context.Context, stockCode string, from, to time.Time) ([]FlowData, error)
}

// FinancialReader 재무 데이터 리더
type FinancialReader interface {
	GetLatestFinancials(ctx context.Context, stockCode string) (*FinancialData, error)
}

// FinancialData 재무 데이터
type FinancialData struct {
	PER       float64
	PBR       float64
	PSR       float64
	ROE       float64
	DebtRatio float64
}

// DisclosureReader 공시 데이터 리더
type DisclosureReader interface {
	GetDisclosures(ctx context.Context, stockCode string, from, to time.Time) ([]signals.EventSignal, error)
}

// NewBuilder 새 빌더 생성
func NewBuilder(
	priceReader PriceReader,
	flowReader FlowReader,
	financialReader FinancialReader,
	disclosureReader DisclosureReader,
) *Builder {
	return &Builder{
		momentum:         NewMomentumCalculator(),
		technical:        NewTechnicalCalculator(),
		value:            NewValueCalculator(),
		quality:          NewQualityCalculator(),
		flow:             NewFlowCalculator(),
		event:            NewEventCalculator(),
		priceReader:      priceReader,
		flowReader:       flowReader,
		financialReader:  financialReader,
		disclosureReader: disclosureReader,
	}
}

// BuildStockSignals 단일 종목의 6팩터 시그널 계산
func (b *Builder) BuildStockSignals(ctx context.Context, stockCode string, date time.Time) (*signals.FactorScoreRecord, error) {
	record := &signals.FactorScoreRecord{
		Symbol:   stockCode,
		CalcDate: date,
	}

	// 1. 가격 데이터 조회 및 모멘텀/기술적 시그널 계산
	prices, err := b.fetchPriceData(ctx, stockCode, date)
	if err != nil {
		log.Warn().Err(err).Str("code", stockCode).Msg("Failed to fetch price data")
	} else {
		// 모멘텀 계산 (최소 60일 필요)
		if len(prices) >= 60 {
			score, _, err := b.momentum.Calculate(ctx, stockCode, prices)
			if err == nil {
				record.Momentum = score
			}
		}

		// 기술적 지표 계산 (최소 120일 필요)
		if len(prices) >= 120 {
			score, _, err := b.technical.Calculate(ctx, stockCode, prices)
			if err == nil {
				record.Technical = score
			}
		}
	}

	// 2. 재무 데이터 조회 및 가치/품질 시그널 계산
	financials, err := b.financialReader.GetLatestFinancials(ctx, stockCode)
	if err != nil {
		log.Warn().Err(err).Str("code", stockCode).Msg("Failed to fetch financial data")
	} else {
		// 가치 계산
		valueMetrics := ValueMetrics{
			PER: financials.PER,
			PBR: financials.PBR,
			PSR: financials.PSR,
		}
		score, _, err := b.value.Calculate(ctx, stockCode, valueMetrics)
		if err == nil {
			record.Value = score
		}

		// 품질 계산
		qualityMetrics := QualityMetrics{
			ROE:       financials.ROE,
			DebtRatio: financials.DebtRatio,
		}
		score, _, err = b.quality.Calculate(ctx, stockCode, qualityMetrics)
		if err == nil {
			record.Quality = score
		}
	}

	// 3. 수급 데이터 조회 및 수급 시그널 계산
	flowData, err := b.fetchFlowData(ctx, stockCode, date)
	if err != nil {
		log.Warn().Err(err).Str("code", stockCode).Msg("Failed to fetch flow data")
	} else if len(flowData) >= 20 {
		score, _, err := b.flow.Calculate(ctx, stockCode, flowData)
		if err == nil {
			record.Flow = score
		}
	}

	// 4. 공시 데이터 조회 및 이벤트 시그널 계산
	events, err := b.fetchEvents(ctx, stockCode, date)
	if err != nil {
		log.Warn().Err(err).Str("code", stockCode).Msg("Failed to fetch events")
	} else {
		score, _, err := b.event.Calculate(ctx, stockCode, events, date)
		if err == nil {
			record.Event = score
		}
	}

	// 5. 종합 점수 계산 (기본 가중치 적용)
	criteria := signals.DefaultSignalCriteria()
	record.TotalScore = record.Momentum*criteria.MomentumWeight +
		record.Technical*criteria.TechnicalWeight +
		record.Value*criteria.ValueWeight +
		record.Quality*criteria.QualityWeight +
		record.Flow*criteria.FlowWeight +
		record.Event*criteria.EventWeight

	record.UpdatedAt = time.Now()

	return record, nil
}

// BuildAllSignals 전체 종목 시그널 계산
func (b *Builder) BuildAllSignals(ctx context.Context, stockCodes []string, date time.Time) ([]*signals.FactorScoreRecord, error) {
	log.Info().
		Time("date", date).
		Int("stock_count", len(stockCodes)).
		Msg("Starting signal generation for all stocks")

	records := make([]*signals.FactorScoreRecord, 0, len(stockCodes))
	successCount := 0

	for _, code := range stockCodes {
		record, err := b.BuildStockSignals(ctx, code, date)
		if err != nil {
			log.Warn().Err(err).Str("code", code).Msg("Failed to build signals")
			continue
		}

		records = append(records, record)
		successCount++
	}

	log.Info().
		Int("total", len(stockCodes)).
		Int("success", successCount).
		Int("failed", len(stockCodes)-successCount).
		Msg("Signal generation completed")

	return records, nil
}

// fetchPriceData 가격 데이터 조회 (최근 200일)
func (b *Builder) fetchPriceData(ctx context.Context, stockCode string, date time.Time) ([]PricePoint, error) {
	if b.priceReader == nil {
		return nil, fmt.Errorf("price reader not configured")
	}

	// 200 캘린더일 → ~120 거래일
	from := date.AddDate(0, 0, -200)
	to := date

	return b.priceReader.GetPriceHistory(ctx, stockCode, from, to)
}

// fetchFlowData 수급 데이터 조회 (최근 40일)
func (b *Builder) fetchFlowData(ctx context.Context, stockCode string, date time.Time) ([]FlowData, error) {
	if b.flowReader == nil {
		return nil, fmt.Errorf("flow reader not configured")
	}

	// 40 캘린더일 → ~20 거래일
	from := date.AddDate(0, 0, -40)
	to := date

	return b.flowReader.GetFlowHistory(ctx, stockCode, from, to)
}

// fetchEvents 이벤트(공시) 데이터 조회 (최근 90일)
func (b *Builder) fetchEvents(ctx context.Context, stockCode string, date time.Time) ([]signals.EventSignal, error) {
	if b.disclosureReader == nil {
		return nil, fmt.Errorf("disclosure reader not configured")
	}

	// 90일 이내 공시
	from := date.AddDate(0, 0, -90)
	to := date

	return b.disclosureReader.GetDisclosures(ctx, stockCode, from, to)
}
