package services

import (
	"log"
	"time"

	"github.com/trading-chitti/market-bridge/internal/broker"
	"github.com/trading-chitti/market-bridge/internal/database"
)

// TokenRefreshService handles automatic token refresh for brokers
type TokenRefreshService struct {
	db     *database.Database
	ticker *time.Ticker
	done   chan bool
}

// NewTokenRefreshService creates a new token refresh service
func NewTokenRefreshService(db *database.Database) *TokenRefreshService {
	return &TokenRefreshService{
		db:   db,
		done: make(chan bool),
	}
}

// Start begins the token refresh loop
func (s *TokenRefreshService) Start(checkInterval time.Duration) {
	log.Printf("🔄 Starting token refresh service (check interval: %v)", checkInterval)

	s.ticker = time.NewTicker(checkInterval)

	go func() {
		// Run once immediately
		s.refreshExpiredTokens()

		// Then run on schedule
		for {
			select {
			case <-s.ticker.C:
				s.refreshExpiredTokens()
			case <-s.done:
				return
			}
		}
	}()
}

// Stop stops the token refresh service
func (s *TokenRefreshService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true
	log.Println("⏹️  Token refresh service stopped")
}

// refreshExpiredTokens checks for expiring tokens and refreshes them
func (s *TokenRefreshService) refreshExpiredTokens() {
	// Get brokers whose tokens expire within 6 hours
	threshold := 6 * time.Hour
	configs, err := s.db.GetExpiringSoonBrokerConfigs(threshold)
	if err != nil {
		log.Printf("❌ Error fetching expiring configs: %v", err)
		return
	}

	if len(configs) == 0 {
		log.Println("✅ No tokens need refresh")
		return
	}

	log.Printf("🔄 Found %d broker(s) with expiring tokens", len(configs))

	for _, config := range configs {
		if err := s.refreshBrokerToken(&config); err != nil {
			log.Printf("❌ Failed to refresh token for %s (ID: %d): %v",
				config.BrokerName, config.ID, err)
		} else {
			log.Printf("✅ Successfully refreshed token for %s (ID: %d)",
				config.BrokerName, config.ID)
		}
	}
}

// refreshBrokerToken refreshes access token for a specific broker
func (s *TokenRefreshService) refreshBrokerToken(config *broker.BrokerConfig) error {
	log.Printf("🔑 Refreshing token for %s broker (ID: %d)", config.BrokerName, config.ID)

	// Create broker instance
	brk, err := broker.NewBroker(config)
	if err != nil {
		return err
	}

	// For Zerodha, we need to use refresh token to get new access token
	zerodhaBroker, ok := brk.(*broker.ZerodhaBroker)
	if !ok {
		log.Printf("⚠️  Token refresh not implemented for %s", config.BrokerName)
		return nil
	}

	// Note: Zerodha Kite Connect v4 supports refresh tokens
	// You would need to implement the refresh logic here
	// For now, this is a placeholder

	_ = zerodhaBroker // Use the variable to avoid unused error

	// TODO: Implement actual token refresh
	// newTokens, err := zerodhaBroker.GetClient().RenewAccessToken(refreshToken, apiSecret)
	// if err != nil {
	//     return err
	// }

	// Update database with new tokens
	// expiresAt := time.Now().Add(24 * time.Hour) // Zerodha tokens expire in 24 hours
	// err = s.db.UpdateBrokerTokens(config.ID, newTokens.AccessToken, newTokens.RefreshToken, expiresAt)

	log.Printf("ℹ️  Token refresh for %s requires manual intervention (Zerodha requires daily login)",
		config.BrokerName)

	return nil
}

// RefreshAllTokensNow forces immediate refresh of all enabled brokers
func (s *TokenRefreshService) RefreshAllTokensNow() error {
	configs, err := s.db.GetAllBrokerConfigs()
	if err != nil {
		return err
	}

	for _, config := range configs {
		if !config.Enabled {
			continue
		}

		if err := s.refreshBrokerToken(&config); err != nil {
			log.Printf("❌ Failed to refresh %s: %v", config.BrokerName, err)
		}
	}

	return nil
}
