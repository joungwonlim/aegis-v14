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
	universeRepo universe.UniverseRepository
	stockRepo    universe.StockRepository
	statsRepo    universe.StatisticsReader

	// External readers
	holdingReader   universe.HoldingReader
	watchlistReader universe.WatchlistReader
	rankingReader   universe.RankingReader

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
