# Universe Selection 설계

> 투자 가능 종목 선정 모듈

**Version**: 1.0.0
**Status**: ✅ 설계 완료
**Last Updated**: 2026-01-14

---

## 📋 개요

### 책임 (Responsibility)
투자 가능 종목(Universe)을 선정하고 관리합니다.

### 핵심 기능
1. **Universe 구성**: Holdings + Watchlist + Market Rankings
2. **필터링**: 유동성, 시가총액, 거래량 기준 필터
3. **계층적 우선순위**: 보유종목 > 관심종목 > 랭킹
4. **Breakdown 제공**: Universe 구성 상세 정보

### 위치
```
backend/internal/service/universe/
backend/internal/domain/universe/
backend/internal/infrastructure/postgres/universe/
backend/internal/api/handlers/universe/
```

### 의존성
- `infra.database` (PostgreSQL)
- Optional: `infra.cache` (성능 최적화)

---

## 🎯 설계 원칙

### 1. SSOT (Single Source of Truth)
```
market.stocks (종목 마스터) - SSOT
└─> universe_snapshots (스냅샷 캐시, 재생성 가능)
```

### 2. 계층적 구성
```
Universe = Holdings (Tier 1, 최우선)
         + Watchlist (Tier 2, 투자 예정)
         + Rankings (Tier 3, 후보군)
         + Fallback (market.stocks, 비상)
```

### 3. 중복 제거
같은 종목이 여러 소스에 있으면 우선순위가 높은 Tier에만 포함

### 4. 필터링 기준 명시
```
유동성 필터: 평균 거래대금 >= 10억원 (5일 평균)
시가총액 필터: >= 100억원
거래량 필터: >= 10만주/일 (5일 평균)
제외 대상: 관리종목, 투자주의, 거래정지
```

---

## 🏗️ 아키텍처

### Domain Layer

