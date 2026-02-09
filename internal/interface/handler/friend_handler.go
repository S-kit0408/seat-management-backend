package handler

import (
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

// SendFriendRequest godoc
// @Summary Send friend request
// @Description Send a friend request to another user by name
// @Tags friends
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body SendFriendRequestRequest true "Friend request details"
// @Success 201 {object} map[string]string
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 404 {object} handler.ErrorResponse "User not found"
// @Router /friends/requests [post]
func (h *FriendHandler) SendFriendRequest(c *gin.Context) {
	var req SendFriendRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	addressee, err := h.userUsecase.GetByName(c.Request.Context(), req.AddresseeName)
	if err != nil {
		HandleUsecaseError(c, err, "ユーザーの取得に失敗しました")
		return
	}

	if err := h.friendUsecase.SendFriendRequest(c.Request.Context(), user.ID, addressee.ID, req.Message); err != nil {
		HandleUsecaseError(c, err, "フレンド申請の送信に失敗しました")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "フレンド申請を送信しました"})
}

// GetReceivedRequests godoc
// @Summary Get received friend requests
// @Description Get list of friend requests received by the authenticated user
// @Tags friends
// @Security BearerAuth
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /friends/requests/received [get]
func (h *FriendHandler) GetReceivedRequests(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	requests, err := h.friendUsecase.GetReceivedRequests(c.Request.Context(), user.ID)
	if err != nil {
		HandleUsecaseError(c, err, "申請一覧の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, requests)
}

// GetSentRequests godoc
// @Summary Get sent friend requests
// @Description Get list of friend requests sent by the authenticated user
// @Tags friends
// @Security BearerAuth
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /friends/requests/sent [get]
func (h *FriendHandler) GetSentRequests(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	requests, err := h.friendUsecase.GetSentRequests(c.Request.Context(), user.ID)
	if err != nil {
		HandleUsecaseError(c, err, "申請一覧の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, requests)
}

// AcceptFriendRequest godoc
// @Summary Accept friend request
// @Description Accept a received friend request
// @Tags friends
// @Security BearerAuth
// @Param id path string true "Friend request ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Router /friends/requests/{id}/accept [post]
func (h *FriendHandler) AcceptFriendRequest(c *gin.Context) {
	requestID := c.Param("id")
	if requestID == "" {
		RespondWithError(c, http.StatusBadRequest, "リクエストIDが必要です", CodeBadRequest)
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	if err := h.friendUsecase.AcceptFriendRequest(c.Request.Context(), requestID, user.ID); err != nil {
		HandleUsecaseError(c, err, "申請の承認に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "フレンド申請を承認しました"})
}

// RejectFriendRequest godoc
// @Summary Reject friend request
// @Description Reject a received friend request
// @Tags friends
// @Security BearerAuth
// @Param id path string true "Friend request ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Router /friends/requests/{id}/reject [post]
func (h *FriendHandler) RejectFriendRequest(c *gin.Context) {
	requestID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	if err := h.friendUsecase.RejectFriendRequest(c.Request.Context(), requestID, user.ID); err != nil {
		HandleUsecaseError(c, err, "申請の拒否に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "フレンド申請を拒否しました"})
}

// CancelFriendRequest godoc
// @Summary Cancel friend request
// @Description Cancel a sent friend request
// @Tags friends
// @Security BearerAuth
// @Param id path string true "Friend request ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Router /friends/requests/{id} [delete]
func (h *FriendHandler) CancelFriendRequest(c *gin.Context) {
	requestID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	if err := h.friendUsecase.CancelFriendRequest(c.Request.Context(), requestID, user.ID); err != nil {
		HandleUsecaseError(c, err, "申請のキャンセルに失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "フレンド申請をキャンセルしました"})
}

// GetFriendsList godoc
// @Summary Get friends list
// @Description Get list of friends for the authenticated user
// @Tags friends
// @Security BearerAuth
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /friends [get]
func (h *FriendHandler) GetFriendsList(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	friends, err := h.friendUsecase.GetFriendsList(c.Request.Context(), user.ID)
	if err != nil {
		HandleUsecaseError(c, err, "フレンドリストの取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, friends)
}

// RemoveFriend godoc
// @Summary Remove friend
// @Description Remove a friend from the authenticated user's friend list
// @Tags friends
// @Security BearerAuth
// @Param id path string true "Friend user ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Router /friends/{id} [delete]
func (h *FriendHandler) RemoveFriend(c *gin.Context) {
	friendID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	if err := h.friendUsecase.RemoveFriend(c.Request.Context(), user.ID, friendID); err != nil {
		HandleUsecaseError(c, err, "フレンド削除に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "フレンドを解除しました"})
}

// CheckFriendship godoc
// @Summary Check friendship status
// @Description Check if a user is a friend of the authenticated user
// @Tags friends
// @Security BearerAuth
// @Param id path string true "Target user ID"
// @Produce json
// @Success 200 {object} map[string]bool
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /friends/{id}/status [get]
func (h *FriendHandler) CheckFriendship(c *gin.Context) {
	targetID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	isFriend, err := h.friendUsecase.CheckFriendship(c.Request.Context(), user.ID, targetID)
	if err != nil {
		HandleUsecaseError(c, err, "フレンド関係の確認に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_friend": isFriend})
}

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
