package exit

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/domain/exit"
)

const (
	// dummyProfileID는 기존 포지션의 더미 값으로, 무시해야 함
	dummyProfileID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
)

// resolveExitProfile resolves exit profile using priority:
// 1. Position override (position.exit_profile_id)
// 2. Symbol override (symbol_exit_overrides.profile_id)
// 3. Strategy override (strategy_id → profile mapping) - 미구현
// 4. Default profile
func (s *Service) resolveExitProfile(ctx context.Context, pos *exit.Position) *exit.ExitProfile {
	// 1. Position override (최우선)
	// 더미 UUID는 무시 (기존 포지션의 placeholder 값)
	if pos.ExitProfileID != nil && *pos.ExitProfileID != "" && *pos.ExitProfileID != dummyProfileID {
		profile, err := s.profileRepo.GetProfile(ctx, *pos.ExitProfileID)
		if err == nil && profile != nil && profile.IsActive {
			log.Debug().
				Str("profile_id", profile.ProfileID).
				Str("symbol", pos.Symbol).
				Str("position_id", pos.PositionID.String()).
				Msg("Using position override profile")
			return profile
		}
		log.Warn().
			Err(err).
			Str("profile_id", *pos.ExitProfileID).
			Str("symbol", pos.Symbol).
			Msg("Failed to load position profile, fallback to symbol override")
	}

	// 2. Symbol override
	override, err := s.symbolOverrideRepo.GetOverride(ctx, pos.Symbol)
	if err == nil && override != nil && override.Enabled {
		profile, err := s.profileRepo.GetProfile(ctx, override.ProfileID)
		if err == nil && profile != nil && profile.IsActive {
			log.Debug().
				Str("profile_id", profile.ProfileID).
				Str("symbol", pos.Symbol).
				Msg("Using symbol override profile")
			return profile
		}
		log.Warn().
			Err(err).
			Str("profile_id", override.ProfileID).
			Str("symbol", pos.Symbol).
			Msg("Failed to load symbol override profile, fallback to default")
	}

	// 3. Strategy override (미구현)
	// if strategyProfile := getStrategyProfile(pos.StrategyID); strategyProfile != "" {
	//     return loadProfile(strategyProfile)
	// }

	// 4. Default
	log.Debug().
		Str("symbol", pos.Symbol).
		Msg("Using default profile")
	return s.defaultProfile
}
