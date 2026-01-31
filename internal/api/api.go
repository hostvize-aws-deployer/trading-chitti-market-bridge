package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/trading-chitti/market-bridge/internal/analyzer"
	"github.com/trading-chitti/market-bridge/internal/broker"
	"github.com/trading-chitti/market-bridge/internal/database"
)

// API handles HTTP requests
type API struct {
	broker            broker.Broker
	db                *database.Database
	analyzer          *analyzer.Analyzer52D
	historicalService *database.HistoricalDataService
	wsHub             *WebSocketHub
	logger            *logrus.Logger
}

// NewAPI creates a new API handler
func NewAPI(b broker.Broker, db *database.Database) *API {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	return &API{
		broker:            b,
		db:                db,
		analyzer:          analyzer.NewAnalyzer52D(),
		historicalService: database.NewHistoricalDataService(db, b),
		logger:            logger,
	}
}

// SetWebSocketHub sets the WebSocket hub for the API
func (a *API) SetWebSocketHub(hub *WebSocketHub) {
	a.wsHub = hub
}

// RegisterRoutes registers all API routes
func (a *API) RegisterRoutes(r *gin.Engine) {
	// Health & Info
	r.GET("/", a.Root)
	r.GET("/health", a.Health)
	
	// Authentication
	auth := r.Group("/auth")
	{
		auth.GET("/login-url", a.GetLoginURL)
		auth.POST("/session", a.GenerateSession)
	}
	
	// Account
	account := r.Group("/account")
	{
		account.GET("/profile", a.GetProfile)
		account.GET("/margins", a.GetMargins)
		account.GET("/positions", a.GetPositions)
		account.GET("/holdings", a.GetHoldings)
		account.GET("/orders", a.GetOrders)
	}
	
	// Market Data
	market := r.Group("/market")
	{
		market.POST("/quote", a.GetQuote)
		market.POST("/ltp", a.GetLTP)
		market.GET("/status", a.GetMarketStatus)
		market.GET("/instruments/:exchange", a.GetInstruments)
	}

	// Instruments
	instruments := r.Group("/instruments")
	{
		instruments.GET("/search", a.SearchInstruments)
		instruments.GET("/:token", a.GetInstrumentByToken)
		instruments.POST("/sync", a.SyncInstruments)
	}

	// Historical Data
	historical := r.Group("/historical")
	{
		historical.POST("/", a.GetHistoricalData)
		historical.GET("/52day", a.Get52DayHistorical)
		historical.POST("/warm-cache", a.WarmCache)
	}

	// Pattern Recognition
	patternHandler := NewPatternHandler(a.broker, a.db)
	patternHandler.RegisterRoutes(r.Group(""))

	// Intraday Data
	intradayHandler := NewIntradayHandler(a.db)
	intradayHandler.RegisterRoutes(r.Group(""))

	// Data Collectors
	collectorHandler := NewCollectorHandler(a.db)
	collectorHandler.RegisterRoutes(r.Group(""))

	// Watchlists
	watchlistHandler := NewWatchlistHandler()
	watchlistHandler.RegisterRoutes(r.Group(""))

	// WebSocket Streaming (if hub is initialized)
	if a.wsHub != nil {
		streamHandler := NewStreamingHandler(a.wsHub)
		streamHandler.RegisterRoutes(r.Group(""))
	}

	// Analysis & Trading
	trade := r.Group("/trade")
	{
		trade.POST("/analyze", a.AnalyzeSymbols)
		trade.POST("/scan", a.ScanAndTrade)
		trade.POST("/order", a.PlaceOrder)
		trade.PUT("/order/:orderID", a.ModifyOrder)
		trade.DELETE("/order/:orderID", a.CancelOrder)
		trade.POST("/positions/close-all", a.CloseAllPositions)
	}
	
	// Broker Management
	brokers := r.Group("/brokers")
	{
		brokers.GET("/", a.ListBrokers)
		brokers.POST("/", a.AddBroker)
		brokers.PUT("/:id", a.UpdateBroker)
		brokers.DELETE("/:id", a.DeleteBroker)
		brokers.POST("/:id/activate", a.ActivateBroker)
	}
}

// Root returns service information
func (a *API) Root(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "Market Bridge API",
		"version": "1.0.0",
		"broker":  a.broker.GetBrokerName(),
		"status":  "running",
	})
}

// Health returns health status
func (a *API) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":        "healthy",
		"broker":        a.broker.GetBrokerName(),
		"market_status": a.broker.GetMarketStatus(),
		"timestamp":     time.Now(),
	})
}

// GetLoginURL returns broker login URL
func (a *API) GetLoginURL(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"login_url": a.broker.GetLoginURL(),
	})
}

