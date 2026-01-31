package api

import (
	"fmt"
	"log"
	"sync"

	"github.com/trading-chitti/market-bridge/internal/broker"
	"github.com/trading-chitti/market-bridge/internal/database"
)

// WebSocketHubManager manages per-user WebSocket hubs
type WebSocketHubManager struct {
	db    *database.Database
	hubs  map[string]*WebSocketHub // userID -> hub
	mu    sync.RWMutex
}

// NewWebSocketHubManager creates a new hub manager
func NewWebSocketHubManager(db *database.Database) *WebSocketHubManager {
	return &WebSocketHubManager{
		db:   db,
		hubs: make(map[string]*WebSocketHub),
	}
}

// GetOrCreateHub gets an existing hub or creates a new one for the user
func (m *WebSocketHubManager) GetOrCreateHub(userID string) (*WebSocketHub, error) {
	m.mu.RLock()
	hub, exists := m.hubs[userID]
	m.mu.RUnlock()

	if exists && hub != nil {
		return hub, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if hub, exists := m.hubs[userID]; exists && hub != nil {
		return hub, nil
	}

	// Get user's default broker config
	configs, err := m.db.GetUserBrokerConfigs(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get broker configs: %w", err)
	}

	var defaultConfig *broker.BrokerConfig
	for _, cfg := range configs {
		if cfg.IsDefault && cfg.IsActive && cfg.AccessToken != "" {
			// Convert database.BrokerConfig to broker.BrokerConfig
			defaultConfig = &broker.BrokerConfig{
				ConfigID:         cfg.ConfigID,
				UserID:           cfg.UserID,
				BrokerName:       cfg.BrokerName,
				APIKey:           cfg.APIKey,
				APISecret:        cfg.APISecret,
				AccessToken:      cfg.AccessToken,
				RefreshToken:     cfg.RefreshToken,
				TokenExpiresAt:   cfg.TokenExpiresAt,
				LastTokenRefresh: cfg.LastTokenRefresh,
				IsActive:         cfg.IsActive,
				AccountName:      cfg.AccountName,
				IsDefault:        cfg.IsDefault,
				CreatedAt:        cfg.CreatedAt,
				UpdatedAt:        cfg.UpdatedAt,
			}
			break
		}
	}

	if defaultConfig == nil {
		return nil, fmt.Errorf("no active default broker account found for user")
	}

	// Create new hub for this user
	hub = NewWebSocketHub(defaultConfig.APIKey, defaultConfig.AccessToken)
	go hub.Run()
	hub.StartTicker()

	m.hubs[userID] = hub

	log.Printf("✅ Created WebSocket hub for user %s (broker: %s)", userID, defaultConfig.BrokerName)

	return hub, nil
}

// GetHub returns an existing hub for the user (without creating)
func (m *WebSocketHubManager) GetHub(userID string) *WebSocketHub {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hubs[userID]
}

// CloseHub closes and removes a user's hub
func (m *WebSocketHubManager) CloseHub(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.hubs[userID]; exists {
		// hub.StopTicker() // TODO: Implement ticker cleanup
		delete(m.hubs, userID)
		log.Printf("🔌 Closed WebSocket hub for user %s", userID)
	}
}

// CloseAllHubs closes all active hubs (for shutdown)
func (m *WebSocketHubManager) CloseAllHubs() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID := range m.hubs {
		// hub.StopTicker() // TODO: Implement ticker cleanup
		log.Printf("🔌 Closed WebSocket hub for user %s", userID)
	}

	m.hubs = make(map[string]*WebSocketHub)
}

// GetActiveUserCount returns the number of users with active hubs
func (m *WebSocketHubManager) GetActiveUserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.hubs)
}

// ListActiveUsers returns list of users with active hubs
func (m *WebSocketHubManager) ListActiveUsers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]string, 0, len(m.hubs))
	for userID := range m.hubs {
		users = append(users, userID)
	}
	return users
}

// UpdateUserBrokerConfig updates a user's hub when their broker config changes
func (m *WebSocketHubManager) UpdateUserBrokerConfig(userID string, newConfig *broker.BrokerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close existing hub if any
	if _, exists := m.hubs[userID]; exists {
		// hub.StopTicker() // TODO: Implement ticker cleanup
		delete(m.hubs, userID)
	}

	// Create new hub with updated config
	if newConfig.IsActive && newConfig.AccessToken != "" {
		hub := NewWebSocketHub(newConfig.APIKey, newConfig.AccessToken)
		go hub.Run()
		hub.StartTicker()
		m.hubs[userID] = hub

		log.Printf("🔄 Updated WebSocket hub for user %s", userID)
	}

	return nil
}
