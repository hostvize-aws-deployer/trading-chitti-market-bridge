-- Intraday Market Data Schema
-- Optimized for high-frequency time-series data using TimescaleDB

-- Enable TimescaleDB extension (if not already enabled)
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- Create schema for market data
CREATE SCHEMA IF NOT EXISTS md;

-- ============================================================================
-- INTRADAY OHLCV BARS
-- ============================================================================

-- Intraday bars (1m, 5m, 15m, 1h candles)
CREATE TABLE IF NOT EXISTS md.intraday_bars (
    bar_id BIGSERIAL,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    instrument_token BIGINT NOT NULL,
    bar_timestamp TIMESTAMPTZ NOT NULL,
    timeframe TEXT NOT NULL CHECK (timeframe IN ('1m', '5m', '15m', '1h', 'day')),

    -- OHLCV data
    open DOUBLE PRECISION NOT NULL,
    high DOUBLE PRECISION NOT NULL,
    low DOUBLE PRECISION NOT NULL,
    close DOUBLE PRECISION NOT NULL,
    volume BIGINT NOT NULL DEFAULT 0,

    -- Additional metrics
    trades_count INTEGER,
    vwap DOUBLE PRECISION,
    oi BIGINT, -- Open Interest (for derivatives)

    -- Metadata
    source TEXT NOT NULL DEFAULT 'zerodha',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint
    UNIQUE(exchange, symbol, bar_timestamp, timeframe)
);

