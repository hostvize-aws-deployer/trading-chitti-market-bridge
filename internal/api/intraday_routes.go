package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/trading-chitti/market-bridge/internal/database"
)

// IntradayHandler handles intraday data requests
type IntradayHandler struct {
	db *database.Database
}

// NewIntradayHandler creates a new intraday handler
func NewIntradayHandler(db *database.Database) *IntradayHandler {
	return &IntradayHandler{db: db}
}

// RegisterRoutes registers intraday data routes
func (h *IntradayHandler) RegisterRoutes(r *gin.RouterGroup) {
	intraday := r.Group("/intraday")
	{
		intraday.GET("/bars/:symbol", h.GetIntradayBars)
		intraday.GET("/latest/:symbol", h.GetLatestBar)
		intraday.GET("/today/:symbol", h.GetTodayBars)
		intraday.GET("/stats/:symbol", h.GetIntradayStats)
		intraday.GET("/vwap/:symbol", h.GetTodayVWAP)
		intraday.GET("/ticks/:symbol", h.GetTickData)
		intraday.GET("/orderbook/:symbol", h.GetLatestOrderBook)
		intraday.GET("/gaps/:symbol", h.GetDataGaps)
		intraday.GET("/completeness/:symbol", h.GetDataCompleteness)
	}
}

// GetIntradayBars retrieves intraday bars for a symbol
// GET /intraday/bars/:symbol?timeframe=1m&from=2024-01-30T09:15:00Z&to=2024-01-30T15:30:00Z&limit=1000
func (h *IntradayHandler) GetIntradayBars(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.DefaultQuery("timeframe", "1m")
	limitStr := c.DefaultQuery("limit", "1000")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 10000 {
		limit = 1000
	}

	// Parse time range
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var fromTime, toTime time.Time

	if fromStr != "" {
		fromTime, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid 'from' time format, use RFC3339",
			})
			return
		}
	} else {
		// Default: last 24 hours
		fromTime = time.Now().Add(-24 * time.Hour)
	}

	if toStr != "" {
		toTime, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid 'to' time format, use RFC3339",
			})
			return
		}
	} else {
		toTime = time.Now()
	}

	// Validate timeframe
	validTimeframes := map[string]bool{
		"1m": true, "5m": true, "15m": true, "1h": true, "day": true,
	}
	if !validTimeframes[timeframe] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid timeframe, must be one of: 1m, 5m, 15m, 1h, day",
		})
		return
	}

	// Fetch data
	bars, err := h.db.GetIntradayBars(symbol, timeframe, fromTime, toTime, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch intraday bars: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":     symbol,
		"timeframe":  timeframe,
		"from":       fromTime,
		"to":         toTime,
		"bars_count": len(bars),
		"bars":       bars,
	})
}

// GetLatestBar retrieves the most recent bar for a symbol
// GET /intraday/latest/:symbol?timeframe=1m
func (h *IntradayHandler) GetLatestBar(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.DefaultQuery("timeframe", "1m")

	bar, err := h.db.GetLatestIntradayBar(symbol, timeframe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch latest bar: " + err.Error(),
		})
		return
	}

	if bar == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no data found for symbol",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":    symbol,
		"timeframe": timeframe,
		"bar":       bar,
	})
}

// GetTodayBars retrieves all bars for current trading day
// GET /intraday/today/:symbol?timeframe=1m
func (h *IntradayHandler) GetTodayBars(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.DefaultQuery("timeframe", "1m")

	bars, err := h.db.GetTodayBars(symbol, timeframe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch today's bars: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":     symbol,
		"timeframe":  timeframe,
		"date":       time.Now().Format("2006-01-02"),
		"bars_count": len(bars),
		"bars":       bars,
	})
}

// GetIntradayStats retrieves intraday statistics for current day
// GET /intraday/stats/:symbol?timeframe=1m
func (h *IntradayHandler) GetIntradayStats(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.DefaultQuery("timeframe", "1m")

	stats, err := h.db.GetIntradayStats(symbol, timeframe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch intraday stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":    symbol,
		"timeframe": timeframe,
		"date":      time.Now().Format("2006-01-02"),
		"stats":     stats,
	})
}

