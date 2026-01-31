package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/trading-chitti/market-bridge/internal/api"
	"github.com/trading-chitti/market-bridge/internal/auth"
	"github.com/trading-chitti/market-bridge/internal/broker"
	"github.com/trading-chitti/market-bridge/internal/database"
	"github.com/trading-chitti/market-bridge/internal/metrics"
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

	// Add CORS middleware
	router.Use(api.CORSMiddleware())

	// Add metrics middleware
	router.Use(api.MetricsMiddleware())

	// Add API key authentication (only if API_KEY is set)
	if os.Getenv("API_KEY") != "" {
		router.Use(api.APIKeyMiddleware())
		log.Println("🔐 API key authentication enabled")
	} else {
		log.Println("⚠️  API key authentication disabled (set API_KEY to enable)")
	}

	// Initialize collector handler
	collectorHandler := api.NewCollectorHandler(db)
	defer collectorHandler.GetManager().StopAll()

	// Initialize metrics (set initial collector count to 0)
	metrics.SetActiveCollectors(0)

	// Check if multi-user mode is enabled
	multiUserMode := os.Getenv("MULTI_USER_MODE") == "true"

	if multiUserMode {
		log.Println("🔐 Multi-user mode enabled")

		// Initialize authentication service
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			log.Fatal("JWT_SECRET environment variable must be set in multi-user mode")
		}
		authService := auth.NewAuthService(jwtSecret)

		// Initialize WebSocket hub manager for per-user hubs
		wsHubManager := api.NewWebSocketHubManager(db)
		defer wsHubManager.CloseAllHubs()

		// Register authentication routes (public)
		authHandler := api.NewAuthHandler(db, authService)
		authHandler.RegisterRoutes(router.Group("/api"))

		// Register broker management routes (authenticated)
		brokerHandler := api.NewBrokerManagementHandler(db, authService)
		authMiddleware := api.AuthMiddleware(authService, db)
		brokerHandler.RegisterRoutes(router.Group("/api"), authMiddleware)

		// Register API handlers with authentication
		apiHandler := api.NewAPI(brk, db)
		apiHandler.RegisterRoutes(router)

		// Register collector routes (authenticated)
		// collectorHandler.RegisterRoutes(router.Group("/api"), authMiddleware)

		log.Println("✅ Multi-user authentication initialized")
	} else {
		log.Println("👤 Single-user mode (backward compatible)")

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

		// Register collector routes (public for backward compatibility)
		collectorHandler.RegisterRoutes(router.Group("/api"))
	}

	// Register Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	log.Println("📊 Prometheus metrics endpoint: /metrics")

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
