package collector

import (
	"fmt"
	"log"
	"sync"

	"github.com/trading-chitti/market-bridge/internal/database"
	"github.com/trading-chitti/market-bridge/internal/metrics"
)

// CollectorInterface defines the interface that all collectors must implement
type CollectorInterface interface {
	Start() error
	Stop()
	IsRunning() bool
	GetMetrics() map[string]interface{}
}

// UnifiedCollectorManager manages both real and mock data collectors
type UnifiedCollectorManager struct {
	db              *database.Database
	realCollectors  map[string]*DataCollector
	mockCollectors  map[string]*MockDataCollector
	mu              sync.RWMutex
}

// NewUnifiedCollectorManager creates a new unified collector manager
func NewUnifiedCollectorManager(db *database.Database) *UnifiedCollectorManager {
	return &UnifiedCollectorManager{
		db:             db,
		realCollectors: make(map[string]*DataCollector),
		mockCollectors: make(map[string]*MockDataCollector),
	}
}

// CreateRealCollector creates a new real data collector (Zerodha WebSocket)
func (ucm *UnifiedCollectorManager) CreateRealCollector(name, apiKey, accessToken string) error {
	ucm.mu.Lock()
	defer ucm.mu.Unlock()

	if _, exists := ucm.realCollectors[name]; exists {
		return fmt.Errorf("real collector '%s' already exists", name)
	}
	if _, exists := ucm.mockCollectors[name]; exists {
		return fmt.Errorf("mock collector '%s' already exists with same name", name)
	}

	collector := NewDataCollector(ucm.db, apiKey, accessToken)
	ucm.realCollectors[name] = collector

	log.Printf("✅ Created real collector: %s", name)
	return nil
}

// CreateMockCollector creates a new mock data collector
func (ucm *UnifiedCollectorManager) CreateMockCollector(name string, symbols []string) error {
	ucm.mu.Lock()
	defer ucm.mu.Unlock()

	if _, exists := ucm.mockCollectors[name]; exists {
		return fmt.Errorf("mock collector '%s' already exists", name)
	}
	if _, exists := ucm.realCollectors[name]; exists {
		return fmt.Errorf("real collector '%s' already exists with same name", name)
	}

	collector := NewMockDataCollector(ucm.db, name, symbols)
	ucm.mockCollectors[name] = collector

	log.Printf("✅ Created mock collector: %s with %d symbols", name, len(symbols))
	return nil
}

// StartCollector starts a collector (real or mock)
func (ucm *UnifiedCollectorManager) StartCollector(name string) error {
	ucm.mu.RLock()
	var err error

	// Check real collectors
	if collector, exists := ucm.realCollectors[name]; exists {
		err = collector.Start()
		ucm.mu.RUnlock()
		if err == nil {
			ucm.updateActiveCollectorsMetric()
		}
		return err
	}

	// Check mock collectors
	if collector, exists := ucm.mockCollectors[name]; exists {
		err = collector.Start()
		ucm.mu.RUnlock()
		if err == nil {
			ucm.updateActiveCollectorsMetric()
		}
		return err
	}

	ucm.mu.RUnlock()
	return fmt.Errorf("collector '%s' not found", name)
}

// StopCollector stops a collector
func (ucm *UnifiedCollectorManager) StopCollector(name string) error {
	ucm.mu.RLock()

	// Check real collectors
	if collector, exists := ucm.realCollectors[name]; exists {
		collector.Stop()
		ucm.mu.RUnlock()
		ucm.updateActiveCollectorsMetric()
		return nil
	}

	// Check mock collectors
	if collector, exists := ucm.mockCollectors[name]; exists {
		collector.Stop()
		ucm.mu.RUnlock()
		ucm.updateActiveCollectorsMetric()
		return nil
	}

	ucm.mu.RUnlock()
	return fmt.Errorf("collector '%s' not found", name)
}

// StopAll stops all collectors
func (ucm *UnifiedCollectorManager) StopAll() {
	ucm.mu.RLock()
	defer ucm.mu.RUnlock()

	for name, collector := range ucm.realCollectors {
		collector.Stop()
		log.Printf("🛑 Stopped real collector: %s", name)
	}

	for name, collector := range ucm.mockCollectors {
		collector.Stop()
		log.Printf("🛑 Stopped mock collector: %s", name)
	}
}

// ListCollectors returns all collector names with their types
func (ucm *UnifiedCollectorManager) ListCollectors() []map[string]interface{} {
	ucm.mu.RLock()
	defer ucm.mu.RUnlock()

	collectors := []map[string]interface{}{}

	for name, collector := range ucm.realCollectors {
		collectors = append(collectors, map[string]interface{}{
			"name":    name,
			"type":    "real",
			"running": collector.IsRunning(),
			"metrics": collector.GetMetrics(),
		})
	}

	for name, collector := range ucm.mockCollectors {
		collectors = append(collectors, map[string]interface{}{
			"name":    name,
			"type":    "mock",
			"running": collector.IsRunning(),
			"metrics": collector.GetMetrics(),
		})
	}

	return collectors
}

