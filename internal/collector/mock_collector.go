package collector

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/trading-chitti/market-bridge/internal/database"
)

// MockDataCollector generates fake market data for testing
type MockDataCollector struct {
	db             *database.Database
	name           string
	symbols        []string
	mode           string // "full" or "bars_only"

	// Control
	ctx            context.Context
	cancel         context.CancelFunc
	running        bool
	mu             sync.RWMutex

	// Metrics
	ticksGenerated int64
	barsGenerated  int64
	errors         int64
	startedAt      time.Time
	lastTickAt     time.Time

	// Price tracking for realistic movements
	basePrices     map[string]float64
	pricesMu       sync.RWMutex
}

// NewMockDataCollector creates a new mock data collector
func NewMockDataCollector(db *database.Database, name string, symbols []string) *MockDataCollector {
	ctx, cancel := context.WithCancel(context.Background())

	return &MockDataCollector{
		db:         db,
		name:       name,
		symbols:    symbols,
		mode:       "full",
		ctx:        ctx,
		cancel:     cancel,
		basePrices: make(map[string]float64),
	}
}

// Start begins mock data generation
func (mc *MockDataCollector) Start() error {
	mc.mu.Lock()
	if mc.running {
		mc.mu.Unlock()
		return fmt.Errorf("mock collector already running")
	}
	mc.running = true
	mc.startedAt = time.Now()
	mc.mu.Unlock()

	// Initialize base prices for symbols
	mc.initializeBasePrices()

	// Start tick generation
	go mc.generateTicks()

	// Start bar aggregation (every minute)
	go mc.aggregateBars()

	log.Printf("✅ Mock collector '%s' started with %d symbols", mc.name, len(mc.symbols))
	return nil
}

// Stop stops mock data generation
func (mc *MockDataCollector) Stop() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.running {
		return
	}

	mc.running = false
	mc.cancel()

	log.Printf("🛑 Mock collector '%s' stopped", mc.name)
}

// IsRunning returns whether collector is active
func (mc *MockDataCollector) IsRunning() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.running
}

// GetMetrics returns collector metrics
func (mc *MockDataCollector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	uptime := int64(0)
	if mc.running {
		uptime = int64(time.Since(mc.startedAt).Seconds())
	}

	return map[string]interface{}{
		"running":           mc.running,
		"symbols":           mc.symbols,
		"symbols_count":     len(mc.symbols),
		"ticks_generated":   mc.ticksGenerated,
		"bars_generated":    mc.barsGenerated,
		"errors":            mc.errors,
		"uptime_seconds":    uptime,
		"started_at":        mc.startedAt,
		"last_tick_at":      mc.lastTickAt,
	}
}

// AddSymbols adds symbols to collection
func (mc *MockDataCollector) AddSymbols(symbols []string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Add unique symbols
	symbolMap := make(map[string]bool)
	for _, sym := range mc.symbols {
		symbolMap[sym] = true
	}
	for _, sym := range symbols {
		symbolMap[sym] = true
	}

	mc.symbols = make([]string, 0, len(symbolMap))
	for sym := range symbolMap {
		mc.symbols = append(mc.symbols, sym)
	}

	// Initialize base prices for new symbols
	mc.initializeBasePrices()
}

// RemoveSymbols removes symbols from collection
func (mc *MockDataCollector) RemoveSymbols(symbols []string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	removeMap := make(map[string]bool)
	for _, sym := range symbols {
		removeMap[sym] = true
	}

	newSymbols := []string{}
	for _, sym := range mc.symbols {
		if !removeMap[sym] {
			newSymbols = append(newSymbols, sym)
		}
	}

	mc.symbols = newSymbols
}

// initializeBasePrices sets realistic base prices for symbols
func (mc *MockDataCollector) initializeBasePrices() {
	mc.pricesMu.Lock()
	defer mc.pricesMu.Unlock()

	// Realistic base prices for common Indian stocks
	knownPrices := map[string]float64{
		"RELIANCE":   2500.0,
		"TCS":        3500.0,
		"INFY":       1450.0,
		"HDFCBANK":   1650.0,
		"ICICIBANK":  1100.0,
		"SBIN":       750.0,
		"BHARTIARTL": 1200.0,
		"ITC":        450.0,
		"HINDUNILVR": 2400.0,
		"LT":         3450.0,
		"KOTAKBANK":  1750.0,
		"AXISBANK":   1050.0,
		"BAJFINANCE": 6800.0,
		"ASIANPAINT": 2900.0,
		"MARUTI":     12500.0,
		"TATASTEEL":  140.0,
		"WIPRO":      450.0,
		"SUNPHARMA":  1650.0,
		"TITAN":      3200.0,
		"NESTLEIND":  2400.0,
	}

	for _, symbol := range mc.symbols {
		if _, exists := mc.basePrices[symbol]; !exists {
			// Use known price if available, otherwise generate random
			if price, ok := knownPrices[symbol]; ok {
				mc.basePrices[symbol] = price
			} else {
				// Random price between 100 and 5000
				mc.basePrices[symbol] = 100 + rand.Float64()*4900
			}
		}
	}
}