#### universe/model.go
```go
package universe

import "time"

// UniverseSnapshot represents a universe snapshot
type UniverseSnapshot struct {
	SnapshotID   string       `json:"snapshot_id"`    // 스냅샷 ID (YYYYMMDD-HHMM)
	GeneratedAt  time.Time    `json:"generated_at"`   // 생성 시각
	TotalCount   int          `json:"total_count"`    // 전체 종목 수
	Holdings     []UniverseStock `json:"holdings"`    // 보유종목
	Watchlist    []UniverseStock `json:"watchlist"`   // 관심종목
	Rankings     RankingBreakdown `json:"rankings"`   // 랭킹 breakdown
	FilterStats  FilterStats     `json:"filter_stats"`// 필터링 통계
}

// UniverseStock represents a stock in the universe
type UniverseStock struct {
	Symbol       string  `json:"symbol"`         // 종목 코드
	Name         string  `json:"name"`           // 종목명
	Market       string  `json:"market"`         // KOSPI | KOSDAQ
	Sector       string  `json:"sector"`         // 섹터
	Tier         string  `json:"tier"`           // HOLDING | WATCHLIST | RANKING
	Source       string  `json:"source"`         // 출처 (holding, watchlist, quantHigh, priceTop, ...)
	MarketCap    int64   `json:"market_cap"`     // 시가총액 (원)
	AvgVolume5D  int64   `json:"avg_volume_5d"`  // 5일 평균 거래량
	AvgValue5D   int64   `json:"avg_value_5d"`   // 5일 평균 거래대금 (원)
	IsActive     bool    `json:"is_active"`      // 활성 여부
}

// RankingBreakdown represents ranking data breakdown
type RankingBreakdown struct {
	QuantHigh       RankingData `json:"quant_high"`      // 거래량급증
	PriceTop        RankingData `json:"price_top"`       // 거래대금
	Upper           RankingData `json:"upper"`           // 상승
	Top             RankingData `json:"top"`             // 인기검색
	Capitalization  RankingData `json:"capitalization"`  // 시가총액
}

// RankingData represents ranking data for a specific category
type RankingData struct {
	Kospi  []UniverseStock `json:"kospi"`   // KOSPI 종목
	Kosdaq []UniverseStock `json:"kosdaq"`  // KOSDAQ 종목
}

// FilterStats represents filtering statistics
type FilterStats struct {
	TotalCandidates    int `json:"total_candidates"`     // 전체 후보
	AfterLiquidity     int `json:"after_liquidity"`      // 유동성 필터 후
	AfterMarketCap     int `json:"after_market_cap"`     // 시총 필터 후
	AfterVolume        int `json:"after_volume"`         // 거래량 필터 후
	AfterExclusions    int `json:"after_exclusions"`     // 제외 대상 필터 후
	Final              int `json:"final"`                // 최종 (중복 제거)
}

// FilterCriteria represents universe filtering criteria
type FilterCriteria struct {
	MinMarketCap       int64   `json:"min_market_cap"`        // 최소 시가총액 (기본: 100억)
	MinAvgValue5D      int64   `json:"min_avg_value_5d"`      // 최소 5일 평균 거래대금 (기본: 10억)
	MinAvgVolume5D     int64   `json:"min_avg_volume_5d"`     // 최소 5일 평균 거래량 (기본: 10만주)
	ExcludeManaged     bool    `json:"exclude_managed"`       // 관리종목 제외 (기본: true)
	ExcludeSuspended   bool    `json:"exclude_suspended"`     // 거래정지 제외 (기본: true)
	RankingLimit       int     `json:"ranking_limit"`         // 랭킹당 종목 수 (기본: 100)
}

// DefaultFilterCriteria returns default filtering criteria
func DefaultFilterCriteria() *FilterCriteria {
	return &FilterCriteria{
		MinMarketCap:       10_000_000_000,  // 100억원
		MinAvgValue5D:      1_000_000_000,   // 10억원
		MinAvgVolume5D:     100_000,         // 10만주
		ExcludeManaged:     true,
		ExcludeSuspended:   true,
		RankingLimit:       100,
	}
}

// Ranking Categories
const (
	CategoryQuantHigh      = "quantHigh"      // 거래량급증
	CategoryPriceTop       = "priceTop"       // 거래대금
	CategoryUpper          = "upper"          // 상승
	CategoryTop            = "top"            // 인기검색
	CategoryCapitalization = "capitalization" // 시가총액
)

// Tiers
const (
	TierHolding  = "HOLDING"   // 보유종목 (Tier 1)
	TierWatchlist = "WATCHLIST" // 관심종목 (Tier 2)
	TierRanking  = "RANKING"   // 랭킹 (Tier 3)
)
```

#### universe/repository.go
```go
package universe

import (
	"context"
	"time"
)

// UniverseRepository manages universe snapshots
type UniverseRepository interface {
	// SaveSnapshot saves a universe snapshot
	SaveSnapshot(ctx context.Context, snapshot *UniverseSnapshot) error

	// GetLatestSnapshot retrieves the latest universe snapshot
	GetLatestSnapshot(ctx context.Context) (*UniverseSnapshot, error)

	// GetSnapshotByID retrieves a snapshot by ID
	GetSnapshotByID(ctx context.Context, snapshotID string) (*UniverseSnapshot, error)

	// ListSnapshots lists snapshots within a time range
	ListSnapshots(ctx context.Context, from, to time.Time) ([]*UniverseSnapshot, error)
}

// StockRepository reads stock master data
type StockRepository interface {
	// GetStockInfo retrieves stock information
	GetStockInfo(ctx context.Context, symbol string) (*StockInfo, error)

	// GetActiveStocks retrieves all active stocks with filters
	GetActiveStocks(ctx context.Context, criteria *FilterCriteria) ([]*StockInfo, error)
}

// StockInfo represents stock master information
type StockInfo struct {
	Symbol       string    `json:"symbol"`
	Name         string    `json:"name"`
	Market       string    `json:"market"`
	Sector       string    `json:"sector"`
	MarketCap    int64     `json:"market_cap"`
	IsActive     bool      `json:"is_active"`
	IsManaged    bool      `json:"is_managed"`     // 관리종목 여부
	IsSuspended  bool      `json:"is_suspended"`   // 거래정지 여부
	ListedDate   time.Time `json:"listed_date"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// HoldingReader reads holdings data
