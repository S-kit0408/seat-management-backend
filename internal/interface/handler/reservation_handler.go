package handler

import (
	"log"
	"net/http"
	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/middleware"
	"time"

	"github.com/gin-gonic/gin"

	"seat-management-backend/internal/usecase"
)

type ReservationHandler struct {
	reservationUsecase usecase.ReservationUsecase
	userUsecase        usecase.UserUsecase
}

func NewReservationHandler(ru usecase.ReservationUsecase, uu usecase.UserUsecase) *ReservationHandler {
	return &ReservationHandler{
		reservationUsecase: ru,
		userUsecase:        uu,
	}
}

type CreateReservationRequest struct {
	SeatID         string    `json:"seat_id" binding:"required"`
	StartTime      time.Time `json:"start_time" binding:"required"`
	EndTime        time.Time `json:"end_time" binding:"required"`
	PrivacySetting string    `json:"privacy_setting"`
	Notes          string    `json:"notes,omitempty"`
}

type CreateInstantReservationRequest struct {
	SeatID          string `json:"seat_id" binding:"required"`
	DurationMinutes int    `json:"duration_minutes" binding:"required"`
	PrivacySetting  string `json:"privacy_setting"`
}

type ExtendReservationRequest struct {
	AdditionalMinutes int `json:"additional_minutes" binding:"required"`
}

type CancelReservationRequest struct {
	Reason string `json:"reason,omitempty"`
}

// 予約作成
func (h *ReservationHandler) CreateReservation(c *gin.Context) {
	var req CreateReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	// プライバシー設定をバリデーション
	var privacySetting *entity.PrivacySetting
	var ps entity.PrivacySetting

	if req.PrivacySetting != "" {
		ps = entity.PrivacySetting(req.PrivacySetting)
		if !ps.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "無効なプライバシー設定です"})
			return
		}
		privacySetting = &ps
	}

	reservation, err := h.reservationUsecase.CreateReservation(
		c.Request.Context(),
		user.ID,
		req.SeatID,
		req.StartTime,
		req.EndTime,
		privacySetting,
	)
	if err != nil {
		log.Printf("Failed to create reservation: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, reservation)
}

// その場の予約作成
func (h *ReservationHandler) CreateInstantReservation(c *gin.Context) {
	var req CreateInstantReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	// プライバシー設定をバリデーション
	var privacySetting *entity.PrivacySetting
	var ps entity.PrivacySetting

	if req.PrivacySetting != "" {
		ps = entity.PrivacySetting(req.PrivacySetting)
		if !ps.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "無効なプライバシー設定です"})
			return
		}
		privacySetting = &ps
	}

	reservation, err := h.reservationUsecase.CreateInstantReservation(
		c.Request.Context(),
		user.ID,
		req.SeatID,
		req.DurationMinutes,
		privacySetting,
	)
	if err != nil {
		log.Printf("Failed to create instant reservation: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, reservation)
}

// 予約取得（ID指定）
func (h *ReservationHandler) GetReservationByID(c *gin.Context) {
	reservationID := c.Param("id")

	reservation, err := h.reservationUsecase.GetReservationByID(c.Request.Context(), reservationID)
	if err != nil {
		log.Printf("Failed to get reservation: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reservation)
}

// 自分の予約取得
func (h *ReservationHandler) GetMyReservations(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	reservations, err := h.reservationUsecase.GetMyReservations(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Failed to get my reservations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reservations)
}

// アクティブな予約取得
func (h *ReservationHandler) GetActiveReservations(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	reservations, err := h.reservationUsecase.GetActiveReservations(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Failed to get active reservations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reservations)
}

// 見える予約取得（プライバシー対応）
func (h *ReservationHandler) GetVisibleReservations(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	// クエリパラメータで時間範囲を指定可能
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	var startTime, endTime *time.Time
	if startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	reservations, err := h.reservationUsecase.GetVisibleReservations(
		c.Request.Context(),
		user.ID,
		startTime,
		endTime,
	)
	if err != nil {
		log.Printf("Failed to get visible reservations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reservations)
}

// チェックイン
func (h *ReservationHandler) CheckIn(c *gin.Context) {
	reservationID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	reservation, err := h.reservationUsecase.GetReservationByID(c.Request.Context(), reservationID)
	if err != nil {
		log.Printf("Failed to get reservation: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 自分の予約のみチェックイン可能
	if reservation.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "権限がありません"})
		return
	}

	reservation, err = h.reservationUsecase.CheckIn(c.Request.Context(), reservationID)
	if err != nil {
		log.Printf("Failed to check in: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reservation)
}

// チェックアウト
func (h *ReservationHandler) CheckOut(c *gin.Context) {
	reservationID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	reservation, err := h.reservationUsecase.GetReservationByID(c.Request.Context(), reservationID)
	if err != nil {
		log.Printf("Failed to get reservation: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 自分の予約のみチェックアウト可能
	if reservation.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "権限がありません"})
		return
	}

	reservation, err = h.reservationUsecase.CheckOut(c.Request.Context(), reservationID)
	if err != nil {
		log.Printf("Failed to check out: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reservation)
}

// キャンセル
func (h *ReservationHandler) CancelReservation(c *gin.Context) {
	reservationID := c.Param("id")

	var req CancelReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	reservation, err := h.reservationUsecase.GetReservationByID(c.Request.Context(), reservationID)
	if err != nil {
		log.Printf("Failed to get reservation: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if reservation.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "権限がありません"})
		return
	}

	reservation, err = h.reservationUsecase.CancelReservation(
		c.Request.Context(),
		reservationID,
		req.Reason,
	)
	if err != nil {
		log.Printf("Failed to cancel reservation: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reservation)
}

// 延長
func (h *ReservationHandler) ExtendReservation(c *gin.Context) {
	reservationID := c.Param("id")

	var req ExtendReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "認証されていません"})
		return
	}

	reservation, err := h.reservationUsecase.GetReservationByID(c.Request.Context(), reservationID)
	if err != nil {
		log.Printf("Failed to get reservation: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if reservation.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "権限がありません"})
		return
	}

	reservation, err = h.reservationUsecase.ExtendReservation(
		c.Request.Context(),
		reservationID,
		req.AdditionalMinutes,
	)
	if err != nil {
		log.Printf("Failed to extend reservation: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reservation)
}

func (h *ReservationHandler) RegisterRoutes(r *gin.Engine) {
	reservation := r.Group("/api/reservations")
	reservation.Use(middleware.ClerkAuthMiddleware())
	{
		reservation.POST("", h.CreateReservation)

		reservation.GET("/my", h.GetMyReservations)
		reservation.GET("/active", h.GetActiveReservations)
		reservation.GET("/visible", h.GetVisibleReservations)

		reservation.POST("/instant", h.CreateInstantReservation)
		reservation.POST("/:id/checkin", h.CheckIn)
		reservation.POST("/:id/checkout", h.CheckOut)
		reservation.POST("/:id/cancel", h.CancelReservation)
		reservation.POST("/:id/extend", h.ExtendReservation)

		reservation.GET("/:id", h.GetReservationByID)
	}
}
