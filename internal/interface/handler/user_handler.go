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

// UserResponse represents user data for API responses
// @Description User profile response
type UserResponse struct {
	ID                    string  `json:"id"`
	ClerkUserID           string  `json:"clerk_user_id"`
	Email                 string  `json:"email"`
	Name                  string  `json:"name"`
	AvatarURL             *string `json:"avatar_url,omitempty"`
	PrimaryAuthProvider   string  `json:"primary_auth_provider"`
	DefaultPrivacySetting string  `json:"default_privacy_setting"`
	Role                  string  `json:"role"`
}

func NewUserHandler(uu usecase.UserUsecase) *UserHandler {
	return &UserHandler{
		userUsecase: uu,
	}
}

// GetMe godoc
// @Summary Get current user profile
// @Description Get authenticated user's profile information
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} UserResponse
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Router /users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	if err := h.userUsecase.UpdateLastLogin(c.Request.Context(), user.ID); err != nil {
		log.Printf("Failed to update last login for user %s: %v", user.ID, err)
	}

	c.JSON(http.StatusOK, user)
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update current user's profile information
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body UpdateProfileRequest true "Update profile request"
// @Success 200 {object} UserResponse
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /users/me [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}
	if req.DefaultPrivacySetting != nil {
		privacySetting := entity.PrivacySetting(*req.DefaultPrivacySetting)
		if !privacySetting.IsValid() {
			HandleUsecaseError(c, entity.ErrInvalidReservationType, "無効なプライバシー設定です")
			return
		}
		user.DefaultPrivacySetting = privacySetting
	}

	if err := h.userUsecase.Update(c.Request.Context(), user); err != nil {
		HandleUsecaseError(c, err, "プロフィールの更新に失敗しました")
		return
	}

	c.JSON(http.StatusOK, user)
}

// SearchUsers godoc
// @Summary Search users by name
// @Description Search for users by their name
// @Tags users
// @Security BearerAuth
// @Produce json
// @Param name query string true "User name to search for"
// @Success 200 {array} UserResponse
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /users/search [get]
func (h *UserHandler) SearchUsers(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		RespondWithError(c, http.StatusBadRequest, "検索キーワードが必要です", CodeValidation)
		return
	}

	users, err := h.userUsecase.SearchByName(c.Request.Context(), name)
	if err != nil {
		HandleUsecaseError(c, err, "ユーザーの検索に失敗しました")
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *UserHandler) RegisterRoutes(r *gin.Engine) {
	users := r.Group("/api/users")
	users.Use(middleware.ClerkAuthMiddleware())
	{
		users.GET("/me", h.GetMe)
		users.PUT("/me", h.UpdateProfile)
		users.GET("/search", h.SearchUsers)
	}
}
