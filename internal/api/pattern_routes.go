package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/trading-chitti/market-bridge/internal/analyzer"
	"github.com/trading-chitti/market-bridge/internal/broker"
	"github.com/trading-chitti/market-bridge/internal/database"
)

// PatternHandler handles pattern detection requests
type PatternHandler struct {
	broker  broker.Broker
	db      *database.Database
	scanner *analyzer.PatternScanner
}

// NewPatternHandler creates a new pattern handler
func NewPatternHandler(brk broker.Broker, db *database.Database) *PatternHandler {
	return &PatternHandler{
		broker:  brk,
		db:      db,
		scanner: analyzer.NewPatternScanner(),
	}
}

// ScanPatternsRequest represents a pattern scan request
type ScanPatternsRequest struct {
	Exchange       string  `form:"exchange" binding:"required"`
	Symbol         string  `form:"symbol" binding:"required"`
	Interval       string  `form:"interval"`
	Days           int     `form:"days"`
	MinConfidence  float64 `form:"min_confidence"`
	PatternTypes   []string `form:"pattern_types"`
	CategoryFilter string  `form:"category"` // "candlestick", "chart", or empty for all
}

// ScanMultipleRequest represents scanning multiple symbols
type ScanMultipleRequest struct {
	Symbols        []string `json:"symbols" binding:"required"`
	Exchange       string   `json:"exchange" binding:"required"`
	Interval       string   `json:"interval"`
	Days           int      `json:"days"`
	MinConfidence  float64  `json:"min_confidence"`
	CategoryFilter string   `json:"category"`
}

// RegisterRoutes registers pattern detection routes
func (h *PatternHandler) RegisterRoutes(r *gin.RouterGroup) {
	patterns := r.Group("/patterns")
	{
		patterns.GET("/scan", h.ScanPatterns)
		patterns.POST("/scan-multiple", h.ScanMultipleSymbols)
		patterns.GET("/types", h.ListPatternTypes)
		patterns.GET("/recent", h.GetRecentPatterns)
	}
}

// ScanPatterns scans for patterns in a symbol's historical data
func (h *PatternHandler) ScanPatterns(c *gin.Context) {
	var req ScanPatternsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// Set defaults
	if req.Interval == "" {
		req.Interval = "day"
	}
	if req.Days == 0 {
		req.Days = 60 // Default to 60 days
	}
	if req.MinConfidence == 0 {
		req.MinConfidence = 0.65
	}

	// Fetch historical data
	toDate := time.Now()
	fromDate := toDate.AddDate(0, 0, -req.Days)

	// Get instrument token
	instrumentToken, err := h.db.GetInstrumentToken(req.Exchange, req.Symbol)
	if err != nil || instrumentToken == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "instrument not found, please sync instruments first",
		})
		return
	}

	// Check cache first
	cachedCandles, err := h.db.GetHistoricalFromCache(instrumentToken, req.Interval, fromDate, toDate)
	var candles []broker.Candle

	if err == nil && len(cachedCandles) > 0 {
		// Convert database candles to broker candles
		candles = make([]broker.Candle, len(cachedCandles))
		for i, cc := range cachedCandles {
			candles[i] = broker.Candle{
				Date:   cc.CandleTimestamp,
				Open:   cc.Open,
				High:   cc.High,
				Low:    cc.Low,
				Close:  cc.Close,
				Volume: cc.Volume,
			}
		}
	} else {
		// Fetch from broker
		symbol := req.Exchange + ":" + req.Symbol
		candles, err = h.broker.GetHistoricalData(symbol, fromDate, toDate, req.Interval)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to fetch historical data: " + err.Error(),
			})
			return
		}

		// Cache the data
		dbCandles := make([]database.HistoricalCandle, len(candles))
		for i, candle := range candles {
			dbCandles[i] = database.HistoricalCandle{
				InstrumentToken: instrumentToken,
				Interval:        req.Interval,
				CandleTimestamp: candle.Date,
				Open:            candle.Open,
				High:            candle.High,
				Low:             candle.Low,
				Close:           candle.Close,
				Volume:          candle.Volume,
			}
		}
		h.db.CacheHistoricalCandles(dbCandles)
	}

	if len(candles) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"patterns": []analyzer.Pattern{},
			"message":  "no data available for analysis",
		})
		return
	}

	// Set minimum confidence
	h.scanner.MinConfidence = req.MinConfidence

	// Scan for patterns
	allPatterns := h.scanner.ScanAllPatterns(candles)

	// Filter by category if specified
	filtered := allPatterns
	if req.CategoryFilter != "" {
		filtered = []analyzer.Pattern{}
		for _, p := range allPatterns {
			if p.Category == req.CategoryFilter {
				filtered = append(filtered, p)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":         req.Symbol,
		"exchange":       req.Exchange,
		"interval":       req.Interval,
		"candles_count":  len(candles),
		"patterns_found": len(filtered),
		"patterns":       filtered,
		"scanned_at":     time.Now(),
	})
}

