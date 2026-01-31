-- Market Bridge Database Schema
-- Multi-broker trading system with 52-day analysis

CREATE SCHEMA IF NOT EXISTS brokers;
CREATE SCHEMA IF NOT EXISTS trades;

-- ============================================================================
-- BROKER CONFIGURATION
-- ============================================================================
CREATE TABLE IF NOT EXISTS brokers.config (
    id SERIAL PRIMARY KEY,
    broker_name VARCHAR(50) NOT NULL, -- zerodha, angelone, upstox, icicidirect
    display_name VARCHAR(100) NOT NULL,
    enabled BOOLEAN DEFAULT FALSE,

    -- API Credentials
    api_key TEXT NOT NULL,
    api_secret TEXT NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    user_id TEXT,

    -- Token Management
    token_expires_at TIMESTAMPTZ,
    last_token_refresh TIMESTAMPTZ,

    -- Trading Configuration
    max_positions INTEGER DEFAULT 5,
    max_risk_per_trade NUMERIC(5,2) DEFAULT 2.0,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(broker_name, user_id)
);

CREATE INDEX idx_brokers_enabled ON brokers.config(enabled);
CREATE INDEX idx_brokers_token_expiry ON brokers.config(token_expires_at) WHERE enabled = TRUE;

-- ============================================================================
-- ANALYSIS RESULTS (52-day analyzer output)
-- ============================================================================
CREATE TABLE IF NOT EXISTS trades.analysis (
    analysis_id SERIAL PRIMARY KEY,
    symbol TEXT NOT NULL,
    analysis_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    period_days INTEGER NOT NULL,
    
    -- Trend Analysis
    trend_direction TEXT,
    trend_slope NUMERIC(10,6),
    trend_r_squared NUMERIC(5,4),
    
    -- Volatility
    volatility NUMERIC(8,4),
    atr NUMERIC(10,2),
    
    -- Technical Indicators
    rsi NUMERIC(5,2),
    macd NUMERIC(10,4),
    sma_20 NUMERIC(12,2),
    sma_50 NUMERIC(12,2),
    
    -- Signals
    signals_count INTEGER DEFAULT 0,
    
    -- Full JSON
    analysis_json JSONB,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_analysis_symbol ON trades.analysis(symbol, analysis_date DESC);
CREATE INDEX idx_analysis_date ON trades.analysis(analysis_date DESC);

-- ============================================================================
-- TRADE EXECUTIONS
-- ============================================================================
CREATE TABLE IF NOT EXISTS trades.executions (
    execution_id SERIAL PRIMARY KEY,
    broker_id INTEGER REFERENCES brokers.config(id),
    
    symbol TEXT NOT NULL,
    exchange TEXT NOT NULL,
    order_id TEXT NOT NULL,
    
    -- Order Details
    action TEXT NOT NULL CHECK (action IN ('BUY', 'SELL')),
    quantity INTEGER NOT NULL,
    entry_price NUMERIC(12,2) NOT NULL,
    order_type TEXT NOT NULL,
    product TEXT NOT NULL, -- MIS, CNC, NRML
    
    -- Risk Management
    stop_loss NUMERIC(12,2),
    take_profit NUMERIC(12,2),
    
    -- Signal Info
    confidence NUMERIC(3,2),
    strategy TEXT NOT NULL,
    
    -- Execution
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Exit Tracking
    exit_price NUMERIC(12,2),
    exit_date TIMESTAMPTZ,
    pnl NUMERIC(15,2),
    pnl_pct NUMERIC(8,4),
    status TEXT DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED', 'CANCELLED')),
    
    -- Metadata
    dry_run BOOLEAN DEFAULT TRUE,
    notes TEXT
);

CREATE INDEX idx_executions_symbol ON trades.executions(symbol, executed_at DESC);
CREATE INDEX idx_executions_status ON trades.executions(status, executed_at DESC);
CREATE INDEX idx_executions_broker ON trades.executions(broker_id, executed_at DESC);

-- ============================================================================
-- TRADING SIGNALS (all generated signals)
-- ============================================================================
CREATE TABLE IF NOT EXISTS trades.signals (
    signal_id SERIAL PRIMARY KEY,
    analysis_id INTEGER REFERENCES trades.analysis(analysis_id),
    
    symbol TEXT NOT NULL,
    signal_type TEXT NOT NULL CHECK (signal_type IN ('BUY', 'SELL')),
    strategy TEXT NOT NULL,
    confidence NUMERIC(3,2) NOT NULL,
    
    entry_price NUMERIC(12,2) NOT NULL,
    stop_loss NUMERIC(12,2),
    take_profit NUMERIC(12,2),
    reason TEXT,
    
    -- Execution Tracking
    executed BOOLEAN DEFAULT FALSE,
    execution_id INTEGER REFERENCES trades.executions(execution_id),
    
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_signals_symbol ON trades.signals(symbol, generated_at DESC);
CREATE INDEX idx_signals_confidence ON trades.signals(confidence DESC);
CREATE INDEX idx_signals_executed ON trades.signals(executed);

-- ============================================================================
-- PERFORMANCE TRACKING
-- ============================================================================
CREATE TABLE IF NOT EXISTS trades.performance (
    date DATE PRIMARY KEY,
    broker_id INTEGER REFERENCES brokers.config(id),
    
    total_trades INTEGER DEFAULT 0,
    winning_trades INTEGER DEFAULT 0,
    losing_trades INTEGER DEFAULT 0,
    
    total_pnl NUMERIC(15,2) DEFAULT 0,
    win_rate NUMERIC(5,4),
    avg_win NUMERIC(15,2),
    avg_loss NUMERIC(15,2),
    
    sharpe_ratio NUMERIC(8,4),
    max_drawdown NUMERIC(8,4),
    
    starting_capital NUMERIC(15,2),
    ending_capital NUMERIC(15,2),
    
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- MARKET DATA CACHE (for quick access)
-- ============================================================================
CREATE TABLE IF NOT EXISTS trades.market_data_cache (
    symbol TEXT NOT NULL,
    exchange TEXT NOT NULL,
    instrument_token BIGINT,
    
    last_price NUMERIC(12,2),
    volume BIGINT,
    timestamp TIMESTAMPTZ,
    
    PRIMARY KEY (symbol, exchange)
);

CREATE INDEX idx_market_cache_timestamp ON trades.market_data_cache(timestamp DESC);

-- ============================================================================
-- WEBSOCKET SUBSCRIPTIONS
-- ============================================================================
CREATE TABLE IF NOT EXISTS trades.ws_subscriptions (
    subscription_id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    symbols TEXT[] NOT NULL,
    instrument_tokens BIGINT[],

    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_active TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- INSTRUMENTS (Symbol to Token Mapping)
-- ============================================================================
CREATE TABLE IF NOT EXISTS trades.instruments (
    instrument_token BIGINT PRIMARY KEY,
    exchange_token BIGINT,
    tradingsymbol TEXT NOT NULL,
    name TEXT,
    exchange TEXT NOT NULL,
    segment TEXT,
    instrument_type TEXT,

    -- Equity specific
    isin TEXT,

    -- F&O specific
    expiry DATE,
    strike NUMERIC(12,2),
    tick_size NUMERIC(8,2),
    lot_size INTEGER,

    -- Trading info
    last_price NUMERIC(12,2),

    -- Metadata
    last_updated TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(exchange, tradingsymbol)
);

CREATE INDEX idx_instruments_tradingsymbol ON trades.instruments(tradingsymbol);
CREATE INDEX idx_instruments_exchange ON trades.instruments(exchange, tradingsymbol);
CREATE INDEX idx_instruments_segment ON trades.instruments(segment);
CREATE INDEX idx_instruments_expiry ON trades.instruments(expiry) WHERE expiry IS NOT NULL;

-- ============================================================================
-- HISTORICAL DATA CACHE
-- ============================================================================
CREATE TABLE IF NOT EXISTS trades.historical_cache (
    instrument_token BIGINT NOT NULL,
    interval TEXT NOT NULL,
    candle_timestamp TIMESTAMPTZ NOT NULL,

    -- OHLCV
    open NUMERIC(12,2) NOT NULL,
    high NUMERIC(12,2) NOT NULL,
    low NUMERIC(12,2) NOT NULL,
    close NUMERIC(12,2) NOT NULL,
    volume BIGINT NOT NULL,

    -- F&O specific
    oi BIGINT,

    -- Metadata
    cached_at TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (instrument_token, interval, candle_timestamp)
);

CREATE INDEX idx_historical_token_interval ON trades.historical_cache(instrument_token, interval, candle_timestamp DESC);
CREATE INDEX idx_historical_timestamp ON trades.historical_cache(candle_timestamp DESC);

-- ============================================================================
-- GRANTS
-- ============================================================================
GRANT USAGE ON SCHEMA brokers TO PUBLIC;
GRANT USAGE ON SCHEMA trades TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA brokers TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA trades TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA brokers TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA trades TO PUBLIC;

-- ============================================================================
-- TRIGGERS
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_brokers_config_updated_at ON brokers.config;
CREATE TRIGGER update_brokers_config_updated_at
    BEFORE UPDATE ON brokers.config
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- ============================================================================
-- SUCCESS MESSAGE
-- ============================================================================
DO $$
BEGIN
    RAISE NOTICE '✅ Market Bridge schema created successfully';
    RAISE NOTICE '   - Created schemas: brokers, trades';
    RAISE NOTICE '   - Created tables: config, analysis, executions, signals, performance';
    RAISE NOTICE '   - Instrument token mapping: trades.instruments';
    RAISE NOTICE '   - Historical data caching: trades.historical_cache';
    RAISE NOTICE '   - Token auto-refresh: brokers.config (refresh_token, token_expires_at)';
    RAISE NOTICE '   - WebSocket support enabled';
    RAISE NOTICE '   - Multi-broker architecture ready';
END $$;
