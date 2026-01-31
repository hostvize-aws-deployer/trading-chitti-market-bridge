package database

import (
	"log"
	"time"

	"github.com/trading-chitti/market-bridge/internal/broker"
)

// HistoricalDataService provides cached historical data access
type HistoricalDataService struct {
	db     *Database
	broker broker.Broker
}

// NewHistoricalDataService creates a new historical data service
func NewHistoricalDataService(db *Database, brk broker.Broker) *HistoricalDataService {
	return &HistoricalDataService{
		db:     db,
		broker: brk,
	}
}

// GetHistoricalData fetches historical data with caching
func (s *HistoricalDataService) GetHistoricalData(
	exchange, symbol, interval string,
	fromDate, toDate time.Time,
) ([]HistoricalCandle, error) {
	// Get instrument token
	token, err := s.db.GetInstrumentToken(exchange, symbol)
	if err != nil {
		return nil, err
	}

	if token == 0 {
		log.Printf("⚠️  Instrument token not found for %s:%s", exchange, symbol)
		return nil, nil
	}

	// Check cache first
	cached, err := s.db.GetHistoricalFromCache(token, interval, fromDate, toDate)
	if err != nil {
		log.Printf("❌ Cache read error: %v", err)
		// Continue to fetch from broker
	}

	// If we have complete data in cache, return it
	if s.isCacheComplete(cached, fromDate, toDate, interval) {
		log.Printf("✅ Returning %d candles from cache for %s", len(cached), symbol)
		return cached, nil
	}

	// Fetch from broker
	log.Printf("🔄 Fetching historical data from broker for %s (%s)", symbol, interval)

	brokerCandles, err := s.broker.GetHistoricalData(symbol, fromDate, toDate, interval)
	if err != nil {
		return nil, err
	}

	// Convert broker candles to database format
	dbCandles := make([]HistoricalCandle, len(brokerCandles))
	for i, candle := range brokerCandles {
		dbCandles[i] = HistoricalCandle{
			InstrumentToken:  token,
			Interval:         interval,
			CandleTimestamp:  candle.Date,
			Open:             candle.Open,
			High:             candle.High,
			Low:              candle.Low,
			Close:            candle.Close,
			Volume:           int64(candle.Volume),
			OI:               0, // OI not available in broker.Candle
			CachedAt:         time.Now(),
		}
	}

	// Cache the data
	if err := s.db.CacheHistoricalCandles(dbCandles); err != nil {
		log.Printf("⚠️  Failed to cache historical data: %v", err)
		// Continue anyway, return the data
	} else {
		log.Printf("💾 Cached %d candles for %s", len(dbCandles), symbol)
	}

	return dbCandles, nil
}

// isCacheComplete checks if cached data is complete for the requested range
func (s *HistoricalDataService) isCacheComplete(
	cached []HistoricalCandle,
	fromDate, toDate time.Time,
	interval string,
) bool {
	if len(cached) == 0 {
		return false
	}

	// Simple check: first and last candle timestamps match requested range
	// More sophisticated gap detection could be added here

	firstCandle := cached[0].CandleTimestamp
	lastCandle := cached[len(cached)-1].CandleTimestamp

	// Allow some tolerance for weekends/holidays
	tolerance := 24 * time.Hour

	fromMatch := firstCandle.Sub(fromDate).Abs() <= tolerance
	toMatch := toDate.Sub(lastCandle).Abs() <= tolerance

	return fromMatch && toMatch
}

// Get52DayHistoricalData fetches 52 trading days of historical data
func (s *HistoricalDataService) Get52DayHistoricalData(
	exchange, symbol string,
) ([]HistoricalCandle, error) {
	// Fetch ~75 calendar days to ensure we get 52 trading days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -75)

	candles, err := s.GetHistoricalData(exchange, symbol, "day", startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Keep only the last 52 candles
	if len(candles) > 52 {
		candles = candles[len(candles)-52:]
	}

	return candles, nil
}

// WarmCache pre-fetches and caches historical data for symbols
func (s *HistoricalDataService) WarmCache(
	exchange string,
	symbols []string,
	interval string,
	days int,
) error {
	log.Printf("🔥 Warming cache for %d symbols (%s, %d days)", len(symbols), interval, days)

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Rate limiter: 3 requests per second for historical data
	rateLimiter := time.NewTicker(350 * time.Millisecond)
	defer rateLimiter.Stop()

	for i, symbol := range symbols {
		<-rateLimiter.C // Wait for rate limiter

		_, err := s.GetHistoricalData(exchange, symbol, interval, startDate, endDate)
		if err != nil {
			log.Printf("❌ Failed to warm cache for %s: %v", symbol, err)
			continue
		}

		if (i+1)%10 == 0 {
			log.Printf("📊 Warmed cache for %d/%d symbols", i+1, len(symbols))
		}
	}

	log.Printf("✅ Cache warming completed for %d symbols", len(symbols))
	return nil
}