// GetTodayVWAP calculates VWAP for current trading day
// GET /intraday/vwap/:symbol?timeframe=1m
func (h *IntradayHandler) GetTodayVWAP(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.DefaultQuery("timeframe", "1m")

	vwap, err := h.db.CalculateTodayVWAP(symbol, timeframe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to calculate VWAP: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":    symbol,
		"timeframe": timeframe,
		"date":      time.Now().Format("2006-01-02"),
		"vwap":      vwap,
	})
}

// GetTickData retrieves tick-level data
// GET /intraday/ticks/:symbol?from=2024-01-30T09:15:00Z&to=2024-01-30T09:20:00Z&limit=1000
func (h *IntradayHandler) GetTickData(c *gin.Context) {
	symbol := c.Param("symbol")
	limitStr := c.DefaultQuery("limit", "1000")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50000 {
		limit = 1000
	}

	// Parse time range
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var fromTime, toTime time.Time

	if fromStr != "" {
		fromTime, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid 'from' time format",
			})
			return
		}
	} else {
		fromTime = time.Now().Add(-1 * time.Hour)
	}

	if toStr != "" {
		toTime, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid 'to' time format",
			})
			return
		}
	} else {
		toTime = time.Now()
	}

	// Fetch data
	ticks, err := h.db.GetTickData(symbol, fromTime, toTime, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch tick data: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":      symbol,
		"from":        fromTime,
		"to":          toTime,
		"ticks_count": len(ticks),
		"ticks":       ticks,
	})
}

// GetLatestOrderBook retrieves the most recent order book snapshot
// GET /intraday/orderbook/:symbol
func (h *IntradayHandler) GetLatestOrderBook(c *gin.Context) {
	symbol := c.Param("symbol")

	orderBook, err := h.db.GetLatestOrderBook(symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch order book: " + err.Error(),
		})
		return
	}

	if orderBook == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no order book data found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":     symbol,
		"order_book": orderBook,
	})
}

// GetDataGaps identifies missing data gaps
// GET /intraday/gaps/:symbol?timeframe=1m&from=2024-01-30T09:15:00Z&to=2024-01-30T15:30:00Z
func (h *IntradayHandler) GetDataGaps(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.DefaultQuery("timeframe", "1m")

	fromStr := c.Query("from")
	toStr := c.Query("to")

	var fromTime, toTime time.Time
	var err error

	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "'from' and 'to' parameters are required",
		})
		return
	}

	fromTime, err = time.Parse(time.RFC3339, fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid 'from' time format",
		})
		return
	}

	toTime, err = time.Parse(time.RFC3339, toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid 'to' time format",
		})
		return
	}

	gaps, err := h.db.GetDataGaps(symbol, timeframe, fromTime, toTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to identify gaps: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":     symbol,
		"timeframe":  timeframe,
		"from":       fromTime,
		"to":         toTime,
		"gaps_count": len(gaps),
		"gaps":       gaps,
	})
}

// GetDataCompleteness calculates data completeness percentage
// GET /intraday/completeness/:symbol?timeframe=1m&from=2024-01-30T09:15:00Z&to=2024-01-30T15:30:00Z
func (h *IntradayHandler) GetDataCompleteness(c *gin.Context) {
	symbol := c.Param("symbol")
	timeframe := c.DefaultQuery("timeframe", "1m")

	fromStr := c.Query("from")
	toStr := c.Query("to")

	var fromTime, toTime time.Time
	var err error

	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "'from' and 'to' parameters are required",
		})
		return
	}

	fromTime, err = time.Parse(time.RFC3339, fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid 'from' time format",
		})
		return
	}

	toTime, err = time.Parse(time.RFC3339, toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid 'to' time format",
		})
		return
	}

	completeness, err := h.db.GetDataCompleteness(symbol, timeframe, fromTime, toTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to calculate completeness: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol":          symbol,
		"timeframe":       timeframe,
		"from":            fromTime,
		"to":              toTime,
		"completeness":    completeness,
		"completeness_pct": completeness,
		"quality":         getQualityRating(completeness),
	})
}

func getQualityRating(completeness float64) string {
	if completeness >= 99.0 {
		return "excellent"
	} else if completeness >= 95.0 {
		return "good"
	} else if completeness >= 90.0 {
		return "fair"
	} else {
		return "poor"
	}
}
