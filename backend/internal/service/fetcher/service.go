package fetcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/fetcher"
)

// CollectorType 수집기 타입
type CollectorType string

const (
	CollectorPrice      CollectorType = "price"
	CollectorFlow       CollectorType = "flow"
	CollectorFundament  CollectorType = "fundamental"
	CollectorMarketCap  CollectorType = "marketcap"
	CollectorDisclosure CollectorType = "disclosure"
	CollectorRanking    CollectorType = "ranking"
)

// Config 서비스 설정
type Config struct {
	// 수집 간격
	PriceInterval       time.Duration
	FlowInterval        time.Duration
	FundamentalInterval time.Duration
	MarketCapInterval   time.Duration
	DisclosureInterval  time.Duration

	// 배치 크기
	BatchSize int

	// 재시도 설정
	MaxRetries   int
	RetryBackoff time.Duration

	// 동시 수집 제한
	MaxConcurrent int
}

// DefaultConfig 기본 설정
func DefaultConfig() *Config {
	return &Config{
		PriceInterval:       1 * time.Hour,
		FlowInterval:        1 * time.Hour,
		FundamentalInterval: 24 * time.Hour,
		MarketCapInterval:   6 * time.Hour,
		DisclosureInterval:  30 * time.Minute,
		BatchSize:           100,
		MaxRetries:          3,
		RetryBackoff:        5 * time.Second,
		MaxConcurrent:       5,
	}
}

// Service Fetcher 서비스
type Service struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Config
	config *Config

	// DB Pool (for direct queries)
	dbPool *pgxpool.Pool

	// External Clients
	naverClient fetcher.NaverClient
	dartClient  fetcher.DartClient

	// Repositories
	stockRepo       fetcher.StockRepository
	priceRepo       fetcher.PriceRepository
	flowRepo        fetcher.FlowRepository
	fundamentalRepo fetcher.FundamentalsRepository
	marketCapRepo   fetcher.MarketCapRepository
	disclosureRepo  fetcher.DisclosureRepository
	fetchLogRepo    fetcher.FetchLogRepository
	rankingRepo     fetcher.RankingRepository

	// State
	running bool
	mu      sync.RWMutex
}

// NewService 서비스 생성
func NewService(
	ctx context.Context,
	config *Config,
	dbPool *pgxpool.Pool,
	naverClient fetcher.NaverClient,
	dartClient fetcher.DartClient,
	stockRepo fetcher.StockRepository,
	priceRepo fetcher.PriceRepository,
	flowRepo fetcher.FlowRepository,
	fundamentalRepo fetcher.FundamentalsRepository,
	marketCapRepo fetcher.MarketCapRepository,
	disclosureRepo fetcher.DisclosureRepository,
	fetchLogRepo fetcher.FetchLogRepository,
	rankingRepo fetcher.RankingRepository,
) *Service {
	ctx, cancel := context.WithCancel(ctx)

	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		ctx:             ctx,
		cancel:          cancel,
		config:          config,
		dbPool:          dbPool,
		naverClient:     naverClient,
		dartClient:      dartClient,
		stockRepo:       stockRepo,
		priceRepo:       priceRepo,
		flowRepo:        flowRepo,
		fundamentalRepo: fundamentalRepo,
		marketCapRepo:   marketCapRepo,
		disclosureRepo:  disclosureRepo,
		fetchLogRepo:    fetchLogRepo,
		rankingRepo:     rankingRepo,
	}
}

// Start 서비스 시작
func (s *Service) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("service already running")
	}
	s.running = true
	s.mu.Unlock()

	log.Info().Msg("Starting Fetcher service")

	// 백그라운드 수집기 시작
	s.wg.Add(6)
	go s.runCollector(CollectorPrice, s.config.PriceInterval, s.collectPrices)
	go s.runCollector(CollectorFlow, s.config.FlowInterval, s.collectFlows)
	go s.runCollector(CollectorFundament, s.config.FundamentalInterval, s.collectFundamentals)
	go s.runCollector(CollectorMarketCap, s.config.MarketCapInterval, s.collectMarketCaps)
	go s.runCollector(CollectorDisclosure, s.config.DisclosureInterval, s.collectDisclosures)
	go s.runCollector(CollectorRanking, 10*time.Minute, s.collectRankings)

	log.Info().Msg("Fetcher service started")
	return nil
}

// Stop 서비스 중지
func (s *Service) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	log.Info().Msg("Stopping Fetcher service")
	s.cancel()
	s.wg.Wait()
	log.Info().Msg("Fetcher service stopped")

	return nil
}

// runCollector 수집기 실행 루프
func (s *Service) runCollector(collectorType CollectorType, interval time.Duration, collectFunc func(context.Context) error) {
	defer s.wg.Done()

	logger := log.With().Str("collector", string(collectorType)).Logger()
	logger.Info().Dur("interval", interval).Msg("Collector started")

	// 초기 수집
	if err := collectFunc(s.ctx); err != nil {
		logger.Error().Err(err).Msg("Initial collection failed")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := collectFunc(s.ctx); err != nil {
				logger.Error().Err(err).Msg("Collection failed")
			}
		case <-s.ctx.Done():
			logger.Info().Msg("Collector stopped")
			return
		}
	}
}

