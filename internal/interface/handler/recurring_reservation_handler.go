package handler

import (
	"log"
	"net/http"
	"seat-management-backend/internal/domain/repository"
	"seat-management-backend/internal/middleware"
	"time"

	"github.com/gin-gonic/gin"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/usecase"
)

type RecurringReservationHandler struct {
	recurringReservationUsecase usecase.RecurringReservationUsecase
	userUsecase                 usecase.UserUsecase
	userRepo                    repository.UserRepository
}

func NewRecurringReservationHandler(rru usecase.RecurringReservationUsecase, uu usecase.UserUsecase, ur repository.UserRepository) *RecurringReservationHandler {
	return &RecurringReservationHandler{
		recurringReservationUsecase: rru,
		userUsecase:                 uu,
		userRepo:                    ur,
	}
}

type CreateRecurringReservationRequest struct {
	SeatID         string  `json:"seat_id" binding:"required"`
	DaysOfWeek     int     `json:"days_of_week" binding:"required"` // ビットフラグ 1-127
	StartTime      string  `json:"start_time" binding:"required"`   // "09:00:00" 形式
	EndTime        string  `json:"end_time" binding:"required"`     // "18:00:00" 形式
	ValidFrom      string  `json:"valid_from" binding:"required"`   // RFC3339形式
	ValidUntil     *string `json:"valid_until,omitempty"`           // RFC3339形式
	PrivacySetting string  `json:"privacy_setting,omitempty"`       // "public", "friends", "private"
	AutoExtend     bool    `json:"auto_extend,omitempty"`
}

type UpdateRecurringReservationRequest struct {
	DaysOfWeek     *int    `json:"days_of_week,omitempty"`
	StartTime      *string `json:"start_time,omitempty"`
	EndTime        *string `json:"end_time,omitempty"`
	ValidFrom      *string `json:"valid_from,omitempty"`
	ValidUntil     *string `json:"valid_until,omitempty"`
	PrivacySetting *string `json:"privacy_setting,omitempty"`
	AutoExtend     *bool   `json:"auto_extend,omitempty"`
	IsActive       *bool   `json:"is_active,omitempty"`
}

type GenerateReservationsRequest struct {
	StartDate string `json:"start_date" binding:"required"` // RFC3339形式
	EndDate   string `json:"end_date" binding:"required"`   // RFC3339形式
}

// CreateRecurringReservation godoc
// @Summary Create recurring reservation
// @Description Create a recurring weekly seat reservation pattern
// @Tags recurring-reservations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateRecurringReservationRequest true "Recurring reservation creation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Router /recurring-reservations [post]
func (h *RecurringReservationHandler) CreateRecurringReservation(c *gin.Context) {
	var req CreateRecurringReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	validFrom, err := time.Parse(time.RFC3339, req.ValidFrom)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "valid_from の日付形式が不正です", CodeValidation)
		return
	}

	var validUntil *time.Time
	if req.ValidUntil != nil {
		vut, err := time.Parse(time.RFC3339, *req.ValidUntil)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "valid_until の日付形式が不正です", CodeValidation)
			return
		}
		validUntil = &vut
	}

	privacySetting := entity.PrivacyPrivate
	if req.PrivacySetting != "" {
		privacySetting = entity.PrivacySetting(req.PrivacySetting)
	}

	recurringReservation, err := h.recurringReservationUsecase.CreateRecurringReservation(
		c.Request.Context(),
		user.ID,
		req.SeatID,
		req.DaysOfWeek,
		req.StartTime,
		req.EndTime,
		validFrom,
		validUntil,
		privacySetting,
		req.AutoExtend,
	)
	if err != nil {
		HandleUsecaseError(c, err, "定期予約の作成に失敗しました")
		return
	}

	c.JSON(http.StatusCreated, recurringReservation)
}