// ScanMultipleSymbols scans multiple symbols for patterns
func (h *PatternHandler) ScanMultipleSymbols(c *gin.Context) {
	var req ScanMultipleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// Set defaults
	if req.Interval == "" {
		req.Interval = "day"
	}
	if req.Days == 0 {
		req.Days = 60
	}
	if req.MinConfidence == 0 {
		req.MinConfidence = 0.65
	}

	results := make([]gin.H, 0, len(req.Symbols))
	toDate := time.Now()
	fromDate := toDate.AddDate(0, 0, -req.Days)

	h.scanner.MinConfidence = req.MinConfidence

	for _, symbol := range req.Symbols {
		// Get instrument token
		instrumentToken, err := h.db.GetInstrumentToken(req.Exchange, symbol)
		if err != nil || instrumentToken == 0 {
			results = append(results, gin.H{
				"symbol": symbol,
				"error":  "instrument not found",
			})
			continue
		}

		// Check cache first
		cachedCandles, err := h.db.GetHistoricalFromCache(instrumentToken, req.Interval, fromDate, toDate)
		var candles []broker.Candle

		if err == nil && len(cachedCandles) > 0 {
			candles = make([]broker.Candle, len(cachedCandles))
			for i, cc := range cachedCandles {
				candles[i] = broker.Candle{
					Date:   cc.CandleTimestamp,
					Open:   cc.Open,
					High:   cc.High,
					Low:    cc.Low,
					Close:  cc.Close,
					Volume: cc.Volume,
				}
			}
		} else {
			// Fetch from broker
			fullSymbol := req.Exchange + ":" + symbol
			candles, err = h.broker.GetHistoricalData(fullSymbol, fromDate, toDate, req.Interval)
			if err != nil {
				results = append(results, gin.H{
					"symbol": symbol,
					"error":  "failed to fetch data",
				})
				continue
			}
		}

		if len(candles) == 0 {
			results = append(results, gin.H{
				"symbol":   symbol,
				"patterns": []analyzer.Pattern{},
			})
			continue
		}

		// Scan for patterns
		allPatterns := h.scanner.ScanAllPatterns(candles)

		// Filter by category
		filtered := allPatterns
		if req.CategoryFilter != "" {
			filtered = []analyzer.Pattern{}
			for _, p := range allPatterns {
				if p.Category == req.CategoryFilter {
					filtered = append(filtered, p)
				}
			}
		}

		results = append(results, gin.H{
			"symbol":         symbol,
			"patterns_found": len(filtered),
			"patterns":       filtered,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"scanned_symbols": len(req.Symbols),
		"results":         results,
		"scanned_at":      time.Now(),
	})
}

// ListPatternTypes lists all supported pattern types
func (h *PatternHandler) ListPatternTypes(c *gin.Context) {
	patternTypes := map[string]interface{}{
		"candlestick_patterns": []gin.H{
			{"type": "Doji", "signal": "neutral", "description": "Indecision candle"},
			{"type": "Hammer", "signal": "bullish", "description": "Bullish reversal with long lower wick"},
			{"type": "Shooting Star", "signal": "bearish", "description": "Bearish reversal with long upper wick"},
			{"type": "Bullish Engulfing", "signal": "bullish", "description": "Bullish candle engulfs bearish"},
			{"type": "Bearish Engulfing", "signal": "bearish", "description": "Bearish candle engulfs bullish"},
			{"type": "Morning Star", "signal": "bullish", "description": "Three-candle bullish reversal"},
			{"type": "Evening Star", "signal": "bearish", "description": "Three-candle bearish reversal"},
			{"type": "Three White Soldiers", "signal": "bullish", "description": "Strong bullish continuation"},
			{"type": "Three Black Crows", "signal": "bearish", "description": "Strong bearish continuation"},
		},
		"chart_patterns": []gin.H{
			{"type": "Head and Shoulders", "signal": "bearish", "description": "Bearish reversal pattern"},
			{"type": "Inverse Head and Shoulders", "signal": "bullish", "description": "Bullish reversal pattern"},
			{"type": "Double Top", "signal": "bearish", "description": "Bearish reversal with two peaks"},
			{"type": "Double Bottom", "signal": "bullish", "description": "Bullish reversal with two troughs"},
			{"type": "Ascending Triangle", "signal": "bullish", "description": "Bullish continuation"},
			{"type": "Descending Triangle", "signal": "bearish", "description": "Bearish continuation"},
			{"type": "Symmetrical Triangle", "signal": "neutral", "description": "Continuation pattern"},
			{"type": "Bullish Flag", "signal": "bullish", "description": "Bullish continuation after rally"},
			{"type": "Bearish Flag", "signal": "bearish", "description": "Bearish continuation after decline"},
			{"type": "Rising Wedge", "signal": "bearish", "description": "Bearish reversal"},
			{"type": "Falling Wedge", "signal": "bullish", "description": "Bullish reversal"},
		},
		"total_patterns": 20,
	}

	c.JSON(http.StatusOK, patternTypes)
}

// GetRecentPatterns gets recent patterns from database (if pattern alerts are stored)
func (h *PatternHandler) GetRecentPatterns(c *gin.Context) {
	// TODO: Implement database storage for pattern alerts
	// For now, return empty array

	c.JSON(http.StatusOK, gin.H{
		"patterns": []analyzer.Pattern{},
		"message":  "Pattern alert storage coming soon",
	})
}