// collectPrices 가격 데이터 수집
func (s *Service) collectPrices(ctx context.Context) error {
	startTime := time.Now()
	log.Info().Msg("Collecting prices")

	// 활성 종목 조회
	stocks, err := s.stockRepo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("get active stocks: %w", err)
	}

	if len(stocks) == 0 {
		log.Warn().Msg("No active stocks to collect prices")
		return nil
	}

	// 배치 수집
	collected := 0
	failed := 0

	for _, stock := range stocks {
		prices, err := s.naverClient.FetchDailyPrices(ctx, stock.Code, 5) // 최근 5일
		if err != nil {
			log.Warn().Err(err).Str("code", stock.Code).Msg("Failed to fetch prices")
			failed++
			continue
		}

		if len(prices) > 0 {
			count, err := s.priceRepo.UpsertBatch(ctx, prices)
			if err != nil {
				log.Warn().Err(err).Str("code", stock.Code).Msg("Failed to save prices")
				failed++
				continue
			}
			collected += count
		}
	}

	// fetch_logs 기록
	duration := time.Since(startTime)
	finishedAt := time.Now()
	durationMs := int(duration.Milliseconds())
	status := "success"
	if failed > 0 && collected == 0 {
		status = "failed"
	}

	fetchLog := &fetcher.FetchLog{
		JobType:         "collector",
		Source:          "naver",
		TargetTable:     "price",
		RecordsFetched:  collected,
		RecordsInserted: collected,
		RecordsUpdated:  0,
		Status:          status,
		StartedAt:       startTime,
		FinishedAt:      &finishedAt,
		DurationMs:      &durationMs,
	}

	if _, err := s.fetchLogRepo.Create(ctx, fetchLog); err != nil {
		log.Warn().Err(err).Msg("Failed to save fetch log")
	}

	log.Info().
		Int("total_stocks", len(stocks)).
		Int("collected", collected).
		Int("failed", failed).
		Msg("Price collection completed")

	return nil
}

// collectFlows 수급 데이터 수집
func (s *Service) collectFlows(ctx context.Context) error {
	startTime := time.Now()
	log.Info().Msg("Collecting investor flows")

	stocks, err := s.stockRepo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("get active stocks: %w", err)
	}

	if len(stocks) == 0 {
		return nil
	}

	collected := 0
	failed := 0

	for _, stock := range stocks {
		flows, err := s.naverClient.FetchInvestorFlow(ctx, stock.Code, 5)
		if err != nil {
			log.Warn().Err(err).Str("code", stock.Code).Msg("Failed to fetch flows")
			failed++
			continue
		}

		if len(flows) > 0 {
			count, err := s.flowRepo.UpsertBatch(ctx, flows)
			if err != nil {
				log.Warn().Err(err).Str("code", stock.Code).Msg("Failed to save flows")
				failed++
				continue
			}
			collected += count
		}
	}

	// fetch_logs 기록
	duration := time.Since(startTime)
	finishedAt := time.Now()
	durationMs := int(duration.Milliseconds())
	status := "success"
	if failed > 0 && collected == 0 {
		status = "failed"
	}

	fetchLog := &fetcher.FetchLog{
		JobType:         "collector",
		Source:          "naver",
		TargetTable:     "flow",
		RecordsFetched:  collected,
		RecordsInserted: collected,
		RecordsUpdated:  0,
		Status:          status,
		StartedAt:       startTime,
		FinishedAt:      &finishedAt,
		DurationMs:      &durationMs,
	}

	if _, err := s.fetchLogRepo.Create(ctx, fetchLog); err != nil {
		log.Warn().Err(err).Msg("Failed to save fetch log")
	}

	log.Info().
		Int("total_stocks", len(stocks)).
		Int("collected", collected).
		Int("failed", failed).
		Msg("Flow collection completed")

	return nil
}

// collectFundamentals 재무 데이터 수집
func (s *Service) collectFundamentals(ctx context.Context) error {
	startTime := time.Now()
	log.Info().Msg("Collecting fundamentals")

	stocks, err := s.stockRepo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("get active stocks: %w", err)
	}

	if len(stocks) == 0 {
		return nil
	}

	collected := 0
	failed := 0

	for _, stock := range stocks {
		fund, err := s.naverClient.FetchFundamentals(ctx, stock.Code)
		if err != nil {
			log.Warn().Err(err).Str("code", stock.Code).Msg("Failed to fetch fundamentals")
			failed++
			continue
		}

		if fund != nil {
			if err := s.fundamentalRepo.Upsert(ctx, fund); err != nil {
				log.Warn().Err(err).Str("code", stock.Code).Msg("Failed to save fundamentals")
				failed++
				continue
			}
			collected++
		}
	}

	// fetch_logs 기록
	duration := time.Since(startTime)
	finishedAt := time.Now()
	durationMs := int(duration.Milliseconds())
	status := "success"
	if failed > 0 && collected == 0 {
		status = "failed"
	}

	fetchLog := &fetcher.FetchLog{
		JobType:         "collector",
		Source:          "naver",
		TargetTable:     "fundamental",
		RecordsFetched:  collected,
		RecordsInserted: collected,
		RecordsUpdated:  0,
		Status:          status,
		StartedAt:       startTime,
		FinishedAt:      &finishedAt,
		DurationMs:      &durationMs,
	}

	if _, err := s.fetchLogRepo.Create(ctx, fetchLog); err != nil {
		log.Warn().Err(err).Msg("Failed to save fetch log")
	}

	log.Info().
		Int("total_stocks", len(stocks)).
		Int("collected", collected).
		Int("failed", failed).
		Msg("Fundamentals collection completed")

	return nil
}

// collectMarketCaps 시가총액 데이터 수집
func (s *Service) collectMarketCaps(ctx context.Context) error {
	// TODO: 네이버 페이지 파싱 로직 수정 필요
	log.Debug().Msg("Market cap collection skipped (parsing logic needs update)")
	return nil

	/*
	log.Info().Msg("Collecting market caps")

	stocks, err := s.stockRepo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("get active stocks: %w", err)
	}

	if len(stocks) == 0 {
		return nil
	}

	collected := 0
	failed := 0

	for _, stock := range stocks {
		mc, err := s.naverClient.FetchMarketCap(ctx, stock.Code)
		if err != nil {
			log.Warn().Err(err).Str("code", stock.Code).Msg("Failed to fetch market cap")
			failed++
			continue
		}

		if mc != nil {
			if err := s.marketCapRepo.Upsert(ctx, mc); err != nil {
				log.Warn().Err(err).Str("code", stock.Code).Msg("Failed to save market cap")
				failed++
				continue
			}
			collected++
		}
	}

	log.Info().
		Int("total_stocks", len(stocks)).
		Int("collected", collected).
		Int("failed", failed).
		Msg("Market cap collection completed")

	return nil
	*/
}

