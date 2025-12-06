package handler

import (
	"encoding/json"
	"log"
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

func (h *SeatHandler) CreateSeat(c *gin.Context) {
	var req CreateSeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "属性の変換に失敗しました"})
			return
		}
		seat.Attributes = attrBytes
	}

	if err := h.seatUsecase.Create(c.Request.Context(), seat); err != nil {
		log.Printf("Failed to create seat: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, seat)
}

// 座席一覧取得
func (h *SeatHandler) GetSeats(c *gin.Context) {
	// クエリパラメータからlimitとoffsetを取得
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
		log.Printf("Failed to get seats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "座席の取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"seats": seats,
		"count": len(seats),
	})
}

// アクティブな座席のみを取得（フロアが設定されている座席のみ）
func (h *SeatHandler) GetActiveSeats(c *gin.Context) {
	seats, err := h.seatUsecase.GetActiveSeatsWithFloor(c.Request.Context())
	if err != nil {
		log.Printf("Failed to get active seats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "座席の取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"seats": seats,
		"count": len(seats),
	})
}

// フロア未設定の座席を取得（管理者のみ）
func (h *SeatHandler) GetUnassignedSeats(c *gin.Context) {
	seats, err := h.seatUsecase.GetUnassignedSeats(c.Request.Context())
	if err != nil {
		log.Printf("Failed to get unassigned seats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "座席の取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"seats": seats,
		"count": len(seats),
	})
}

// 座席詳細を取得
func (h *SeatHandler) GetSeat(c *gin.Context) {
	seatID := c.Param("id")
	if seatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "座席IDが必要です"})
		return
	}

	seat, err := h.seatUsecase.GetByID(c.Request.Context(), seatID)
	if err != nil {
		if err == entity.ErrSeatNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "座席が見つかりません"})
			return
		}
		log.Printf("Failed to get seat %s: %v", seatID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "座席の取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, seat)
}

// 座席番号で座席を取得
func (h *SeatHandler) GetSeatByNumber(c *gin.Context) {
	seatNumber := c.Query("seat_number")
	if seatNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "座席番号が必要です"})
		return
	}

	seat, err := h.seatUsecase.GetBySeatNumber(c.Request.Context(), seatNumber)
	if err != nil {
		if err == entity.ErrSeatNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "座席が見つかりません"})
			return
		}
		log.Printf("Failed to get seat by number %s: %v", seatNumber, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "座席の取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, seat)
}

// 座席を更新（管理者のみ）
func (h *SeatHandler) UpdateSeat(c *gin.Context) {
	seatID := c.Param("id")
	if seatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "座席IDが必要です"})
		return
	}

	var req UpdateSeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新用エンティティを作成（変更されたフィールドのみ設定）
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "属性の変換に失敗しました"})
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
		log.Printf("Failed to update seat %s: %v", seatID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新後の座席の取得
	updatedSeat, err := h.seatUsecase.GetByID(c.Request.Context(), seatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新後の座席取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "座席を更新しました",
		"seat":    updatedSeat,
	})
}

// 座席を削除（管理者のみ）
func (h *SeatHandler) DeleteSeat(c *gin.Context) {
	seatID := c.Param("id")
	if seatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "座席IDが必要です"})
		return
	}

	if err := h.seatUsecase.Delete(c.Request.Context(), seatID); err != nil {
		if err == entity.ErrSeatNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "座席が見つかりません"})
			return
		}
		log.Printf("Failed to delete seat %s: %v", seatID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "座席の削除に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "座席を削除しました"})
}

// フロアIDで取得
func (h *SeatHandler) GetSeatsByFloor(c *gin.Context) {
	floorID := c.Query("floor_id")
	if floorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "フロアIDが必要です"})
		return
	}

	seats, err := h.seatUsecase.GetSeatsByFloorID(c.Request.Context(), floorID)
	if err != nil {
		log.Printf("Failed to get seats by floor %s: %v", floorID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "座席の取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"seats": seats,
		"count": len(seats),
	})
}

// スペースIDで取得
func (h *SeatHandler) GetSeatsBySpace(c *gin.Context) {
	spaceID := c.Query("space_id")
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "スペースIDが必要です"})
		return
	}

	seats, err := h.seatUsecase.GetSeatsBySpaceID(c.Request.Context(), spaceID)
	if err != nil {
		log.Printf("Failed to get seats by space %s: %v", spaceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "座席の取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"seats": seats,
		"count": len(seats),
	})
}

// ルート登録
func (h *SeatHandler) RegisterRoutes(r *gin.Engine) {
	// ユーザーエンドポイント（認証必須、閲覧のみ）
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

	// 管理者エンドポイント
	admin := r.Group("/api/admin/seats")
	admin.Use(middleware.ClerkAuthMiddleware())
	admin.Use(middleware.RequireAdmin(h.userRepo))
	{
		admin.POST("", h.CreateSeat)
		admin.PUT("/:id", h.UpdateSeat)
		admin.DELETE("/:id", h.DeleteSeat)
		admin.GET("/unassigned", h.GetUnassignedSeats) // フロア未設定の座席
	}
}
