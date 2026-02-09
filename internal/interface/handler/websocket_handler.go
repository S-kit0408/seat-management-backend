package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"seat-management-backend/internal/domain/repository"
	wsinf "seat-management-backend/internal/infrastructure/websocket"
	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandler struct {
	hub            *wsinf.Hub
	userUsecase    usecase.UserUsecase
	friendshipRepo repository.FriendshipRepository
}

func NewWebSocketHandler(hub *wsinf.Hub, uu usecase.UserUsecase, fr repository.FriendshipRepository) *WebSocketHandler {
	return &WebSocketHandler{
		hub:            hub,
		userUsecase:    uu,
		friendshipRepo: fr,
	}
}

// HandleConnection godoc
// @Summary WebSocket connection endpoint
// @Description Establish WebSocket connection for real-time seat updates
// @Tags websocket
// @Security BearerAuth
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /ws [get]
func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "WebSocket接続のアップグレードに失敗しました", CodeInternalError)
		return
	}

	conn := wsinf.NewConnection(h.hub, wsConn, user.ID, h.friendshipRepo)
	h.hub.RegisterConnection(conn)

	go conn.WritePump()
	go conn.ReadPump()
}

func (h *WebSocketHandler) RegisterRoutes(r *gin.Engine) {
	wsGroup := r.Group("/api/ws")
	wsGroup.Use(middleware.ClerkAuthMiddleware())
	{
		wsGroup.GET("", h.HandleConnection)
	}
}