// collectDisclosures 공시 데이터 수집
func (s *Service) collectDisclosures(ctx context.Context) error {
	startTime := time.Now()
	log.Info().Msg("Collecting disclosures")

	// DART 클라이언트가 없으면 스킵
	if s.dartClient == nil {
		log.Warn().Msg("DART client not configured, skipping disclosure collection")
		return nil
	}

	// 오늘 날짜 기준 공시 수집
	today := time.Now()

	disclosures, err := s.dartClient.FetchAllDisclosures(ctx, today, today)
	if err != nil {
		// fetch_logs에 실패 기록
		duration := time.Since(startTime)
		finishedAt := time.Now()
		durationMs := int(duration.Milliseconds())
		errMsg := err.Error()
		fetchLog := &fetcher.FetchLog{
			JobType:      "collector",
			Source:       "dart",
			TargetTable:  "disclosure",
			Status:       "failed",
			ErrorMessage: &errMsg,
			StartedAt:    startTime,
			FinishedAt:   &finishedAt,
			DurationMs:   &durationMs,
		}
		if _, logErr := s.fetchLogRepo.Create(ctx, fetchLog); logErr != nil {
			log.Warn().Err(logErr).Msg("Failed to save fetch log")
		}
		return fmt.Errorf("fetch disclosures: %w", err)
	}

	if len(disclosures) == 0 {
		log.Info().Msg("No new disclosures found")
		// fetch_logs에 성공 기록 (0건)
		duration := time.Since(startTime)
		finishedAt := time.Now()
		durationMs := int(duration.Milliseconds())
		fetchLog := &fetcher.FetchLog{
			JobType:     "collector",
			Source:      "dart",
			TargetTable: "disclosure",
			Status:      "success",
			StartedAt:   startTime,
			FinishedAt:  &finishedAt,
			DurationMs:  &durationMs,
		}
		if _, err := s.fetchLogRepo.Create(ctx, fetchLog); err != nil {
			log.Warn().Err(err).Msg("Failed to save fetch log")
		}
		return nil
	}

	count, err := s.disclosureRepo.SaveBatch(ctx, disclosures)
	if err != nil {
		// fetch_logs에 실패 기록
		duration := time.Since(startTime)
		finishedAt := time.Now()
		durationMs := int(duration.Milliseconds())
		errMsg := err.Error()
		fetchLog := &fetcher.FetchLog{
			JobType:         "collector",
			Source:          "dart",
			TargetTable:     "disclosure",
			RecordsFetched:  len(disclosures),
			Status:          "failed",
			ErrorMessage:    &errMsg,
			StartedAt:       startTime,
			FinishedAt:      &finishedAt,
			DurationMs:      &durationMs,
		}
		if _, logErr := s.fetchLogRepo.Create(ctx, fetchLog); logErr != nil {
			log.Warn().Err(logErr).Msg("Failed to save fetch log")
		}
		return fmt.Errorf("save disclosures: %w", err)
	}

	// fetch_logs에 성공 기록
	duration := time.Since(startTime)
	finishedAt := time.Now()
	durationMs := int(duration.Milliseconds())
	fetchLog := &fetcher.FetchLog{
		JobType:         "collector",
		Source:          "dart",
		TargetTable:     "disclosure",
		RecordsFetched:  len(disclosures),
		RecordsInserted: count,
		RecordsUpdated:  0,
		Status:          "success",
		StartedAt:       startTime,
		FinishedAt:      &finishedAt,
		DurationMs:      &durationMs,
	}

	if _, err := s.fetchLogRepo.Create(ctx, fetchLog); err != nil {
		log.Warn().Err(err).Msg("Failed to save fetch log")
	}

	log.Info().
		Int("fetched", len(disclosures)).
		Int("saved", count).
		Msg("Disclosure collection completed")

	return nil
}

// CollectNow 즉시 수집 (특정 타입)
func (s *Service) CollectNow(ctx context.Context, collectorType CollectorType) error {
	switch collectorType {
	case CollectorPrice:
		return s.collectPrices(ctx)
	case CollectorFlow:
		return s.collectFlows(ctx)
	case CollectorFundament:
		return s.collectFundamentals(ctx)
	case CollectorMarketCap:
		return s.collectMarketCaps(ctx)
	case CollectorDisclosure:
		return s.collectDisclosures(ctx)
	case CollectorRanking:
		return s.collectRankings(ctx)
	default:
		return fmt.Errorf("unknown collector type: %s", collectorType)
	}
}

