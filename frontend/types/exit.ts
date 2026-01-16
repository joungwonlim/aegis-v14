/**
 * Exit Engine Types
 *
 * Custom Exit Rules 관련 타입 정의
 */

export type CustomRuleCondition = 'profit_above' | 'profit_below';

export interface CustomExitRule {
  id: string;
  enabled: boolean;
  condition: CustomRuleCondition;
  threshold: number;    // e.g., 7 for +7%
  exitPercent: number;  // e.g., 20 for 20%
  priority: number;
  description?: string;
}

export interface ExitProfileConfig {
  // ATR settings
  atr: {
    ref: number;
    factor_min: number;
    factor_max: number;
  };

  // Triggers
  sl1: TriggerConfig;
  sl2: TriggerConfig;
  tp1: TriggerConfig;
  tp2: TriggerConfig;
  tp3: TriggerConfig;

  // Trailing
  trailing: {
    pct_trail: number;
    atr_k: number;
  };

  // Time stop
  time_stop: {
    max_hold_days: number;
    no_momentum_days: number;
    no_momentum_profit: number;
  };

  // HardStop
  hardstop: {
    enabled: boolean;
    pct: number;
  };

  // Custom Rules
  custom_rules?: CustomExitRule[];
}

export interface TriggerConfig {
  base_pct: number;
  min_pct: number;
  max_pct: number;
  qty_pct: number;
  stop_floor_profit?: number;
  start_trailing?: boolean;
}

export interface ExitProfile {
  profile_id: string;
  name: string;
  description: string;
  config: ExitProfileConfig;
  is_active: boolean;
  created_by: string;
  created_ts: string;
}