type HoldingReader interface {
	// GetHoldings retrieves current holdings
	GetHoldings(ctx context.Context) ([]string, error)
}

// WatchlistReader reads watchlist data
type WatchlistReader interface {
	// GetWatchlist retrieves watchlist
	GetWatchlist(ctx context.Context) ([]string, error)
}

// RankingReader reads ranking data
type RankingReader interface {
	// GetRanking retrieves ranking by category and market
	GetRanking(ctx context.Context, category, market string, limit int) ([]string, error)
}

// StatisticsReader reads stock statistics (volume, value)
type StatisticsReader interface {
	// GetStockStatistics retrieves stock statistics
	GetStockStatistics(ctx context.Context, symbol string, days int) (*StockStatistics, error)

	// GetBatchStatistics retrieves statistics for multiple symbols
	GetBatchStatistics(ctx context.Context, symbols []string, days int) (map[string]*StockStatistics, error)
}

// StockStatistics represents stock statistics
type StockStatistics struct {
	Symbol       string `json:"symbol"`
	AvgVolume5D  int64  `json:"avg_volume_5d"`   // 5일 평균 거래량
	AvgValue5D   int64  `json:"avg_value_5d"`    // 5일 평균 거래대금
	AvgVolume20D int64  `json:"avg_volume_20d"`  // 20일 평균 거래량
	AvgValue20D  int64  `json:"avg_value_20d"`   // 20일 평균 거래대금
}
```

#### universe/errors.go
```go
package universe

import "errors"

var (
	ErrSnapshotNotFound = errors.New("universe snapshot not found")
	ErrInvalidCriteria  = errors.New("invalid filter criteria")
	ErrNoActiveStocks   = errors.New("no active stocks found")
)
```

---

## 🔧 Service Layer

#### universe/service.go
```go
package universe

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/universe"
)

const (
	snapshotInterval = 1 * time.Hour // 1시간마다 Universe 갱신
)

// Service is the Universe service
type Service struct {
	ctx context.Context

	// Repositories
	universeRepo   universe.UniverseRepository
	stockRepo      universe.StockRepository
	statsRepo      universe.StatisticsReader

	// External readers
	holdingReader  universe.HoldingReader
	watchlistReader universe.WatchlistReader
	rankingReader  universe.RankingReader

	// Config
	filterCriteria *universe.FilterCriteria

	// Cache
	latestSnapshot *universe.UniverseSnapshot
}

// NewService creates a new Universe service
func NewService(
	ctx context.Context,
	universeRepo universe.UniverseRepository,
	stockRepo universe.StockRepository,
	statsRepo universe.StatisticsReader,
	holdingReader universe.HoldingReader,
	watchlistReader universe.WatchlistReader,
	rankingReader universe.RankingReader,
) *Service {
	return &Service{
		ctx:             ctx,
		universeRepo:    universeRepo,
		stockRepo:       stockRepo,
		statsRepo:       statsRepo,
		holdingReader:   holdingReader,
		watchlistReader: watchlistReader,
		rankingReader:   rankingReader,
		filterCriteria:  universe.DefaultFilterCriteria(),
	}
}