// CollectStock 특정 종목 데이터 수집
func (s *Service) CollectStock(ctx context.Context, stockCode string) (*fetcher.FetchResult, error) {
	log.Info().Str("stock_code", stockCode).Msg("Collecting data for stock")

	startTime := time.Now()
	result := &fetcher.FetchResult{
		Source: "naver",
		Target: stockCode,
	}

	// 가격 수집
	prices, err := s.naverClient.FetchDailyPrices(ctx, stockCode, 30)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch prices")
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	} else if len(prices) > 0 {
		count, _ := s.priceRepo.UpsertBatch(ctx, prices)
		result.SuccessCount += count
		result.TotalCount += len(prices)
	}

	// 수급 수집
	flows, err := s.naverClient.FetchInvestorFlow(ctx, stockCode, 30)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch flows")
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	} else if len(flows) > 0 {
		count, _ := s.flowRepo.UpsertBatch(ctx, flows)
		result.SuccessCount += count
		result.TotalCount += len(flows)
	}

	// 재무 수집
	fund, err := s.naverClient.FetchFundamentals(ctx, stockCode)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch fundamentals")
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	} else if fund != nil {
		if err := s.fundamentalRepo.Upsert(ctx, fund); err == nil {
			result.SuccessCount++
			result.TotalCount++
		}
	}

	// 시가총액 수집
	mc, err := s.naverClient.FetchMarketCap(ctx, stockCode)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch market cap")
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	} else if mc != nil {
		if err := s.marketCapRepo.Upsert(ctx, mc); err == nil {
			result.SuccessCount++
			result.TotalCount++
		}
	}

	result.Duration = time.Since(startTime).Seconds()
	result.CompletedAt = time.Now()

	log.Info().
		Str("stock_code", stockCode).
		Int("success_count", result.SuccessCount).
		Float64("duration", result.Duration).
		Msg("Stock data collection completed")

	return result, nil
}

// GetLatestPrice 최신 가격 조회
func (s *Service) GetLatestPrice(ctx context.Context, stockCode string) (*fetcher.DailyPrice, error) {
	return s.priceRepo.GetLatest(ctx, stockCode)
}

// GetPriceRange 기간별 가격 조회
func (s *Service) GetPriceRange(ctx context.Context, stockCode string, from, to time.Time) ([]*fetcher.DailyPrice, error) {
	return s.priceRepo.GetRange(ctx, stockCode, from, to)
}

// GetLatestFlow 최신 수급 조회
func (s *Service) GetLatestFlow(ctx context.Context, stockCode string) (*fetcher.InvestorFlow, error) {
	return s.flowRepo.GetLatest(ctx, stockCode)
}

// GetFlowRange 기간별 수급 조회
func (s *Service) GetFlowRange(ctx context.Context, stockCode string, from, to time.Time) ([]*fetcher.InvestorFlow, error) {
	return s.flowRepo.GetRange(ctx, stockCode, from, to)
}

// GetLatestFundamentals 최신 재무 조회
func (s *Service) GetLatestFundamentals(ctx context.Context, stockCode string) (*fetcher.Fundamentals, error) {
	return s.fundamentalRepo.GetLatest(ctx, stockCode)
}

// GetLatestMarketCap 최신 시가총액 조회
func (s *Service) GetLatestMarketCap(ctx context.Context, stockCode string) (*fetcher.MarketCap, error) {
	return s.marketCapRepo.GetLatest(ctx, stockCode)
}

// GetRecentDisclosures 최근 공시 조회
func (s *Service) GetRecentDisclosures(ctx context.Context, limit int) ([]*fetcher.Disclosure, error) {
	return s.disclosureRepo.GetRecent(ctx, limit)
}

// GetStockDisclosures 종목별 공시 조회
func (s *Service) GetStockDisclosures(ctx context.Context, stockCode string, from, to time.Time) ([]*fetcher.Disclosure, error) {
	return s.disclosureRepo.GetByStock(ctx, stockCode, from, to)
}

// RefreshStockMaster 종목 마스터 갱신
func (s *Service) RefreshStockMaster(ctx context.Context) error {
	log.Info().Msg("Refreshing stock master")

	// 시가총액 상위 종목 수집 (KOSPI/KOSDAQ)
	for _, market := range []string{"KOSPI", "KOSDAQ"} {
		stocks, err := s.naverClient.FetchMarketCapRanking(ctx, market, 500)
		if err != nil {
			log.Warn().Err(err).Str("market", market).Msg("Failed to fetch market cap ranking")
			continue
		}

		count, err := s.stockRepo.UpsertBatch(ctx, stocks)
		if err != nil {
			log.Warn().Err(err).Str("market", market).Msg("Failed to save stocks")
			continue
		}

		log.Info().
			Str("market", market).
			Int("count", count).
			Msg("Stock master updated")
	}

	return nil
}

// GetStock 종목 조회
func (s *Service) GetStock(ctx context.Context, code string) (*fetcher.Stock, error) {
	return s.stockRepo.GetByCode(ctx, code)
}

// ListStocks 종목 목록 조회
func (s *Service) ListStocks(ctx context.Context, filter *fetcher.StockFilter) ([]*fetcher.Stock, error) {
	return s.stockRepo.List(ctx, filter)
}

// collectRankings 순위 데이터 수집
func (s *Service) collectRankings(ctx context.Context) error {
	startTime := time.Now()
	log.Info().Msg("Collecting stock rankings")

	categories := []string{"volume", "trading_value", "gainers", "foreign_net_buy", "inst_net_buy", "volume_surge", "high_52week"}
	markets := []string{"ALL", "KOSPI", "KOSDAQ"}

	totalCollected := 0
	totalFailed := 0

	for _, category := range categories {
		for _, market := range markets {
			rankings, err := s.collectRankingByCategory(ctx, category, market, 100)
			if err != nil {
				log.Error().
					Err(err).
					Str("category", category).
					Str("market", market).
					Msg("Failed to collect ranking")
				totalFailed++
				continue
			}

			if len(rankings) == 0 {
				log.Warn().
					Str("category", category).
					Str("market", market).
					Msg("No rankings collected")
				continue
			}

			// SaveBatch로 저장
			err = s.rankingRepo.SaveBatch(ctx, category, market, rankings)
			if err != nil {
				log.Error().
					Err(err).
					Str("category", category).
					Str("market", market).
					Int("count", len(rankings)).
					Msg("Failed to save rankings")
				totalFailed++
				continue
			}

			log.Debug().
				Str("category", category).
				Str("market", market).
				Int("count", len(rankings)).
				Msg("Rankings saved")

			totalCollected += len(rankings)
		}
	}

	// fetch_logs 기록
	duration := time.Since(startTime)
	finishedAt := time.Now()
	durationMs := int(duration.Milliseconds())
	status := "success"
	if totalFailed > 0 && totalCollected == 0 {
		status = "failed"
	}

	fetchLog := &fetcher.FetchLog{
		JobType:         "collector",
		Source:          "database",
		TargetTable:     "ranking",
		RecordsFetched:  totalCollected,
		RecordsInserted: totalCollected,
		RecordsUpdated:  0,
		Status:          status,
		StartedAt:       startTime,
		FinishedAt:      &finishedAt,
		DurationMs:      &durationMs,
	}

	if _, err := s.fetchLogRepo.Create(ctx, fetchLog); err != nil {
		log.Warn().Err(err).Msg("Failed to save fetch log")
	}

	log.Info().
		Int("total_collected", totalCollected).
		Int("total_failed", totalFailed).
		Msg("Ranking collection completed")

	return nil
}

