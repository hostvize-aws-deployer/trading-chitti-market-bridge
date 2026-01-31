package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrSessionRevoked     = errors.New("session has been revoked")
)

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

// User represents a user account
type User struct {
	UserID        string    `json:"user_id" db:"user_id"`
	Email         string    `json:"email" db:"email"`
	PasswordHash  string    `json:"-" db:"password_hash"`
	FullName      string    `json:"full_name" db:"full_name"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	EmailVerified bool      `json:"email_verified" db:"email_verified"`
}

// Session represents an active user session
type Session struct {
	SessionID        string    `json:"session_id" db:"session_id"`
	UserID           string    `json:"user_id" db:"user_id"`
	TokenHash        string    `json:"-" db:"token_hash"`
	RefreshTokenHash string    `json:"-" db:"refresh_token_hash"`
	ExpiresAt        time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	LastUsedAt       time.Time `json:"last_used_at" db:"last_used_at"`
	IPAddress        string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent        string    `json:"user_agent,omitempty" db:"user_agent"`
	IsRevoked        bool      `json:"is_revoked" db:"is_revoked"`
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// AuthService handles authentication operations
type AuthService struct {
	jwtSecret          []byte
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewAuthService creates a new authentication service
func NewAuthService(jwtSecret string) *AuthService {
	return &AuthService{
		jwtSecret:          []byte(jwtSecret),
		accessTokenExpiry:  15 * time.Minute,
		refreshTokenExpiry: 7 * 24 * time.Hour, // 7 days
	}
}

// HashPassword creates a bcrypt hash of the password
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword checks if the provided password matches the hash
func (s *AuthService) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateTokenPair creates a new JWT access token and refresh token
func (s *AuthService) GenerateTokenPair(user *User) (*TokenPair, string, error) {
	sessionID := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(s.accessTokenExpiry)

	// Create access token claims
	claims := &JWTClaims{
		UserID:    user.UserID,
		Email:     user.Email,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "market-bridge",
			Subject:   user.UserID,
		},
	}

	// Generate access token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token (random string)
	refreshToken, err := s.generateRandomToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	tokenPair := &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}

	// Hash tokens for storage (currently not persisted)
	// tokenHash := s.hashToken(accessToken)

	return tokenPair, sessionID, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// generateRandomToken creates a cryptographically secure random token
func (s *AuthService) generateRandomToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken creates a SHA-256 hash of the token for storage
func (s *AuthService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// HashRefreshToken creates a hash of the refresh token for storage
func (s *AuthService) HashRefreshToken(refreshToken string) string {
	return s.hashToken(refreshToken)
}
