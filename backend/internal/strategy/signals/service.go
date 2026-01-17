package signals

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/wonny/aegis/v14/internal/domain/signals"
	"github.com/wonny/aegis/v14/internal/domain/universe"
)

// Service Signals 서비스
type Service struct {
	ctx context.Context

	// Repositories
	signalRepo signals.SignalRepository
	factorRepo signals.FactorRepository

	// External readers
	universeReader UniverseReader

	// Config
	criteria *signals.SignalCriteria

	// Cache
	latestSnapshot *signals.SignalSnapshot

	// Components
	evaluator *Evaluator
	ranker    *Ranker
}

// UniverseReader Universe 데이터 Reader
type UniverseReader interface {
	// 최신 Universe 스냅샷 조회
	GetLatestSnapshot(ctx context.Context) (*universe.UniverseSnapshot, error)

	// 특정 Universe 스냅샷 조회
	GetSnapshot(ctx context.Context, snapshotID string) (*universe.UniverseSnapshot, error)
}

// NewService 새 서비스 생성
func NewService(
	ctx context.Context,
	signalRepo signals.SignalRepository,
	factorRepo signals.FactorRepository,
	universeReader UniverseReader,
) *Service {
	criteria := signals.DefaultSignalCriteria()

	return &Service{
		ctx:            ctx,
		signalRepo:     signalRepo,
		factorRepo:     factorRepo,
		universeReader: universeReader,
		criteria:       criteria,
		evaluator:      NewEvaluator(factorRepo, criteria),
		ranker:         NewRanker(criteria),
	}
}

// Start 서비스 시작
func (s *Service) Start() error {
	log.Info().Msg("Starting Signals service")

	// Load latest snapshot on startup
	snapshot, err := s.signalRepo.GetLatestSnapshot(s.ctx)
	if err != nil {
		log.Warn().Err(err).Msg("No existing snapshot")
	} else {
		s.latestSnapshot = snapshot
		log.Info().
			Str("snapshot_id", snapshot.SnapshotID).
			Int("total_count", snapshot.TotalCount).
			Msg("Loaded latest signal snapshot")
	}

	return nil
}

// Stop 서비스 정지
func (s *Service) Stop() error {
	log.Info().Msg("Stopping Signals service")
	return nil
}

// GenerateSignals Universe에서 신호 생성
func (s *Service) GenerateSignals(ctx context.Context) (*signals.SignalSnapshot, error) {
	log.Info().Msg("Generating signals from latest universe")

	// 1. Load latest universe snapshot
	universeSnapshot, err := s.universeReader.GetLatestSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("load universe snapshot: %w", err)
	}

	// Universe의 모든 종목 수집 (Holdings + Watchlist)
	allStocks := make([]universe.UniverseStock, 0, len(universeSnapshot.Holdings)+len(universeSnapshot.Watchlist))
	allStocks = append(allStocks, universeSnapshot.Holdings...)
	allStocks = append(allStocks, universeSnapshot.Watchlist...)

	if len(allStocks) == 0 {
		log.Warn().Msg("Universe is empty, no signals to generate")
		return nil, signals.ErrUniverseNotReady
	}

	log.Info().
		Str("universe_id", universeSnapshot.SnapshotID).
		Int("stock_count", len(allStocks)).
		Msg("Universe loaded")

	// 2. Generate signals for each stock
	allSignals := make([]signals.Signal, 0, len(allStocks))

	for _, stock := range allStocks {
		signal, err := s.evaluator.EvaluateStock(ctx, stock)
		if err != nil {
			log.Warn().
				Err(err).
				Str("symbol", stock.Symbol).
				Msg("Failed to evaluate stock, skipping")
			continue
		}

		// Set snapshot metadata
		signal.SnapshotID = universeSnapshot.SnapshotID
		signal.GeneratedAt = time.Now()

		allSignals = append(allSignals, *signal)
	}

	log.Info().
		Int("evaluated", len(allSignals)).
		Msg("Stock evaluation complete")

	// 3. Filter by conviction
	filteredSignals := s.filterByConviction(allSignals)

	log.Info().
		Int("filtered", len(filteredSignals)).
		Int("min_conviction", s.criteria.MinConviction).
		Msg("Filtered by conviction")

	// 4. Rank signals
	rankedSignals := s.ranker.RankSignals(filteredSignals)

	// 5. Limit to MaxSignals
	if len(rankedSignals) > s.criteria.MaxSignals {
		rankedSignals = rankedSignals[:s.criteria.MaxSignals]
	}

	// 6. Split by signal type
	buySignals := make([]signals.Signal, 0)
	sellSignals := make([]signals.Signal, 0)

	for _, sig := range rankedSignals {
		switch sig.SignalType {
		case signals.SignalBuy:
			buySignals = append(buySignals, sig)
		case signals.SignalSell:
			sellSignals = append(sellSignals, sig)
		}
	}

	// 7. Create snapshot
	snapshot := &signals.SignalSnapshot{
		SnapshotID:  generateSnapshotID(),
		UniverseID:  universeSnapshot.SnapshotID,
		GeneratedAt: time.Now(),
		TotalCount:  len(rankedSignals),
		BuySignals:  buySignals,
		SellSignals: sellSignals,
		Stats:       s.calculateStats(rankedSignals),
	}

	// 8. Save snapshot
	if err := s.signalRepo.SaveSnapshot(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("save snapshot: %w", err)
	}

	// 9. Update cache
	s.latestSnapshot = snapshot

	log.Info().
		Str("snapshot_id", snapshot.SnapshotID).
		Int("buy_count", len(buySignals)).
		Int("sell_count", len(sellSignals)).
		Msg("Signals generated successfully")

	return snapshot, nil
}

