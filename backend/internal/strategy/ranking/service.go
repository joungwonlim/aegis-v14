package ranking

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"aegis/internal/domain/ranking"
	"aegis/internal/domain/signals"
)

// Service Ranking 서비스
type Service struct {
	ctx context.Context

	// Repositories
	rankingRepo ranking.RankingRepository
	riskRepo    ranking.RiskDataRepository

	// External readers
	signalReader SignalReader

	// Config
	criteria *ranking.RankingCriteria

	// Cache
	latestSnapshot *ranking.RankingSnapshot

	// Components
	scorer      *Scorer
	diversifier *Diversifier
}

// SignalReader Signal 데이터 Reader
type SignalReader interface {
	// 최신 Signal 스냅샷 조회
	GetLatestSnapshot(ctx context.Context) (*signals.SignalSnapshot, error)

	// 특정 Signal 스냅샷 조회
	GetSnapshotByID(ctx context.Context, snapshotID string) (*signals.SignalSnapshot, error)
}

// NewService 새 서비스 생성
func NewService(
	ctx context.Context,
	rankingRepo ranking.RankingRepository,
	riskRepo ranking.RiskDataRepository,
	signalReader SignalReader,
) *Service {
	criteria := ranking.DefaultRankingCriteria()

	return &Service{
		ctx:          ctx,
		rankingRepo:  rankingRepo,
		riskRepo:     riskRepo,
		signalReader: signalReader,
		criteria:     criteria,
		scorer:       NewScorer(riskRepo, criteria),
		diversifier:  NewDiversifier(criteria),
	}
}

// Start 서비스 시작
func (s *Service) Start() error {
	log.Info().Msg("Starting Ranking service")

	// Load latest snapshot on startup
	snapshot, err := s.rankingRepo.GetLatestSnapshot(s.ctx)
	if err != nil {
		log.Warn().Err(err).Msg("No existing snapshot")
	} else {
		s.latestSnapshot = snapshot
		log.Info().
			Str("snapshot_id", snapshot.SnapshotID).
			Int("selected_count", snapshot.SelectedCount).
			Msg("Loaded latest ranking snapshot")
	}

	return nil
}

// Stop 서비스 정지
func (s *Service) Stop() error {
	log.Info().Msg("Stopping Ranking service")
	return nil
}

// GenerateRankings Signals에서 순위 생성
func (s *Service) GenerateRankings(ctx context.Context) (*ranking.RankingSnapshot, error) {
	log.Info().Msg("Generating rankings from latest signals")

	// 1. Load latest signal snapshot
	signalSnapshot, err := s.signalReader.GetLatestSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("load signal snapshot: %w", err)
	}

	// Only process BUY signals
	buySignals := signalSnapshot.BuySignals
	if len(buySignals) == 0 {
		log.Warn().Msg("No BUY signals available")
		return nil, ranking.ErrSignalsNotReady
	}

	log.Info().
		Str("signal_id", signalSnapshot.SnapshotID).
		Int("buy_signals", len(buySignals)).
		Msg("Signals loaded")

	// 2. Score each signal with risk adjustment
	rankedStocks := make([]ranking.RankedStock, 0, len(buySignals))

	for _, sig := range buySignals {
		ranked, err := s.scorer.ScoreSignal(ctx, sig)
		if err != nil {
			log.Warn().
				Err(err).
				Str("symbol", sig.Symbol).
				Msg("Failed to score signal, skipping")
			continue
		}

		// Set snapshot metadata
		ranked.SnapshotID = signalSnapshot.SnapshotID
		ranked.SignalID = sig.SignalID
		ranked.GeneratedAt = time.Now()

		rankedStocks = append(rankedStocks, *ranked)
	}

	log.Info().
		Int("scored", len(rankedStocks)).
		Msg("Signal scoring complete")

	// 3. Filter by minimum score
	filteredStocks := s.filterByMinScore(rankedStocks)

	log.Info().
		Int("filtered", len(filteredStocks)).
		Float64("min_score", s.criteria.MinTotalScore).
		Msg("Filtered by minimum score")

	if len(filteredStocks) == 0 {
		return nil, ranking.ErrNoValidStocks
	}

	// 4. Apply diversity constraints
	diversifiedStocks := s.diversifier.ApplyDiversityConstraints(filteredStocks)

	log.Info().
		Int("after_diversity", len(diversifiedStocks)).
		Msg("Diversity constraints applied")

	// 5. Final ranking
	finalRanked := s.rankStocks(diversifiedStocks)

	// 6. Select top N
	selectedStocks := s.selectTopN(finalRanked)

	log.Info().
		Int("selected", len(selectedStocks)).
		Int("max", s.criteria.MaxSelections).
		Msg("Top stocks selected")

	// 7. Create snapshot
	snapshot := &ranking.RankingSnapshot{
		SnapshotID:    generateSnapshotID(),
		SignalID:      signalSnapshot.SnapshotID,
		GeneratedAt:   time.Now(),
		TotalCount:    len(finalRanked),
		SelectedCount: len(selectedStocks),
		Rankings:      finalRanked,
		Stats:         s.calculateStats(selectedStocks),
	}

	// 8. Save snapshot
	if err := s.rankingRepo.SaveSnapshot(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("save snapshot: %w", err)
	}

	// 9. Update cache
	s.latestSnapshot = snapshot

	log.Info().
		Str("snapshot_id", snapshot.SnapshotID).
		Int("selected", len(selectedStocks)).
		Float64("avg_score", snapshot.Stats.AvgTotalScore).
		Msg("Rankings generated successfully")

	return snapshot, nil
}