// GetCollectorMetrics returns metrics for a specific collector
func (ucm *UnifiedCollectorManager) GetCollectorMetrics(name string) (map[string]interface{}, error) {
	ucm.mu.RLock()
	defer ucm.mu.RUnlock()

	// Check real collectors
	if collector, exists := ucm.realCollectors[name]; exists {
		metrics := collector.GetMetrics()
		metrics["type"] = "real"
		metrics["name"] = name
		return metrics, nil
	}

	// Check mock collectors
	if collector, exists := ucm.mockCollectors[name]; exists {
		metrics := collector.GetMetrics()
		metrics["type"] = "mock"
		metrics["name"] = name
		return metrics, nil
	}

	return nil, fmt.Errorf("collector '%s' not found", name)
}

// GetAllMetrics returns metrics for all collectors
func (ucm *UnifiedCollectorManager) GetAllMetrics() map[string]interface{} {
	ucm.mu.RLock()
	defer ucm.mu.RUnlock()

	metrics := make(map[string]interface{})

	for name, collector := range ucm.realCollectors {
		collectorMetrics := collector.GetMetrics()
		collectorMetrics["type"] = "real"
		metrics[name] = collectorMetrics
	}

	for name, collector := range ucm.mockCollectors {
		collectorMetrics := collector.GetMetrics()
		collectorMetrics["type"] = "mock"
		metrics[name] = collectorMetrics
	}

	return metrics
}

// SubscribeSymbols subscribes to symbols (real collectors only)
func (ucm *UnifiedCollectorManager) SubscribeSymbols(collectorName string, symbols []string) error {
	ucm.mu.RLock()
	defer ucm.mu.RUnlock()

	// Check if it's a real collector
	if collector, exists := ucm.realCollectors[collectorName]; exists {
		// Get instrument tokens from database
		tokens := []uint32{}
		for _, symbol := range symbols {
			// Try NSE first
			token, err := ucm.db.GetInstrumentToken("NSE", symbol)
			if err != nil || token == 0 {
				// Try BSE
				token, err = ucm.db.GetInstrumentToken("BSE", symbol)
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

	// Check if it's a mock collector
	if collector, exists := ucm.mockCollectors[collectorName]; exists {
		collector.AddSymbols(symbols)
		return nil
	}

	return fmt.Errorf("collector '%s' not found", collectorName)
}

// UnsubscribeSymbols unsubscribes from symbols
func (ucm *UnifiedCollectorManager) UnsubscribeSymbols(collectorName string, symbols []string) error {
	ucm.mu.RLock()
	defer ucm.mu.RUnlock()

	// Check if it's a real collector
	if collector, exists := ucm.realCollectors[collectorName]; exists {
		tokens := []uint32{}
		for _, symbol := range symbols {
			token, err := ucm.db.GetInstrumentToken("NSE", symbol)
			if err != nil || token == 0 {
				token, err = ucm.db.GetInstrumentToken("BSE", symbol)
				if err != nil || token == 0 {
					continue
				}
			}
			tokens = append(tokens, token)
		}

		return collector.Unsubscribe(tokens)
	}

	// Check if it's a mock collector
	if collector, exists := ucm.mockCollectors[collectorName]; exists {
		collector.RemoveSymbols(symbols)
		return nil
	}

	return fmt.Errorf("collector '%s' not found", collectorName)
}

// DeleteCollector removes a collector
func (ucm *UnifiedCollectorManager) DeleteCollector(name string) error {
	ucm.mu.Lock()
	defer ucm.mu.Unlock()

	// Check real collectors
	if collector, exists := ucm.realCollectors[name]; exists {
		if collector.IsRunning() {
			return fmt.Errorf("cannot delete running collector, stop it first")
		}
		delete(ucm.realCollectors, name)
		log.Printf("🗑️  Deleted real collector: %s", name)
		return nil
	}

	// Check mock collectors
	if collector, exists := ucm.mockCollectors[name]; exists {
		if collector.IsRunning() {
			return fmt.Errorf("cannot delete running collector, stop it first")
		}
		delete(ucm.mockCollectors, name)
		log.Printf("🗑️  Deleted mock collector: %s", name)
		return nil
	}

	return fmt.Errorf("collector '%s' not found", name)
}

// GetCollectorType returns the type of a collector ("real", "mock", or error)
func (ucm *UnifiedCollectorManager) GetCollectorType(name string) (string, error) {
	ucm.mu.RLock()
	defer ucm.mu.RUnlock()

	if _, exists := ucm.realCollectors[name]; exists {
		return "real", nil
	}

	if _, exists := ucm.mockCollectors[name]; exists {
		return "mock", nil
	}

	return "", fmt.Errorf("collector '%s' not found", name)
}

// updateActiveCollectorsMetric updates the Prometheus metric for active collectors
func (ucm *UnifiedCollectorManager) updateActiveCollectorsMetric() {
	ucm.mu.RLock()
	defer ucm.mu.RUnlock()

	activeCount := 0

	// Count running real collectors
	for _, collector := range ucm.realCollectors {
		if collector.IsRunning() {
			activeCount++
		}
	}

	// Count running mock collectors
	for _, collector := range ucm.mockCollectors {
		if collector.IsRunning() {
			activeCount++
		}
	}

	metrics.SetActiveCollectors(activeCount)
}
