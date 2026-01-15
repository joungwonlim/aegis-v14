package exit

import (
	"context"

	"github.com/wonny/aegis/v14/internal/domain/exit"
)

// resolveExitProfile resolves exit profile using priority:
// 1. Position override (position.exit_profile_id)
// 2. Symbol override (symbol_exit_overrides.profile_id)
// 3. Strategy override (strategy_id → profile mapping) - 미구현
// 4. Default profile
func (s *Service) resolveExitProfile(ctx context.Context, pos *exit.Position) *exit.ExitProfile {
	// 1. Position override
	if pos.ExitProfileID != nil && *pos.ExitProfileID != "" {
		// TODO: Load profile from repository
		// For now, return default if override is set
		return s.defaultProfile
	}

	// 2. Symbol override
	// TODO: Implement symbol override lookup
	// override, err := s.symbolOverrideRepo.GetOverride(ctx, pos.Symbol)
	// if err == nil && override.Enabled {
	//     return loadProfile(override.ProfileID)
	// }

	// 3. Strategy override (미구현)
	// if strategyProfile := getStrategyProfile(pos.StrategyID); strategyProfile != "" {
	//     return loadProfile(strategyProfile)
	// }

	// 4. Default
	return s.defaultProfile
}
