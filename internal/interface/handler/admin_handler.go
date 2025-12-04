package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"
)

type AdminHandler struct {
	userUsecase usecase.UserUsecase
	userRepo    repository.UserRepository
}

func NewAdminHandler(uu usecase.UserUsecase, ur repository.UserRepository) *AdminHandler {
	return &AdminHandler{
		userUsecase: uu,
		userRepo:    ur,
	}
}

// GetAllUsers は全ユーザーを取得（管理者のみ）
func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	// デフォルトの取得件数を設定
	limit := 50
	offset := 0

	users, err := h.userUsecase.List(c.Request.Context(), limit, offset)
	if err != nil {
		log.Printf("Admin: Failed to list users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ユーザーの取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

// ユーザーロール更新リクエスト
type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// ユーザーのロールを更新（管理者のみ）
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ユーザーIDが必要です"})
		return
	}

	var req UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無効なリクエストです"})
		return
	}

	role := entity.UserRole(req.Role)
	if !role.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無効なロールです"})
		return
	}

	if err := h.userUsecase.UpdateRole(c.Request.Context(), userID, role); err != nil {
		log.Printf("Admin: Failed to update role for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ロールの更新に失敗しました"})
		return
	}

	// 更新後ユーザー情報取得
	user, err := h.userUsecase.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ユーザーの取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ロールを更新しました",
		"user":    user,
	})
}

// ユーザー詳細を取得（管理者のみ）
func (h *AdminHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ユーザーIDが必要です"})
		return
	}

	user, err := h.userUsecase.GetByID(c.Request.Context(), userID)
	if err != nil {
		if err == entity.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "ユーザーが見つかりません"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ユーザーの取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ユーザーを削除（管理者のみ）
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ユーザーIDが必要です"})
		return
	}

	// （管理者）自身を削除しようとしていないか
	currentUser, exists := c.Get("currentUser")
	if exists {
		if user, ok := currentUser.(*entity.User); ok && user.ID == userID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "自分自身を削除することはできません"})
			return
		}
	}

	if err := h.userUsecase.Delete(c.Request.Context(), userID); err != nil {
		log.Printf("Admin: Failed to delete user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ユーザーの削除に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ユーザーを削除しました"})
}

// ユーザールートを登録
func (h *AdminHandler) RegisterRoutes(r *gin.Engine) {
	admin := r.Group("/api/admin")
	admin.Use(middleware.ClerkAuthMiddleware())
	admin.Use(middleware.RequireAdmin(h.userRepo))
	{
		admin.GET("/users", h.GetAllUsers)
		admin.GET("/users/:id", h.GetUser)
		admin.PUT("/users/:id/role", h.UpdateUserRole)
		admin.DELETE("/users/:id", h.DeleteUser)
	}
}
