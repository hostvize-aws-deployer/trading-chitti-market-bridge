package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trading-chitti/market-bridge/internal/collector"
	"github.com/trading-chitti/market-bridge/internal/database"
)

// CollectorHandler handles data collector API requests
type CollectorHandler struct {
	manager *collector.CollectorManager
}

// NewCollectorHandler creates a new collector handler
func NewCollectorHandler(db *database.Database) *CollectorHandler {
	return &CollectorHandler{
		manager: collector.NewCollectorManager(db),
	}
}

// RegisterRoutes registers collector routes
func (h *CollectorHandler) RegisterRoutes(r *gin.RouterGroup) {
	collectors := r.Group("/collectors")
	{
		collectors.POST("", h.CreateCollector)
		collectors.GET("", h.ListCollectors)
		collectors.GET("/:name", h.GetCollectorStatus)
		collectors.POST("/:name/start", h.StartCollector)
		collectors.POST("/:name/stop", h.StopCollector)
		collectors.POST("/:name/subscribe", h.SubscribeSymbols)
		collectors.POST("/:name/unsubscribe", h.UnsubscribeSymbols)
		collectors.GET("/metrics", h.GetMetrics)
	}
}

// CreateCollectorRequest represents collector creation request
type CreateCollectorRequest struct {
	Name        string `json:"name" binding:"required"`
	APIKey      string `json:"api_key" binding:"required"`
	AccessToken string `json:"access_token" binding:"required"`
}

// SubscribeRequest represents symbol subscription request
type SubscribeRequest struct {
	Symbols []string `json:"symbols" binding:"required"`
}

// CreateCollector creates a new data collector
// POST /collectors
func (h *CollectorHandler) CreateCollector(c *gin.Context) {
	var req CreateCollectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	_, err := h.manager.CreateCollector(req.Name, req.APIKey, req.AccessToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "collector created successfully",
		"name":    req.Name,
	})
}

// ListCollectors lists all collectors
// GET /collectors
func (h *CollectorHandler) ListCollectors(c *gin.Context) {
	names := h.manager.ListCollectors()

	collectors := make([]gin.H, 0, len(names))
	for _, name := range names {
		collector, err := h.manager.GetCollector(name)
		if err != nil {
			continue
		}

		collectors = append(collectors, gin.H{
			"name":    name,
			"running": collector.IsRunning(),
			"metrics": collector.GetMetrics(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"collectors": collectors,
		"total":      len(collectors),
	})
}

// GetCollectorStatus gets status of a specific collector
// GET /collectors/:name
func (h *CollectorHandler) GetCollectorStatus(c *gin.Context) {
	name := c.Param("name")

	collector, err := h.manager.GetCollector(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":    name,
		"running": collector.IsRunning(),
		"metrics": collector.GetMetrics(),
	})
}

// StartCollector starts a data collector
// POST /collectors/:name/start
func (h *CollectorHandler) StartCollector(c *gin.Context) {
	name := c.Param("name")

	if err := h.manager.StartCollector(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to start collector: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "collector started successfully",
		"name":    name,
	})
}

// StopCollector stops a data collector
// POST /collectors/:name/stop
func (h *CollectorHandler) StopCollector(c *gin.Context) {
	name := c.Param("name")

	if err := h.manager.StopCollector(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to stop collector: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "collector stopped successfully",
		"name":    name,
	})
}

// SubscribeSymbols subscribes to symbols
// POST /collectors/:name/subscribe
func (h *CollectorHandler) SubscribeSymbols(c *gin.Context) {
	name := c.Param("name")

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	if err := h.manager.SubscribeSymbols(name, req.Symbols); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to subscribe: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "subscribed successfully",
		"collector":      name,
		"symbols":        req.Symbols,
		"symbols_count":  len(req.Symbols),
	})
}

// UnsubscribeSymbols unsubscribes from symbols
// POST /collectors/:name/unsubscribe
func (h *CollectorHandler) UnsubscribeSymbols(c *gin.Context) {
	name := c.Param("name")

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	if err := h.manager.UnsubscribeSymbols(name, req.Symbols); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to unsubscribe: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "unsubscribed successfully",
		"collector": name,
		"symbols": req.Symbols,
	})
}

// GetMetrics returns metrics for all collectors
// GET /collectors/metrics
func (h *CollectorHandler) GetMetrics(c *gin.Context) {
	metrics := h.manager.GetAllMetrics()

	c.JSON(http.StatusOK, gin.H{
		"metrics": metrics,
	})
}

// GetManager returns the collector manager (for main.go integration)
func (h *CollectorHandler) GetManager() *collector.CollectorManager {
	return h.manager
}
