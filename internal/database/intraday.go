package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/trading-chitti/market-bridge/internal/broker"
)

// IntradayBar represents a single OHLCV bar
type IntradayBar struct {
	BarID           int64     `json:"bar_id" db:"bar_id"`
	Exchange        string    `json:"exchange" db:"exchange"`
	Symbol          string    `json:"symbol" db:"symbol"`
	InstrumentToken int64     `json:"instrument_token" db:"instrument_token"`
	BarTimestamp    time.Time `json:"bar_timestamp" db:"bar_timestamp"`
	Timeframe       string    `json:"timeframe" db:"timeframe"`
	Open            float64   `json:"open" db:"open"`
	High            float64   `json:"high" db:"high"`
	Low             float64   `json:"low" db:"low"`
	Close           float64   `json:"close" db:"close"`
	Volume          int64     `json:"volume" db:"volume"`
	TradesCount     *int      `json:"trades_count,omitempty" db:"trades_count"`
	VWAP            *float64  `json:"vwap,omitempty" db:"vwap"`
	OI              *int64    `json:"oi,omitempty" db:"oi"`
	Source          string    `json:"source" db:"source"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// TickData represents a single tick/trade
type TickData struct {
	TickID          int64     `json:"tick_id" db:"tick_id"`
	Exchange        string    `json:"exchange" db:"exchange"`
	Symbol          string    `json:"symbol" db:"symbol"`
	InstrumentToken int64     `json:"instrument_token" db:"instrument_token"`
	TickTimestamp   time.Time `json:"tick_timestamp" db:"tick_timestamp"`
	Price           float64   `json:"price" db:"price"`
	Quantity        int64     `json:"quantity" db:"quantity"`
	TradeType       string    `json:"trade_type" db:"trade_type"`
	Source          string    `json:"source" db:"source"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// OrderBookSnapshot represents L2 order book depth
type OrderBookSnapshot struct {
	SnapshotID        int64     `json:"snapshot_id" db:"snapshot_id"`
	Exchange          string    `json:"exchange" db:"exchange"`
	Symbol            string    `json:"symbol" db:"symbol"`
	InstrumentToken   int64     `json:"instrument_token" db:"instrument_token"`
	SnapshotTimestamp time.Time `json:"snapshot_timestamp" db:"snapshot_timestamp"`
	Bids              string    `json:"bids" db:"bids"` // JSONB as string
	Asks              string    `json:"asks" db:"asks"` // JSONB as string
	BidQuantity       *int64    `json:"bid_quantity,omitempty" db:"bid_quantity"`
	AskQuantity       *int64    `json:"ask_quantity,omitempty" db:"ask_quantity"`
	Spread            *float64  `json:"spread,omitempty" db:"spread"`
	Source            string    `json:"source" db:"source"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// ============================================================================
// INTRADAY BAR OPERATIONS
// ============================================================================

// InsertIntradayBar inserts a single intraday bar
func (db *Database) InsertIntradayBar(bar *IntradayBar) error {
	query := `
		INSERT INTO md.intraday_bars (
			exchange, symbol, instrument_token, bar_timestamp, timeframe,
			open, high, low, close, volume, trades_count, vwap, oi, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (exchange, symbol, bar_timestamp, timeframe)
		DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			trades_count = EXCLUDED.trades_count,
			vwap = EXCLUDED.vwap,
			oi = EXCLUDED.oi
		RETURNING bar_id
	`

	err := db.conn.QueryRow(
		query,
		bar.Exchange,
		bar.Symbol,
		bar.InstrumentToken,
		bar.BarTimestamp,
		bar.Timeframe,
		bar.Open,
		bar.High,
		bar.Low,
		bar.Close,
		bar.Volume,
		bar.TradesCount,
		bar.VWAP,
		bar.OI,
		bar.Source,
	).Scan(&bar.BarID)

	return err
}

// BulkInsertIntradayBars efficiently inserts multiple bars
func (db *Database) BulkInsertIntradayBars(bars []IntradayBar) error {
	if len(bars) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO md.intraday_bars (
			exchange, symbol, instrument_token, bar_timestamp, timeframe,
			open, high, low, close, volume, trades_count, vwap, oi, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (exchange, symbol, bar_timestamp, timeframe)
		DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			trades_count = EXCLUDED.trades_count,
			vwap = EXCLUDED.vwap,
			oi = EXCLUDED.oi
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, bar := range bars {
		_, err := stmt.Exec(
			bar.Exchange,
			bar.Symbol,
			bar.InstrumentToken,
			bar.BarTimestamp,
			bar.Timeframe,
			bar.Open,
			bar.High,
			bar.Low,
			bar.Close,
			bar.Volume,
			bar.TradesCount,
			bar.VWAP,
			bar.OI,
			bar.Source,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetIntradayBars retrieves intraday bars for a symbol
func (db *Database) GetIntradayBars(symbol, timeframe string, fromTime, toTime time.Time, limit int) ([]IntradayBar, error) {
	query := `
		SELECT
			bar_id, exchange, symbol, instrument_token, bar_timestamp, timeframe,
			open, high, low, close, volume, trades_count, vwap, oi, source, created_at
		FROM md.intraday_bars
		WHERE symbol = $1
		  AND timeframe = $2
		  AND bar_timestamp >= $3
		  AND bar_timestamp <= $4
		ORDER BY bar_timestamp ASC
		LIMIT $5
	`

	rows, err := db.conn.Query(query, symbol, timeframe, fromTime, toTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bars := []IntradayBar{}
	for rows.Next() {
		var bar IntradayBar
		err := rows.Scan(
			&bar.BarID,
			&bar.Exchange,
			&bar.Symbol,
			&bar.InstrumentToken,
			&bar.BarTimestamp,
			&bar.Timeframe,
			&bar.Open,
			&bar.High,
			&bar.Low,
			&bar.Close,
			&bar.Volume,
			&bar.TradesCount,
			&bar.VWAP,
			&bar.OI,
			&bar.Source,
			&bar.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		bars = append(bars, bar)
	}

	return bars, nil
}

// GetLatestIntradayBar retrieves the most recent bar for a symbol
func (db *Database) GetLatestIntradayBar(symbol, timeframe string) (*IntradayBar, error) {
	query := `
		SELECT
			bar_id, exchange, symbol, instrument_token, bar_timestamp, timeframe,
			open, high, low, close, volume, trades_count, vwap, oi, source, created_at
		FROM md.intraday_bars
		WHERE symbol = $1 AND timeframe = $2
		ORDER BY bar_timestamp DESC
		LIMIT 1
	`

	var bar IntradayBar
	err := db.conn.QueryRow(query, symbol, timeframe).Scan(
		&bar.BarID,
		&bar.Exchange,
		&bar.Symbol,
		&bar.InstrumentToken,
		&bar.BarTimestamp,
		&bar.Timeframe,
		&bar.Open,
		&bar.High,
		&bar.Low,
		&bar.Close,
		&bar.Volume,
		&bar.TradesCount,
		&bar.VWAP,
		&bar.OI,
		&bar.Source,
		&bar.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &bar, nil
}

// GetTodayBars retrieves all bars for current trading day
func (db *Database) GetTodayBars(symbol, timeframe string) ([]IntradayBar, error) {
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	return db.GetIntradayBars(symbol, timeframe, today, tomorrow, 1000)
}

// ============================================================================
// TICK DATA OPERATIONS
// ============================================================================

// InsertTickData inserts a single tick
func (db *Database) InsertTickData(tick *TickData) error {
	query := `
		INSERT INTO md.tick_data (
			exchange, symbol, instrument_token, tick_timestamp,
			price, quantity, trade_type, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING tick_id
	`

	err := db.conn.QueryRow(
		query,
		tick.Exchange,
		tick.Symbol,
		tick.InstrumentToken,
		tick.TickTimestamp,
		tick.Price,
		tick.Quantity,
		tick.TradeType,
		tick.Source,
	).Scan(&tick.TickID)

	return err
}

// BulkInsertTickData efficiently inserts multiple ticks
func (db *Database) BulkInsertTickData(ticks []TickData) error {
	if len(ticks) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO md.tick_data (
			exchange, symbol, instrument_token, tick_timestamp,
			price, quantity, trade_type, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, tick := range ticks {
		_, err := stmt.Exec(
			tick.Exchange,
			tick.Symbol,
			tick.InstrumentToken,
			tick.TickTimestamp,
			tick.Price,
			tick.Quantity,
			tick.TradeType,
			tick.Source,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetTickData retrieves tick data for a symbol
func (db *Database) GetTickData(symbol string, fromTime, toTime time.Time, limit int) ([]TickData, error) {
	query := `
		SELECT
			tick_id, exchange, symbol, instrument_token, tick_timestamp,
			price, quantity, trade_type, source, created_at
		FROM md.tick_data
		WHERE symbol = $1
		  AND tick_timestamp >= $2
		  AND tick_timestamp <= $3
		ORDER BY tick_timestamp ASC
		LIMIT $4
	`

	rows, err := db.conn.Query(query, symbol, fromTime, toTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ticks := []TickData{}
	for rows.Next() {
		var tick TickData
		err := rows.Scan(
			&tick.TickID,
			&tick.Exchange,
			&tick.Symbol,
			&tick.InstrumentToken,
			&tick.TickTimestamp,
			&tick.Price,
			&tick.Quantity,
			&tick.TradeType,
			&tick.Source,
			&tick.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		ticks = append(ticks, tick)
	}

	return ticks, nil
}

// ============================================================================
// ORDER BOOK OPERATIONS
// ============================================================================

// InsertOrderBookSnapshot inserts an order book snapshot
func (db *Database) InsertOrderBookSnapshot(snapshot *OrderBookSnapshot) error {
	query := `
		INSERT INTO md.order_book (
			exchange, symbol, instrument_token, snapshot_timestamp,
			bids, asks, bid_quantity, ask_quantity, spread, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING snapshot_id
	`

	err := db.conn.QueryRow(
		query,
		snapshot.Exchange,
		snapshot.Symbol,
		snapshot.InstrumentToken,
		snapshot.SnapshotTimestamp,
		snapshot.Bids,
		snapshot.Asks,
		snapshot.BidQuantity,
		snapshot.AskQuantity,
		snapshot.Spread,
		snapshot.Source,
	).Scan(&snapshot.SnapshotID)

	return err
}

// GetLatestOrderBook retrieves the most recent order book snapshot
func (db *Database) GetLatestOrderBook(symbol string) (*OrderBookSnapshot, error) {
	query := `
		SELECT
			snapshot_id, exchange, symbol, instrument_token, snapshot_timestamp,
			bids, asks, bid_quantity, ask_quantity, spread, source, created_at
		FROM md.order_book
		WHERE symbol = $1
		ORDER BY snapshot_timestamp DESC
		LIMIT 1
	`

	var snapshot OrderBookSnapshot
	err := db.conn.QueryRow(query, symbol).Scan(
		&snapshot.SnapshotID,
		&snapshot.Exchange,
		&snapshot.Symbol,
		&snapshot.InstrumentToken,
		&snapshot.SnapshotTimestamp,
		&snapshot.Bids,
		&snapshot.Asks,
		&snapshot.BidQuantity,
		&snapshot.AskQuantity,
		&snapshot.Spread,
		&snapshot.Source,
		&snapshot.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// ============================================================================
// AGGREGATION & ANALYTICS
// ============================================================================

// CalculateTodayVWAP calculates VWAP for current trading day
func (db *Database) CalculateTodayVWAP(symbol, timeframe string) (float64, error) {
	query := `
		SELECT md.calculate_today_vwap($1, $2)
	`

	var vwap float64
	err := db.conn.QueryRow(query, symbol, timeframe).Scan(&vwap)
	return vwap, err
}

// GetIntradayStats retrieves statistics for current trading day
func (db *Database) GetIntradayStats(symbol, timeframe string) (map[string]interface{}, error) {
	query := `
		SELECT
			MIN(low) AS day_low,
			MAX(high) AS day_high,
			first(open, bar_timestamp) AS day_open,
			last(close, bar_timestamp) AS current_price,
			SUM(volume) AS total_volume,
			COUNT(*) AS bars_count
		FROM md.intraday_bars
		WHERE symbol = $1
		  AND timeframe = $2
		  AND bar_timestamp >= date_trunc('day', NOW())
	`

	var dayLow, dayHigh, dayOpen, currentPrice float64
	var totalVolume int64
	var barsCount int

	err := db.conn.QueryRow(query, symbol, timeframe).Scan(
		&dayLow,
		&dayHigh,
		&dayOpen,
		&currentPrice,
		&totalVolume,
		&barsCount,
	)

	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"day_low":       dayLow,
		"day_high":      dayHigh,
		"day_open":      dayOpen,
		"current_price": currentPrice,
		"total_volume":  totalVolume,
		"bars_count":    barsCount,
		"day_change":    currentPrice - dayOpen,
		"day_change_pct": ((currentPrice - dayOpen) / dayOpen) * 100,
	}

	return stats, nil
}

// ConvertBrokerCandlesToIntradayBars converts broker candles to intraday bars
func ConvertBrokerCandlesToIntradayBars(
	candles []broker.Candle,
	exchange, symbol string,
	instrumentToken int64,
	timeframe, source string,
) []IntradayBar {
	bars := make([]IntradayBar, len(candles))

	for i, candle := range candles {
		bars[i] = IntradayBar{
			Exchange:        exchange,
			Symbol:          symbol,
			InstrumentToken: instrumentToken,
			BarTimestamp:    candle.Date,
			Timeframe:       timeframe,
			Open:            candle.Open,
			High:            candle.High,
			Low:             candle.Low,
			Close:           candle.Close,
			Volume:          candle.Volume,
			Source:          source,
		}
	}

	return bars
}

// GetDataGaps identifies missing data gaps in intraday bars
func (db *Database) GetDataGaps(symbol, timeframe string, startTime, endTime time.Time) ([]map[string]interface{}, error) {
	query := `
		WITH expected_bars AS (
			SELECT generate_series(
				date_trunc('minute', $3),
				$4,
				CASE $2
					WHEN '1m' THEN INTERVAL '1 minute'
					WHEN '5m' THEN INTERVAL '5 minutes'
					WHEN '15m' THEN INTERVAL '15 minutes'
					WHEN '1h' THEN INTERVAL '1 hour'
					ELSE INTERVAL '1 day'
				END
			) AS expected_time
		)
		SELECT expected_time
		FROM expected_bars
		WHERE NOT EXISTS (
			SELECT 1 FROM md.intraday_bars
			WHERE symbol = $1
			  AND timeframe = $2
			  AND bar_timestamp = expected_bars.expected_time
		)
		ORDER BY expected_time
	`

	rows, err := db.conn.Query(query, symbol, timeframe, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	gaps := []map[string]interface{}{}
	for rows.Next() {
		var gapTime time.Time
		if err := rows.Scan(&gapTime); err != nil {
			return nil, err
		}
		gaps = append(gaps, map[string]interface{}{
			"missing_timestamp": gapTime,
			"symbol":            symbol,
			"timeframe":         timeframe,
		})
	}

	return gaps, nil
}

// GetDataCompleteness calculates data completeness percentage
func (db *Database) GetDataCompleteness(symbol, timeframe string, startTime, endTime time.Time) (float64, error) {
	gaps, err := db.GetDataGaps(symbol, timeframe, startTime, endTime)
	if err != nil {
		return 0, fmt.Errorf("failed to get gaps: %w", err)
	}

	// Calculate expected number of bars
	var expectedBars int64
	switch timeframe {
	case "1m":
		expectedBars = int64(endTime.Sub(startTime).Minutes())
	case "5m":
		expectedBars = int64(endTime.Sub(startTime).Minutes() / 5)
	case "15m":
		expectedBars = int64(endTime.Sub(startTime).Minutes() / 15)
	case "1h":
		expectedBars = int64(endTime.Sub(startTime).Hours())
	default:
		expectedBars = int64(endTime.Sub(startTime).Hours() / 24)
	}

	if expectedBars == 0 {
		return 0, nil
	}

	completeness := float64(expectedBars-int64(len(gaps))) / float64(expectedBars) * 100
	return completeness, nil
}
