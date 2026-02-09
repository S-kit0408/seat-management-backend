package handler

import (
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

// GetAllUsers godoc
// @Summary Get all users
// @Description Get list of all users (admin only)
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /admin/users [get]
func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	limit := 50
	offset := 0

	users, err := h.userUsecase.List(c.Request.Context(), limit, offset)
	if err != nil {
		HandleUsecaseError(c, err, "ユーザー一覧の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// UpdateUserRole godoc
// @Summary Update user role
// @Description Update the role of a user (admin only)
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Param id path string true "User ID"
// @Param request body UpdateUserRoleRequest true "Role update request"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /admin/users/{id}/role [put]
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		RespondWithError(c, http.StatusBadRequest, "ユーザーIDが必要です", CodeValidation)
		return
	}

	var req UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	role := entity.UserRole(req.Role)
	if !role.IsValid() {
		HandleUsecaseError(c, entity.ErrInvalidRole, "無効なロールです")
		return
	}

	if err := h.userUsecase.UpdateRole(c.Request.Context(), userID, role); err != nil {
		HandleUsecaseError(c, err, "ロールの更新に失敗しました")
		return
	}

	user, err := h.userUsecase.GetByID(c.Request.Context(), userID)
	if err != nil {
		HandleUsecaseError(c, err, "ユーザーの取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ロールを更新しました",
		"user":    user,
	})
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get user details by ID (admin only)
// @Tags admin
// @Security BearerAuth
// @Param id path string true "User ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 404 {object} handler.ErrorResponse "User not found"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /admin/users/{id} [get]
func (h *AdminHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		RespondWithError(c, http.StatusBadRequest, "ユーザーIDが必要です", CodeValidation)
		return
	}

	user, err := h.userUsecase.GetByID(c.Request.Context(), userID)
	if err != nil {
		HandleUsecaseError(c, err, "ユーザーの取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser godoc
// @Summary Delete user
// @Description Delete a user (soft delete) (admin only, cannot delete self)
// @Tags admin
// @Security BearerAuth
// @Param id path string true "User ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		RespondWithError(c, http.StatusBadRequest, "ユーザーIDが必要です", CodeValidation)
		return
	}

	currentUser, exists := c.Get("currentUser")
	if exists {
		if user, ok := currentUser.(*entity.User); ok && user.ID == userID {
			RespondWithError(c, http.StatusForbidden, "自分自身を削除することはできません", CodeForbidden)
			return
		}
	}

	if err := h.userUsecase.Delete(c.Request.Context(), userID); err != nil {
		HandleUsecaseError(c, err, "ユーザーの削除に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ユーザーを削除しました"})
}

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
