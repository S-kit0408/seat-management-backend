package websocket

import (
	"context"
	"log"
	"sync"
)

type Hub struct {
	// Registered connections
	connections map[*Connection]bool

	// Inbound messages from connections
	broadcast chan []byte

	// Register requests from connections
	register chan *Connection

	// Unregister requests from connections
	unregister chan *Connection

	// Event channel for seat updates
	eventChannel chan SeatEvent

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		connections:  make(map[*Connection]bool),
		broadcast:    make(chan []byte, 512),
		register:     make(chan *Connection),
		unregister:   make(chan *Connection),
		eventChannel: make(chan SeatEvent, 512),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.connections[conn] = true
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.connections[conn]; ok {
				delete(h.connections, conn)
				close(conn.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.connections {
				select {
				case conn.send <- message:
				default:
					// Connection send buffer full, close and remove connection
					log.Printf("WebSocket broadcast send buffer full, closing connection for user: %s", conn.userID)
					close(conn.send)
					delete(h.connections, conn)
				}
			}
			h.mu.RUnlock()

		case event := <-h.eventChannel:
			h.handleSeatEvent(event)

		case <-ctx.Done():
			return
		}
	}
}

func (h *Hub) BroadcastSeatEvent(event SeatEvent) {
	select {
	case h.eventChannel <- event:
	default:
		// Event channel full, discard event - log for monitoring
		log.Printf("WebSocket event channel full, discarding event for seat: %s, status: %s", event.SeatID, event.Status)
	}
}

func (h *Hub) RegisterConnection(conn *Connection) {
	h.register <- conn
}

func (h *Hub) UnregisterConnection(conn *Connection) {
	h.unregister <- conn
}

func (h *Hub) handleSeatEvent(event SeatEvent) {
	// Convert event to JSON message
	message := createMessageFromEvent(event)

	// Broadcast to all connections (privacy filtering done per connection)
	h.mu.RLock()
	for conn := range h.connections {
		// Apply privacy filtering based on connection's user and event privacy
		filteredMessage := conn.applyPrivacyFilter(message, event)

		select {
		case conn.send <- filteredMessage:
		default:
			// Connection send buffer full, close and remove connection
			log.Printf("WebSocket connection send buffer full, closing connection for user: %s", conn.userID)
			close(conn.send)
			delete(h.connections, conn)
		}
	}
	h.mu.RUnlock()
}
