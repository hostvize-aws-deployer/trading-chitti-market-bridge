package collector

import (
	"context"
	"log"
	"sync"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"
	"github.com/zerodha/gokiteconnect/v4/models"
	"github.com/trading-chitti/market-bridge/internal/database"
)

// DataCollector manages real-time market data collection
type DataCollector struct {
	db             *database.Database
	ticker         *kiteticker.Ticker
	apiKey         string
	accessToken    string

	// Subscribed instruments
	subscribedTokens []uint32
	tokenToSymbol    map[uint32]string
	mu               sync.RWMutex

	// Candle aggregation
	candleBuilders   map[uint32]*CandleBuilder
	builderMu        sync.RWMutex

	// Control
	ctx              context.Context
	cancel           context.CancelFunc
	running          bool

	// Metrics
	ticksReceived    int64
	barsCreated      int64
	errors           int64
}

// CandleBuilder aggregates ticks into OHLCV candles
type CandleBuilder struct {
	InstrumentToken int64
	Symbol          string
	Exchange        string
	Timeframe       string

	// Current candle data
	CurrentOpen      float64
	CurrentHigh      float64
	CurrentLow       float64
	CurrentClose     float64
	CurrentVolume    int64
	CurrentTimestamp time.Time

	mu sync.Mutex
}

// NewDataCollector creates a new data collector
func NewDataCollector(db *database.Database, apiKey, accessToken string) *DataCollector {
	ctx, cancel := context.WithCancel(context.Background())

	return &DataCollector{
		db:               db,
		apiKey:           apiKey,
		accessToken:      accessToken,
		tokenToSymbol:    make(map[uint32]string),
		candleBuilders:   make(map[uint32]*CandleBuilder),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start begins data collection
func (dc *DataCollector) Start() error {
	dc.mu.Lock()
	if dc.running {
		dc.mu.Unlock()
		return nil
	}
	dc.running = true
	dc.mu.Unlock()

	// Initialize Kite Ticker
	dc.ticker = kiteticker.New(dc.apiKey, dc.accessToken)

	// Set up callbacks
	dc.ticker.OnConnect(dc.onConnect)
	dc.ticker.OnTick(dc.onTick)
	dc.ticker.OnReconnect(dc.onReconnect)
	dc.ticker.OnNoReconnect(dc.onNoReconnect)
	dc.ticker.OnError(dc.onError)
	dc.ticker.OnClose(dc.onClose)
	dc.ticker.OnOrderUpdate(dc.onOrderUpdate)

	// Enable auto-reconnect
	dc.ticker.SetAutoReconnect(true)
	dc.ticker.SetReconnectMaxRetries(10)
	dc.ticker.SetReconnectMaxDelay(60 * time.Second)

	// Start periodic candle flushing
	go dc.flushCandlesPeriodically()

	// Serve (blocking call)
	go func() {
		dc.ticker.Serve()
	}()

	log.Println("✅ Data collector started")
	return nil
}

// Stop stops data collection
func (dc *DataCollector) Stop() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if !dc.running {
		return
	}

	dc.running = false
	dc.cancel()

	if dc.ticker != nil {
		dc.ticker.Stop()
	}

	// Flush remaining candles
	dc.flushAllCandles()

	log.Println("🛑 Data collector stopped")
}

// Subscribe adds instruments to collect data for
func (dc *DataCollector) Subscribe(tokens []uint32) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.subscribedTokens = append(dc.subscribedTokens, tokens...)

	if dc.ticker != nil && dc.running {
		// Subscribe to full mode (OHLC + depth + LTP)
		return dc.ticker.Subscribe(tokens)
	}

	return nil
}

// Unsubscribe removes instruments from collection
func (dc *DataCollector) Unsubscribe(tokens []uint32) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.ticker != nil && dc.running {
		return dc.ticker.Unsubscribe(tokens)
	}

	return nil
}

// SetMode sets subscription mode for instruments
func (dc *DataCollector) SetMode(mode string, tokens []uint32) error {
	if dc.ticker == nil || !dc.running {
		return nil
	}

	switch mode {
	case "ltp":
		return dc.ticker.SetMode(kiteticker.ModeLTP, tokens)
	case "quote":
		return dc.ticker.SetMode(kiteticker.ModeQuote, tokens)
	case "full":
		return dc.ticker.SetMode(kiteticker.ModeFull, tokens)
	default:
		return dc.ticker.SetMode(kiteticker.ModeLTP, tokens)
	}
}

// RegisterSymbol maps a token to a symbol
func (dc *DataCollector) RegisterSymbol(token uint32, exchange, symbol string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.tokenToSymbol[token] = symbol

	// Initialize candle builders for different timeframes
	dc.builderMu.Lock()
	dc.candleBuilders[token] = &CandleBuilder{
		InstrumentToken: int64(token),
		Symbol:          symbol,
		Exchange:        exchange,
		Timeframe:       "1m",
	}
	dc.builderMu.Unlock()
}

// ============================================================================
// CALLBACKS
// ============================================================================

func (dc *DataCollector) onConnect() {
	log.Println("✅ Connected to Kite Ticker")

	// Resubscribe to instruments
	dc.mu.RLock()
	tokens := dc.subscribedTokens
	dc.mu.RUnlock()

	if len(tokens) > 0 {
		if err := dc.ticker.Subscribe(tokens); err != nil {
			log.Printf("❌ Failed to subscribe: %v", err)
		}

		// Set to full mode for complete data
		if err := dc.ticker.SetMode(kiteticker.ModeFull, tokens); err != nil {
			log.Printf("❌ Failed to set mode: %v", err)
		}

		log.Printf("📊 Subscribed to %d instruments", len(tokens))
	}
}

