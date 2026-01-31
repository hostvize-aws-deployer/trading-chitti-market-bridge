package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/trading-chitti/market-bridge/internal/broker"
	"github.com/trading-chitti/market-bridge/internal/database"
	"github.com/trading-chitti/market-bridge/internal/watchlist"
)

var (
	// Flags
	symbolsFlag    = flag.String("symbols", "", "Comma-separated list of symbols (e.g., RELIANCE,TCS,INFY)")
	watchlistFlag  = flag.String("watchlist", "", "Watchlist name (e.g., NIFTY50, BANKNIFTY)")
	fromDateFlag   = flag.String("from", "", "Start date (YYYY-MM-DD)")
	toDateFlag     = flag.String("to", "", "End date (YYYY-MM-DD)")
	timeframeFlag  = flag.String("timeframe", "day", "Timeframe (minute, 5minute, 15minute, day)")
	dryRunFlag     = flag.Bool("dry-run", false, "Dry run mode (don't insert data)")
	concurrentFlag = flag.Int("concurrent", 5, "Number of concurrent requests")
)

func main() {
	flag.Parse()

	// Validate flags
	if *symbolsFlag == "" && *watchlistFlag == "" {
		fmt.Println("Error: Either -symbols or -watchlist must be specified")
		flag.Usage()
		os.Exit(1)
	}

	if *fromDateFlag == "" {
		fmt.Println("Error: -from date is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse dates
	fromDate, err := time.Parse("2006-01-02", *fromDateFlag)
	if err != nil {
		log.Fatalf("Invalid from date: %v", err)
	}

	var toDate time.Time
	if *toDateFlag == "" {
		toDate = time.Now()
	} else {
		toDate, err = time.Parse("2006-01-02", *toDateFlag)
		if err != nil {
			log.Fatalf("Invalid to date: %v", err)
		}
	}

	// Initialize database
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./market_bridge.db"
	}

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize broker
	apiKey := os.Getenv("ZERODHA_API_KEY")
	if apiKey == "" {
		log.Fatal("ZERODHA_API_KEY environment variable not set")
	}

	zerodha := broker.NewZerodha(apiKey, "", db)

	// Determine symbols to backfill
	var symbols []string
	if *symbolsFlag != "" {
		symbols = strings.Split(*symbolsFlag, ",")
		for i := range symbols {
			symbols[i] = strings.TrimSpace(symbols[i])
		}
	} else if *watchlistFlag != "" {
		wl := watchlist.GetWatchlist(*watchlistFlag)
		if wl == nil {
			log.Fatalf("Watchlist not found: %s", *watchlistFlag)
		}
		symbols = wl.Symbols
	}

	log.Printf("📊 Backfill Configuration")
	log.Printf("   Symbols: %d", len(symbols))
	log.Printf("   From: %s", fromDate.Format("2006-01-02"))
	log.Printf("   To: %s", toDate.Format("2006-01-02"))
	log.Printf("   Timeframe: %s", *timeframeFlag)
	log.Printf("   Concurrent: %d", *concurrentFlag)
	log.Printf("   Dry Run: %v", *dryRunFlag)
	log.Println()

	// Create backfill worker
	backfiller := &Backfiller{
		broker:      zerodha,
		db:          db,
		timeframe:   *timeframeFlag,
		dryRun:      *dryRunFlag,
		concurrent:  *concurrentFlag,
	}

	// Run backfill
	stats := backfiller.Backfill(symbols, fromDate, toDate)

	// Print summary
	log.Println()
	log.Println("📈 Backfill Summary")
	log.Printf("   Total Symbols: %d", stats.TotalSymbols)
	log.Printf("   Successful: %d", stats.Successful)
	log.Printf("   Failed: %d", stats.Failed)
	log.Printf("   Total Bars: %d", stats.TotalBars)
	log.Printf("   Duration: %v", stats.Duration)
}

// Backfiller handles data backfilling
type Backfiller struct {
	broker     broker.Broker
	db         *database.Database
	timeframe  string
	dryRun     bool
	concurrent int
}

// BackfillStats contains backfill statistics
type BackfillStats struct {
	TotalSymbols int
	Successful   int
	Failed       int
	TotalBars    int
	Duration     time.Duration
}

// Backfill fills historical data for symbols
func (b *Backfiller) Backfill(symbols []string, fromDate, toDate time.Time) *BackfillStats {
	startTime := time.Now()
	stats := &BackfillStats{
		TotalSymbols: len(symbols),
	}

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, b.concurrent)
	results := make(chan BackfillResult, len(symbols))

	// Process each symbol
	for _, symbol := range symbols {
		semaphore <- struct{}{} // Acquire semaphore

		go func(sym string) {
			defer func() { <-semaphore }() // Release semaphore

			result := b.backfillSymbol(sym, fromDate, toDate)
			results <- result
		}(symbol)
	}

	// Collect results
	for i := 0; i < len(symbols); i++ {
		result := <-results

		if result.Error != nil {
			stats.Failed++
			log.Printf("❌ %s: %v", result.Symbol, result.Error)
		} else {
			stats.Successful++
			stats.TotalBars += result.BarsInserted
			log.Printf("✅ %s: %d bars inserted", result.Symbol, result.BarsInserted)
		}
	}

	stats.Duration = time.Since(startTime)
	return stats
}

// BackfillResult contains result for a single symbol
type BackfillResult struct {
	Symbol       string
	BarsInserted int
	Error        error
}

// backfillSymbol backfills data for a single symbol
func (b *Backfiller) backfillSymbol(symbol string, fromDate, toDate time.Time) BackfillResult {
	// Get instrument token
	token, err := b.db.GetInstrumentToken("NSE", symbol)
	if err != nil || token == 0 {
		token, err = b.db.GetInstrumentToken("BSE", symbol)
		if err != nil || token == 0 {
			return BackfillResult{
				Symbol: symbol,
				Error:  fmt.Errorf("instrument token not found"),
			}
		}
	}

	// Fetch historical data
	// Note: The broker interface needs to support historical data fetching
	// This is a placeholder implementation
	log.Printf("🔄 Fetching data for %s (token: %d)...", symbol, token)

	// In a real implementation, you would:
	// 1. Fetch historical OHLCV data from broker
	// 2. Convert to IntradayBar format
	// 3. Insert into database

	// For now, return success with 0 bars (placeholder)
	if b.dryRun {
		log.Printf("   [DRY RUN] Would fetch %s from %s to %s",
			symbol, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
		return BackfillResult{
			Symbol:       symbol,
			BarsInserted: 0,
		}
	}

	// TODO: Implement actual data fetching and insertion
	// bars := b.broker.GetHistoricalData(token, fromDate, toDate, b.timeframe)
	// if !b.dryRun {
	//     err := b.db.BulkInsertIntradayBars(bars)
	//     if err != nil {
	//         return BackfillResult{Symbol: symbol, Error: err}
	//     }
	// }

	return BackfillResult{
		Symbol:       symbol,
		BarsInserted: 0,
		Error:        fmt.Errorf("historical data fetching not yet implemented"),
	}
}
