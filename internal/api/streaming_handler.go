package api

import (
	"github.com/gin-gonic/gin"
)

// StreamingHandler handles WebSocket streaming requests
type StreamingHandler struct {
	hub *StreamingHub
}

// NewStreamingHandler creates a new streaming handler
func NewStreamingHandler(hub *StreamingHub) *StreamingHandler {
	return &StreamingHandler{
		hub: hub,
	}
}

// RegisterRoutes registers streaming WebSocket routes
func (h *StreamingHandler) RegisterRoutes(r *gin.RouterGroup) {
	stream := r.Group("/stream")
	{
		stream.GET("/ws", h.HandleWebSocket)
	}
}

// HandleWebSocket handles WebSocket connections for streaming
// GET /stream/ws
func (h *StreamingHandler) HandleWebSocket(c *gin.Context) {
	h.hub.HandleWebSocket(c)
}