func (dc *DataCollector) onTick(tick models.Tick) {
	dc.ticksReceived++

	// Store tick data
	go dc.storeTick(tick)

	// Update candle builders
	go dc.updateCandles(tick)
}

func (dc *DataCollector) onReconnect(attempt int, delay time.Duration) {
	log.Printf("🔄 Reconnecting (attempt %d, delay %v)", attempt, delay)
}

func (dc *DataCollector) onNoReconnect(attempt int) {
	log.Printf("❌ Reconnection failed after %d attempts", attempt)
	dc.errors++
}

func (dc *DataCollector) onError(err error) {
	log.Printf("❌ Ticker error: %v", err)
	dc.errors++
}

func (dc *DataCollector) onClose(code int, reason string) {
	log.Printf("🔌 Connection closed: code=%d, reason=%s", code, reason)
}

func (dc *DataCollector) onOrderUpdate(order kiteconnect.Order) {
	log.Printf("📋 Order update: %s - %s", order.OrderID, order.Status)
	// TODO: Store order updates in database
}

// ============================================================================
// DATA STORAGE
// ============================================================================

func (dc *DataCollector) storeTick(tickData interface{}) {
	// Type assert to kiteticker.Tick
	tick, ok := tickData.(map[string]interface{})
	if !ok {
		return
	}
	// Extract instrument token
	instrumentToken, ok := tick["instrument_token"].(uint32)
	if !ok {
		return
	}

	dc.mu.RLock()
	symbol, exists := dc.tokenToSymbol[instrumentToken]
	dc.mu.RUnlock()

	if !exists {
		return
	}

	// Extract price and quantity
	lastPrice, _ := tick["last_price"].(float64)
	lastQuantity, _ := tick["last_quantity"].(uint32)
	timestamp, _ := tick["timestamp"].(time.Time)

	dbTickData := &database.TickData{
		Exchange:        "NSE", // TODO: Get from instrument lookup
		Symbol:          symbol,
		InstrumentToken: int64(instrumentToken),
		TickTimestamp:   timestamp,
		Price:           lastPrice,
		Quantity:        int64(lastQuantity),
		TradeType:       "unknown",
		Source:          "zerodha",
	}

	if err := dc.db.InsertTickData(dbTickData); err != nil {
		log.Printf("❌ Failed to store tick: %v", err)
		dc.errors++
	}
}

func (dc *DataCollector) updateCandles(tick models.Tick) {
	dc.builderMu.RLock()
	builder, exists := dc.candleBuilders[tick.InstrumentToken]
	dc.builderMu.RUnlock()

	if !exists {
		return
	}

	builder.mu.Lock()
	defer builder.mu.Unlock()

	now := time.Now()
	currentMinute := now.Truncate(time.Minute)

	// Check if we need to start a new candle
	if builder.CurrentTimestamp.IsZero() || !builder.CurrentTimestamp.Equal(currentMinute) {
		// Flush old candle if exists
		if !builder.CurrentTimestamp.IsZero() {
			dc.flushCandle(builder)
		}

		// Start new candle
		builder.CurrentTimestamp = currentMinute
		builder.CurrentOpen = tick.LastPrice
		builder.CurrentHigh = tick.LastPrice
		builder.CurrentLow = tick.LastPrice
		builder.CurrentClose = tick.LastPrice
		builder.CurrentVolume = int64(tick.LastTradedQuantity)
	} else {
		// Update existing candle
		if tick.LastPrice > builder.CurrentHigh {
			builder.CurrentHigh = tick.LastPrice
		}
		if tick.LastPrice < builder.CurrentLow {
			builder.CurrentLow = tick.LastPrice
		}
		builder.CurrentClose = tick.LastPrice
		builder.CurrentVolume += int64(tick.LastTradedQuantity)
	}
}

func (dc *DataCollector) flushCandle(builder *CandleBuilder) {
	if builder.CurrentTimestamp.IsZero() {
		return
	}

	bar := &database.IntradayBar{
		Exchange:        builder.Exchange,
		Symbol:          builder.Symbol,
		InstrumentToken: builder.InstrumentToken,
		BarTimestamp:    builder.CurrentTimestamp,
		Timeframe:       builder.Timeframe,
		Open:            builder.CurrentOpen,
		High:            builder.CurrentHigh,
		Low:             builder.CurrentLow,
		Close:           builder.CurrentClose,
		Volume:          builder.CurrentVolume,
		Source:          "zerodha_websocket",
	}

	if err := dc.db.InsertIntradayBar(bar); err != nil {
		log.Printf("❌ Failed to store bar: %v", err)
		dc.errors++
	} else {
		dc.barsCreated++
	}
}

func (dc *DataCollector) flushAllCandles() {
	dc.builderMu.RLock()
	defer dc.builderMu.RUnlock()

	for _, builder := range dc.candleBuilders {
		builder.mu.Lock()
		dc.flushCandle(builder)
		builder.mu.Unlock()
	}

	log.Printf("💾 Flushed all candles")
}

func (dc *DataCollector) flushCandlesPeriodically() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dc.flushAllCandles()
		case <-dc.ctx.Done():
			return
		}
	}
}

// ============================================================================
// METRICS
// ============================================================================

// GetMetrics returns collector metrics
func (dc *DataCollector) GetMetrics() map[string]interface{} {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return map[string]interface{}{
		"running":           dc.running,
		"subscribed_tokens": len(dc.subscribedTokens),
		"ticks_received":    dc.ticksReceived,
		"bars_created":      dc.barsCreated,
		"errors":            dc.errors,
	}
}

// IsRunning returns whether collector is active
func (dc *DataCollector) IsRunning() bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.running
}
