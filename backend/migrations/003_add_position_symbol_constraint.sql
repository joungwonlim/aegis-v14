-- Migration: Add NOT NULL constraint to positions.symbol
-- Purpose: Prevent data integrity issues with empty symbols
-- Date: 2026-01-17

-- Add NOT NULL constraint to symbol column
ALTER TABLE trade.positions
ALTER COLUMN symbol SET NOT NULL;

-- Add check constraint to ensure symbol is not empty string
ALTER TABLE trade.positions
ADD CONSTRAINT positions_symbol_not_empty CHECK (symbol != '');

-- Create index on symbol for performance (if not exists)
CREATE INDEX IF NOT EXISTS idx_positions_symbol ON trade.positions(symbol);
