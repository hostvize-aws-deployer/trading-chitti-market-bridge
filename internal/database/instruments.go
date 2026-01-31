package database

import (
	"log"
	"time"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	"github.com/trading-chitti/market-bridge/internal/broker"
)

// SyncInstrumentsFromBroker fetches all instruments from broker and syncs to database
func (db *Database) SyncInstrumentsFromBroker(brk broker.Broker) error {
	log.Println("🔄 Starting instrument sync...")

	// Type assertion to get underlying Kite client
	zerodhaBroker, ok := brk.(*broker.ZerodhaBroker)
	if !ok {
		log.Println("⚠️  Instrument sync currently only supports Zerodha")
		return nil
	}

	// Fetch instruments from Zerodha
	instruments, err := zerodhaBroker.GetClient().GetInstruments()
	if err != nil {
		return err
	}

	log.Printf("📥 Fetched %d instruments from broker", len(instruments))

	// Sync to database in batches
	batchSize := 1000
	synced := 0

	for i := 0; i < len(instruments); i += batchSize {
		end := i + batchSize
		if end > len(instruments) {
			end = len(instruments)
		}

		batch := instruments[i:end]
		for _, inst := range batch {
			dbInst := convertToDBInstrument(inst)
			if err := db.UpsertInstrument(dbInst); err != nil {
				log.Printf("❌ Error syncing %s: %v", inst.Tradingsymbol, err)
				continue
			}
			synced++
		}

		log.Printf("📊 Synced %d/%d instruments", synced, len(instruments))
	}

	log.Printf("✅ Instrument sync completed: %d instruments synced", synced)
	return nil
}

// convertToDBInstrument converts Kite instrument to database instrument
func convertToDBInstrument(kiteInst kiteconnect.Instrument) Instrument {
	inst := Instrument{
		InstrumentToken: kiteInst.InstrumentToken,
		ExchangeToken:   kiteInst.ExchangeToken,
		Tradingsymbol:   kiteInst.Tradingsymbol,
		Name:            kiteInst.Name,
		Exchange:        kiteInst.Exchange,
		Segment:         kiteInst.Segment,
		InstrumentType:  kiteInst.InstrumentType,
		TickSize:        kiteInst.TickSize,
		LotSize:         int(kiteInst.LotSize),
		LastPrice:       kiteInst.LastPrice,
		LastUpdated:     time.Now(),
	}

	// Handle expiry
	if !kiteInst.Expiry.IsZero() {
		inst.Expiry = &kiteInst.Expiry.Time
	}

	// Handle strike
	if kiteInst.Strike > 0 {
		inst.Strike = kiteInst.Strike
	}

	return inst
}

// SyncInstrumentsByExchange syncs instruments for specific exchange
func (db *Database) SyncInstrumentsByExchange(brk broker.Broker, exchange string) error {
	log.Printf("🔄 Starting instrument sync for exchange: %s", exchange)

	zerodhaBroker, ok := brk.(*broker.ZerodhaBroker)
	if !ok {
		return nil
	}

	instruments, err := zerodhaBroker.GetClient().GetInstruments()
	if err != nil {
		return err
	}

	synced := 0
	for _, inst := range instruments {
		if inst.Exchange != exchange {
			continue
		}

		dbInst := convertToDBInstrument(inst)
		if err := db.UpsertInstrument(dbInst); err != nil {
			log.Printf("❌ Error syncing %s: %v", inst.Tradingsymbol, err)
			continue
		}
		synced++
	}

	log.Printf("✅ Synced %d instruments for exchange %s", synced, exchange)
	return nil
}

// GetInstrumentTokensForSymbols returns instrument tokens for given symbols
func (db *Database) GetInstrumentTokensForSymbols(exchange string, symbols []string) ([]uint32, error) {
	tokens := []uint32{}

	for _, symbol := range symbols {
		token, err := db.GetInstrumentToken(exchange, symbol)
		if err != nil {
			return nil, err
		}
		if token > 0 {
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}
