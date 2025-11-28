package handler

import (
	"log"
	"net/http"

	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userUsecase usecase.UserUsecase
}

func NewUserHandler(uu usecase.UserUsecase) *UserHandler {
	return &UserHandler{
		userUsecase: uu,
	}
}

// GetMe は現在のユーザー情報を取得
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

// RegisterRoutes はユーザールートを登録
func (h *UserHandler) RegisterRoutes(r *gin.Engine) {
	users := r.Group("/api/users")
	users.Use(middleware.ClerkAuthMiddleware())
	{
		users.GET("/me", h.GetMe)
	}
}