// ScheduleInfo 스케줄 정보
type ScheduleInfo struct {
	CollectorType CollectorType `json:"collector_type"`
	DisplayName   string        `json:"display_name"`
	Interval      string        `json:"interval"` // "10m", "1h", "24h" 등
	IntervalSec   int64         `json:"interval_sec"`
}

// GetSchedules 모든 수집기 스케줄 정보 반환
func (s *Service) GetSchedules() []ScheduleInfo {
	schedules := []ScheduleInfo{
		{
			CollectorType: CollectorRanking,
			DisplayName:   "네이버 순위",
			Interval:      "10분",
			IntervalSec:   int64(10 * time.Minute / time.Second),
		},
		{
			CollectorType: CollectorPrice,
			DisplayName:   "가격 데이터",
			Interval:      formatDuration(s.config.PriceInterval),
			IntervalSec:   int64(s.config.PriceInterval / time.Second),
		},
		{
			CollectorType: CollectorFlow,
			DisplayName:   "수급 데이터",
			Interval:      formatDuration(s.config.FlowInterval),
			IntervalSec:   int64(s.config.FlowInterval / time.Second),
		},
		{
			CollectorType: CollectorFundament,
			DisplayName:   "재무 데이터",
			Interval:      formatDuration(s.config.FundamentalInterval),
			IntervalSec:   int64(s.config.FundamentalInterval / time.Second),
		},
		{
			CollectorType: CollectorDisclosure,
			DisplayName:   "DART 공시",
			Interval:      formatDuration(s.config.DisclosureInterval),
			IntervalSec:   int64(s.config.DisclosureInterval / time.Second),
		},
	}
	return schedules
}

// formatDuration Duration을 읽기 쉬운 문자열로 변환
func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d분", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d시간", int(d.Hours()))
	}
	return fmt.Sprintf("%d일", int(d.Hours()/24))
}

