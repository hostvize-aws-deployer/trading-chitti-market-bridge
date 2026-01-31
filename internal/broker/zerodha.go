package broker

import (
	"fmt"
	"time"
	
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	"github.com/sirupsen/logrus"
)

// ZerodhaBroker implements the Broker interface for Zerodha Kite Connect
type ZerodhaBroker struct {
	config *BrokerConfig
	kite   *kiteconnect.Client
	logger *logrus.Logger
}

// NewZerodhaBroker creates a new Zerodha broker instance
func NewZerodhaBroker(config *BrokerConfig) (*ZerodhaBroker, error) {
	kite := kiteconnect.New(config.APIKey)
	
	if config.AccessToken != "" {
		kite.SetAccessToken(config.AccessToken)
	}
	
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	
	broker := &ZerodhaBroker{
		config: config,
		kite:   kite,
		logger: logger,
	}
	
	broker.logger.Info("✅ Zerodha broker initialized")
	
	return broker, nil
}

// GetLoginURL returns the Zerodha login URL
func (z *ZerodhaBroker) GetLoginURL() string {
	return z.kite.GetLoginURL()
}

// GenerateSession generates a session using request token
func (z *ZerodhaBroker) GenerateSession(requestToken string) (*Session, error) {
	data, err := z.kite.GenerateSession(requestToken, z.config.APISecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session: %w", err)
	}
	
	z.logger.Infof("✅ Session generated for user: %s", data.UserID)
	
	return &Session{
		UserID:      data.UserID,
		AccessToken: data.AccessToken,
		ExpiresAt:   time.Now().Add(24 * time.Hour), // Expires daily
	}, nil
}

// SetAccessToken sets the access token
func (z *ZerodhaBroker) SetAccessToken(token string) {
	z.kite.SetAccessToken(token)
	z.config.AccessToken = token
}

// GetProfile returns user profile
func (z *ZerodhaBroker) GetProfile() (*Profile, error) {
	profile, err := z.kite.GetUserProfile()
	if err != nil {
		return nil, err
	}
	
	return &Profile{
		UserID:    profile.UserID,
		UserName:  profile.UserName,
		Email:     profile.Email,
		Broker:    "zerodha",
		Products:  profile.Products,
		Exchanges: profile.Exchanges,
	}, nil
}

// GetMargins returns account margins
func (z *ZerodhaBroker) GetMargins() (*Margins, error) {
	margins, err := z.kite.GetUserMargins()
	if err != nil {
		return nil, err
	}
	
	result := &Margins{}
	
	if equity, ok := margins["equity"]; ok {
		result.Equity.Available = equity.Available.Cash
		result.Equity.Used = equity.Utilised.Debits
		result.Equity.Net = equity.Net
	}
	
	if commodity, ok := margins["commodity"]; ok {
		result.Commodity.Available = commodity.Available.Cash
		result.Commodity.Used = commodity.Utilised.Debits
		result.Commodity.Net = commodity.Net
	}
	
	z.logger.Infof("💰 Equity Available: ₹%.2f", result.Equity.Available)
	
	return result, nil
}

// GetPositions returns current positions
func (z *ZerodhaBroker) GetPositions() (*Positions, error) {
	positions, err := z.kite.GetPositions()
	if err != nil {
		return nil, err
	}
	
	result := &Positions{
		Net: make([]Position, 0, len(positions.Net)),
		Day: make([]Position, 0, len(positions.Day)),
	}
	
	for _, p := range positions.Net {
		result.Net = append(result.Net, Position{
			Symbol:       p.Tradingsymbol,
			Exchange:     p.Exchange,
			Product:      p.Product,
			Quantity:     p.Quantity,
			AveragePrice: p.AveragePrice,
			LastPrice:    p.LastPrice,
			PNL:          p.PNL,
			Overnight:    p.Overnight,
		})
	}
	
	for _, p := range positions.Day {
		result.Day = append(result.Day, Position{
			Symbol:       p.Tradingsymbol,
			Exchange:     p.Exchange,
			Product:      p.Product,
			Quantity:     p.Quantity,
			AveragePrice: p.AveragePrice,
			LastPrice:    p.LastPrice,
			PNL:          p.PNL,
			Overnight:    false,
		})
	}
	
	z.logger.Infof("📊 Positions: %d net, %d day", len(result.Net), len(result.Day))
	
	return result, nil
}