// GenerateSession generates session from request token
func (a *API) GenerateSession(c *gin.Context) {
	var req struct {
		RequestToken string `json:"request_token" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	session, err := a.broker.GenerateSession(req.RequestToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, session)
}

// GetProfile returns user profile
func (a *API) GetProfile(c *gin.Context) {
	profile, err := a.broker.GetProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, profile)
}

// GetMargins returns account margins
func (a *API) GetMargins(c *gin.Context) {
	margins, err := a.broker.GetMargins()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, margins)
}

// GetPositions returns current positions
func (a *API) GetPositions(c *gin.Context) {
	positions, err := a.broker.GetPositions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, positions)
}

// GetHoldings returns holdings
func (a *API) GetHoldings(c *gin.Context) {
	holdings, err := a.broker.GetHoldings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, holdings)
}

// GetOrders returns orders
func (a *API) GetOrders(c *gin.Context) {
	orders, err := a.broker.GetOrders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, orders)
}

// GetQuote returns real-time quotes
func (a *API) GetQuote(c *gin.Context) {
	var req struct {
		Symbols []string `json:"symbols" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	quotes, err := a.broker.GetQuote(req.Symbols)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, quotes)
}

// GetLTP returns last traded price
func (a *API) GetLTP(c *gin.Context) {
	var req struct {
		Symbols []string `json:"symbols" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	ltp, err := a.broker.GetLTP(req.Symbols)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, ltp)
}

// GetMarketStatus returns market status
func (a *API) GetMarketStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  a.broker.GetMarketStatus(),
		"is_open": a.broker.IsMarketOpen(),
	})
}

// GetInstruments returns tradable instruments
func (a *API) GetInstruments(c *gin.Context) {
	exchange := c.Param("exchange")
	
	instruments, err := a.broker.GetInstruments(exchange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"exchange": exchange,
		"count":    len(instruments),
		"instruments": instruments,
	})
}

// AnalyzeSymbols analyzes symbols and generates signals
func (a *API) AnalyzeSymbols(c *gin.Context) {
	var req struct {
		Symbols []string `json:"symbols" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// TODO: Fetch historical data and analyze
	// This is a placeholder - need to implement instrument token lookup and historical data fetching
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Analysis endpoint - implementation in progress",
		"symbols": req.Symbols,
	})
}

// ScanAndTrade scans symbols and executes trades
func (a *API) ScanAndTrade(c *gin.Context) {
	var req struct {
		Symbols []string `json:"symbols" binding:"required"`
		DryRun  bool     `json:"dry_run"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// TODO: Implement trading logic
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Trade scan initiated",
		"symbols": req.Symbols,
		"dry_run": req.DryRun,
	})
}

// PlaceOrder places a new order
func (a *API) PlaceOrder(c *gin.Context) {
	var order broker.OrderRequest
	
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	orderID, err := a.broker.PlaceOrder(&order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"order_id": orderID,
		"status":   "placed",
	})
}

// ModifyOrder modifies an existing order
func (a *API) ModifyOrder(c *gin.Context) {
	orderID := c.Param("orderID")
	
	var modify broker.OrderModify
	if err := c.ShouldBindJSON(&modify); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	newOrderID, err := a.broker.ModifyOrder(orderID, &modify)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"order_id": newOrderID,
		"status":   "modified",
	})
}

// CancelOrder cancels an order
func (a *API) CancelOrder(c *gin.Context) {
	orderID := c.Param("orderID")
	
	cancelledID, err := a.broker.CancelOrder(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"order_id": cancelledID,
		"status":   "cancelled",
	})
}

// CloseAllPositions closes all open positions
func (a *API) CloseAllPositions(c *gin.Context) {
	positions, err := a.broker.GetPositions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	closedCount := 0
	for _, pos := range positions.Net {
		if pos.Quantity == 0 {
			continue
		}
		
		transactionType := "SELL"
		if pos.Quantity < 0 {
			transactionType = "BUY"
		}
		
		order := &broker.OrderRequest{
			Symbol:          pos.Symbol,
			Exchange:        pos.Exchange,
			TransactionType: transactionType,
			OrderType:       "MARKET",
			Product:         pos.Product,
			Quantity:        abs(pos.Quantity),
		}
		
		if _, err := a.broker.PlaceOrder(order); err == nil {
			closedCount++
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"closed": closedCount,
		"total":  len(positions.Net),
	})
}

// Broker Management Endpoints
func (a *API) ListBrokers(c *gin.Context) {
	brokers, err := a.db.GetAllBrokerConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, brokers)
}

func (a *API) AddBroker(c *gin.Context) {
	var config broker.BrokerConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	id, err := a.db.SaveBrokerConfig(&config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Broker added successfully"})
}

func (a *API) UpdateBroker(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update broker endpoint"})
}

func (a *API) DeleteBroker(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Delete broker endpoint"})
}

func (a *API) ActivateBroker(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Activate broker endpoint"})
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
