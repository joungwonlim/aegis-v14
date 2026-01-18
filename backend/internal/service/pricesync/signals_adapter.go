package pricesync

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/signals"
)

// SignalsRepositoryAdapter adapts signals.SignalRepository to pricesync.SignalsRepository
type SignalsRepositoryAdapter struct {
	repo signals.SignalRepository
}

// NewSignalsRepositoryAdapter creates a new adapter
func NewSignalsRepositoryAdapter(repo signals.SignalRepository) *SignalsRepositoryAdapter {
	return &SignalsRepositoryAdapter{
		repo: repo,
	}
}

// GetSignalSymbols returns symbols with active buy signals
func (a *SignalsRepositoryAdapter) GetSignalSymbols(ctx context.Context) ([]string, error) {
	// Get latest snapshot
	snapshot, err := a.repo.GetLatestSnapshot(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get latest signal snapshot")
		return nil, nil // Return empty, not error - signals are optional
	}

	if snapshot == nil {
		return nil, nil
	}

	// Extract symbols from buy signals
	symbols := make([]string, 0, len(snapshot.BuySignals))
	for _, sig := range snapshot.BuySignals {
		symbols = append(symbols, sig.Symbol)
	}

	log.Debug().
		Int("count", len(symbols)).
		Msg("Signal symbols loaded for price sync")

	return symbols, nil
}
