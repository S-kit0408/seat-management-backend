package websocket

import (
	"context"
	"time"

	"github.com/gorilla/websocket"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Connection struct {
	hub            *Hub
	ws             *websocket.Conn
	send           chan []byte
	userID         string
	friendshipRepo repository.FriendshipRepository
}

func NewConnection(hub *Hub, ws *websocket.Conn, userID string, friendshipRepo repository.FriendshipRepository) *Connection {
	return &Connection{
		hub:            hub,
		ws:             ws,
		send:           make(chan []byte, 256),
		userID:         userID,
		friendshipRepo: friendshipRepo,
	}
}

func (c *Connection) ReadPump() {
	defer func() {
		c.hub.UnregisterConnection(c)
		c.ws.Close()
	}()

	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		// Handle client messages (ping, etc.) - future extensibility
		_ = message
	}
}

func (c *Connection) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.ws.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Connection) applyPrivacyFilter(message []byte, event SeatEvent) []byte {
	// If public or user is the reservation owner, return full message
	if event.PrivacySetting == entity.PrivacyPublic || event.UserID == c.userID {
		return message
	}

	// If private, return only seat status (no user details)
	if event.PrivacySetting == entity.PrivacyPrivate {
		return createPrivateMessage(event)
	}

	// If friends-only, check friendship
	if event.PrivacySetting == entity.PrivacyFriends {
		// Create a context with timeout to prevent indefinite wait
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		isFriend, err := c.friendshipRepo.CheckFriendship(ctx, c.userID, event.UserID)
		if err != nil {
			// On error, default to private message for privacy safety
			return createPrivateMessage(event)
		}

		if isFriend {
			return message // Full message for friends
		}
		return createPrivateMessage(event) // Limited info for non-friends
	}

	return message
}
