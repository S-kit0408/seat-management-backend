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

type ReservationSettingsHandler struct {
	settingsUsecase usecase.ReservationSettingsUsecase
	userRepo        repository.UserRepository
}

type CreateSettingsRequest struct {
	CheckInMinutesBeforeStart   int    `json:"check_in_minutes_before_start" binding:"required,min=0"`
	CheckInGracePeriodMinutes   int    `json:"check_in_grace_period_minutes" binding:"required,min=0"`
	MinReservationMinutes       int    `json:"min_reservation_minutes" binding:"required,min=1"`
	MaxReservationMinutes       int    `json:"max_reservation_minutes" binding:"required,min=1"`
	MaxAdvanceBookingDays       int    `json:"max_advance_booking_days" binding:"required,min=1"`
	CancellationDeadlineMinutes int    `json:"cancellation_deadline_minutes" binding:"required,min=0"`
	MaxExtensionMinutes         int    `json:"max_extension_minutes" binding:"required,min=1"`
	MaxExtensionCount           int    `json:"max_extension_count" binding:"required,min=0"`
	AllowInstantReservation     bool   `json:"allow_instant_reservation"`
	Description                 string `json:"description,omitempty"`
	IsActive                    bool   `json:"is_active"`
}

type UpdateSettingsRequest struct {
	CheckInMinutesBeforeStart   *int    `json:"check_in_minutes_before_start,omitempty" binding:"omitempty,min=0"`
	CheckInGracePeriodMinutes   *int    `json:"check_in_grace_period_minutes,omitempty" binding:"omitempty,min=0"`
	MinReservationMinutes       *int    `json:"min_reservation_minutes,omitempty" binding:"omitempty,min=1"`
	MaxReservationMinutes       *int    `json:"max_reservation_minutes,omitempty" binding:"omitempty,min=1"`
	MaxAdvanceBookingDays       *int    `json:"max_advance_booking_days,omitempty" binding:"omitempty,min=1"`
	CancellationDeadlineMinutes *int    `json:"cancellation_deadline_minutes,omitempty" binding:"omitempty,min=0"`
	MaxExtensionMinutes         *int    `json:"max_extension_minutes,omitempty" binding:"omitempty,min=1"`
	MaxExtensionCount           *int    `json:"max_extension_count,omitempty" binding:"omitempty,min=0"`
	AllowInstantReservation     *bool   `json:"allow_instant_reservation,omitempty"`
	Description                 *string `json:"description,omitempty"`
}

func NewReservationSettingsHandler(
	su usecase.ReservationSettingsUsecase,
	ur repository.UserRepository,
) *ReservationSettingsHandler {
	return &ReservationSettingsHandler{
		settingsUsecase: su,
		userRepo:        ur,
	}
}