// generateTicks generates fake tick data
func (mc *MockDataCollector) generateTicks() {
	// Generate a tick every 1-3 seconds for each symbol
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.mu.RLock()
			symbols := mc.symbols
			mode := mc.mode
			mc.mu.RUnlock()

			if mode == "bars_only" {
				continue // Skip tick generation in bars_only mode
			}

			// Generate tick for each symbol
			for _, symbol := range symbols {
				if err := mc.generateTickForSymbol(symbol); err != nil {
					log.Printf("❌ Failed to generate tick for %s: %v", symbol, err)
					mc.mu.Lock()
					mc.errors++
					mc.mu.Unlock()
				}
			}
		}
	}
}

// generateTickForSymbol generates a single tick for a symbol
func (mc *MockDataCollector) generateTickForSymbol(symbol string) error {
	mc.pricesMu.RLock()
	basePrice := mc.basePrices[symbol]
	mc.pricesMu.RUnlock()

	// Generate realistic price movement (±0.5%)
	priceChange := (rand.Float64() - 0.5) * basePrice * 0.01
	currentPrice := basePrice + priceChange

	// Update base price slowly to simulate trends
	mc.pricesMu.Lock()
	mc.basePrices[symbol] = mc.basePrices[symbol]*0.99 + currentPrice*0.01
	mc.pricesMu.Unlock()

	// Determine trade type based on price movement
	tradeType := "buy"
	if priceChange < 0 {
		tradeType = "sell"
	}

	// Generate random quantity
	quantity := int64(rand.Intn(900) + 100) // 100-1000

	tick := &database.TickData{
		Exchange:      "NSE",
		Symbol:        symbol,
		TickTimestamp: time.Now(),
		Price:         currentPrice,
		Quantity:      quantity,
		TradeType:     tradeType,
		Source:        fmt.Sprintf("mock_%s", mc.name),
	}

	if err := mc.db.InsertTickData(tick); err != nil {
		return err
	}

	mc.mu.Lock()
	mc.ticksGenerated++
	mc.lastTickAt = time.Now()
	mc.mu.Unlock()

	return nil
}

// aggregateBars aggregates ticks into 1-minute bars
func (mc *MockDataCollector) aggregateBars() {
	// Wait for the next minute boundary to start
	now := time.Now()
	nextMinute := now.Truncate(time.Minute).Add(time.Minute)
	waitDuration := nextMinute.Sub(now)

	time.Sleep(waitDuration)

	// Now run every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.mu.RLock()
			symbols := mc.symbols
			mc.mu.RUnlock()

			// Aggregate bars for each symbol
			for _, symbol := range symbols {
				if err := mc.aggregateBarForSymbol(symbol); err != nil {
					log.Printf("❌ Failed to aggregate bar for %s: %v", symbol, err)
					mc.mu.Lock()
					mc.errors++
					mc.mu.Unlock()
				}
			}
		}
	}
}

// aggregateBarForSymbol aggregates the last minute of ticks into a bar
func (mc *MockDataCollector) aggregateBarForSymbol(symbol string) error {
	now := time.Now().Truncate(time.Minute)
	oneMinuteAgo := now.Add(-1 * time.Minute)

	// Get ticks from the last minute
	ticks, err := mc.db.GetTickData(symbol, oneMinuteAgo, now, 10000)
	if err != nil {
		return fmt.Errorf("failed to get ticks: %w", err)
	}

	if len(ticks) == 0 {
		// No ticks in last minute, skip
		return nil
	}

	// Calculate OHLCV
	open := ticks[0].Price
	close := ticks[len(ticks)-1].Price
	high := open
	low := open
	var volume int64

	for _, tick := range ticks {
		if tick.Price > high {
			high = tick.Price
		}
		if tick.Price < low {
			low = tick.Price
		}
		volume += tick.Quantity
	}

	tradesCount := len(ticks)

	bar := &database.IntradayBar{
		Exchange:     "NSE",
		Symbol:       symbol,
		BarTimestamp: oneMinuteAgo,
		Timeframe:    "1m",
		Open:         open,
		High:         high,
		Low:          low,
		Close:        close,
		Volume:       volume,
		TradesCount:  &tradesCount,
		Source:       fmt.Sprintf("mock_%s", mc.name),
	}

	if err := mc.db.InsertIntradayBar(bar); err != nil {
		return fmt.Errorf("failed to insert bar: %w", err)
	}

	mc.mu.Lock()
	mc.barsGenerated++
	mc.mu.Unlock()

	log.Printf("📊 Generated 1m bar for %s: O=%.2f H=%.2f L=%.2f C=%.2f V=%d",
		symbol, open, high, low, close, volume)

	return nil
}
