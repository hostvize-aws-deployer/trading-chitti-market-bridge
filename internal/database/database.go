package database

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"github.com/trading-chitti/market-bridge/internal/broker"
)

// BrokerConfig is an alias for broker.BrokerConfig used in database operations
type BrokerConfig = broker.BrokerConfig

// Database handles PostgreSQL operations
type Database struct {
	conn *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(dsn string) (*Database, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	
	return &Database{conn: conn}, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	return db.conn.Close()
}

// GetActiveBrokerConfig returns the active broker configuration
func (db *Database) GetActiveBrokerConfig() (*broker.BrokerConfig, error) {
	query := `
		SELECT id, broker_name, display_name, enabled, api_key, api_secret, 
		       access_token, user_id, max_positions, max_risk_per_trade, 
		       created_at, updated_at
		FROM brokers.config
		WHERE enabled = true
		ORDER BY updated_at DESC
		LIMIT 1
	`
	
	config := &broker.BrokerConfig{}
	err := db.conn.QueryRow(query).Scan(
		&config.ID,
		&config.BrokerName,
		&config.DisplayName,
		&config.Enabled,
		&config.APIKey,
		&config.APISecret,
		&config.AccessToken,
		&config.UserID,
		&config.MaxPositions,
		&config.MaxRiskPerTrade,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	
	if err != nil {
		return nil, err
	}
	
	return config, nil
}

// GetAllBrokerConfigs returns all broker configurations
func (db *Database) GetAllBrokerConfigs() ([]broker.BrokerConfig, error) {
	query := `
		SELECT id, broker_name, display_name, enabled, api_key, api_secret,
		       access_token, user_id, max_positions, max_risk_per_trade,
		       created_at, updated_at
		FROM brokers.config
		ORDER BY created_at DESC
	`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	configs := []broker.BrokerConfig{}
	for rows.Next() {
		config := broker.BrokerConfig{}
		err := rows.Scan(
			&config.ID,
			&config.BrokerName,
			&config.DisplayName,
			&config.Enabled,
			&config.APIKey,
			&config.APISecret,
			&config.AccessToken,
			&config.UserID,
			&config.MaxPositions,
			&config.MaxRiskPerTrade,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	
	return configs, nil
}

// SaveBrokerConfig saves a broker configuration
func (db *Database) SaveBrokerConfig(config *broker.BrokerConfig) (int, error) {
	query := `
		INSERT INTO brokers.config (
			broker_name, display_name, enabled, api_key, api_secret,
			access_token, user_id, max_positions, max_risk_per_trade
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	
	var id int
	err := db.conn.QueryRow(
		query,
		config.BrokerName,
		config.DisplayName,
		config.Enabled,
		config.APIKey,
		config.APISecret,
		config.AccessToken,
		config.UserID,
		config.MaxPositions,
		config.MaxRiskPerTrade,
	).Scan(&id)
	
	return id, err
}

// SaveAnalysis saves analysis results
func (db *Database) SaveAnalysis(symbol string, analysis interface{}) error {
	query := `
		INSERT INTO trades.analysis (
			symbol, analysis_date, period_days, analysis_json
		) VALUES ($1, $2, $3, $4)
	`
	
	_, err := db.conn.Exec(query, symbol, time.Now(), 52, analysis)
	return err
}

// SaveTrade saves a trade execution
func (db *Database) SaveTrade(trade interface{}) error {
	// TODO: Implement trade saving
	return nil
}

// ============================================================================
// INSTRUMENT MANAGEMENT
// ============================================================================

// Instrument represents a trading instrument
type Instrument struct {
	InstrumentToken uint32
	ExchangeToken   uint32
	Tradingsymbol   string
	Name            string
	Exchange        string
	Segment         string
	InstrumentType  string
	ISIN            string
	Expiry          *time.Time
	Strike          float64
	TickSize        float64
	LotSize         int
	LastPrice       float64
	LastUpdated     time.Time
}

// UpsertInstrument inserts or updates an instrument
func (db *Database) UpsertInstrument(inst Instrument) error {
	query := `
		INSERT INTO trades.instruments (
			instrument_token, exchange_token, tradingsymbol, name, exchange,
			segment, instrument_type, isin, expiry, strike, tick_size, lot_size,
			last_price, last_updated
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (exchange, tradingsymbol)
		DO UPDATE SET
			instrument_token = EXCLUDED.instrument_token,
			exchange_token = EXCLUDED.exchange_token,
			name = EXCLUDED.name,
			segment = EXCLUDED.segment,
			instrument_type = EXCLUDED.instrument_type,
			isin = EXCLUDED.isin,
			expiry = EXCLUDED.expiry,
			strike = EXCLUDED.strike,
			tick_size = EXCLUDED.tick_size,
			lot_size = EXCLUDED.lot_size,
			last_price = EXCLUDED.last_price,
			last_updated = EXCLUDED.last_updated
	`

	_, err := db.conn.Exec(
		query,
		inst.InstrumentToken,
		inst.ExchangeToken,
		inst.Tradingsymbol,
		inst.Name,
		inst.Exchange,
		inst.Segment,
		inst.InstrumentType,
		inst.ISIN,
		inst.Expiry,
		inst.Strike,
		inst.TickSize,
		inst.LotSize,
		inst.LastPrice,
		inst.LastUpdated,
	)

	return err
}

// GetInstrumentToken returns instrument token for a given exchange and symbol
func (db *Database) GetInstrumentToken(exchange, symbol string) (uint32, error) {
	query := `
		SELECT instrument_token
		FROM trades.instruments
		WHERE exchange = $1 AND tradingsymbol = $2
	`

	var token uint32
	err := db.conn.QueryRow(query, exchange, symbol).Scan(&token)
	if err == sql.ErrNoRows {
		return 0, nil
	}

	return token, err
}

// GetInstrumentByToken returns instrument details for a given token
func (db *Database) GetInstrumentByToken(token uint32) (*Instrument, error) {
	query := `
		SELECT instrument_token, exchange_token, tradingsymbol, name, exchange,
		       segment, instrument_type, isin, expiry, strike, tick_size, lot_size,
		       last_price, last_updated
		FROM trades.instruments
		WHERE instrument_token = $1
	`

	inst := &Instrument{}
	err := db.conn.QueryRow(query, token).Scan(
		&inst.InstrumentToken,
		&inst.ExchangeToken,
		&inst.Tradingsymbol,
		&inst.Name,
		&inst.Exchange,
		&inst.Segment,
		&inst.InstrumentType,
		&inst.ISIN,
		&inst.Expiry,
		&inst.Strike,
		&inst.TickSize,
		&inst.LotSize,
		&inst.LastPrice,
		&inst.LastUpdated,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return inst, err
}

// SearchInstruments searches instruments by symbol pattern
func (db *Database) SearchInstruments(pattern string, limit int) ([]Instrument, error) {
	query := `
		SELECT instrument_token, exchange_token, tradingsymbol, name, exchange,
		       segment, instrument_type, isin, expiry, strike, tick_size, lot_size,
		       last_price, last_updated
		FROM trades.instruments
		WHERE tradingsymbol ILIKE $1 OR name ILIKE $1
		ORDER BY tradingsymbol
		LIMIT $2
	`

	rows, err := db.conn.Query(query, "%"+pattern+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	instruments := []Instrument{}
	for rows.Next() {
		inst := Instrument{}
		err := rows.Scan(
			&inst.InstrumentToken,
			&inst.ExchangeToken,
			&inst.Tradingsymbol,
			&inst.Name,
			&inst.Exchange,
			&inst.Segment,
			&inst.InstrumentType,
			&inst.ISIN,
			&inst.Expiry,
			&inst.Strike,
			&inst.TickSize,
			&inst.LotSize,
			&inst.LastPrice,
			&inst.LastUpdated,
		)
		if err != nil {
			return nil, err
		}
		instruments = append(instruments, inst)
	}

	return instruments, nil
}

// ============================================================================
// HISTORICAL DATA CACHE
// ============================================================================

// HistoricalCandle represents a single OHLCV candle
type HistoricalCandle struct {
	InstrumentToken  uint32
	Interval         string
	CandleTimestamp  time.Time
	Open             float64
	High             float64
	Low              float64
	Close            float64
	Volume           int64
	OI               int64
	CachedAt         time.Time
}

// CacheHistoricalCandles stores historical candles in cache
func (db *Database) CacheHistoricalCandles(candles []HistoricalCandle) error {
	if len(candles) == 0 {
		return nil
	}

	query := `
		INSERT INTO trades.historical_cache (
			instrument_token, interval, candle_timestamp,
			open, high, low, close, volume, oi
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (instrument_token, interval, candle_timestamp)
		DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			oi = EXCLUDED.oi,
			cached_at = NOW()
	`

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, candle := range candles {
		_, err := stmt.Exec(
			candle.InstrumentToken,
			candle.Interval,
			candle.CandleTimestamp,
			candle.Open,
			candle.High,
			candle.Low,
			candle.Close,
			candle.Volume,
			candle.OI,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetHistoricalFromCache retrieves cached historical candles
func (db *Database) GetHistoricalFromCache(
	instrumentToken uint32,
	interval string,
	fromDate, toDate time.Time,
) ([]HistoricalCandle, error) {
	query := `
		SELECT instrument_token, interval, candle_timestamp,
		       open, high, low, close, volume, oi, cached_at
		FROM trades.historical_cache
		WHERE instrument_token = $1
		  AND interval = $2
		  AND candle_timestamp >= $3
		  AND candle_timestamp <= $4
		ORDER BY candle_timestamp ASC
	`

	rows, err := db.conn.Query(query, instrumentToken, interval, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candles := []HistoricalCandle{}
	for rows.Next() {
		candle := HistoricalCandle{}
		err := rows.Scan(
			&candle.InstrumentToken,
			&candle.Interval,
			&candle.CandleTimestamp,
			&candle.Open,
			&candle.High,
			&candle.Low,
			&candle.Close,
			&candle.Volume,
			&candle.OI,
			&candle.CachedAt,
		)
		if err != nil {
			return nil, err
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

// ============================================================================
// TOKEN MANAGEMENT
// ============================================================================

// UpdateBrokerTokens updates access and refresh tokens for a broker
func (db *Database) UpdateBrokerTokens(brokerID int, accessToken, refreshToken string, expiresAt time.Time) error {
	query := `
		UPDATE brokers.config
		SET access_token = $1,
		    refresh_token = $2,
		    token_expires_at = $3,
		    last_token_refresh = NOW(),
		    updated_at = NOW()
		WHERE id = $4
	`

	_, err := db.conn.Exec(query, accessToken, refreshToken, expiresAt, brokerID)
	return err
}

// GetExpiringSoonBrokerConfigs returns brokers whose tokens expire within threshold
func (db *Database) GetExpiringSoonBrokerConfigs(threshold time.Duration) ([]broker.BrokerConfig, error) {
	query := `
		SELECT id, broker_name, display_name, enabled, api_key, api_secret,
		       access_token, user_id, max_positions, max_risk_per_trade,
		       created_at, updated_at
		FROM brokers.config
		WHERE enabled = true
		  AND token_expires_at IS NOT NULL
		  AND token_expires_at < $1
		ORDER BY token_expires_at ASC
	`

	expiryThreshold := time.Now().Add(threshold)
	rows, err := db.conn.Query(query, expiryThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := []broker.BrokerConfig{}
	for rows.Next() {
		config := broker.BrokerConfig{}
		err := rows.Scan(
			&config.ID,
			&config.BrokerName,
			&config.DisplayName,
			&config.Enabled,
			&config.APIKey,
			&config.APISecret,
			&config.AccessToken,
			&config.UserID,
			&config.MaxPositions,
			&config.MaxRiskPerTrade,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, nil
}
