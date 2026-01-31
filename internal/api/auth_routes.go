package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trading-chitti/market-bridge/internal/auth"
	"github.com/trading-chitti/market-bridge/internal/database"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	db          *database.Database
	authService *auth.AuthService
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db *database.Database, authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		db:          db,
		authService: authService,
	}
}

// RegisterRequest represents user registration request
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
}

// LoginRequest represents user login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest represents token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RegisterRoutes registers authentication routes
func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/logout", h.Logout)
		auth.POST("/refresh", h.RefreshToken)
		auth.GET("/me", AuthMiddleware(h.authService, h.db), h.GetCurrentUser)
	}
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// Check if user already exists
	existingUser, _ := h.db.GetUserByEmail(req.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "user with this email already exists",
		})
		return
	}

	// Hash password
	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to process registration",
		})
		return
	}

	// Create user
	user, err := h.db.CreateUser(req.Email, passwordHash, req.FullName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}

	// Generate tokens
	tokenPair, sessionID, err := h.authService.GenerateTokenPair(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate tokens",
		})
		return
	}

	// Create session
	session := &auth.Session{
		SessionID:        sessionID,
		UserID:           user.UserID,
		TokenHash:        h.authService.HashRefreshToken(tokenPair.AccessToken),
		RefreshTokenHash: h.authService.HashRefreshToken(tokenPair.RefreshToken),
		ExpiresAt:        tokenPair.ExpiresAt,
		IPAddress:        c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	}

	if err := h.db.CreateSession(session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create session",
		})
		return
	}

	// Audit log
	h.db.CreateAuditLog(user.UserID, "user.register", "user", user.UserID, c.ClientIP(), c.GetHeader("User-Agent"), nil)

	c.JSON(http.StatusCreated, gin.H{
		"user": gin.H{
			"user_id":   user.UserID,
			"email":     user.Email,
			"full_name": user.FullName,
		},
		"token": tokenPair,
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// Get user by email
	user, err := h.db.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid email or password",
		})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "account is disabled",
		})
		return
	}

	// Verify password
	if !h.authService.VerifyPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid email or password",
		})
		return
	}

	// Update last login
	h.db.UpdateLastLogin(user.UserID)

	// Generate tokens
	tokenPair, sessionID, err := h.authService.GenerateTokenPair(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate tokens",
		})
		return
	}

	// Create session
	session := &auth.Session{
		SessionID:        sessionID,
		UserID:           user.UserID,
		TokenHash:        h.authService.HashRefreshToken(tokenPair.AccessToken),
		RefreshTokenHash: h.authService.HashRefreshToken(tokenPair.RefreshToken),
		ExpiresAt:        tokenPair.ExpiresAt,
		IPAddress:        c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	}

	if err := h.db.CreateSession(session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create session",
		})
		return
	}

	// Audit log
	h.db.CreateAuditLog(user.UserID, "user.login", "user", user.UserID, c.ClientIP(), c.GetHeader("User-Agent"), nil)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"user_id":   user.UserID,
			"email":     user.Email,
			"full_name": user.FullName,
		},
		"token": tokenPair,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, exists := c.Get("session_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"message": "logged out",
		})
		return
	}

	sessionIDStr, ok := sessionID.(string)
	if ok {
		h.db.RevokeSession(sessionIDStr)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	// Hash the refresh token to look up session
	tokenHash := h.authService.HashRefreshToken(req.RefreshToken)

	// Get session by refresh token
	session, err := h.db.GetSessionByToken(tokenHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid refresh token",
		})
		return
	}

	// Get user
	user, err := h.db.GetUserByID(session.UserID)
	if err != nil || !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not found or inactive",
		})
		return
	}

	// Generate new token pair
	tokenPair, newSessionID, err := h.authService.GenerateTokenPair(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate tokens",
		})
		return
	}

	// Revoke old session
	h.db.RevokeSession(session.SessionID)

	// Create new session
	newSession := &auth.Session{
		SessionID:        newSessionID,
		UserID:           user.UserID,
		TokenHash:        h.authService.HashRefreshToken(tokenPair.AccessToken),
		RefreshTokenHash: h.authService.HashRefreshToken(tokenPair.RefreshToken),
		ExpiresAt:        tokenPair.ExpiresAt,
		IPAddress:        c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	}

	if err := h.db.CreateSession(newSession); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenPair,
	})
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authentication required",
		})
		return
	}

	user, err := h.db.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"user_id":        user.UserID,
			"email":          user.Email,
			"full_name":      user.FullName,
			"created_at":     user.CreatedAt,
			"last_login_at":  user.LastLoginAt,
			"email_verified": user.EmailVerified,
		},
	})
}
