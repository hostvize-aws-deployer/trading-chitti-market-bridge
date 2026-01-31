package collector

import (
	"fmt"
	"log"
	"sync"

	"github.com/trading-chitti/market-bridge/internal/database"
	"github.com/trading-chitti/market-bridge/internal/watchlist"
)

// CollectorManager manages multiple data collectors
type CollectorManager struct {
	db         *database.Database
	collectors map[string]*DataCollector
	mu         sync.RWMutex
}

// NewCollectorManager creates a new collector manager
func NewCollectorManager(db *database.Database) *CollectorManager {
	return &CollectorManager{
		db:         db,
		collectors: make(map[string]*DataCollector),
	}
}

// CreateCollector creates a new data collector
func (cm *CollectorManager) CreateCollector(name, apiKey, accessToken string) (*DataCollector, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.collectors[name]; exists {
		return nil, fmt.Errorf("collector '%s' already exists", name)
	}

	collector := NewDataCollector(cm.db, apiKey, accessToken)
	cm.collectors[name] = collector

	log.Printf("✅ Created collector: %s", name)
	return collector, nil
}

// GetCollector retrieves a collector by name
func (cm *CollectorManager) GetCollector(name string) (*DataCollector, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	collector, exists := cm.collectors[name]
	if !exists {
		return nil, fmt.Errorf("collector '%s' not found", name)
	}

	return collector, nil
}

// StartCollector starts a specific collector
func (cm *CollectorManager) StartCollector(name string) error {
	collector, err := cm.GetCollector(name)
	if err != nil {
		return err
	}

	return collector.Start()
}

// StopCollector stops a specific collector
func (cm *CollectorManager) StopCollector(name string) error {
	collector, err := cm.GetCollector(name)
	if err != nil {
		return err
	}

	collector.Stop()
	return nil
}

// StopAll stops all collectors
func (cm *CollectorManager) StopAll() {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for name, collector := range cm.collectors {
		collector.Stop()
		log.Printf("🛑 Stopped collector: %s", name)
	}
}

// ListCollectors returns all collector names
func (cm *CollectorManager) ListCollectors() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	names := make([]string, 0, len(cm.collectors))
	for name := range cm.collectors {
		names = append(names, name)
	}

	return names
}

// GetAllMetrics returns metrics for all collectors
func (cm *CollectorManager) GetAllMetrics() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	metrics := make(map[string]interface{})
	for name, collector := range cm.collectors {
		metrics[name] = collector.GetMetrics()
	}

	return metrics
}

// SubscribeSymbols subscribes to symbols across collectors
func (cm *CollectorManager) SubscribeSymbols(collectorName string, symbols []string) error {
	collector, err := cm.GetCollector(collectorName)
	if err != nil {
		return err
	}

	// Get instrument tokens from database
	tokens := []uint32{}
	for _, symbol := range symbols {
		// Try NSE first
		token, err := cm.db.GetInstrumentToken("NSE", symbol)
		if err != nil || token == 0 {
			// Try BSE
			token, err = cm.db.GetInstrumentToken("BSE", symbol)
			if err != nil || token == 0 {
				log.Printf("⚠️  Symbol not found: %s", symbol)
				continue
			}
		}

		tokens = append(tokens, token)
		collector.RegisterSymbol(token, "NSE", symbol)
	}

	if len(tokens) == 0 {
		return fmt.Errorf("no valid symbols found")
	}

	return collector.Subscribe(tokens)
}

// UnsubscribeSymbols unsubscribes from symbols
func (cm *CollectorManager) UnsubscribeSymbols(collectorName string, symbols []string) error {
	collector, err := cm.GetCollector(collectorName)
	if err != nil {
		return err
	}

	tokens := []uint32{}
	for _, symbol := range symbols {
		token, err := cm.db.GetInstrumentToken("NSE", symbol)
		if err != nil || token == 0 {
			token, err = cm.db.GetInstrumentToken("BSE", symbol)
			if err != nil || token == 0 {
				continue
			}
		}
		tokens = append(tokens, token)
	}

	return collector.Unsubscribe(tokens)
}

// LoadAndStartFromConfig loads configuration and starts auto-start collectors
func (cm *CollectorManager) LoadAndStartFromConfig(configPath string) error {
	config, err := LoadConfigFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Expand environment variables
	ExpandEnvVars(config)

	started := 0
	for _, collectorCfg := range config.Collectors {
		if !collectorCfg.AutoStart {
			log.Printf("⏭️  Skipping collector: %s (auto_start=false)", collectorCfg.Name)
			continue
		}

		// Create collector
		collector, err := cm.CreateCollector(collectorCfg.Name, collectorCfg.APIKey, collectorCfg.AccessToken)
		if err != nil {
			log.Printf("❌ Failed to create collector '%s': %v", collectorCfg.Name, err)
			continue
		}

		// Subscribe to symbols from watchlists
		var allSymbols []string
		for _, watchlistName := range collectorCfg.Watchlists {
			wl := GetWatchlistFromName(watchlistName)
			if wl != nil {
				allSymbols = append(allSymbols, wl.Symbols...)
			} else {
				log.Printf("⚠️  Watchlist not found: %s", watchlistName)
			}
		}

		// Add individual symbols
		allSymbols = append(allSymbols, collectorCfg.Symbols...)

		// Remove duplicates
		uniqueSymbols := removeDuplicates(allSymbols)

		// Subscribe to symbols
		if len(uniqueSymbols) > 0 {
			if err := cm.SubscribeSymbols(collectorCfg.Name, uniqueSymbols); err != nil {
				log.Printf("❌ Failed to subscribe symbols for '%s': %v", collectorCfg.Name, err)
				continue
			}
		}

		// Set mode
		if collectorCfg.Mode != "" {
			tokens := collector.subscribedTokens
			if err := collector.SetMode(collectorCfg.Mode, tokens); err != nil {
				log.Printf("⚠️  Failed to set mode for '%s': %v", collectorCfg.Name, err)
			}
		}

		// Start collector
		if err := cm.StartCollector(collectorCfg.Name); err != nil {
			log.Printf("❌ Failed to start collector '%s': %v", collectorCfg.Name, err)
			continue
		}

		started++
		log.Printf("✅ Auto-started collector: %s (%d symbols, mode: %s)",
			collectorCfg.Name, len(uniqueSymbols), collectorCfg.Mode)
	}

	log.Printf("🚀 Auto-started %d/%d collectors", started, len(config.Collectors))
	return nil
}

// Helper function to get watchlist by name
func GetWatchlistFromName(name string) *watchlist.Watchlist {
	return watchlist.GetWatchlist(name)
}

// Helper function to remove duplicates from string slice
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