// GetActiveSettings - Available to all authenticated users
func (h *ReservationSettingsHandler) GetActiveSettings(c *gin.Context) {
	settings, err := h.settingsUsecase.GetOrCreateActiveSettings(c.Request.Context())
	if err != nil {
		log.Printf("Failed to get active settings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "設定の取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// CreateSettings - Admin only
func (h *ReservationSettingsHandler) CreateSettings(c *gin.Context) {
	var req CreateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings := &entity.ReservationSettings{
		CheckInMinutesBeforeStart:   req.CheckInMinutesBeforeStart,
		CheckInGracePeriodMinutes:   req.CheckInGracePeriodMinutes,
		MinReservationMinutes:       req.MinReservationMinutes,
		MaxReservationMinutes:       req.MaxReservationMinutes,
		MaxAdvanceBookingDays:       req.MaxAdvanceBookingDays,
		CancellationDeadlineMinutes: req.CancellationDeadlineMinutes,
		MaxExtensionMinutes:         req.MaxExtensionMinutes,
		MaxExtensionCount:           req.MaxExtensionCount,
		AllowInstantReservation:     req.AllowInstantReservation,
		Description:                 req.Description,
		IsActive:                    req.IsActive,
	}

	if err := h.settingsUsecase.Create(c.Request.Context(), settings); err != nil {
		log.Printf("Failed to create settings: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, settings)
}

// GetAllSettings - Admin only
func (h *ReservationSettingsHandler) GetAllSettings(c *gin.Context) {
	settings, err := h.settingsUsecase.GetAll(c.Request.Context())
	if err != nil {
		log.Printf("Failed to get all settings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "設定の取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"settings": settings,
		"count":    len(settings),
	})
}

// UpdateSettings - Admin only
func (h *ReservationSettingsHandler) UpdateSettings(c *gin.Context) {
	settingsID := c.Param("id")
	if settingsID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "設定IDが必要です"})
		return
	}

	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update entity
	settings := &entity.ReservationSettings{}
	if req.CheckInMinutesBeforeStart != nil {
		settings.CheckInMinutesBeforeStart = *req.CheckInMinutesBeforeStart
	}
	if req.CheckInGracePeriodMinutes != nil {
		settings.CheckInGracePeriodMinutes = *req.CheckInGracePeriodMinutes
	}
	if req.MinReservationMinutes != nil {
		settings.MinReservationMinutes = *req.MinReservationMinutes
	}
	if req.MaxReservationMinutes != nil {
		settings.MaxReservationMinutes = *req.MaxReservationMinutes
	}
	if req.MaxAdvanceBookingDays != nil {
		settings.MaxAdvanceBookingDays = *req.MaxAdvanceBookingDays
	}
	if req.CancellationDeadlineMinutes != nil {
		settings.CancellationDeadlineMinutes = *req.CancellationDeadlineMinutes
	}
	if req.MaxExtensionMinutes != nil {
		settings.MaxExtensionMinutes = *req.MaxExtensionMinutes
	}
	if req.MaxExtensionCount != nil {
		settings.MaxExtensionCount = *req.MaxExtensionCount
	}
	if req.AllowInstantReservation != nil {
		settings.AllowInstantReservation = *req.AllowInstantReservation
	}
	if req.Description != nil {
		settings.Description = *req.Description
	}

	if err := h.settingsUsecase.Update(c.Request.Context(), settingsID, settings); err != nil {
		log.Printf("Failed to update settings %s: %v", settingsID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedSettings, err := h.settingsUsecase.GetByID(c.Request.Context(), settingsID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新後の設定取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "設定を更新しました",
		"settings": updatedSettings,
	})
}

// ActivateSettings - Admin only
func (h *ReservationSettingsHandler) ActivateSettings(c *gin.Context) {
	settingsID := c.Param("id")
	if settingsID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "設定IDが必要です"})
		return
	}

	if err := h.settingsUsecase.Activate(c.Request.Context(), settingsID); err != nil {
		if err == entity.ErrReservationSettingsNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "設定が見つかりません"})
			return
		}
		log.Printf("Failed to activate settings %s: %v", settingsID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "設定の有効化に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "設定を有効化しました"})
}

// DeleteSettings - Admin only
func (h *ReservationSettingsHandler) DeleteSettings(c *gin.Context) {
	settingsID := c.Param("id")
	if settingsID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "設定IDが必要です"})
		return
	}

	if err := h.settingsUsecase.Delete(c.Request.Context(), settingsID); err != nil {
		if err == entity.ErrReservationSettingsNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "設定が見つかりません"})
			return
		}
		log.Printf("Failed to delete settings %s: %v", settingsID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "設定を削除しました"})
}

// RegisterRoutes registers all routes for reservation settings
func (h *ReservationSettingsHandler) RegisterRoutes(r *gin.Engine) {
	// User endpoint (read-only access to active settings)
	settings := r.Group("/api/settings/reservation")
	settings.Use(middleware.ClerkAuthMiddleware())
	{
		settings.GET("", h.GetActiveSettings)
	}

	// Admin endpoints (full CRUD)
	admin := r.Group("/api/admin/settings/reservation")
	admin.Use(middleware.ClerkAuthMiddleware())
	admin.Use(middleware.RequireAdmin(h.userRepo))
	{
		admin.POST("", h.CreateSettings)
		admin.GET("", h.GetAllSettings)
		admin.PUT("/:id", h.UpdateSettings)
		admin.POST("/:id/activate", h.ActivateSettings)
		admin.DELETE("/:id", h.DeleteSettings)
	}
}
