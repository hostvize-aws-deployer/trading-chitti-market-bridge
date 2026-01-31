package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// SearchInstruments searches for instruments by symbol or name
func (a *API) SearchInstruments(c *gin.Context) {
	pattern := c.Query("q")
	if pattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "query parameter 'q' is required",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	instruments, err := a.db.SearchInstruments(pattern, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to search instruments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":       pattern,
		"count":       len(instruments),
		"instruments": instruments,
	})
}

// GetInstrumentByToken returns instrument details for a given token
func (a *API) GetInstrumentByToken(c *gin.Context) {
	tokenStr := c.Param("token")
	token64, err := strconv.ParseUint(tokenStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid instrument token",
		})
		return
	}

	token := uint32(token64)
	instrument, err := a.db.GetInstrumentByToken(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch instrument",
		})
		return
	}

	if instrument == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "instrument not found",
		})
		return
	}

	c.JSON(http.StatusOK, instrument)
}

// SyncInstruments syncs instruments from broker to database
func (a *API) SyncInstruments(c *gin.Context) {
	exchange := c.Query("exchange")

	if exchange != "" {
		// Sync specific exchange
		err := a.db.SyncInstrumentsByExchange(a.broker, exchange)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to sync instruments",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "instruments synced successfully",
			"exchange": exchange,
		})
	} else {
		// Sync all instruments
		err := a.db.SyncInstrumentsFromBroker(a.broker)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to sync instruments",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "all instruments synced successfully",
		})
	}
}

// GetHistoricalData returns historical candle data with caching
func (a *API) GetHistoricalData(c *gin.Context) {
	type HistoricalRequest struct {
		Exchange  string `json:"exchange" binding:"required"`
		Symbol    string `json:"symbol" binding:"required"`
		Interval  string `json:"interval" binding:"required"`
		FromDate  string `json:"from_date" binding:"required"`
		ToDate    string `json:"to_date" binding:"required"`
	}

	var req HistoricalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Parse dates
	fromDate, err := time.Parse("2006-01-02", req.FromDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid from_date format (use YYYY-MM-DD)",
		})
		return
	}

	toDate, err := time.Parse("2006-01-02", req.ToDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid to_date format (use YYYY-MM-DD)",
		})
		return
	}

	// Fetch historical data (with caching)
	if a.historicalService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "historical data service not available",
		})
		return
	}

	candles, err := a.historicalService.GetHistoricalData(
		req.Exchange,
		req.Symbol,
		req.Interval,
		fromDate,
		toDate,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch historical data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exchange": req.Exchange,
		"symbol":   req.Symbol,
		"interval": req.Interval,
		"count":    len(candles),
		"candles":  candles,
	})
}

// Get52DayHistorical returns 52 trading days of historical data
func (a *API) Get52DayHistorical(c *gin.Context) {
	exchange := c.Query("exchange")
	symbol := c.Query("symbol")

	if exchange == "" || symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "exchange and symbol query parameters are required",
		})
		return
	}

	if a.historicalService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "historical data service not available",
		})
		return
	}

	candles, err := a.historicalService.Get52DayHistoricalData(exchange, symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch 52-day historical data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exchange": exchange,
		"symbol":   symbol,
		"days":     len(candles),
		"candles":  candles,
	})
}

// WarmCache pre-fetches and caches historical data
func (a *API) WarmCache(c *gin.Context) {
	type WarmCacheRequest struct {
		Exchange string   `json:"exchange" binding:"required"`
		Symbols  []string `json:"symbols" binding:"required"`
		Interval string   `json:"interval" binding:"required"`
		Days     int      `json:"days" binding:"required,min=1,max=2000"`
	}

	var req WarmCacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if a.historicalService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "historical data service not available",
		})
		return
	}

	// Run cache warming in background
	go func() {
		err := a.historicalService.WarmCache(req.Exchange, req.Symbols, req.Interval, req.Days)
		if err != nil {
			a.logger.Error("Cache warming failed: ", err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "cache warming started in background",
		"symbols": len(req.Symbols),
		"days":    req.Days,
	})
}
