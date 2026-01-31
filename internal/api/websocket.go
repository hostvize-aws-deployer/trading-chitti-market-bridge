package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	"github.com/zerodha/gokiteconnect/v4/models"
	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// WebSocketClient represents a connected client
type WebSocketClient struct {
	conn         *websocket.Conn
	send         chan []byte
	hub          *WebSocketHub
	subscriptions map[string]bool
	mu           sync.RWMutex
}

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
	
	// Zerodha ticker for real-time market data
	ticker *kiteticker.Ticker
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(apiKey, accessToken string) *WebSocketHub {
	hub := &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
	
	// Initialize Zerodha WebSocket ticker
	ticker := kiteticker.New(apiKey, accessToken)
	hub.ticker = ticker

	// Enable auto-reconnect with retry logic
	ticker.SetAutoReconnect(true)
	ticker.SetReconnectMaxRetries(10)
	ticker.SetReconnectMaxDelay(60 * time.Second)

	// Set up ticker callbacks
	ticker.OnConnect(hub.onTickerConnect)
	ticker.OnTick(hub.onTick)
	ticker.OnError(hub.onTickerError)
	ticker.OnClose(hub.onTickerClose)
	ticker.OnReconnect(hub.onTickerReconnect)
	ticker.OnNoReconnect(hub.onTickerNoReconnect)
	ticker.OnOrderUpdate(hub.onOrderUpdate)

	return hub
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("📡 WebSocket client connected (total: %d)", len(h.clients))
			
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("📡 WebSocket client disconnected (total: %d)", len(h.clients))
			
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// StartTicker starts the Zerodha WebSocket ticker
func (h *WebSocketHub) StartTicker() {
	go h.ticker.Serve()
}

// Subscribe subscribes to instrument tokens
func (h *WebSocketHub) Subscribe(tokens []uint32) {
	if h.ticker != nil {
		h.ticker.Subscribe(tokens)
		h.ticker.SetMode(kiteticker.ModeFull, tokens)
	}
}

// Ticker callbacks
func (h *WebSocketHub) onTickerConnect() {
	log.Println("✅ Zerodha WebSocket ticker connected")
}

func (h *WebSocketHub) onTick(tick models.Tick) {
	data := map[string]interface{}{
		"type":          "tick",
		"instrument_token": tick.InstrumentToken,
		"last_price":    tick.LastPrice,
		"last_quantity": tick.LastTradedQuantity,
		"volume":        tick.VolumeTraded,
		"timestamp":     tick.Timestamp.Time,
		"ohlc": map[string]float64{
			"open":  tick.OHLC.Open,
			"high":  tick.OHLC.High,
			"low":   tick.OHLC.Low,
			"close": tick.OHLC.Close,
		},
	}

	if msg, err := json.Marshal(data); err == nil {
		h.broadcast <- msg
	}
}

func (h *WebSocketHub) onTickerError(err error) {
	log.Printf("❌ Ticker error: %v", err)
}

func (h *WebSocketHub) onTickerClose(code int, reason string) {
	log.Printf("⚠️  Ticker closed: %d - %s", code, reason)
}

func (h *WebSocketHub) onTickerReconnect(attempt int, delay time.Duration) {
	log.Printf("🔄 Reconnecting to ticker... attempt %d, delay %v", attempt, delay)

	// Broadcast reconnection status to all clients
	data := map[string]interface{}{
		"type":    "status",
		"status":  "reconnecting",
		"attempt": attempt,
		"delay":   delay.String(),
	}

	if msg, err := json.Marshal(data); err == nil {
		h.broadcast <- msg
	}
}

func (h *WebSocketHub) onTickerNoReconnect(attempt int) {
	log.Printf("❌ Max reconnection attempts reached (%d). Connection failed.", attempt)

	// Broadcast connection failure to all clients
	data := map[string]interface{}{
		"type":    "status",
		"status":  "disconnected",
		"message": "Max reconnection attempts reached",
	}

	if msg, err := json.Marshal(data); err == nil {
		h.broadcast <- msg
	}
}

func (h *WebSocketHub) onOrderUpdate(order kiteconnect.Order) {
	log.Printf("📋 Order Update: %s | Status: %s | Filled: %d/%d",
		order.OrderID,
		order.Status,
		order.FilledQuantity,
		order.Quantity)

	// Broadcast order update to all clients
	data := map[string]interface{}{
		"type":            "order_update",
		"order_id":        order.OrderID,
		"status":          order.Status,
		"tradingsymbol":   order.TradingSymbol,
		"exchange":        order.Exchange,
		"transaction_type": order.TransactionType,
		"quantity":        order.Quantity,
		"filled_quantity": order.FilledQuantity,
		"pending_quantity": order.PendingQuantity,
		"price":           order.Price,
		"average_price":   order.AveragePrice,
		"status_message":  order.StatusMessage,
		"timestamp":       order.OrderTimestamp.Time,
	}

	if msg, err := json.Marshal(data); err == nil {
		h.broadcast <- msg
	}
}

// HandleWebSocket handles WebSocket connections
func (a *API) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	
	client := &WebSocketClient{
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}
	
	// Register client
	if a.wsHub != nil {
		a.wsHub.register <- client
	}
	
	// Start goroutines for reading and writing
	go client.readPump()
	go client.writePump()
}

// readPump reads messages from WebSocket client
func (c *WebSocketClient) readPump() {
	defer func() {
		if c.hub != nil {
			c.hub.unregister <- c
		}
		c.conn.Close()
	}()
	
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		
		// Handle subscription messages
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			if action, ok := msg["action"].(string); ok {
				switch action {
				case "subscribe":
					if tokens, ok := msg["tokens"].([]interface{}); ok {
						// Convert to uint32 slice and subscribe
						var tokenList []uint32
						for _, t := range tokens {
							if token, ok := t.(float64); ok {
								tokenList = append(tokenList, uint32(token))
							}
						}
						if c.hub != nil {
							c.hub.Subscribe(tokenList)
						}
					}
				case "unsubscribe":
					// Handle unsubscribe
				}
			}
		}
	}
}

// writePump writes messages to WebSocket client
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			
			// Add queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}
			
			if err := w.Close(); err != nil {
				return
			}
			
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Add WebSocket route to API
func (a *API) RegisterWebSocketRoutes(r *gin.Engine) {
	// WebSocket endpoint
	r.GET("/ws", a.HandleWebSocket)
	
	// WebSocket for market data streaming
	r.GET("/ws/market", a.HandleWebSocket)
	
	// WebSocket for order updates
	r.GET("/ws/orders", a.HandleWebSocket)
	
	// WebSocket for position updates
	r.GET("/ws/positions", a.HandleWebSocket)
}