// GetHoldings returns holdings
func (z *ZerodhaBroker) GetHoldings() ([]Holding, error) {
	holdings, err := z.kite.GetHoldings()
	if err != nil {
		return nil, err
	}
	
	result := make([]Holding, 0, len(holdings))
	for _, h := range holdings {
		result = append(result, Holding{
			Symbol:       h.Tradingsymbol,
			Exchange:     h.Exchange,
			Quantity:     h.Quantity,
			AveragePrice: h.AveragePrice,
			LastPrice:    h.LastPrice,
			PNL:          h.PNL,
			PNLPercent:   (h.LastPrice - h.AveragePrice) / h.AveragePrice * 100,
		})
	}
	
	z.logger.Infof("💼 Holdings: %d stocks", len(result))
	
	return result, nil
}

// GetOrders returns orders for the day
func (z *ZerodhaBroker) GetOrders() ([]Order, error) {
	orders, err := z.kite.GetOrders()
	if err != nil {
		return nil, err
	}
	
	result := make([]Order, 0, len(orders))
	for _, o := range orders {
		result = append(result, Order{
			OrderID:          o.OrderID,
			Symbol:           o.Tradingsymbol,
			Exchange:         o.Exchange,
			TransactionType:  o.TransactionType,
			OrderType:        o.OrderType,
			Product:          o.Product,
			Quantity:         o.Quantity,
			Price:            o.Price,
			TriggerPrice:     o.TriggerPrice,
			Status:           o.Status,
			FilledQuantity:   o.FilledQuantity,
			PendingQuantity:  o.PendingQuantity,
			AveragePrice:     o.AveragePrice,
			PlacedAt:         o.OrderTimestamp.Time,
			UpdatedAt:        o.ExchangeUpdateTimestamp.Time,
		})
	}
	
	z.logger.Infof("📝 Orders today: %d", len(result))
	
	return result, nil
}

// GetQuote returns real-time quotes
func (z *ZerodhaBroker) GetQuote(symbols []string) (map[string]Quote, error) {
	quotes, err := z.kite.GetQuote(symbols...)
	if err != nil {
		return nil, err
	}
	
	result := make(map[string]Quote)
	for symbol, q := range quotes {
		result[symbol] = Quote{
			Symbol:        symbol,
			LastPrice:     q.LastPrice,
			Open:          q.OHLC.Open,
			High:          q.OHLC.High,
			Low:           q.OHLC.Low,
			Close:         q.OHLC.Close,
			Change:        q.NetChange,
			ChangePercent: (q.LastPrice - q.OHLC.Close) / q.OHLC.Close * 100,
			Volume:        int64(q.Volume),
			BuyQuantity:   int64(q.BuyQuantity),
			SellQuantity:  int64(q.SellQuantity),
			Timestamp:     q.Timestamp.Time,
		}
	}
	
	return result, nil
}

// GetLTP returns last traded prices
func (z *ZerodhaBroker) GetLTP(symbols []string) (map[string]float64, error) {
	ltp, err := z.kite.GetLTP(symbols...)
	if err != nil {
		return nil, err
	}
	
	result := make(map[string]float64)
	for symbol, data := range ltp {
		result[symbol] = data.LastPrice
	}
	
	return result, nil
}

// GetHistoricalData returns historical OHLCV data
func (z *ZerodhaBroker) GetHistoricalData(instrument string, from, to time.Time, interval string) ([]Candle, error) {
	// instrument should be instrument_token as string
	// For now, return error - need to implement instrument token lookup
	return nil, fmt.Errorf("not implemented - need instrument token")
}

