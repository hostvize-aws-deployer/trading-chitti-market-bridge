-- ============================================================================
-- Trading Chitti - Intraday Market Data Schema (TimescaleDB)
-- ============================================================================
--
-- This schema creates tables for real-time and intraday market data storage
-- using TimescaleDB hypertables for time-series optimization.
--
-- Features:
-- - Intraday OHLCV bars (1m, 5m, 15m, 1h timeframes)
-- - Tick-level data (raw trades)
-- - Order book snapshots
-- - Automatic compression (90% space reduction)
-- - Data retention policies
-- - Optimized indexes for fast queries
--
-- ============================================================================

-- Enable TimescaleDB extension (if not already enabled)
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ==================================================================================================
-- TABLE: md.intraday_bars - Stores aggregated OHLCV bars at multiple timeframes
-- ================================================================================================

CREATE TABLE IF NOT EXISTS md.intraday_bars (
    bar_id BIGSERIAL,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    instrument_token INTEGER,
    bar_timestamp TIMESTAMPTZ NOT NULL,
    timeframe TEXT NOT NULL CHECK (timeframe IN ('1m', '5m', '15m', '1h', '1d')),
    open DOUBLE PRECISION NOT NULL,
    high DOUBLE PRECISION NOT NULL,
    low DOUBLE PRECISION NOT NULL,
    close DOUBLE PRECISION NOT NULL,
    volume BIGINT NOT NULL DEFAULT 0,
    trades_count INTEGER DEFAULT 0,
    vwap DOUBLE PRECISION,
    oi BIGINT DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'collector',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (exchange, symbol, bar_timestamp, timeframe),
    FOREIGN KEY (exchange, symbol) REFERENCES md.symbols(exchange, symbol) ON DELETE CASCADE
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable('md.intraday_bars', 'bar_timestamp', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);

-- Add compression policy
SELECT add_compression_policy('md.intraday_bars', compress_after => INTERVAL '7 days', if_not_exists => TRUE);

-- Retention policy
SELECT add_retention_policy('md.intraday_bars', drop_after => INTERVAL '365 days', if_not_exists => TRUE);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_intraday_bars_symbol_time ON md.intraday_bars (symbol, bar_timestamp DESC) WHERE exchange = 'NSE';
CREATE INDEX IF NOT EXISTS idx_intraday_bars_timeframe ON md.intraday_bars (timeframe, bar_timestamp DESC);

-- ==============================================================================================
-- TABLE: md.tick_data - Stores raw tick-level trade data
-- ==============================================================================================

CREATE TABLE IF NOT EXISTS md.tick_data (
    tick_id BIGSERIAL,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    instrument_token INTEGER,
    tick_timestamp TIMESTAMPTZ NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    quantity BIGINT NOT NULL,
    trade_type TEXT CHECK (trade_type IN ('buy', 'sell', 'unknown')),
    source TEXT NOT NULL DEFAULT 'collector',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (exchange, symbol, tick_timestamp, tick_id),
    FOREIGN KEY (exchange, symbol) REFERENCES md.symbols(exchange, symbol) ON DELETE CASCADE
);

-- Convert to hypertable
SELECT create_hypertable('md.tick_data', 'tick_timestamp', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);
SELECT add_compression_policy('md.tick_data', compress_after => INTERVAL '3 days', if_not_exists => TRUE);
SELECT add_retention_policy('md.tick_data', drop_after => INTERVAL '30 days', if_not_exists => TRUE);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_tick_data_symbol_time ON md.tick_data (symbol, tick_timestamp DESC) WHERE exchange = 'NSE';

-- ==============================================================================================
-- TABLE: md.order_book - Stores order book snapshots
-- ==============================================================================================

CREATE TABLE IF NOT EXISTS md.order_book (
    snapshot_id BIGSERIAL PRIMARY KEY,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    snapshot_timestamp TIMESTAMPTZ NOT NULL,
    bids JSONB NOT NULL,
    asks JSONB NOT NULL,
    source TEXT NOT NULL DEFAULT 'collector',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (exchange, symbol) REFERENCES md.symbols(exchange, symbol) ON DELETE CASCADE
);

-- Convert to hypertable
SELECT create_hypertable('md.order_book', 'snapshot_timestamp', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);
SELECT add_compression_policy('md.order_book', compress_after => INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('md.order_book', drop_after => INTERVAL '90 days', if_not_exists => TRUE);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_order_book_symbol_time ON md.order_book (symbol, snapshot_timestamp DESC);

-- ==============================================================================================
-- VIEWS
-- ==============================================================================================

CREATE OR REPLACE VIEW md.latest_1m_bars AS
SELECT DISTINCT ON (exchange, symbol)
    exchange, symbol, bar_timestamp, open, high, low, close, volume, vwap, trades_count
FROM md.intraday_bars
WHERE timeframe = '1m'
ORDER BY exchange, symbol, bar_timestamp DESC;

CREATE OR REPLACE VIEW md.today_bars AS
SELECT exchange, symbol, timeframe, bar_timestamp, open, high, low, close, volume, vwap, trades_count
FROM md.intraday_bars
WHERE bar_timestamp >= CURRENT_DATE
ORDER BY symbol, timeframe, bar_timestamp DESC;
