package fetcher

import (
	"context"
	"fmt"
	"sync"
	"time"

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

	// State
	running bool
	mu      sync.RWMutex
}

// NewService 서비스 생성
func NewService(
	ctx context.Context,
	config *Config,
	naverClient fetcher.NaverClient,
	dartClient fetcher.DartClient,
	stockRepo fetcher.StockRepository,
	priceRepo fetcher.PriceRepository,
	flowRepo fetcher.FlowRepository,
	fundamentalRepo fetcher.FundamentalsRepository,
	marketCapRepo fetcher.MarketCapRepository,
	disclosureRepo fetcher.DisclosureRepository,
	fetchLogRepo fetcher.FetchLogRepository,
) *Service {
	ctx, cancel := context.WithCancel(ctx)

	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		ctx:             ctx,
		cancel:          cancel,
		config:          config,
		naverClient:     naverClient,
		dartClient:      dartClient,
		stockRepo:       stockRepo,
		priceRepo:       priceRepo,
		flowRepo:        flowRepo,
		fundamentalRepo: fundamentalRepo,
		marketCapRepo:   marketCapRepo,
		disclosureRepo:  disclosureRepo,
		fetchLogRepo:    fetchLogRepo,
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
	s.wg.Add(5)
	go s.runCollector(CollectorPrice, s.config.PriceInterval, s.collectPrices)
	go s.runCollector(CollectorFlow, s.config.FlowInterval, s.collectFlows)
	go s.runCollector(CollectorFundament, s.config.FundamentalInterval, s.collectFundamentals)
	go s.runCollector(CollectorMarketCap, s.config.MarketCapInterval, s.collectMarketCaps)
	go s.runCollector(CollectorDisclosure, s.config.DisclosureInterval, s.collectDisclosures)

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

	log.Info().
		Int("total_stocks", len(stocks)).
		Int("collected", collected).
		Int("failed", failed).
		Msg("Price collection completed")

	return nil
}

// collectFlows 수급 데이터 수집
func (s *Service) collectFlows(ctx context.Context) error {
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

	log.Info().
		Int("total_stocks", len(stocks)).
		Int("collected", collected).
		Int("failed", failed).
		Msg("Flow collection completed")

	return nil
}

// collectFundamentals 재무 데이터 수집
func (s *Service) collectFundamentals(ctx context.Context) error {
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
		return fmt.Errorf("fetch disclosures: %w", err)
	}

	if len(disclosures) == 0 {
		log.Info().Msg("No new disclosures found")
		return nil
	}

	count, err := s.disclosureRepo.SaveBatch(ctx, disclosures)
	if err != nil {
		return fmt.Errorf("save disclosures: %w", err)
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
