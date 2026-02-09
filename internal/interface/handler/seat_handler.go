package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"
)

type SeatHandler struct {
	seatUsecase usecase.SeatUsecase
	userRepo    repository.UserRepository // 管理者認証用
}

type CreateSeatRequest struct {
	SeatNumber    string                 `json:"seat_number" binding:"required"`
	Description   string                 `json:"description,omitempty"`
	PositionX     float64                `json:"position_x"`
	PositionY     float64                `json:"position_y"`
	RotationAngle int                    `json:"rotation_angle"`
	Width         float64                `json:"width"`
	Height        float64                `json:"height"`
	Shape         string                 `json:"shape,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	FloorID       *string                `json:"floor_id,omitempty"`
	SpaceID       *string                `json:"space_id,omitempty"`
	IsActive      bool                   `json:"is_active"`
}

type UpdateSeatRequest struct {
	SeatNumber    *string                `json:"seat_number,omitempty"`
	Description   *string                `json:"description,omitempty"`
	PositionX     *float64               `json:"position_x,omitempty"`
	PositionY     *float64               `json:"position_y,omitempty"`
	RotationAngle *int                   `json:"rotation_angle,omitempty"`
	Width         *float64               `json:"width,omitempty"`
	Height        *float64               `json:"height,omitempty"`
	Shape         *string                `json:"shape,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	FloorID       *string                `json:"floor_id,omitempty"`
	SpaceID       *string                `json:"space_id,omitempty"`
	IsActive      *bool                  `json:"is_active,omitempty"`
}

func NewSeatHandler(su usecase.SeatUsecase, ur repository.UserRepository) *SeatHandler {
	return &SeatHandler{
		seatUsecase: su,
		userRepo:    ur,
	}
}

// CreateSeat godoc
// @Summary Create seat
// @Description Create a new seat (admin only)
// @Tags seats
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateSeatRequest true "Seat creation request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Router /admin/seats [post]
func (h *SeatHandler) CreateSeat(c *gin.Context) {
	var req CreateSeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	seat := &entity.Seat{
		SeatNumber:    req.SeatNumber,
		Description:   req.Description,
		PositionX:     req.PositionX,
		PositionY:     req.PositionY,
		RotationAngle: req.RotationAngle,
		Width:         req.Width,
		Height:        req.Height,
		Shape:         entity.SeatShape(req.Shape),
		FloorID:       req.FloorID,
		SpaceID:       req.SpaceID,
		IsActive:      req.IsActive,
	}

	if req.Attributes != nil {
		attrBytes, err := json.Marshal(req.Attributes)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "属性の変換に失敗しました", CodeValidation)
			return
		}
		seat.Attributes = attrBytes
	}

	if err := h.seatUsecase.Create(c.Request.Context(), seat); err != nil {
		HandleUsecaseError(c, err, "座席の作成に失敗しました")
		return
	}

	c.JSON(http.StatusOK, seat)
}

// GetSeats godoc
// @Summary Get all seats
// @Description Get list of seats (all authenticated users)
// @Tags seats
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit (default: 100)"
// @Param offset query int false "Offset (default: 0)"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /seats [get]
func (h *SeatHandler) GetSeats(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	seats, err := h.seatUsecase.List(c.Request.Context(), limit, offset)
	if err != nil {
		HandleUsecaseError(c, err, "座席の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"seats": seats,
		"count": len(seats),
	})
}

// GetActiveSeats godoc
// @Summary Get active seats
// @Description Get only active seats with floor assignments (all authenticated users)
// @Tags seats
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /seats/active [get]
func (h *SeatHandler) GetActiveSeats(c *gin.Context) {
	seats, err := h.seatUsecase.GetActiveSeatsWithFloor(c.Request.Context())
	if err != nil {
		HandleUsecaseError(c, err, "アクティブな座席の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"seats": seats,
		"count": len(seats),
	})
}

// GetUnassignedSeats godoc
// @Summary Get unassigned seats
// @Description Get seats without floor assignments (admin only)
// @Tags seats
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /admin/seats/unassigned [get]
func (h *SeatHandler) GetUnassignedSeats(c *gin.Context) {
	seats, err := h.seatUsecase.GetUnassignedSeats(c.Request.Context())
	if err != nil {
		HandleUsecaseError(c, err, "未割り当て座席の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"seats": seats,
		"count": len(seats),
	})
}