-- Convert to TimescaleDB hypertable for time-series optimization
SELECT create_hypertable('md.intraday_bars', 'bar_timestamp',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_intraday_bars_symbol_time
    ON md.intraday_bars(symbol, bar_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_intraday_bars_token_time
    ON md.intraday_bars(instrument_token, bar_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_intraday_bars_timeframe
    ON md.intraday_bars(timeframe, bar_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_intraday_bars_exchange
    ON md.intraday_bars(exchange, symbol, timeframe, bar_timestamp DESC);

-- Enable compression for older data (compress data older than 7 days)
ALTER TABLE md.intraday_bars SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'exchange, symbol, timeframe',
    timescaledb.compress_orderby = 'bar_timestamp DESC'
);

SELECT add_compression_policy('md.intraday_bars', INTERVAL '7 days', if_not_exists => TRUE);

-- ============================================================================
-- TICK DATA (Raw Trades)
-- ============================================================================

-- Tick-level data for granular analysis
CREATE TABLE IF NOT EXISTS md.tick_data (
    tick_id BIGSERIAL,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    instrument_token BIGINT NOT NULL,
    tick_timestamp TIMESTAMPTZ NOT NULL,

    -- Trade data
    price DOUBLE PRECISION NOT NULL,
    quantity BIGINT NOT NULL,
    trade_type TEXT CHECK (trade_type IN ('buy', 'sell', 'unknown')),

    -- Metadata
    source TEXT NOT NULL DEFAULT 'zerodha',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Convert to hypertable
SELECT create_hypertable('md.tick_data', 'tick_timestamp',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_tick_data_symbol_time
    ON md.tick_data(symbol, tick_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_tick_data_token_time
    ON md.tick_data(instrument_token, tick_timestamp DESC);

-- Compression for tick data (compress after 3 days)
ALTER TABLE md.tick_data SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'exchange, symbol',
    timescaledb.compress_orderby = 'tick_timestamp DESC'
);

SELECT add_compression_policy('md.tick_data', INTERVAL '3 days', if_not_exists => TRUE);

-- ============================================================================
-- ORDER BOOK SNAPSHOTS (Level 2 Depth)
-- ============================================================================

-- Order book depth snapshots for advanced strategies
CREATE TABLE IF NOT EXISTS md.order_book (
    snapshot_id BIGSERIAL,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    instrument_token BIGINT NOT NULL,
    snapshot_timestamp TIMESTAMPTZ NOT NULL,

    -- Order book data (stored as JSONB for flexibility)
    bids JSONB NOT NULL,  -- [{price: 2500.50, quantity: 100, orders: 5}, ...]
    asks JSONB NOT NULL,  -- [{price: 2500.75, quantity: 80, orders: 3}, ...]

    -- Aggregated metrics
    bid_quantity BIGINT,
    ask_quantity BIGINT,
    spread DOUBLE PRECISION,

    -- Metadata
    source TEXT NOT NULL DEFAULT 'zerodha',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Convert to hypertable
SELECT create_hypertable('md.order_book', 'snapshot_timestamp',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_order_book_symbol_time
    ON md.order_book(symbol, snapshot_timestamp DESC);

-- Compression (compress after 1 day)
ALTER TABLE md.order_book SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'exchange, symbol',
    timescaledb.compress_orderby = 'snapshot_timestamp DESC'
);

SELECT add_compression_policy('md.order_book', INTERVAL '1 day', if_not_exists => TRUE);

-- ============================================================================
-- DATA SOURCE CONFIGURATION
-- ============================================================================

-- Configuration for different data sources
CREATE TABLE IF NOT EXISTS md.data_sources (
    source_id SERIAL PRIMARY KEY,
    source_name TEXT NOT NULL UNIQUE,
    source_type TEXT NOT NULL CHECK (source_type IN ('websocket', 'http_poll', 'api')),
    exchange TEXT NOT NULL,

    -- Connection details
    endpoint_url TEXT,
    api_key_env TEXT,  -- Environment variable name for API key
    connection_params JSONB,

    -- Rate limiting
    rate_limit_per_second INTEGER DEFAULT 10,
    max_symbols INTEGER DEFAULT 100,

    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_connected_at TIMESTAMPTZ,
    last_error TEXT,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default data sources
INSERT INTO md.data_sources (source_name, source_type, exchange, endpoint_url, max_symbols, rate_limit_per_second)
VALUES
    ('zerodha_websocket', 'websocket', 'NSE', 'wss://ws.zerodha.com', 200, 100),
    ('nse_http_poller', 'http_poll', 'NSE', 'https://www.nseindia.com/api/quote-equity', 50, 3),
    ('polygon_websocket', 'websocket', 'US', 'wss://socket.polygon.io/stocks', 500, 100)
ON CONFLICT (source_name) DO NOTHING;

-- ============================================================================
-- DATA COLLECTION JOBS
-- ============================================================================

-- Track data collection job runs
CREATE TABLE IF NOT EXISTS md.collection_runs (
    run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id INTEGER REFERENCES md.data_sources(source_id),
    job_type TEXT NOT NULL CHECK (job_type IN ('intraday', 'tick', 'orderbook')),

    -- Time range
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,

    -- Results
    status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed', 'cancelled')),
    bars_collected INTEGER DEFAULT 0,
    ticks_collected INTEGER DEFAULT 0,
    symbols_processed INTEGER DEFAULT 0,
    error_message TEXT,

    -- Metadata
    metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_collection_runs_started
    ON md.collection_runs(started_at DESC);

CREATE INDEX IF NOT EXISTS idx_collection_runs_status
    ON md.collection_runs(status, started_at DESC);

-- ============================================================================
-- CONTINUOUS AGGREGATES (Pre-computed Views)
-- ============================================================================

-- Materialized view for 1-hour bars from 1-minute bars (for performance)
CREATE MATERIALIZED VIEW IF NOT EXISTS md.hourly_bars_mv
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', bar_timestamp) AS hour_bucket,
    exchange,
    symbol,
    instrument_token,
    timeframe,
    first(open, bar_timestamp) AS open,
    max(high) AS high,
    min(low) AS low,
    last(close, bar_timestamp) AS close,
    sum(volume) AS volume,
    count(*) AS bars_count
FROM md.intraday_bars
WHERE timeframe = '1m'
GROUP BY hour_bucket, exchange, symbol, instrument_token, timeframe;

-- Refresh policy (refresh every 15 minutes)
SELECT add_continuous_aggregate_policy('md.hourly_bars_mv',
    start_offset => INTERVAL '2 hours',
    end_offset => INTERVAL '15 minutes',
    schedule_interval => INTERVAL '15 minutes',
    if_not_exists => TRUE
);

-- ============================================================================
-- DATA RETENTION POLICIES
-- ============================================================================

-- Auto-delete tick data older than 30 days
SELECT add_retention_policy('md.tick_data', INTERVAL '30 days', if_not_exists => TRUE);

-- Auto-delete order book snapshots older than 7 days
SELECT add_retention_policy('md.order_book', INTERVAL '7 days', if_not_exists => TRUE);

-- Keep intraday bars for 1 year
SELECT add_retention_policy('md.intraday_bars', INTERVAL '365 days', if_not_exists => TRUE);

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to get latest bar for a symbol
CREATE OR REPLACE FUNCTION md.get_latest_bar(
    p_symbol TEXT,
    p_timeframe TEXT DEFAULT '1m'
)
RETURNS TABLE (
    symbol TEXT,
    bar_timestamp TIMESTAMPTZ,
    open DOUBLE PRECISION,
    high DOUBLE PRECISION,
    low DOUBLE PRECISION,
    close DOUBLE PRECISION,
    volume BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        b.symbol,
        b.bar_timestamp,
        b.open,
        b.high,
        b.low,
        b.close,
        b.volume
    FROM md.intraday_bars b
    WHERE b.symbol = p_symbol
      AND b.timeframe = p_timeframe
    ORDER BY b.bar_timestamp DESC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate VWAP for current day
CREATE OR REPLACE FUNCTION md.calculate_today_vwap(
    p_symbol TEXT,
    p_timeframe TEXT DEFAULT '1m'
)
RETURNS DOUBLE PRECISION AS $$
DECLARE
    v_vwap DOUBLE PRECISION;
BEGIN
    SELECT
        SUM((high + low + close) / 3 * volume) / NULLIF(SUM(volume), 0)
    INTO v_vwap
    FROM md.intraday_bars
    WHERE symbol = p_symbol
      AND timeframe = p_timeframe
      AND bar_timestamp >= date_trunc('day', NOW());

    RETURN COALESCE(v_vwap, 0);
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE md.intraday_bars IS 'Intraday OHLCV bars (1m, 5m, 15m, 1h)';
COMMENT ON TABLE md.tick_data IS 'Raw tick-level trade data';
COMMENT ON TABLE md.order_book IS 'Level 2 order book depth snapshots';
COMMENT ON TABLE md.data_sources IS 'Configuration for market data sources';
COMMENT ON TABLE md.collection_runs IS 'Data collection job tracking';

-- Create maintenance script reminder
DO $$
BEGIN
    RAISE NOTICE 'TimescaleDB intraday data schema created successfully!';
    RAISE NOTICE '';
    RAISE NOTICE 'Compression policies:';
    RAISE NOTICE '  - intraday_bars: compress after 7 days';
    RAISE NOTICE '  - tick_data: compress after 3 days';
    RAISE NOTICE '  - order_book: compress after 1 day';
    RAISE NOTICE '';
    RAISE NOTICE 'Retention policies:';
    RAISE NOTICE '  - tick_data: delete after 30 days';
    RAISE NOTICE '  - order_book: delete after 7 days';
    RAISE NOTICE '  - intraday_bars: delete after 365 days';
    RAISE NOTICE '';
    RAISE NOTICE 'Next steps:';
    RAISE NOTICE '  1. Start data collector service';
    RAISE NOTICE '  2. Monitor md.collection_runs table';
    RAISE NOTICE '  3. Query intraday data via API';
END $$;