// Start starts the Universe service
func (s *Service) Start() error {
	log.Info().Msg("Starting Universe service")

	// Load latest snapshot on startup
	snapshot, err := s.universeRepo.GetLatestSnapshot(s.ctx)
	if err != nil {
		log.Warn().Err(err).Msg("No existing snapshot, will generate new one")
	} else {
		s.latestSnapshot = snapshot
		log.Info().
			Str("snapshot_id", snapshot.SnapshotID).
			Int("total_count", snapshot.TotalCount).
			Msg("Loaded latest universe snapshot")
	}

	// Generate initial snapshot if none exists
	if s.latestSnapshot == nil {
		if err := s.GenerateSnapshot(s.ctx); err != nil {
			log.Error().Err(err).Msg("Failed to generate initial snapshot")
		}
	}

	// Start background snapshot generation
	go s.snapshotLoop()

	log.Info().Msg("Universe service started")
	return nil
}

// snapshotLoop generates universe snapshots periodically
func (s *Service) snapshotLoop() {
	ticker := time.NewTicker(snapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.GenerateSnapshot(s.ctx); err != nil {
				log.Error().Err(err).Msg("Failed to generate universe snapshot")
			}
		case <-s.ctx.Done():
			log.Info().Msg("Universe snapshot loop stopped")
			return
		}
	}
}

// GenerateSnapshot generates a new universe snapshot
func (s *Service) GenerateSnapshot(ctx context.Context) error {
	log.Info().Msg("Generating universe snapshot")

	// 1. Collect holdings
	holdings, err := s.collectHoldings(ctx)
	if err != nil {
		return fmt.Errorf("collect holdings: %w", err)
	}

	// 2. Collect watchlist
	watchlist, err := s.collectWatchlist(ctx)
	if err != nil {
		return fmt.Errorf("collect watchlist: %w", err)
	}

	// 3. Collect rankings
	rankings, err := s.collectRankings(ctx)
	if err != nil {
		return fmt.Errorf("collect rankings: %w", err)
	}

	// 4. Apply filters and build snapshot
	snapshot, err := s.buildSnapshot(ctx, holdings, watchlist, rankings)
	if err != nil {
		return fmt.Errorf("build snapshot: %w", err)
	}

	// 5. Save snapshot
	if err := s.universeRepo.SaveSnapshot(ctx, snapshot); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}

	// 6. Update cache
	s.latestSnapshot = snapshot

	log.Info().
		Str("snapshot_id", snapshot.SnapshotID).
		Int("total_count", snapshot.TotalCount).
		Int("holdings", len(snapshot.Holdings)).
		Int("watchlist", len(snapshot.Watchlist)).
		Msg("Universe snapshot generated")

	return nil
}

// GetLatestSnapshot retrieves the latest universe snapshot
func (s *Service) GetLatestSnapshot() (*universe.UniverseSnapshot, error) {
	if s.latestSnapshot != nil {
		return s.latestSnapshot, nil
	}
	return s.universeRepo.GetLatestSnapshot(s.ctx)
}

// GetSnapshot retrieves a snapshot by ID
func (s *Service) GetSnapshot(snapshotID string) (*universe.UniverseSnapshot, error) {
	return s.universeRepo.GetSnapshotByID(s.ctx, snapshotID)
}

// ListSnapshots lists snapshots within a time range
func (s *Service) ListSnapshots(from, to time.Time) ([]*universe.UniverseSnapshot, error) {
	return s.universeRepo.ListSnapshots(s.ctx, from, to)
}

// GetUniverseSymbols returns all symbols in the latest universe
func (s *Service) GetUniverseSymbols() ([]string, error) {
	snapshot, err := s.GetLatestSnapshot()
	if err != nil {
		return nil, err
	}

	// Collect all symbols (deduped)
	seen := make(map[string]bool)
	var symbols []string

	addStocks := func(stocks []universe.UniverseStock) {
		for _, stock := range stocks {
			if !seen[stock.Symbol] {
				seen[stock.Symbol] = true
				symbols = append(symbols, stock.Symbol)
			}
		}
	}

	addStocks(snapshot.Holdings)
	addStocks(snapshot.Watchlist)

	for _, rd := range []universe.RankingData{
		snapshot.Rankings.QuantHigh,
		snapshot.Rankings.PriceTop,
		snapshot.Rankings.Upper,
		snapshot.Rankings.Top,
		snapshot.Rankings.Capitalization,
	} {
		addStocks(rd.Kospi)
		addStocks(rd.Kosdaq)
	}

	return symbols, nil
}
```

#### universe/collector.go
```go
package universe