// GetSeat godoc
// @Summary Get seat by ID
// @Description Get seat details by ID (all authenticated users)
// @Tags seats
// @Security BearerAuth
// @Param id path string true "Seat ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 404 {object} handler.ErrorResponse "Seat not found"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /seats/{id} [get]
func (h *SeatHandler) GetSeat(c *gin.Context) {
	seatID := c.Param("id")
	if seatID == "" {
		RespondWithError(c, http.StatusBadRequest, "座席IDが必要です", CodeValidation)
		return
	}

	seat, err := h.seatUsecase.GetByID(c.Request.Context(), seatID)
	if err != nil {
		HandleUsecaseError(c, err, "座席の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, seat)
}

// GetSeatByNumber godoc
// @Summary Search seat by number
// @Description Get seat by seat number (all authenticated users)
// @Tags seats
// @Security BearerAuth
// @Produce json
// @Param seat_number query string true "Seat number"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 404 {object} handler.ErrorResponse "Seat not found"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /seats/search [get]
func (h *SeatHandler) GetSeatByNumber(c *gin.Context) {
	seatNumber := c.Query("seat_number")
	if seatNumber == "" {
		RespondWithError(c, http.StatusBadRequest, "座席番号が必要です", CodeValidation)
		return
	}

	seat, err := h.seatUsecase.GetBySeatNumber(c.Request.Context(), seatNumber)
	if err != nil {
		HandleUsecaseError(c, err, "座席の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, seat)
}

// UpdateSeat godoc
// @Summary Update seat
// @Description Update seat details (admin only)
// @Tags seats
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Seat ID"
// @Param request body UpdateSeatRequest true "Seat update request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /admin/seats/{id} [put]
func (h *SeatHandler) UpdateSeat(c *gin.Context) {
	seatID := c.Param("id")
	if seatID == "" {
		RespondWithError(c, http.StatusBadRequest, "座席IDが必要です", CodeValidation)
		return
	}

	var req UpdateSeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	seat := &entity.Seat{}

	if req.SeatNumber != nil {
		seat.SeatNumber = *req.SeatNumber
	}
	if req.Description != nil {
		seat.Description = *req.Description
	}
	if req.PositionX != nil {
		seat.PositionX = *req.PositionX
	}
	if req.PositionY != nil {
		seat.PositionY = *req.PositionY
	}
	if req.RotationAngle != nil {
		seat.RotationAngle = *req.RotationAngle
	}
	if req.Width != nil {
		seat.Width = *req.Width
	}
	if req.Height != nil {
		seat.Height = *req.Height
	}
	if req.Shape != nil {
		seat.Shape = entity.SeatShape(*req.Shape)
	}
	if req.Attributes != nil {
		attrBytes, err := json.Marshal(req.Attributes)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "属性の変換に失敗しました", CodeValidation)
			return
		}
		seat.Attributes = attrBytes
	}
	if req.FloorID != nil {
		seat.FloorID = req.FloorID
	}
	if req.SpaceID != nil {
		seat.SpaceID = req.SpaceID
	}
	if req.IsActive != nil {
		seat.IsActive = *req.IsActive
	}

	if err := h.seatUsecase.Update(c.Request.Context(), seatID, seat); err != nil {
		HandleUsecaseError(c, err, "座席の更新に失敗しました")
		return
	}

	updatedSeat, err := h.seatUsecase.GetByID(c.Request.Context(), seatID)
	if err != nil {
		HandleUsecaseError(c, err, "更新後の座席取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "座席を更新しました",
		"seat":    updatedSeat,
	})
}

// DeleteSeat godoc
// @Summary Delete seat
// @Description Delete a seat (admin only)
// @Tags seats
// @Security BearerAuth
// @Param id path string true "Seat ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 404 {object} handler.ErrorResponse "Seat not found"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /admin/seats/{id} [delete]
func (h *SeatHandler) DeleteSeat(c *gin.Context) {
	seatID := c.Param("id")
	if seatID == "" {
		RespondWithError(c, http.StatusBadRequest, "座席IDが必要です", CodeValidation)
		return
	}

	if err := h.seatUsecase.Delete(c.Request.Context(), seatID); err != nil {
		HandleUsecaseError(c, err, "座席の削除に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "座席を削除しました"})
}

// GetSeatsByFloor godoc
// @Summary Get seats by floor
// @Description Get seats assigned to a specific floor (all authenticated users)
// @Tags seats
// @Security BearerAuth
// @Produce json
// @Param floor_id query string true "Floor ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /seats/floor [get]
func (h *SeatHandler) GetSeatsByFloor(c *gin.Context) {
	floorID := c.Query("floor_id")
	if floorID == "" {
		RespondWithError(c, http.StatusBadRequest, "フロアIDが必要です", CodeValidation)
		return
	}

	seats, err := h.seatUsecase.GetSeatsByFloorID(c.Request.Context(), floorID)
	if err != nil {
		HandleUsecaseError(c, err, "座席の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"seats": seats,
		"count": len(seats),
	})
}

// GetSeatsBySpace godoc
// @Summary Get seats by space
// @Description Get seats assigned to a specific space (all authenticated users)
// @Tags seats
// @Security BearerAuth
// @Produce json
// @Param space_id query string true "Space ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /seats/space [get]
func (h *SeatHandler) GetSeatsBySpace(c *gin.Context) {
	spaceID := c.Query("space_id")
	if spaceID == "" {
		RespondWithError(c, http.StatusBadRequest, "スペースIDが必要です", CodeValidation)
		return
	}

	seats, err := h.seatUsecase.GetSeatsBySpaceID(c.Request.Context(), spaceID)
	if err != nil {
		HandleUsecaseError(c, err, "座席の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"seats": seats,
		"count": len(seats),
	})
}

func (h *SeatHandler) RegisterRoutes(r *gin.Engine) {
	seats := r.Group("/api/seats")
	seats.Use(middleware.ClerkAuthMiddleware())
	{
		seats.GET("", h.GetSeats)
		seats.GET("/active", h.GetActiveSeats)
		seats.GET("/:id", h.GetSeat)
		seats.GET("/search", h.GetSeatByNumber)
		seats.GET("/floor", h.GetSeatsByFloor)
		seats.GET("/space", h.GetSeatsBySpace)
	}

	admin := r.Group("/api/admin/seats")
	admin.Use(middleware.ClerkAuthMiddleware())
	admin.Use(middleware.RequireAdmin(h.userRepo))
	{
		admin.POST("", h.CreateSeat)
		admin.PUT("/:id", h.UpdateSeat)
		admin.DELETE("/:id", h.DeleteSeat)
		admin.GET("/unassigned", h.GetUnassignedSeats)
	}
}