// collectRankingByCategory 카테고리별 순위 데이터 수집 (stock_rankings.go 쿼리 재사용)
func (s *Service) collectRankingByCategory(ctx context.Context, category, market string, limit int) ([]*fetcher.RankingStock, error) {
	marketFilter := ""
	if market == "KOSPI" {
		marketFilter = "AND s.market = 'KOSPI'"
	} else if market == "KOSDAQ" {
		marketFilter = "AND s.market = 'KOSDAQ'"
	}

	var query string

	switch category {
	case "volume":
		query = fmt.Sprintf(`
			WITH stock_prices AS (
				SELECT
					stock_code,
					trade_date,
					close_price,
					volume,
					high_price,
					low_price,
					ROW_NUMBER() OVER (PARTITION BY stock_code ORDER BY trade_date DESC) as rn
				FROM data.daily_prices
				WHERE trade_date >= NOW() - INTERVAL '30 days'
			),
			latest_two AS (
				SELECT
					stock_code,
					MAX(CASE WHEN rn = 1 THEN close_price END) as current_price,
					MAX(CASE WHEN rn = 2 THEN close_price END) as prev_close,
					MAX(CASE WHEN rn = 1 THEN volume END) as volume,
					MAX(CASE WHEN rn = 1 THEN high_price END) as high_price,
					MAX(CASE WHEN rn = 1 THEN low_price END) as low_price
				FROM stock_prices
				WHERE rn <= 2
				GROUP BY stock_code
				HAVING MAX(CASE WHEN rn = 1 THEN close_price END) IS NOT NULL
			)
			SELECT
				ROW_NUMBER() OVER (ORDER BY lt.volume DESC) as rank,
				s.code as stock_code,
				s.name as stock_name,
				s.market,
				lt.current_price,
				COALESCE(((lt.current_price - lt.prev_close) / NULLIF(lt.prev_close, 0) * 100), 0) as change_rate,
				lt.volume,
				(lt.current_price * lt.volume)::bigint as trading_value,
				lt.high_price,
				lt.low_price
			FROM data.stocks s
			JOIN latest_two lt ON s.code = lt.stock_code
			WHERE s.status = 'active'
			  AND s.market NOT IN ('ETF', 'ETN')
			  AND s.name NOT LIKE '%%스팩%%'
			  AND s.name NOT LIKE '%%SPAC%%'
			  AND lt.volume > 0
			  %s
			ORDER BY lt.volume DESC
			LIMIT $1
		`, marketFilter)

	case "trading_value":
		query = fmt.Sprintf(`
			WITH stock_prices AS (
				SELECT
					stock_code,
					trade_date,
					close_price,
					volume,
					high_price,
					low_price,
					ROW_NUMBER() OVER (PARTITION BY stock_code ORDER BY trade_date DESC) as rn
				FROM data.daily_prices
				WHERE trade_date >= NOW() - INTERVAL '30 days'
			),
			latest_two AS (
				SELECT
					stock_code,
					MAX(CASE WHEN rn = 1 THEN close_price END) as current_price,
					MAX(CASE WHEN rn = 2 THEN close_price END) as prev_close,
					MAX(CASE WHEN rn = 1 THEN volume END) as volume,
					MAX(CASE WHEN rn = 1 THEN high_price END) as high_price,
					MAX(CASE WHEN rn = 1 THEN low_price END) as low_price
				FROM stock_prices
				WHERE rn <= 2
				GROUP BY stock_code
				HAVING MAX(CASE WHEN rn = 1 THEN close_price END) IS NOT NULL
			)
			SELECT
				ROW_NUMBER() OVER (ORDER BY (lt.current_price * lt.volume) DESC) as rank,
				s.code as stock_code,
				s.name as stock_name,
				s.market,
				lt.current_price,
				COALESCE(((lt.current_price - lt.prev_close) / NULLIF(lt.prev_close, 0) * 100), 0) as change_rate,
				lt.volume,
				(lt.current_price * lt.volume)::bigint as trading_value,
				lt.high_price,
				lt.low_price
			FROM data.stocks s
			JOIN latest_two lt ON s.code = lt.stock_code
			WHERE s.status = 'active'
			  AND s.market NOT IN ('ETF', 'ETN')
			  AND s.name NOT LIKE '%%스팩%%'
			  AND s.name NOT LIKE '%%SPAC%%'
			  AND lt.volume > 0
			  %s
			ORDER BY (lt.current_price * lt.volume) DESC
			LIMIT $1
		`, marketFilter)

	case "gainers":
		query = fmt.Sprintf(`
			WITH stock_prices AS (
				SELECT
					stock_code,
					trade_date,
					close_price,
					volume,
					high_price,
					low_price,
					ROW_NUMBER() OVER (PARTITION BY stock_code ORDER BY trade_date DESC) as rn
				FROM data.daily_prices
				WHERE trade_date >= NOW() - INTERVAL '30 days'
			),
			latest_two AS (
				SELECT
					stock_code,
					MAX(CASE WHEN rn = 1 THEN close_price END) as current_price,
					MAX(CASE WHEN rn = 2 THEN close_price END) as prev_close,
					MAX(CASE WHEN rn = 1 THEN volume END) as volume,
					MAX(CASE WHEN rn = 1 THEN high_price END) as high_price,
					MAX(CASE WHEN rn = 1 THEN low_price END) as low_price
				FROM stock_prices
				WHERE rn <= 2
				GROUP BY stock_code
				HAVING MAX(CASE WHEN rn = 1 THEN close_price END) IS NOT NULL
				   AND MAX(CASE WHEN rn = 2 THEN close_price END) IS NOT NULL
			)
			SELECT
				ROW_NUMBER() OVER (ORDER BY ((lt.current_price - lt.prev_close) / lt.prev_close * 100) DESC) as rank,
				s.code as stock_code,
				s.name as stock_name,
				s.market,
				lt.current_price,
				((lt.current_price - lt.prev_close) / lt.prev_close * 100) as change_rate,
				lt.volume,
				(lt.current_price * lt.volume) as trading_value,
				lt.high_price,
				lt.low_price
			FROM data.stocks s
			JOIN latest_two lt ON s.code = lt.stock_code
			WHERE s.status = 'active'
			  AND s.market NOT IN ('ETF', 'ETN')
			  AND s.name NOT LIKE '%%스팩%%'
			  AND s.name NOT LIKE '%%SPAC%%'
			  AND lt.prev_close > 0
			  AND lt.current_price > lt.prev_close
			  %s
			ORDER BY change_rate DESC
			LIMIT $1
		`, marketFilter)

	case "foreign_net_buy":
		query = fmt.Sprintf(`
			WITH latest_flow AS (
				SELECT DISTINCT ON (stock_code)
					stock_code,
					trade_date,
					foreign_net_value
				FROM data.investor_flow
				WHERE trade_date >= NOW() - INTERVAL '365 days'
				  AND foreign_net_value > 0
				ORDER BY stock_code, trade_date DESC
			),
			stock_prices AS (
				SELECT
					stock_code,
					trade_date,
					close_price,
					high_price,
					low_price,
					volume,
					ROW_NUMBER() OVER (PARTITION BY stock_code ORDER BY trade_date DESC) as rn
				FROM data.daily_prices
				WHERE trade_date >= NOW() - INTERVAL '30 days'
			),
			latest_two AS (
				SELECT
					stock_code,
					MAX(CASE WHEN rn = 1 THEN close_price END) as current_price,
					MAX(CASE WHEN rn = 2 THEN close_price END) as prev_close,
					MAX(CASE WHEN rn = 1 THEN high_price END) as high_price,
					MAX(CASE WHEN rn = 1 THEN low_price END) as low_price,
					MAX(CASE WHEN rn = 1 THEN volume END) as volume
				FROM stock_prices
				WHERE rn <= 2
				GROUP BY stock_code
			)
			SELECT
				ROW_NUMBER() OVER (ORDER BY lf.foreign_net_value DESC) as rank,
				s.code as stock_code,
				s.name as stock_name,
				s.market,
				lp.current_price,
				COALESCE(((lp.current_price - lp.prev_close) / NULLIF(lp.prev_close, 0) * 100), 0) as change_rate,
				lp.high_price,
				lp.low_price,
				lp.volume,
				(lp.current_price * lp.volume)::bigint as trading_value,
				lf.foreign_net_value
			FROM data.stocks s
			JOIN latest_flow lf ON s.code = lf.stock_code
			LEFT JOIN latest_two lp ON s.code = lp.stock_code
			WHERE s.status = 'active'
			  AND s.market NOT IN ('ETF', 'ETN')
			  AND s.name NOT LIKE '%%스팩%%'
			  AND s.name NOT LIKE '%%SPAC%%'
			  %s
			ORDER BY lf.foreign_net_value DESC
			LIMIT $1
		`, marketFilter)

	case "inst_net_buy":
		query = fmt.Sprintf(`
			WITH latest_flow AS (
				SELECT DISTINCT ON (stock_code)
					stock_code,
					trade_date,
					inst_net_value
				FROM data.investor_flow
				WHERE trade_date >= NOW() - INTERVAL '365 days'
				  AND inst_net_value > 0
				ORDER BY stock_code, trade_date DESC
			),
			stock_prices AS (
				SELECT
					stock_code,
					trade_date,
					close_price,
					high_price,
					low_price,
					volume,
					ROW_NUMBER() OVER (PARTITION BY stock_code ORDER BY trade_date DESC) as rn
				FROM data.daily_prices
				WHERE trade_date >= NOW() - INTERVAL '30 days'
			),
			latest_two AS (
				SELECT
					stock_code,
					MAX(CASE WHEN rn = 1 THEN close_price END) as current_price,
					MAX(CASE WHEN rn = 2 THEN close_price END) as prev_close,
					MAX(CASE WHEN rn = 1 THEN high_price END) as high_price,
					MAX(CASE WHEN rn = 1 THEN low_price END) as low_price,
					MAX(CASE WHEN rn = 1 THEN volume END) as volume
				FROM stock_prices
				WHERE rn <= 2
				GROUP BY stock_code
			)
			SELECT
				ROW_NUMBER() OVER (ORDER BY lf.inst_net_value DESC) as rank,
				s.code as stock_code,
				s.name as stock_name,
				s.market,
				lp.current_price,
				COALESCE(((lp.current_price - lp.prev_close) / NULLIF(lp.prev_close, 0) * 100), 0) as change_rate,
				lp.high_price,
				lp.low_price,
				lp.volume,
				(lp.current_price * lp.volume)::bigint as trading_value,
				lf.inst_net_value
			FROM data.stocks s
			JOIN latest_flow lf ON s.code = lf.stock_code
			LEFT JOIN latest_two lp ON s.code = lp.stock_code
			WHERE s.status = 'active'
			  AND s.market NOT IN ('ETF', 'ETN')
			  AND s.name NOT LIKE '%%스팩%%'
			  AND s.name NOT LIKE '%%SPAC%%'
			  %s
			ORDER BY lf.inst_net_value DESC
			LIMIT $1
		`, marketFilter)

	case "volume_surge":
		query = fmt.Sprintf(`
			WITH stock_volumes AS (
				SELECT
					stock_code,
					trade_date,
					volume,
					close_price,
					high_price,
					low_price,
					ROW_NUMBER() OVER (PARTITION BY stock_code ORDER BY trade_date DESC) as rn
				FROM data.daily_prices
				WHERE trade_date >= NOW() - INTERVAL '30 days'
			),
			latest_two AS (
				SELECT
					stock_code,
					MAX(CASE WHEN rn = 1 THEN volume END) as current_volume,
					MAX(CASE WHEN rn = 2 THEN volume END) as prev_volume,
					MAX(CASE WHEN rn = 1 THEN close_price END) as current_price,
					MAX(CASE WHEN rn = 2 THEN close_price END) as prev_close,
					MAX(CASE WHEN rn = 1 THEN high_price END) as high_price,
					MAX(CASE WHEN rn = 1 THEN low_price END) as low_price,
					MAX(CASE WHEN rn = 1 THEN (close_price * volume) END) as trading_value
				FROM stock_volumes
				WHERE rn <= 2
				GROUP BY stock_code
				HAVING MAX(CASE WHEN rn = 1 THEN volume END) IS NOT NULL
				   AND MAX(CASE WHEN rn = 2 THEN volume END) IS NOT NULL
				   AND MAX(CASE WHEN rn = 2 THEN volume END) > 0
			)
			SELECT
				ROW_NUMBER() OVER (ORDER BY ((lt.current_volume - lt.prev_volume)::float / lt.prev_volume * 100) DESC) as rank,
				s.code as stock_code,
				s.name as stock_name,
				s.market,
				lt.current_price,
				COALESCE(((lt.current_price - lt.prev_close) / NULLIF(lt.prev_close, 0) * 100), 0) as change_rate,
				lt.current_volume,
				lt.high_price,
				lt.low_price,
				lt.trading_value,
				((lt.current_volume - lt.prev_volume)::float / lt.prev_volume * 100) as volume_surge_rate
			FROM data.stocks s
			JOIN latest_two lt ON s.code = lt.stock_code
			WHERE s.status = 'active'
			  AND s.market NOT IN ('ETF', 'ETN')
			  AND s.name NOT LIKE '%%스팩%%'
			  AND s.name NOT LIKE '%%SPAC%%'
			  AND lt.current_volume > lt.prev_volume
			  %s
			ORDER BY volume_surge_rate DESC
			LIMIT $1
		`, marketFilter)

	case "high_52week":
		query = fmt.Sprintf(`
			WITH stock_prices AS (
				SELECT
					stock_code,
					trade_date,
					close_price,
					high_price,
					low_price,
					volume,
					ROW_NUMBER() OVER (PARTITION BY stock_code ORDER BY trade_date DESC) as rn
				FROM data.daily_prices
				WHERE trade_date >= NOW() - INTERVAL '30 days'
				  AND high_price > 0
				  AND low_price > 0
				  AND volume > 0
			),
			latest_two AS (
				SELECT
					stock_code,
					MAX(CASE WHEN rn = 1 THEN close_price END) as current_price,
					MAX(CASE WHEN rn = 2 THEN close_price END) as prev_close,
					MAX(CASE WHEN rn = 1 THEN high_price END) as high_price,
					MAX(CASE WHEN rn = 1 THEN low_price END) as low_price,
					MAX(CASE WHEN rn = 1 THEN volume END) as volume
				FROM stock_prices
				WHERE rn <= 2
				GROUP BY stock_code
			),
			stock_52week_high AS (
				SELECT
					stock_code,
					MAX(high_price) as high_52week
				FROM data.daily_prices
				WHERE trade_date >= NOW() - INTERVAL '52 weeks'
				  AND high_price > 0
				GROUP BY stock_code
			)
			SELECT
				ROW_NUMBER() OVER (ORDER BY (lp.current_price / s52.high_52week) DESC) as rank,
				s.code as stock_code,
				s.name as stock_name,
				s.market,
				lp.current_price,
				COALESCE(((lp.current_price - lp.prev_close) / NULLIF(lp.prev_close, 0) * 100), 0) as change_rate,
				lp.high_price,
				lp.low_price,
				lp.volume,
				(lp.current_price * lp.volume)::bigint as trading_value,
				s52.high_52week
			FROM data.stocks s
			JOIN stock_52week_high s52 ON s.code = s52.stock_code
			JOIN latest_two lp ON s.code = lp.stock_code
			WHERE s.status = 'active'
			  AND s.market NOT IN ('ETF', 'ETN')
			  AND s.name NOT LIKE '%%스팩%%'
			  AND s.name NOT LIKE '%%SPAC%%'
			  AND s52.high_52week > 0
			  AND lp.current_price > 0
			  %s
			ORDER BY (lp.current_price / s52.high_52week) DESC
			LIMIT $1
		`, marketFilter)

	default:
		return nil, fmt.Errorf("unsupported category: %s", category)
	}

	// DB에서 직접 조회
	rows, err := s.dbPool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query rankings: %w", err)
	}
	defer rows.Close()

	var rankings []*fetcher.RankingStock
	for rows.Next() {
		r := &fetcher.RankingStock{
			Category: category,
			Market:   market,
		}

		// 카테고리별 스캔
		switch category {
		case "volume", "trading_value":
			// SELECT: rank, stock_code, stock_name, market, current_price, change_rate, volume, trading_value, high_price, low_price
			var currentPrice, changeRate, highPrice, lowPrice float64
			var volume, tradingValue int64
			err = rows.Scan(
				&r.Rank, &r.StockCode, &r.StockName, &r.Market,
				&currentPrice, &changeRate, &volume, &tradingValue, &highPrice, &lowPrice,
			)
			r.CurrentPrice = &currentPrice
			r.ChangeRate = &changeRate
			r.Volume = &volume
			r.TradingValue = &tradingValue
			r.HighPrice = &highPrice
			r.LowPrice = &lowPrice

		case "gainers":
			// SELECT: rank, stock_code, stock_name, market, current_price, change_rate, volume, trading_value, high_price, low_price
			var currentPrice, changeRate, highPrice, lowPrice float64
			var volume, tradingValue int64
			err = rows.Scan(
				&r.Rank, &r.StockCode, &r.StockName, &r.Market,
				&currentPrice, &changeRate, &volume, &tradingValue, &highPrice, &lowPrice,
			)
			r.CurrentPrice = &currentPrice
			r.ChangeRate = &changeRate
			r.Volume = &volume
			r.TradingValue = &tradingValue
			r.HighPrice = &highPrice
			r.LowPrice = &lowPrice

		case "foreign_net_buy", "inst_net_buy":
			// SELECT: rank, stock_code, stock_name, market, current_price, change_rate, high_price, low_price, volume, trading_value, net_value
			var currentPrice, changeRate, highPrice, lowPrice *float64
			var volume, tradingValue, netValue *int64
			err = rows.Scan(
				&r.Rank, &r.StockCode, &r.StockName, &r.Market,
				&currentPrice, &changeRate, &highPrice, &lowPrice, &volume, &tradingValue, &netValue,
			)
			r.CurrentPrice = currentPrice
			r.ChangeRate = changeRate
			r.HighPrice = highPrice
			r.LowPrice = lowPrice
			r.Volume = volume
			r.TradingValue = tradingValue
			if category == "foreign_net_buy" {
				r.ForeignNetValue = netValue
			} else {
				r.InstNetValue = netValue
			}

		case "volume_surge":
			// SELECT: rank, stock_code, stock_name, market, current_price, change_rate, current_volume, high_price, low_price, trading_value, volume_surge_rate
			var currentPrice, changeRate, highPrice, lowPrice, volumeSurgeRate float64
			var volume, tradingValue int64
			err = rows.Scan(
				&r.Rank, &r.StockCode, &r.StockName, &r.Market,
				&currentPrice, &changeRate, &volume, &highPrice, &lowPrice, &tradingValue, &volumeSurgeRate,
			)
			r.CurrentPrice = &currentPrice
			r.ChangeRate = &changeRate
			r.Volume = &volume
			r.HighPrice = &highPrice
			r.LowPrice = &lowPrice
			r.TradingValue = &tradingValue
			r.VolumeSurgeRate = &volumeSurgeRate

		case "high_52week":
			// SELECT: rank, stock_code, stock_name, market, current_price, change_rate, high_price, low_price, volume, trading_value, high_52week
			var currentPrice, changeRate, highPrice, lowPrice, high52Week float64
			var volume, tradingValue int64
			err = rows.Scan(
				&r.Rank, &r.StockCode, &r.StockName, &r.Market,
				&currentPrice, &changeRate, &highPrice, &lowPrice, &volume, &tradingValue, &high52Week,
			)
			r.CurrentPrice = &currentPrice
			r.ChangeRate = &changeRate
			r.HighPrice = &highPrice
			r.LowPrice = &lowPrice
			r.Volume = &volume
			r.TradingValue = &tradingValue
			r.High52Week = &high52Week
		}

		if err != nil {
			log.Error().Err(err).Msg("Failed to scan ranking row")
			continue
		}

		rankings = append(rankings, r)
	}

	return rankings, nil
}