import (
	"context"
	"fmt"

	"github.com/wonny/aegis/v14/internal/domain/universe"
)

// collectHoldings collects holdings with enriched data
func (s *Service) collectHoldings(ctx context.Context) ([]universe.UniverseStock, error) {
	// Get holdings symbols
	symbols, err := s.holdingReader.GetHoldings(ctx)
	if err != nil {
		return nil, fmt.Errorf("get holdings: %w", err)
	}

	// Enrich with stock info and statistics
	return s.enrichStocks(ctx, symbols, universe.TierHolding, "holding")
}

// collectWatchlist collects watchlist with enriched data
func (s *Service) collectWatchlist(ctx context.Context) ([]universe.UniverseStock, error) {
	symbols, err := s.watchlistReader.GetWatchlist(ctx)
	if err != nil {
		return nil, fmt.Errorf("get watchlist: %w", err)
	}

	return s.enrichStocks(ctx, symbols, universe.TierWatchlist, "watchlist")
}

// collectRankings collects rankings with enriched data
func (s *Service) collectRankings(ctx context.Context) (universe.RankingBreakdown, error) {
	var breakdown universe.RankingBreakdown

	categories := []struct {
		code   string
		target *universe.RankingData
	}{
		{universe.CategoryQuantHigh, &breakdown.QuantHigh},
		{universe.CategoryPriceTop, &breakdown.PriceTop},
		{universe.CategoryUpper, &breakdown.Upper},
		{universe.CategoryTop, &breakdown.Top},
		{universe.CategoryCapitalization, &breakdown.Capitalization},
	}

	for _, cat := range categories {
		// KOSPI
		kospiSymbols, err := s.rankingReader.GetRanking(ctx, cat.code, "KOSPI", s.filterCriteria.RankingLimit)
		if err == nil {
			cat.target.Kospi, _ = s.enrichStocks(ctx, kospiSymbols, universe.TierRanking, cat.code)
		}

		// KOSDAQ
		kosdaqSymbols, err := s.rankingReader.GetRanking(ctx, cat.code, "KOSDAQ", s.filterCriteria.RankingLimit)
		if err == nil {
			cat.target.Kosdaq, _ = s.enrichStocks(ctx, kosdaqSymbols, universe.TierRanking, cat.code)
		}
	}

	return breakdown, nil
}

// enrichStocks enriches symbols with stock info and statistics
func (s *Service) enrichStocks(ctx context.Context, symbols []string, tier, source string) ([]universe.UniverseStock, error) {
	if len(symbols) == 0 {
		return []universe.UniverseStock{}, nil
	}

	// Get batch statistics
	statsMap, err := s.statsRepo.GetBatchStatistics(ctx, symbols, 5)
	if err != nil {
		return nil, fmt.Errorf("get batch statistics: %w", err)
	}

	var stocks []universe.UniverseStock
	for _, symbol := range symbols {
		// Get stock info
		info, err := s.stockRepo.GetStockInfo(ctx, symbol)
		if err != nil {
			continue // Skip if stock info not found
		}

		// Get statistics
		stats := statsMap[symbol]
		if stats == nil {
			// No statistics, use defaults
			stats = &universe.StockStatistics{
				Symbol:      symbol,
				AvgVolume5D: 0,
				AvgValue5D:  0,
			}
		}

		stock := universe.UniverseStock{
			Symbol:      symbol,
			Name:        info.Name,
			Market:      info.Market,
			Sector:      info.Sector,
			Tier:        tier,
			Source:      source,
			MarketCap:   info.MarketCap,
			AvgVolume5D: stats.AvgVolume5D,
			AvgValue5D:  stats.AvgValue5D,
			IsActive:    info.IsActive,
		}

		stocks = append(stocks, stock)
	}

	return stocks, nil
}
```

#### universe/filter.go
```go
package universe

