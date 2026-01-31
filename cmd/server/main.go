package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/trading-chitti/market-bridge/internal/api"
	"github.com/trading-chitti/market-bridge/internal/broker"
	"github.com/trading-chitti/market-bridge/internal/database"
	"github.com/trading-chitti/market-bridge/internal/services"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
	
	// Initialize database
	db, err := database.NewDatabase(os.Getenv("TRADING_CHITTI_PG_DSN"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	
	// Load broker configuration from database
	brokerConfig, err := db.GetActiveBrokerConfig()
	if err != nil {
		log.Println("⚠️  No active broker configured, using environment variables")
		brokerConfig = &broker.BrokerConfig{
			BrokerName:  "zerodha",
			APIKey:      os.Getenv("ZERODHA_API_KEY"),
			APISecret:   os.Getenv("ZERODHA_API_SECRET"),
			AccessToken: os.Getenv("ZERODHA_ACCESS_TOKEN"),
		}
	}
	
	// Initialize broker
	brk, err := broker.NewBroker(brokerConfig)
	if err != nil {
		log.Fatalf("Failed to initialize broker: %v", err)
	}

	// Initialize WebSocket hub
	var wsHub *api.WebSocketHub
	if brokerConfig.APIKey != "" && brokerConfig.AccessToken != "" {
		wsHub = api.NewWebSocketHub(brokerConfig.APIKey, brokerConfig.AccessToken)
		go wsHub.Run()
		wsHub.StartTicker()
		log.Println("✅ WebSocket hub initialized and started")
	} else {
		log.Println("⚠️  WebSocket hub not started (missing API credentials)")
	}

	// Initialize token refresh service
	tokenRefreshService := services.NewTokenRefreshService(db)
	tokenRefreshService.Start(1 * time.Hour) // Check every hour
	defer tokenRefreshService.Stop()
	log.Println("✅ Token refresh service started")

	// Optionally sync instruments on startup
	if os.Getenv("SYNC_INSTRUMENTS_ON_START") == "true" {
		log.Println("🔄 Syncing instruments from broker...")
		go func() {
			if err := db.SyncInstrumentsFromBroker(brk); err != nil {
				log.Printf("❌ Failed to sync instruments: %v", err)
			}
		}()
	}

	// Create Gin router
	router := gin.Default()

	// Initialize API handlers
	apiHandler := api.NewAPI(brk, db)
	if wsHub != nil {
		apiHandler.SetWebSocketHub(wsHub)
	}

	// Register routes
	apiHandler.RegisterRoutes(router)

	// Register WebSocket routes
	if wsHub != nil {
		apiHandler.RegisterWebSocketRoutes(router)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "6005"
	}

	log.Printf("🚀 Market Bridge API starting on port %s", port)
	log.Printf("📊 Active Broker: %s", brk.GetBrokerName())
	log.Printf("📈 Market Status: %s", brk.GetMarketStatus())
	log.Printf("🔌 WebSocket: ws://localhost:%s/ws/market", port)
	log.Printf("📖 API Docs: http://localhost:%s/", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
