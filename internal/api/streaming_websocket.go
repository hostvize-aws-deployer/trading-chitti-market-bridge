package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/trading-chitti/market-bridge/internal/database"
)

// StreamingHub manages WebSocket connections for live data streaming
type StreamingHub struct {
	clients    map[*StreamingClient]bool
	broadcast  chan *StreamMessage
	register   chan *StreamingClient
	unregister chan *StreamingClient
	mu         sync.RWMutex
	db         *database.Database
}

// StreamingClient represents a connected WebSocket client
type StreamingClient struct {
	hub           *StreamingHub
	conn          *websocket.Conn
	send          chan *StreamMessage
	subscriptions map[string]bool // symbol -> subscribed
	mu            sync.RWMutex
}

// StreamMessage represents a message to stream to clients
type StreamMessage struct {
	Type      string                 `json:"type"`
	Symbol    string                 `json:"symbol,omitempty"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

var streamingUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// NewStreamingHub creates a new streaming hub
func NewStreamingHub(db *database.Database) *StreamingHub {
	return &StreamingHub{
		clients:    make(map[*StreamingClient]bool),
		broadcast:  make(chan *StreamMessage, 256),
		register:   make(chan *StreamingClient),
		unregister: make(chan *StreamingClient),
		db:         db,
	}
}

// Run starts the streaming hub
func (h *StreamingHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("📱 Client connected (total: %d)", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("📱 Client disconnected (total: %d)", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// Check if client is subscribed to this symbol
				if message.Symbol != "" {
					client.mu.RLock()
					subscribed := client.subscriptions[message.Symbol]
					client.mu.RUnlock()

					if !subscribed {
						continue
					}
				}

				select {
				case client.send <- message:
				default:
					// Client buffer full, disconnect
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastTick broadcasts a tick update to all subscribed clients
func (h *StreamingHub) BroadcastTick(symbol string, tick *database.TickData) {
	message := &StreamMessage{
		Type:      "tick",
		Symbol:    symbol,
		Data:      tick,
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- message:
	default:
		// Channel full, skip this tick
	}
}

// BroadcastBar broadcasts a new candle to all subscribed clients
func (h *StreamingHub) BroadcastBar(symbol string, bar *database.IntradayBar) {
	message := &StreamMessage{
		Type:      "bar",
		Symbol:    symbol,
		Data:      bar,
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- message:
	default:
		// Channel full, skip
	}
}

// BroadcastStats broadcasts intraday stats update
func (h *StreamingHub) BroadcastStats(symbol string, stats map[string]interface{}) {
	message := &StreamMessage{
		Type:      "stats",
		Symbol:    symbol,
		Data:      stats,
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- message:
	default:
	}
}

// GetClientCount returns the number of connected clients
func (h *StreamingHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ============================================================================
// CLIENT METHODS
// ============================================================================

func (c *StreamingClient) readPump() {
	defer func() {
		c.hub.unregister <- c
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

		// Handle client messages (subscribe/unsubscribe)
		c.handleMessage(message)
	}
}

func (c *StreamingClient) writePump() {
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

			// Write message
			data, _ := json.Marshal(message)
			w.Write(data)

			// Add queued messages to current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				msg := <-c.send
				data, _ := json.Marshal(msg)
				w.Write([]byte("\n"))
				w.Write(data)
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

func (c *StreamingClient) handleMessage(message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Invalid message: %v", err)
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "subscribe":
		symbols, ok := msg["symbols"].([]interface{})
		if !ok {
			return
		}

		c.mu.Lock()
		for _, sym := range symbols {
			if symbol, ok := sym.(string); ok {
				c.subscriptions[symbol] = true
				log.Printf("📊 Client subscribed to %s", symbol)
			}
		}
		c.mu.Unlock()

		// Send confirmation
		c.send <- &StreamMessage{
			Type: "subscribed",
			Data: map[string]interface{}{
				"symbols": symbols,
				"count":   len(symbols),
			},
			Timestamp: time.Now(),
		}

	case "unsubscribe":
		symbols, ok := msg["symbols"].([]interface{})
		if !ok {
			return
		}

		c.mu.Lock()
		for _, sym := range symbols {
			if symbol, ok := sym.(string); ok {
				delete(c.subscriptions, symbol)
				log.Printf("📊 Client unsubscribed from %s", symbol)
			}
		}
		c.mu.Unlock()

		// Send confirmation
		c.send <- &StreamMessage{
			Type: "unsubscribed",
			Data: map[string]interface{}{
				"symbols": symbols,
			},
			Timestamp: time.Now(),
		}

	case "get_latest":
		// Client requesting latest data for subscribed symbols
		c.mu.RLock()
		symbols := make([]string, 0, len(c.subscriptions))
		for symbol := range c.subscriptions {
			symbols = append(symbols, symbol)
		}
		c.mu.RUnlock()

		// Send latest bars for each symbol
		for _, symbol := range symbols {
			bar, err := c.hub.db.GetLatestIntradayBar(symbol, "1m")
			if err == nil && bar != nil {
				c.send <- &StreamMessage{
					Type:      "bar",
					Symbol:    symbol,
					Data:      bar,
					Timestamp: time.Now(),
				}
			}
		}
	}
}

// ============================================================================
// HTTP HANDLER
// ============================================================================

// StreamingHandler handles WebSocket streaming requests
type StreamingHandler struct {
	hub *StreamingHub
}

// NewStreamingHandler creates a new streaming handler
func NewStreamingHandler(db *database.Database) *StreamingHandler {
	hub := NewStreamingHub(db)
	go hub.Run()

	return &StreamingHandler{
		hub: hub,
	}
}

// RegisterRoutes registers streaming routes
func (h *StreamingHandler) RegisterRoutes(r *gin.RouterGroup) {
	stream := r.Group("/stream")
	{
		stream.GET("/ws", h.HandleWebSocket)
		stream.GET("/stats", h.GetStats)
	}
}

// HandleWebSocket handles WebSocket connections
func (h *StreamingHandler) HandleWebSocket(c *gin.Context) {
	conn, err := streamingUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &StreamingClient{
		hub:           h.hub,
		conn:          conn,
		send:          make(chan *StreamMessage, 256),
		subscriptions: make(map[string]bool),
	}

	client.hub.register <- client

	// Send welcome message
	client.send <- &StreamMessage{
		Type: "connected",
		Data: map[string]interface{}{
			"message": "Connected to Market Bridge streaming",
			"server":  "market-bridge",
			"version": "1.0.0",
		},
		Timestamp: time.Now(),
	}

	// Start read and write pumps
	go client.writePump()
	go client.readPump()
}

// GetStats returns streaming statistics
func (h *StreamingHandler) GetStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"connected_clients": h.hub.GetClientCount(),
		"channel_size":      cap(h.hub.broadcast),
		"active":            true,
	})
}

// GetHub returns the streaming hub
func (h *StreamingHandler) GetHub() *StreamingHub {
	return h.hub
}