import (
	"context"
	"time"

	"github.com/wonny/aegis/v14/internal/domain/universe"
)

// buildSnapshot builds a universe snapshot with filtering
func (s *Service) buildSnapshot(
	ctx context.Context,
	holdings, watchlist []universe.UniverseStock,
	rankings universe.RankingBreakdown,
) (*universe.UniverseSnapshot, error) {

	seen := make(map[string]bool)
	filterStats := universe.FilterStats{}

	var finalHoldings, finalWatchlist []universe.UniverseStock
	var finalRankings universe.RankingBreakdown

	// 1. Holdings (Tier 1) - NO FILTER (always include)
	for _, stock := range holdings {
		if !seen[stock.Symbol] {
			seen[stock.Symbol] = true
			finalHoldings = append(finalHoldings, stock)
		}
	}

	// 2. Watchlist (Tier 2) - Light filter
	for _, stock := range watchlist {
		if seen[stock.Symbol] {
			continue // Already in holdings
		}

		// Apply filters
		if !s.passesFilter(stock) {
			continue
		}

		seen[stock.Symbol] = true
		finalWatchlist = append(finalWatchlist, stock)
	}

	// 3. Rankings (Tier 3) - Full filter
	filterRanking := func(stocks []universe.UniverseStock) []universe.UniverseStock {
		var filtered []universe.UniverseStock
		for _, stock := range stocks {
			if seen[stock.Symbol] {
				continue // Already included
			}

			filterStats.TotalCandidates++

			if !s.passesFilter(stock) {
				continue
			}

			seen[stock.Symbol] = true
			filtered = append(filtered, stock)
		}
		return filtered
	}

	finalRankings.QuantHigh.Kospi = filterRanking(rankings.QuantHigh.Kospi)
	finalRankings.QuantHigh.Kosdaq = filterRanking(rankings.QuantHigh.Kosdaq)
	finalRankings.PriceTop.Kospi = filterRanking(rankings.PriceTop.Kospi)
	finalRankings.PriceTop.Kosdaq = filterRanking(rankings.PriceTop.Kosdaq)
	finalRankings.Upper.Kospi = filterRanking(rankings.Upper.Kospi)
	finalRankings.Upper.Kosdaq = filterRanking(rankings.Upper.Kosdaq)
	finalRankings.Top.Kospi = filterRanking(rankings.Top.Kospi)
	finalRankings.Top.Kosdaq = filterRanking(rankings.Top.Kosdaq)
	finalRankings.Capitalization.Kospi = filterRanking(rankings.Capitalization.Kospi)
	finalRankings.Capitalization.Kosdaq = filterRanking(rankings.Capitalization.Kosdaq)

	// Update filter stats
	filterStats.Final = len(seen)

	// Generate snapshot ID
	snapshotID := time.Now().Format("20060102-1504")

	snapshot := &universe.UniverseSnapshot{
		SnapshotID:  snapshotID,
		GeneratedAt: time.Now(),
		TotalCount:  filterStats.Final,
		Holdings:    finalHoldings,
		Watchlist:   finalWatchlist,
		Rankings:    finalRankings,
		FilterStats: filterStats,
	}

	return snapshot, nil
}

