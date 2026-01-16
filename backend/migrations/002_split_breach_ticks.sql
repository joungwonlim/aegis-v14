-- Split breach_ticks into stop_floor_breach_ticks and trailing_breach_ticks
-- This prevents cross-contamination between StopFloor and Trailing consecutive conditions

-- Add new columns
ALTER TABLE trade.position_state
ADD COLUMN stop_floor_breach_ticks INTEGER NOT NULL DEFAULT 0,
ADD COLUMN trailing_breach_ticks INTEGER NOT NULL DEFAULT 0;

-- Copy existing breach_ticks to both (safe migration)
UPDATE trade.position_state
SET
    stop_floor_breach_ticks = breach_ticks,
    trailing_breach_ticks = breach_ticks;

-- Drop old column (commented out for safety - uncomment after verification)
-- ALTER TABLE trade.position_state DROP COLUMN breach_ticks;
