package main

import (
	"fmt"
	"log"
	"os"
	
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	
	"github.com/trading-chitti/market-bridge/internal/api"
	"github.com/trading-chitti/market-bridge/internal/broker"
	"github.com/trading-chitti/market-bridge/internal/database"
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
	
	// Create Gin router
	router := gin.Default()
	
	// Initialize API handlers
	apiHandler := api.NewAPI(brk, db)
	
	// Register routes
	apiHandler.RegisterRoutes(router)
	
	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "6005"
	}
	
	log.Printf("🚀 Market Bridge API starting on port %s", port)
	log.Printf("📊 Active Broker: %s", brk.GetBrokerName())
	log.Printf("📈 Market Status: %s", brk.GetMarketStatus())
	
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
