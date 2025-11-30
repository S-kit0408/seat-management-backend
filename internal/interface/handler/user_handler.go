package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"
)

type UserHandler struct {
	userUsecase usecase.UserUsecase
}

type UpdateProfileRequest struct {
	Name                  *string `json:"name,omitempty"`
	AvatarURL             *string `json:"avatar_url,omitempty"`
	DefaultPrivacySetting *string `json:"default_privacy_setting,omitempty"`
}

func NewUserHandler(uu usecase.UserUsecase) *UserHandler {
	return &UserHandler{
		userUsecase: uu,
	}
}

// ユーザー情報を取得
func (h *UserHandler) GetMe(c *gin.Context) {
	clerkUserID, err := middleware.GetClerkUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	user, err := h.userUsecase.GetByClerkUserID(c.Request.Context(), clerkUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ユーザーが見つかりません"})
		return
	}

	// 最終ログイン時刻を更新
	if err := h.userUsecase.UpdateLastLogin(c.Request.Context(), user.ID); err != nil {
		log.Println("Failed to update last login for user :", user.ID, err)
	}

	c.JSON(http.StatusOK, user)
}

// ユーザー情報の更新
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	clerkUserID, err := middleware.GetClerkUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userUsecase.GetByClerkUserID(c.Request.Context(), clerkUserID)
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 変更フィールドのみ適用
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}
	if req.DefaultPrivacySetting != nil {
		user.DefaultPrivacySetting = entity.PrivacySetting(*req.DefaultPrivacySetting)
	}

	if err := h.userUsecase.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusOK, user)
}

// RegisterRoutes はユーザールートを登録
func (h *UserHandler) RegisterRoutes(r *gin.Engine) {
	users := r.Group("/api/users")
	users.Use(middleware.ClerkAuthMiddleware())
	{
		users.GET("/me", h.GetMe)
		users.PUT("/me", h.UpdateProfile)
	}
}