// GetLatestSnapshot 최신 스냅샷 조회
func (s *Service) GetLatestSnapshot(ctx context.Context) (*signals.SignalSnapshot, error) {
	// Return cached if available
	if s.latestSnapshot != nil {
		return s.latestSnapshot, nil
	}

	// Otherwise load from repository
	snapshot, err := s.signalRepo.GetLatestSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	s.latestSnapshot = snapshot
	return snapshot, nil
}

// GetSnapshotByID 특정 스냅샷 조회
func (s *Service) GetSnapshotByID(ctx context.Context, snapshotID string) (*signals.SignalSnapshot, error) {
	return s.signalRepo.GetSnapshotByID(ctx, snapshotID)
}

// GetSignalBySymbol 특정 종목의 신호 조회
func (s *Service) GetSignalBySymbol(ctx context.Context, snapshotID, symbol string) (*signals.Signal, error) {
	return s.signalRepo.GetSignalBySymbol(ctx, snapshotID, symbol)
}

// filterByConviction 신뢰도로 필터링
func (s *Service) filterByConviction(allSignals []signals.Signal) []signals.Signal {
	filtered := make([]signals.Signal, 0, len(allSignals))

	for _, sig := range allSignals {
		if sig.Conviction >= s.criteria.MinConviction {
			filtered = append(filtered, sig)
		}
	}

	return filtered
}

// calculateStats 통계 계산
func (s *Service) calculateStats(allSignals []signals.Signal) signals.SignalStats {
	if len(allSignals) == 0 {
		return signals.SignalStats{}
	}

	stats := signals.SignalStats{}
	totalStrength := 0.0
	totalConviction := 0.0

	for _, sig := range allSignals {
		totalStrength += float64(sig.Strength)
		totalConviction += float64(sig.Conviction)

		switch sig.SignalType {
		case signals.SignalBuy:
			stats.BuyCount++
		case signals.SignalSell:
			stats.SellCount++
		case signals.SignalHold:
			stats.HoldCount++
		}
	}

	stats.AvgStrength = totalStrength / float64(len(allSignals))
	stats.AvgConviction = totalConviction / float64(len(allSignals))

	return stats
}

// generateSnapshotID 스냅샷 ID 생성
func generateSnapshotID() string {
	now := time.Now()
	return fmt.Sprintf("%s-%s", now.Format("20060102"), uuid.New().String()[:8])
}
