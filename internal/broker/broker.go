package broker

import (
	"time"
)

// Broker defines the interface that all broker implementations must satisfy
// This allows pluggable support for Zerodha, Angel Broking, Upstox, etc.
type Broker interface {
	// Authentication
	GetLoginURL() string
	GenerateSession(requestToken string) (*Session, error)
	SetAccessToken(token string)
	
	// Account Info
	GetProfile() (*Profile, error)
	GetMargins() (*Margins, error)
	GetPositions() (*Positions, error)
	GetHoldings() ([]Holding, error)
	GetOrders() ([]Order, error)
	
	// Market Data
	GetQuote(symbols []string) (map[string]Quote, error)
	GetLTP(symbols []string) (map[string]float64, error)
	GetHistoricalData(instrument string, from, to time.Time, interval string) ([]Candle, error)
	GetInstruments(exchange string) ([]Instrument, error)
	
	// Trading
	PlaceOrder(order *OrderRequest) (string, error)
	ModifyOrder(orderID string, order *OrderModify) (string, error)
	CancelOrder(orderID string) (string, error)
	
	// Utility
	IsMarketOpen() bool
	GetMarketStatus() string
	GetBrokerName() string
}

// Session represents authentication session
type Session struct {
	UserID      string
	AccessToken string
	ExpiresAt   time.Time
}

// Profile represents user profile
type Profile struct {
	UserID    string
	UserName  string
	Email     string
	Phone     string
	Broker    string
	Products  []string
	Exchanges []string
}

// Margins represents account margins
type Margins struct {
	Equity struct {
		Available float64
		Used      float64
		Net       float64
	}
	Commodity struct {
		Available float64
		Used      float64
		Net       float64
	}
}

// Positions represents current positions
type Positions struct {
	Net []Position
	Day []Position
}

// Position represents a single position
type Position struct {
	Symbol         string
	Exchange       string
	Product        string
	Quantity       int
	AveragePrice   float64
	LastPrice      float64
	PNL            float64
	Overnight      bool
}

// Holding represents a long-term holding
type Holding struct {
	Symbol       string
	Exchange     string
	Quantity     int
	AveragePrice float64
	LastPrice    float64
	PNL          float64
	PNLPercent   float64
}

// Order represents an order
type Order struct {
	OrderID          string
	Symbol           string
	Exchange         string
	TransactionType  string // BUY or SELL
	OrderType        string // MARKET, LIMIT, SL, SL-M
	Product          string // MIS, CNC, NRML
	Quantity         int
	Price            float64
	TriggerPrice     float64
	Status           string
	FilledQuantity   int
	PendingQuantity  int
	AveragePrice     float64
	PlacedAt         time.Time
	UpdatedAt        time.Time
}

// OrderRequest represents a new order request
type OrderRequest struct {
	Symbol          string
	Exchange        string
	TransactionType string
	OrderType       string
	Product         string
	Quantity        int
	Price           float64
	TriggerPrice    float64
	Validity        string // DAY, IOC
	Tag             string
}

// OrderModify represents order modification
type OrderModify struct {
	Quantity     *int
	Price        *float64
	TriggerPrice *float64
	OrderType    *string
}

// Quote represents real-time quote
type Quote struct {
	Symbol       string
	LastPrice    float64
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Change       float64
	ChangePercent float64
	Volume       int64
	BuyQuantity  int64
	SellQuantity int64
	Timestamp    time.Time
}

// Candle represents OHLCV candle
type Candle struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// Instrument represents a tradable instrument
type Instrument struct {
	InstrumentToken int64
	ExchangeToken   int64
	TradingSymbol   string
	Name            string
	Exchange        string
	InstrumentType  string
	Segment         string
	Expiry          *time.Time
	Strike          float64
	TickSize        float64
	LotSize         int
}

// BrokerConfig represents broker configuration from database
type BrokerConfig struct {
	ID             int
	BrokerName     string    // zerodha, angelone, upstox, icicidirect
	DisplayName    string
	Enabled        bool
	APIKey         string
	APISecret      string
	AccessToken    string
	UserID         string
	MaxPositions   int
	MaxRiskPerTrade float64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Factory creates broker instances based on config
func NewBroker(config *BrokerConfig) (Broker, error) {
	switch config.BrokerName {
	case "zerodha":
		return NewZerodhaBroker(config)
	case "angelone":
		// return NewAngelOneBroker(config)
		return nil, ErrBrokerNotSupported
	case "upstox":
		// return NewUpstoxBroker(config)
		return nil, ErrBrokerNotSupported
	default:
		return nil, ErrBrokerNotSupported
	}
}