// GetRecurringReservationByID godoc
// @Summary Get recurring reservation by ID
// @Description Get recurring reservation details by ID
// @Tags recurring-reservations
// @Security BearerAuth
// @Param id path string true "Recurring reservation ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 404 {object} handler.ErrorResponse "Recurring reservation not found"
// @Router /recurring-reservations/{id} [get]
func (h *RecurringReservationHandler) GetRecurringReservationByID(c *gin.Context) {
	id := c.Param("id")

	recurringReservation, err := h.recurringReservationUsecase.GetRecurringReservationByID(c.Request.Context(), id)
	if err != nil {
		HandleUsecaseError(c, err, "定期予約の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, recurringReservation)
}

// GetMyRecurringReservations godoc
// @Summary Get my recurring reservations
// @Description Get all recurring reservations for the authenticated user
// @Tags recurring-reservations
// @Security BearerAuth
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /recurring-reservations/my [get]
func (h *RecurringReservationHandler) GetMyRecurringReservations(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	recurringReservations, err := h.recurringReservationUsecase.GetMyRecurringReservations(c.Request.Context(), user.ID)
	if err != nil {
		HandleUsecaseError(c, err, "定期予約一覧の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, recurringReservations)
}

// GetActiveRecurringReservations godoc
// @Summary Get active recurring reservations
// @Description Get active recurring reservations for the authenticated user
// @Tags recurring-reservations
// @Security BearerAuth
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /recurring-reservations/active [get]
func (h *RecurringReservationHandler) GetActiveRecurringReservations(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	recurringReservations, err := h.recurringReservationUsecase.GetActiveRecurringReservations(c.Request.Context(), user.ID)
	if err != nil {
		HandleUsecaseError(c, err, "アクティブな定期予約の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, recurringReservations)
}

// UpdateRecurringReservation godoc
// @Summary Update recurring reservation
// @Description Update a recurring reservation pattern
// @Tags recurring-reservations
// @Security BearerAuth
// @Accept json
// @Param id path string true "Recurring reservation ID"
// @Param request body UpdateRecurringReservationRequest true "Update request"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 404 {object} handler.ErrorResponse "Recurring reservation not found"
// @Router /recurring-reservations/{id} [put]
func (h *RecurringReservationHandler) UpdateRecurringReservation(c *gin.Context) {
	id := c.Param("id")

	var req UpdateRecurringReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	rr, err := h.recurringReservationUsecase.GetRecurringReservationByID(c.Request.Context(), id)
	if err != nil {
		HandleUsecaseError(c, err, "定期予約の取得に失敗しました")
		return
	}

	if rr.UserID != user.ID {
		RespondWithError(c, http.StatusForbidden, "権限がありません", CodeForbidden)
		return
	}

	updates := make(map[string]interface{})
	if req.DaysOfWeek != nil {
		updates["days_of_week"] = *req.DaysOfWeek
	}
	if req.StartTime != nil {
		updates["start_time"] = *req.StartTime
	}
	if req.EndTime != nil {
		updates["end_time"] = *req.EndTime
	}
	if req.ValidFrom != nil {
		vf, err := time.Parse(time.RFC3339, *req.ValidFrom)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "valid_from の日付形式が不正です", CodeValidation)
			return
		}
		updates["valid_from"] = vf
	}
	if req.ValidUntil != nil {
		vu, err := time.Parse(time.RFC3339, *req.ValidUntil)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "valid_until の日付形式が不正です", CodeValidation)
			return
		}
		updates["valid_until"] = vu
	}
	if req.PrivacySetting != nil {
		updates["privacy_setting"] = *req.PrivacySetting
	}
	if req.AutoExtend != nil {
		updates["auto_extend"] = *req.AutoExtend
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	updatedRr, err := h.recurringReservationUsecase.UpdateRecurringReservation(c.Request.Context(), id, updates)
	if err != nil {
		HandleUsecaseError(c, err, "定期予約の更新に失敗しました")
		return
	}

	c.JSON(http.StatusOK, updatedRr)
}

// DeleteRecurringReservation godoc
// @Summary Delete recurring reservation
// @Description Delete a recurring reservation pattern
// @Tags recurring-reservations
// @Security BearerAuth
// @Param id path string true "Recurring reservation ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 404 {object} handler.ErrorResponse "Recurring reservation not found"
// @Router /recurring-reservations/{id} [delete]
func (h *RecurringReservationHandler) DeleteRecurringReservation(c *gin.Context) {
	id := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	rr, err := h.recurringReservationUsecase.GetRecurringReservationByID(c.Request.Context(), id)
	if err != nil {
		HandleUsecaseError(c, err, "定期予約の取得に失敗しました")
		return
	}

	if rr.UserID != user.ID {
		RespondWithError(c, http.StatusForbidden, "権限がありません", CodeForbidden)
		return
	}

	if err := h.recurringReservationUsecase.DeleteRecurringReservation(c.Request.Context(), id); err != nil {
		HandleUsecaseError(c, err, "定期予約の削除に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "定期予約を削除しました"})
}

// DisableRecurringReservation godoc
// @Summary Disable recurring reservation
// @Description Disable a recurring reservation (soft disable without deletion)
// @Tags recurring-reservations
// @Security BearerAuth
// @Param id path string true "Recurring reservation ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 404 {object} handler.ErrorResponse "Recurring reservation not found"
// @Router /recurring-reservations/{id}/disable [post]
func (h *RecurringReservationHandler) DisableRecurringReservation(c *gin.Context) {
	id := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	rr, err := h.recurringReservationUsecase.GetRecurringReservationByID(c.Request.Context(), id)
	if err != nil {
		HandleUsecaseError(c, err, "定期予約の取得に失敗しました")
		return
	}

	if rr.UserID != user.ID {
		RespondWithError(c, http.StatusForbidden, "権限がありません", CodeForbidden)
		return
	}

	disabledRr, err := h.recurringReservationUsecase.DisableRecurringReservation(c.Request.Context(), id)
	if err != nil {
		HandleUsecaseError(c, err, "定期予約の無効化に失敗しました")
		return
	}

	c.JSON(http.StatusOK, disabledRr)
}

// GenerateReservations godoc
// @Summary Generate reservations from recurring patterns
// @Description Generate one-time reservations from recurring patterns for a date range (admin only)
// @Tags recurring-reservations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body GenerateReservationsRequest true "Generation request with date range"
// @Success 200 {object} map[string]string
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Router /admin/recurring-reservations/generate [post]
func (h *RecurringReservationHandler) GenerateReservations(c *gin.Context) {
	var req GenerateReservationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "start_date の日付形式が不正です", CodeValidation)
		return
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "end_date の日付形式が不正です", CodeValidation)
		return
	}

	if err := h.recurringReservationUsecase.GenerateReservationsForDateRange(
		c.Request.Context(),
		startDate,
		endDate,
	); err != nil {
		log.Printf("Failed to generate reservations by admin %s (email: %s): %v",
			user.ID, user.Email, err)
		HandleUsecaseError(c, err, "予約の生成に失敗しました")
		return
	}

	log.Printf("Admin %s (email: %s) successfully generated reservations from %s to %s", user.ID, user.Email, req.StartDate, req.EndDate)
	c.JSON(http.StatusOK, gin.H{"message": "予約を生成しました"})
}

func (h *RecurringReservationHandler) RegisterRoutes(r *gin.Engine) {
	recurring := r.Group("/api/recurring-reservations")
	recurring.Use(middleware.ClerkAuthMiddleware())
	{
		recurring.POST("", h.CreateRecurringReservation)
		recurring.GET("/my", h.GetMyRecurringReservations)
		recurring.GET("/active", h.GetActiveRecurringReservations)
		recurring.GET("/:id", h.GetRecurringReservationByID)
		recurring.PUT("/:id", h.UpdateRecurringReservation)
		recurring.DELETE("/:id", h.DeleteRecurringReservation)

		recurring.POST("/:id/disable", h.DisableRecurringReservation)
	}

	admin := r.Group("/api/admin/recurring-reservations")
	admin.Use(middleware.ClerkAuthMiddleware())
	admin.Use(middleware.RequireAdmin(h.userRepo))
	{
		admin.POST("/generate", h.GenerateReservations)
	}
}