// GetLatestSnapshot 최신 스냅샷 조회
func (s *Service) GetLatestSnapshot(ctx context.Context) (*ranking.RankingSnapshot, error) {
	// Return cached if available
	if s.latestSnapshot != nil {
		return s.latestSnapshot, nil
	}

	// Otherwise load from repository
	snapshot, err := s.rankingRepo.GetLatestSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	s.latestSnapshot = snapshot
	return snapshot, nil
}

// GetSnapshotByID 특정 스냅샷 조회
func (s *Service) GetSnapshotByID(ctx context.Context, snapshotID string) (*ranking.RankingSnapshot, error) {
	return s.rankingRepo.GetSnapshotByID(ctx, snapshotID)
}

// GetRankBySymbol 특정 종목의 순위 조회
func (s *Service) GetRankBySymbol(ctx context.Context, snapshotID, symbol string) (*ranking.RankedStock, error) {
	return s.rankingRepo.GetRankBySymbol(ctx, snapshotID, symbol)
}

// filterByMinScore 최소 점수로 필터링
func (s *Service) filterByMinScore(stocks []ranking.RankedStock) []ranking.RankedStock {
	filtered := make([]ranking.RankedStock, 0, len(stocks))

	for _, stock := range stocks {
		if stock.TotalScore >= s.criteria.MinTotalScore {
			filtered = append(filtered, stock)
		} else {
			stock.Selected = false
			stock.Reason = fmt.Sprintf("Score too low: %.2f < %.2f", stock.TotalScore, s.criteria.MinTotalScore)
			filtered = append(filtered, stock)
		}
	}

	return filtered
}

// rankStocks 최종 순위 매기기
func (s *Service) rankStocks(stocks []ranking.RankedStock) []ranking.RankedStock {
	// Sort by total score descending
	sortByTotalScore(stocks)

	// Assign rank
	for i := range stocks {
		stocks[i].Rank = i + 1
	}

	return stocks
}

// selectTopN 상위 N개 선정
func (s *Service) selectTopN(stocks []ranking.RankedStock) []ranking.RankedStock {
	selected := make([]ranking.RankedStock, 0, s.criteria.MaxSelections)

	count := 0
	for i := range stocks {
		if count >= s.criteria.MaxSelections {
			stocks[i].Selected = false
			stocks[i].Reason = fmt.Sprintf("Exceeds max selections: %d", s.criteria.MaxSelections)
		} else if stocks[i].TotalScore >= s.criteria.MinTotalScore {
			stocks[i].Selected = true
			stocks[i].Reason = "Selected"
			selected = append(selected, stocks[i])
			count++
		}
	}

	return selected
}

// calculateStats 통계 계산
func (s *Service) calculateStats(selectedStocks []ranking.RankedStock) ranking.RankingStats {
	if len(selectedStocks) == 0 {
		return ranking.RankingStats{
			SectorDistribution: make(map[string]int),
			MarketDistribution: make(map[string]int),
		}
	}

	stats := ranking.RankingStats{
		SectorDistribution: make(map[string]int),
		MarketDistribution: make(map[string]int),
	}

	totalScore := 0.0
	totalRiskScore := 0.0

	for _, stock := range selectedStocks {
		totalScore += stock.TotalScore
		totalRiskScore += stock.RiskScore

		stats.SectorDistribution[stock.Sector]++
		stats.MarketDistribution[stock.Market]++
	}

	stats.AvgTotalScore = totalScore / float64(len(selectedStocks))
	stats.AvgRiskScore = totalRiskScore / float64(len(selectedStocks))

	return stats
}

// generateSnapshotID 스냅샷 ID 생성
func generateSnapshotID() string {
	now := time.Now()
	return fmt.Sprintf("%s-%s", now.Format("20060102"), uuid.New().String()[:8])
}

// sortByTotalScore 점수로 정렬 (내림차순)
func sortByTotalScore(stocks []ranking.RankedStock) {
	// Simple bubble sort for now
	n := len(stocks)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if stocks[j].TotalScore < stocks[j+1].TotalScore {
				stocks[j], stocks[j+1] = stocks[j+1], stocks[j]
			}
		}
	}
}
