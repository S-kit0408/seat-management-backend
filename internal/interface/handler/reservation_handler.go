package handler

import (
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

// CreateReservation godoc
// @Summary Create scheduled reservation
// @Description Create a scheduled seat reservation with start and end times
// @Tags reservations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateReservationRequest true "Reservation creation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Router /reservations [post]
func (h *ReservationHandler) CreateReservation(c *gin.Context) {
	var req CreateReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	var privacySetting *entity.PrivacySetting
	var ps entity.PrivacySetting

	if req.PrivacySetting != "" {
		ps = entity.PrivacySetting(req.PrivacySetting)
		if !ps.IsValid() {
			RespondWithValidationError(c, err)
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
		HandleUsecaseError(c, err, "予約の作成に失敗しました")
		return
	}

	c.JSON(http.StatusCreated, reservation)
}

// CreateInstantReservation godoc
// @Summary Create instant reservation
// @Description Create an instant seat reservation for a specified duration
// @Tags reservations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateInstantReservationRequest true "Instant reservation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Router /reservations/instant [post]
func (h *ReservationHandler) CreateInstantReservation(c *gin.Context) {
	var req CreateInstantReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	var privacySetting *entity.PrivacySetting
	var ps entity.PrivacySetting

	if req.PrivacySetting != "" {
		ps = entity.PrivacySetting(req.PrivacySetting)
		if !ps.IsValid() {
			HandleUsecaseError(c, entity.ErrInvalidReservationType, "無効なプライバシー設定です")
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
		HandleUsecaseError(c, err, "インスタント予約の作成に失敗しました")
		return
	}

	c.JSON(http.StatusCreated, reservation)
}

// GetReservationByID godoc
// @Summary Get reservation by ID
// @Description Get reservation details by ID
// @Tags reservations
// @Security BearerAuth
// @Param id path string true "Reservation ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 404 {object} handler.ErrorResponse "Reservation not found"
// @Router /reservations/{id} [get]
func (h *ReservationHandler) GetReservationByID(c *gin.Context) {
	reservationID := c.Param("id")

	reservation, err := h.reservationUsecase.GetReservationByID(c.Request.Context(), reservationID)
	if err != nil {
		HandleUsecaseError(c, err, "予約の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, reservation)
}

// GetMyReservations godoc
// @Summary Get my reservations
// @Description Get all reservations for the authenticated user
// @Tags reservations
// @Security BearerAuth
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /reservations/my [get]
func (h *ReservationHandler) GetMyReservations(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	reservations, err := h.reservationUsecase.GetMyReservations(c.Request.Context(), user.ID)
	if err != nil {
		HandleUsecaseError(c, err, "予約一覧の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, reservations)
}

// GetActiveReservations godoc
// @Summary Get active reservations
// @Description Get active (reserved/in-use) reservations for the authenticated user
// @Tags reservations
// @Security BearerAuth
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /reservations/active [get]
func (h *ReservationHandler) GetActiveReservations(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	reservations, err := h.reservationUsecase.GetActiveReservations(c.Request.Context(), user.ID)
	if err != nil {
		HandleUsecaseError(c, err, "アクティブな予約の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, reservations)
}

// GetVisibleReservations godoc
// @Summary Get visible reservations
// @Description Get reservations visible to the authenticated user (privacy-aware)
// @Tags reservations
// @Security BearerAuth
// @Produce json
// @Param start_time query string false "Start time (RFC3339 format)"
// @Param end_time query string false "End time (RFC3339 format)"
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /reservations/visible [get]
func (h *ReservationHandler) GetVisibleReservations(c *gin.Context) {
	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

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
		HandleUsecaseError(c, err, "可視予約の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, reservations)
}

// CheckIn godoc
// @Summary Check in to reservation
// @Description Check in to a reserved seat
// @Tags reservations
// @Security BearerAuth
// @Param id path string true "Reservation ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 404 {object} handler.ErrorResponse "Reservation not found"
// @Router /reservations/{id}/checkin [post]
func (h *ReservationHandler) CheckIn(c *gin.Context) {
	reservationID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	reservation, err := h.reservationUsecase.GetReservationByID(c.Request.Context(), reservationID)
	if err != nil {
		HandleUsecaseError(c, err, "予約の取得に失敗しました")
		return
	}

	if reservation.UserID != user.ID {
		RespondWithError(c, http.StatusForbidden, "権限がありません", CodeForbidden)
		return
	}

	reservation, err = h.reservationUsecase.CheckIn(c.Request.Context(), reservationID)
	if err != nil {
		HandleUsecaseError(c, err, "チェックインに失敗しました")
		return
	}

	c.JSON(http.StatusOK, reservation)
}

// CheckOut godoc
// @Summary Check out from reservation
// @Description Check out from a seat reservation
// @Tags reservations
// @Security BearerAuth
// @Param id path string true "Reservation ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 404 {object} handler.ErrorResponse "Reservation not found"
// @Router /reservations/{id}/checkout [post]
func (h *ReservationHandler) CheckOut(c *gin.Context) {
	reservationID := c.Param("id")

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	reservation, err := h.reservationUsecase.GetReservationByID(c.Request.Context(), reservationID)
	if err != nil {
		HandleUsecaseError(c, err, "予約の取得に失敗しました")
		return
	}

	if reservation.UserID != user.ID {
		RespondWithError(c, http.StatusForbidden, "権限がありません", CodeForbidden)
		return
	}

	reservation, err = h.reservationUsecase.CheckOut(c.Request.Context(), reservationID)
	if err != nil {
		HandleUsecaseError(c, err, "チェックアウトに失敗しました")
		return
	}

	c.JSON(http.StatusOK, reservation)
}

// CancelReservation godoc
// @Summary Cancel reservation
// @Description Cancel a reservation with optional reason
// @Tags reservations
// @Security BearerAuth
// @Accept json
// @Param id path string true "Reservation ID"
// @Param request body CancelReservationRequest true "Cancellation details"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 404 {object} handler.ErrorResponse "Reservation not found"
// @Router /reservations/{id}/cancel [post]
func (h *ReservationHandler) CancelReservation(c *gin.Context) {
	reservationID := c.Param("id")

	var req CancelReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	reservation, err := h.reservationUsecase.GetReservationByID(c.Request.Context(), reservationID)
	if err != nil {
		HandleUsecaseError(c, err, "予約の取得に失敗しました")
		return
	}

	if reservation.UserID != user.ID {
		RespondWithError(c, http.StatusForbidden, "権限がありません", CodeForbidden)
		return
	}

	reservation, err = h.reservationUsecase.CancelReservation(
		c.Request.Context(),
		reservationID,
		req.Reason,
	)
	if err != nil {
		HandleUsecaseError(c, err, "予約のキャンセルに失敗しました")
		return
	}

	c.JSON(http.StatusOK, reservation)
}

// ExtendReservation godoc
// @Summary Extend reservation
// @Description Extend a reservation by additional minutes
// @Tags reservations
// @Security BearerAuth
// @Accept json
// @Param id path string true "Reservation ID"
// @Param request body ExtendReservationRequest true "Extension details"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 404 {object} handler.ErrorResponse "Reservation not found"
// @Router /reservations/{id}/extend [post]
func (h *ReservationHandler) ExtendReservation(c *gin.Context) {
	reservationID := c.Param("id")

	var req ExtendReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	user, err := GetAuthenticatedUser(c, h.userUsecase)
	if err != nil {
		RespondWithUnauthorized(c)
		return
	}

	reservation, err := h.reservationUsecase.GetReservationByID(c.Request.Context(), reservationID)
	if err != nil {
		HandleUsecaseError(c, err, "予約の取得に失敗しました")
		return
	}

	if reservation.UserID != user.ID {
		RespondWithError(c, http.StatusForbidden, "権限がありません", CodeForbidden)
		return
	}

	reservation, err = h.reservationUsecase.ExtendReservation(
		c.Request.Context(),
		reservationID,
		req.AdditionalMinutes,
	)
	if err != nil {
		HandleUsecaseError(c, err, "予約の延長に失敗しました")
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