// passesFilter checks if a stock passes filtering criteria
func (s *Service) passesFilter(stock universe.UniverseStock) bool {
	// Market cap filter
	if stock.MarketCap < s.filterCriteria.MinMarketCap {
		return false
	}

	// Liquidity filter (average value)
	if stock.AvgValue5D < s.filterCriteria.MinAvgValue5D {
		return false
	}

	// Volume filter
	if stock.AvgVolume5D < s.filterCriteria.MinAvgVolume5D {
		return false
	}

	// Active check
	if !stock.IsActive {
		return false
	}

	return true
}
```

---

## 📊 Database Schema

### universe_snapshots 테이블

```sql
CREATE TABLE market.universe_snapshots (
    snapshot_id VARCHAR(20) PRIMARY KEY,       -- YYYYMMDD-HHMM
    generated_at TIMESTAMP NOT NULL,
    total_count INT NOT NULL,
    holdings JSONB NOT NULL,                   -- []UniverseStock
    watchlist JSONB NOT NULL,                  -- []UniverseStock
    rankings JSONB NOT NULL,                   -- RankingBreakdown
    filter_stats JSONB NOT NULL,               -- FilterStats
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_universe_snapshots_generated_at ON market.universe_snapshots(generated_at DESC);
```

### 참조 테이블 (이미 존재)

- `market.stocks` - 종목 마스터 (SSOT)
- `portfolio.positions` - 보유종목
- `portfolio.watchlist` - 관심종목
- `ranking.naver` - 네이버 랭킹

---

## 🔌 API Endpoints

### GET /api/v1/universe/latest
최신 Universe 스냅샷 조회

**Response**:
```json
{
  "snapshot_id": "20260114-1500",
  "generated_at": "2026-01-14T15:00:00Z",
  "total_count": 523,
  "holdings": [...],
  "watchlist": [...],
  "rankings": {
    "quant_high": {...},
    "price_top": {...},
    ...
  },
  "filter_stats": {...}
}
```

### GET /api/v1/universe/snapshots/{snapshotId}
특정 스냅샷 조회

### GET /api/v1/universe/symbols
현재 Universe 종목 코드 목록

**Response**:
```json
{
  "symbols": ["005930", "000660", ...],
  "count": 523
}
```

---

## ⚠️ v10 참고 및 개선사항

### v10의 좋은 점
- 계층적 Universe 구성 (Holdings > Watchlist > Rankings)
- 중복 제거 로직
- 5개 랭킹 카테고리 활용

### v14 개선사항
1. **명확한 필터링 기준**: v10은 암묵적, v14는 명시적
2. **Snapshot 관리**: 이력 추적 및 재생성 가능
3. **SSOT 준수**: market.stocks를 SSOT로 명확히
4. **통계 기반 필터**: 유동성, 거래량을 통계 데이터로 검증
5. **Tier 개념 명시화**: HOLDING/WATCHLIST/RANKING

---

## 📝 주의사항

### 1. Holdings는 필터 제외
보유종목은 현재 포트폴리오이므로 어떤 조건이든 Universe에 포함

### 2. Snapshot 갱신 주기
- 기본: 1시간
- 장중: 필요시 30분으로 단축 가능
- 장외: 6시간으로 연장 가능

### 3. Ranking 데이터 신선도
네이버 랭킹 데이터가 오래된 경우 (> 24시간) 경고 로그

### 4. 메모리 사용
Snapshot 전체를 메모리에 캐싱하므로, 너무 크면 (> 10,000 종목) 문제 가능

---

## 🚀 성능 최적화

### 1. Batch 조회
통계 데이터는 GetBatchStatistics로 일괄 조회

### 2. Snapshot 캐싱
최신 snapshot은 메모리에 캐싱, DB 조회 최소화

### 3. 비동기 생성
Snapshot 생성은 background goroutine에서 비동기 처리

### 4. PostgreSQL JSONB
Breakdown 데이터를 JSONB로 저장하여 유연성 확보

---

## 🧪 테스트 전략

### 단위 테스트
- Filter 로직 테스트
- Enrichment 테스트

### 통합 테스트
- Snapshot 생성 전체 흐름
- DB 저장/조회

### 성능 테스트
- 10,000개 종목 처리 시간 측정
- 메모리 사용량 측정

---

**Version**: 1.0.0
**Status**: ✅ 설계 완료
**Next**: 구현 (Implementation Phase)
