package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"
)

type FriendHandler struct {
	friendUsecase usecase.FriendUsecase
	userUsecase   usecase.UserUsecase
}

func NewFriendHandler(fu usecase.FriendUsecase, uu usecase.UserUsecase) *FriendHandler {
	return &FriendHandler{
		friendUsecase: fu,
		userUsecase:   uu,
	}
}

// リクエスト構造体
type SendFriendRequestRequest struct {
	AddresseeName string  `json:"addressee_name" binding:"required"`
	Message       *string `json:"message"`
}

// フレンド申請を送信
func (h *FriendHandler) SendFriendRequest(c *gin.Context) {
	var req SendFriendRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	addressee, err := h.userUsecase.GetByName(c.Request.Context(), req.AddresseeName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "指定されたユーザーが見つかりません"})
		return
	}

	if err := h.friendUsecase.SendFriendRequest(c.Request.Context(), user.ID, addressee.ID, req.Message); err != nil {
		log.Printf("Failed to send friend request from %s to %s: %v", user.ID, addressee.ID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "フレンド申請を送信しました"})
}

// 受信した申請一覧を取得
func (h *FriendHandler) GetReceivedRequests(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	requests, err := h.friendUsecase.GetReceivedRequests(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Failed to get received requests for user %s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requests)
}

// 送信した申請一覧を取得
func (h *FriendHandler) GetSentRequests(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	requests, err := h.friendUsecase.GetSentRequests(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requests)
}

// 申請を承認
func (h *FriendHandler) AcceptFriendRequest(c *gin.Context) {
	requestID := c.Param("id")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストIDが必要です"})
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	if err := h.friendUsecase.AcceptFriendRequest(c.Request.Context(), requestID, user.ID); err != nil {
		log.Printf("Failed to accept friend request %s by user %s: %v", requestID, user.ID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "フレンド申請を承認しました"})
}

// 申請を拒否
func (h *FriendHandler) RejectFriendRequest(c *gin.Context) {
	requestID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	if err := h.friendUsecase.RejectFriendRequest(c.Request.Context(), requestID, user.ID); err != nil {
		log.Printf("Failed to reject friend request %s by user %s: %v", requestID, user.ID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "フレンド申請を拒否しました"})
}

// 申請をキャンセル
func (h *FriendHandler) CancelFriendRequest(c *gin.Context) {
	requestID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	if err := h.friendUsecase.CancelFriendRequest(c.Request.Context(), requestID, user.ID); err != nil {
		log.Printf("Failed to cancel friend request %s by user %s: %v", requestID, user.ID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "フレンド申請をキャンセルしました"})
}

// フレンドリストを取得
func (h *FriendHandler) GetFriendsList(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	friends, err := h.friendUsecase.GetFriendsList(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, friends)
}

// フレンドを解除
func (h *FriendHandler) RemoveFriend(c *gin.Context) {
	friendID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	if err := h.friendUsecase.RemoveFriend(c.Request.Context(), user.ID, friendID); err != nil {
		log.Printf("Failed to remove friend %s for user %s: %v", friendID, user.ID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "フレンドを解除しました"})
}

// フレンド関係を確認
func (h *FriendHandler) CheckFriendship(c *gin.Context) {
	targetID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	isFriend, err := h.friendUsecase.CheckFriendship(c.Request.Context(), user.ID, targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_friend": isFriend})
}

// ルート登録
func (h *FriendHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/friends")
	api.Use(middleware.ClerkAuthMiddleware())
	{
		api.POST("/requests", h.SendFriendRequest)
		api.GET("/requests/received", h.GetReceivedRequests)
		api.GET("/requests/sent", h.GetSentRequests)
		api.POST("/requests/:id/accept", h.AcceptFriendRequest)
		api.POST("/requests/:id/reject", h.RejectFriendRequest)
		api.DELETE("/requests/:id", h.CancelFriendRequest)
		api.GET("", h.GetFriendsList)
		api.DELETE("/:id", h.RemoveFriend)
		api.GET("/:id/status", h.CheckFriendship)
	}
}
