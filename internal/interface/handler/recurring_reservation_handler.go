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

// 定期予約作成
func (h *RecurringReservationHandler) CreateRecurringReservation(c *gin.Context) {
	var req CreateRecurringReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	// validFromをパース
	validFrom, err := time.Parse(time.RFC3339, req.ValidFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid_from の日付形式が不正です"})
		return
	}

	// validUntilをパース
	var validUntil *time.Time
	if req.ValidUntil != nil {
		vut, err := time.Parse(time.RFC3339, *req.ValidUntil)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "valid_until の日付形式が不正です"})
			return
		}
		validUntil = &vut
	}

	// プライバシー設定のデフォルト
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
		log.Printf("Failed to create recurring reservation: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, recurringReservation)
}

// 定期予約取得（ID指定）
func (h *RecurringReservationHandler) GetRecurringReservationByID(c *gin.Context) {
	id := c.Param("id")

	recurringReservation, err := h.recurringReservationUsecase.GetRecurringReservationByID(c.Request.Context(), id)
	if err != nil {
		log.Printf("Failed to get recurring reservation: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, recurringReservation)
}

// 自分の定期予約取得
func (h *RecurringReservationHandler) GetMyRecurringReservations(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	recurringReservations, err := h.recurringReservationUsecase.GetMyRecurringReservations(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Failed to get my recurring reservations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, recurringReservations)
}

// アクティブな定期予約取得
func (h *RecurringReservationHandler) GetActiveRecurringReservations(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	recurringReservations, err := h.recurringReservationUsecase.GetActiveRecurringReservations(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Failed to get active recurring reservations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, recurringReservations)
}

// 定期予約更新
func (h *RecurringReservationHandler) UpdateRecurringReservation(c *gin.Context) {
	id := c.Param("id")

	var req UpdateRecurringReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	// 自分の定期予約のみ更新可能
	rr, err := h.recurringReservationUsecase.GetRecurringReservationByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if rr.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "権限がありません"})
		return
	}

	// 更新マップを構築
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "valid_from の日付形式が不正です"})
			return
		}
		updates["valid_from"] = vf
	}
	if req.ValidUntil != nil {
		vu, err := time.Parse(time.RFC3339, *req.ValidUntil)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "valid_until の日付形式が不正です"})
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
		log.Printf("Failed to update recurring reservation: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedRr)
}

// 定期予約削除
func (h *RecurringReservationHandler) DeleteRecurringReservation(c *gin.Context) {
	id := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	// 自分の定期予約のみ削除可能
	rr, err := h.recurringReservationUsecase.GetRecurringReservationByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if rr.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "権限がありません"})
		return
	}

	if err := h.recurringReservationUsecase.DeleteRecurringReservation(c.Request.Context(), id); err != nil {
		log.Printf("Failed to delete recurring reservation: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "定期予約を削除しました"})
}

// 定期予約無効化
func (h *RecurringReservationHandler) DisableRecurringReservation(c *gin.Context) {
	id := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	// 自分の定期予約のみ無効化可能
	rr, err := h.recurringReservationUsecase.GetRecurringReservationByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if rr.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "権限がありません"})
		return
	}

	disabledRr, err := h.recurringReservationUsecase.DisableRecurringReservation(c.Request.Context(), id)
	if err != nil {
		log.Printf("Failed to disable recurring reservation: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, disabledRr)
}

// 予約生成（管理者用）
func (h *RecurringReservationHandler) GenerateReservations(c *gin.Context) {
	var req GenerateReservationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	// 日付をパース
	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date の日付形式が不正です"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_date の日付形式が不正です"})
		return
	}

	if err := h.recurringReservationUsecase.GenerateReservationsForDateRange(
		c.Request.Context(),
		startDate,
		endDate,
	); err != nil {
		log.Printf("Failed to generate reservations by admin %s (email: %s): %v",
			user.ID, user.Email, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
