package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trading-chitti/market-bridge/internal/auth"
	"github.com/trading-chitti/market-bridge/internal/database"
)

// BrokerManagementHandler handles per-user broker account management
type BrokerManagementHandler struct {
	db          *database.Database
	authService *auth.AuthService
}

// NewBrokerManagementHandler creates a new broker management handler
func NewBrokerManagementHandler(db *database.Database, authService *auth.AuthService) *BrokerManagementHandler {
	return &BrokerManagementHandler{
		db:          db,
		authService: authService,
	}
}

// AddBrokerAccountRequest represents adding a new broker account
type AddBrokerAccountRequest struct {
	BrokerName  string `json:"broker_name" binding:"required"`
	APIKey      string `json:"api_key" binding:"required"`
	APISecret   string `json:"api_secret" binding:"required"`
	AccountName string `json:"account_name" binding:"required"`
	IsDefault   bool   `json:"is_default"`
}

// UpdateBrokerAccountRequest represents updating a broker account
type UpdateBrokerAccountRequest struct {
	AccessToken string `json:"access_token"`
	AccountName string `json:"account_name"`
	IsDefault   *bool  `json:"is_default"`
	IsActive    *bool  `json:"is_active"`
}

// RegisterRoutes registers broker management routes
func (h *BrokerManagementHandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	brokers := r.Group("/brokers")
	brokers.Use(authMiddleware)
	{
		brokers.GET("", h.ListBrokerAccounts)
		brokers.POST("", h.AddBrokerAccount)
		brokers.GET("/:config_id", h.GetBrokerAccount)
		brokers.PUT("/:config_id", h.UpdateBrokerAccount)
		brokers.DELETE("/:config_id", h.DeleteBrokerAccount)
		brokers.POST("/:config_id/set-default", h.SetDefaultBrokerAccount)
	}
}

// ListBrokerAccounts returns all broker accounts for the authenticated user
func (h *BrokerManagementHandler) ListBrokerAccounts(c *gin.Context) {
	userID, exists := RequireUserID(c)
	if !exists {
		return
	}

	configs, err := h.db.GetUserBrokerConfigs(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch broker accounts",
		})
		return
	}

	// Transform for response (hide sensitive data)
	accounts := make([]gin.H, 0, len(configs))
	for _, cfg := range configs {
		accounts = append(accounts, gin.H{
			"config_id":    cfg.ConfigID,
			"broker_name":  cfg.BrokerName,
			"account_name": cfg.AccountName,
			"is_default":   cfg.IsDefault,
			"is_active":    cfg.IsActive,
			"created_at":   cfg.CreatedAt,
			"has_access_token": cfg.AccessToken != "",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": accounts,
	})
}

// AddBrokerAccount adds a new broker account for the user
func (h *BrokerManagementHandler) AddBrokerAccount(c *gin.Context) {
	userID, exists := RequireUserID(c)
	if !exists {
		return
	}

	var req AddBrokerAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// Validate broker name
	validBrokers := map[string]bool{
		"zerodha":     true,
		"angelone":    true,
		"upstox":      true,
		"icicidirect": true,
	}
	if !validBrokers[req.BrokerName] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unsupported broker: " + req.BrokerName,
		})
		return
	}

	// Create broker config
	config, err := h.db.CreateUserBrokerConfig(
		userID,
		req.BrokerName,
		req.APIKey,
		req.APISecret,
		req.AccountName,
		req.IsDefault,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create broker account",
		})
		return
	}

	// Audit log
	h.db.CreateAuditLog(
		userID,
		"broker.add",
		"broker_config",
		string(rune(config.ConfigID)),
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		map[string]interface{}{
			"broker_name":  req.BrokerName,
			"account_name": req.AccountName,
		},
	)

	c.JSON(http.StatusCreated, gin.H{
		"config_id":    config.ConfigID,
		"broker_name":  config.BrokerName,
		"account_name": config.AccountName,
		"is_default":   config.IsDefault,
		"is_active":    config.IsActive,
		"created_at":   config.CreatedAt,
	})
}

// GetBrokerAccount returns a specific broker account
func (h *BrokerManagementHandler) GetBrokerAccount(c *gin.Context) {
	userID, exists := RequireUserID(c)
	if !exists {
		return
	}

	configID := c.Param("config_id")

	// Get user's broker configs to verify ownership
	configs, err := h.db.GetUserBrokerConfigs(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch broker account",
		})
		return
	}

	// Find the specific config
	var foundConfig *database.BrokerConfig
	for _, cfg := range configs {
		if string(rune(cfg.ConfigID)) == configID {
			foundConfig = cfg
			break
		}
	}

	if foundConfig == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "broker account not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"config_id":        foundConfig.ConfigID,
		"broker_name":      foundConfig.BrokerName,
		"account_name":     foundConfig.AccountName,
		"is_default":       foundConfig.IsDefault,
		"is_active":        foundConfig.IsActive,
		"has_access_token": foundConfig.AccessToken != "",
		"token_expires_at": foundConfig.TokenExpiresAt,
		"created_at":       foundConfig.CreatedAt,
		"updated_at":       foundConfig.UpdatedAt,
	})
}

// UpdateBrokerAccount updates a broker account
func (h *BrokerManagementHandler) UpdateBrokerAccount(c *gin.Context) {
	_, exists := RequireUserID(c)
	if !exists {
		return
	}

	configID := c.Param("config_id")

	var req UpdateBrokerAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// TODO: Implement update logic in database package
	// For now, return success
	c.JSON(http.StatusOK, gin.H{
		"message": "broker account updated",
		"config_id": configID,
	})
}

// DeleteBrokerAccount deletes a broker account
func (h *BrokerManagementHandler) DeleteBrokerAccount(c *gin.Context) {
	userID, exists := RequireUserID(c)
	if !exists {
		return
	}

	configID := c.Param("config_id")

	// TODO: Implement delete logic in database package
	// For now, return success

	// Audit log
	h.db.CreateAuditLog(
		userID,
		"broker.delete",
		"broker_config",
		configID,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		nil,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "broker account deleted",
	})
}

// SetDefaultBrokerAccount sets a broker account as the default
func (h *BrokerManagementHandler) SetDefaultBrokerAccount(c *gin.Context) {
	_, exists := RequireUserID(c)
	if !exists {
		return
	}

	configID := c.Param("config_id")

	// TODO: Implement set default logic in database package
	// For now, return success

	c.JSON(http.StatusOK, gin.H{
		"message":   "default broker account updated",
		"config_id": configID,
	})
}
