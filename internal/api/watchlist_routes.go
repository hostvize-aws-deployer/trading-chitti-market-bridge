package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trading-chitti/market-bridge/internal/watchlist"
)

// WatchlistHandler handles watchlist requests
type WatchlistHandler struct{}

// NewWatchlistHandler creates a new watchlist handler
func NewWatchlistHandler() *WatchlistHandler {
	return &WatchlistHandler{}
}

// RegisterRoutes registers watchlist routes
func (h *WatchlistHandler) RegisterRoutes(r *gin.RouterGroup) {
	wl := r.Group("/watchlists")
	{
		wl.GET("", h.ListWatchlists)
		wl.GET("/names", h.ListWatchlistNames)
		wl.GET("/categories", h.ListCategories)
		wl.GET("/category/:category", h.GetWatchlistsByCategory)
		wl.GET("/:name", h.GetWatchlist)
		wl.POST("/merge", h.MergeWatchlists)
	}
}

// ListWatchlists returns all predefined watchlists
// GET /watchlists
func (h *WatchlistHandler) ListWatchlists(c *gin.Context) {
	watchlists := watchlist.GetAllWatchlists()

	c.JSON(http.StatusOK, gin.H{
		"count":      len(watchlists),
		"watchlists": watchlists,
	})
}

// ListWatchlistNames returns names of all watchlists
// GET /watchlists/names
func (h *WatchlistHandler) ListWatchlistNames(c *gin.Context) {
	names := watchlist.ListWatchlistNames()

	c.JSON(http.StatusOK, gin.H{
		"count": len(names),
		"names": names,
	})
}

// GetWatchlist returns a specific watchlist by name
// GET /watchlists/:name
func (h *WatchlistHandler) GetWatchlist(c *gin.Context) {
	name := c.Param("name")

	wl := watchlist.GetWatchlist(name)
	if wl == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "watchlist not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"watchlist": wl,
	})
}

// ListCategories returns all watchlist categories
// GET /watchlists/categories
func (h *WatchlistHandler) ListCategories(c *gin.Context) {
	categories := watchlist.GetCategories()

	c.JSON(http.StatusOK, gin.H{
		"count":      len(categories),
		"categories": categories,
	})
}

// GetWatchlistsByCategory returns watchlists in a specific category
// GET /watchlists/category/:category
func (h *WatchlistHandler) GetWatchlistsByCategory(c *gin.Context) {
	category := c.Param("category")

	watchlists := watchlist.GetWatchlistsByCategory(category)

	c.JSON(http.StatusOK, gin.H{
		"category":   category,
		"count":      len(watchlists),
		"watchlists": watchlists,
	})
}

// MergeWatchlistsRequest represents a merge request
type MergeWatchlistsRequest struct {
	Names []string `json:"names" binding:"required"`
}

// MergeWatchlists combines multiple watchlists into one
// POST /watchlists/merge
// Body: {"names": ["NIFTY50", "BANKNIFTY"]}
func (h *WatchlistHandler) MergeWatchlists(c *gin.Context) {
	var req MergeWatchlistsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	if len(req.Names) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "at least one watchlist name required",
		})
		return
	}

	merged := watchlist.MergeWatchlists(req.Names)

	c.JSON(http.StatusOK, gin.H{
		"watchlist": merged,
	})
}