// GetInstruments returns all tradable instruments
func (z *ZerodhaBroker) GetInstruments(exchange string) ([]Instrument, error) {
	instruments, err := z.kite.GetInstruments()
	if err != nil {
		return nil, err
	}
	
	result := make([]Instrument, 0)
	for _, inst := range instruments {
		if exchange == "" || inst.Exchange == exchange {
			var expiry *time.Time
			if !inst.Expiry.IsZero() {
				t := inst.Expiry.Time
				expiry = &t
			}
			
			result = append(result, Instrument{
				InstrumentToken: int64(inst.InstrumentToken),
				ExchangeToken:   int64(inst.ExchangeToken),
				TradingSymbol:   inst.Tradingsymbol,
				Name:            inst.Name,
				Exchange:        inst.Exchange,
				InstrumentType:  inst.InstrumentType,
				Segment:         inst.Segment,
				Expiry:          expiry,
				Strike:          inst.Strike,
				TickSize:        inst.TickSize,
				LotSize:         inst.LotSize,
			})
		}
	}
	
	z.logger.Infof("🏢 Loaded %d instruments from %s", len(result), exchange)
	
	return result, nil
}

// PlaceOrder places a new order
func (z *ZerodhaBroker) PlaceOrder(order *OrderRequest) (string, error) {
	params := kiteconnect.OrderParams{
		Exchange:        order.Exchange,
		Tradingsymbol:   order.Symbol,
		TransactionType: order.TransactionType,
		OrderType:       order.OrderType,
		Product:         order.Product,
		Quantity:        order.Quantity,
		Price:           order.Price,
		TriggerPrice:    order.TriggerPrice,
		Validity:        order.Validity,
		Tag:             order.Tag,
	}
	
	response, err := z.kite.PlaceOrder(kiteconnect.VarietyRegular, params)
	if err != nil {
		return "", err
	}
	
	z.logger.Infof("📤 Order placed: %s - %s %d %s @ %s", 
		response.OrderID, order.TransactionType, order.Quantity, order.Symbol, order.OrderType)
	
	return response.OrderID, nil
}

// ModifyOrder modifies an existing order
func (z *ZerodhaBroker) ModifyOrder(orderID string, modify *OrderModify) (string, error) {
	params := kiteconnect.OrderParams{}
	
	if modify.Quantity != nil {
		params.Quantity = *modify.Quantity
	}
	if modify.Price != nil {
		params.Price = *modify.Price
	}
	if modify.TriggerPrice != nil {
		params.TriggerPrice = *modify.TriggerPrice
	}
	if modify.OrderType != nil {
		params.OrderType = *modify.OrderType
	}
	
	response, err := z.kite.ModifyOrder(kiteconnect.VarietyRegular, orderID, params)
	if err != nil {
		return "", err
	}
	
	z.logger.Infof("✏️  Order modified: %s", response.OrderID)
	
	return response.OrderID, nil
}

// CancelOrder cancels an order
func (z *ZerodhaBroker) CancelOrder(orderID string) (string, error) {
	response, err := z.kite.CancelOrder(kiteconnect.VarietyRegular, orderID)
	if err != nil {
		return "", err
	}
	
	z.logger.Infof("❌ Order cancelled: %s", response.OrderID)
	
	return response.OrderID, nil
}

// IsMarketOpen checks if market is open
func (z *ZerodhaBroker) IsMarketOpen() bool {
	loc, _ := time.LoadLocation("Asia/Kolkata")
	now := time.Now().In(loc)
	
	// Weekend check
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}
	
	// Market hours: 9:15 AM - 3:30 PM IST
	marketOpen := time.Date(now.Year(), now.Month(), now.Day(), 9, 15, 0, 0, loc)
	marketClose := time.Date(now.Year(), now.Month(), now.Day(), 15, 30, 0, 0, loc)
	
	return now.After(marketOpen) && now.Before(marketClose)
}

// GetMarketStatus returns current market status
func (z *ZerodhaBroker) GetMarketStatus() string {
	if z.IsMarketOpen() {
		return "OPEN"
	}
	
	loc, _ := time.LoadLocation("Asia/Kolkata")
	now := time.Now().In(loc)
	
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return "WEEKEND"
	}
	
	if now.Hour() < 9 {
		return "PRE_MARKET"
	}
	
	return "CLOSED"
}

// GetBrokerName returns the broker name
func (z *ZerodhaBroker) GetBrokerName() string {
	return "zerodha"
}
