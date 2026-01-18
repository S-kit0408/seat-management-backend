package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"
)

type FloorOperationHoursHandler struct {
	operationHoursUsecase usecase.FloorOperationHoursUsecase
	userRepo              repository.UserRepository
}

type CreateFloorOperationHoursRequest struct {
	FloorID   string `json:"floor_id" binding:"required"`
	DayOfWeek int    `json:"day_of_week" binding:"required,min=0,max=6"`
	OpenTime  string `json:"open_time" binding:"required"`
	CloseTime string `json:"close_time" binding:"required"`
	IsClosed  bool   `json:"is_closed"`
}

type UpdateFloorOperationHoursRequest struct {
	DayOfWeek *int    `json:"day_of_week,omitempty" binding:"omitempty,min=0,max=6"`
	OpenTime  *string `json:"open_time,omitempty"`
	CloseTime *string `json:"close_time,omitempty"`
	IsClosed  *bool   `json:"is_closed,omitempty"`
}

type UpdateFloorOperationHoursBatchRequest struct {
	OperationHours []struct {
		DayOfWeek int    `json:"day_of_week" binding:"required,min=0,max=6"`
		OpenTime  string `json:"open_time" binding:"required"`
		CloseTime string `json:"close_time" binding:"required"`
		IsClosed  bool   `json:"is_closed"`
	} `json:"operation_hours" binding:"required,min=0,max=7"`
}

func NewFloorOperationHoursHandler(
	ohu usecase.FloorOperationHoursUsecase,
	ur repository.UserRepository,
) *FloorOperationHoursHandler {
	return &FloorOperationHoursHandler{
		operationHoursUsecase: ohu,
		userRepo:              ur,
	}
}

// CreateOperationHours - POST /api/admin/operation-hours
func (h *FloorOperationHoursHandler) CreateOperationHours(c *gin.Context) {
	var req CreateFloorOperationHoursRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hours := &entity.FloorOperationHours{
		FloorID:   req.FloorID,
		DayOfWeek: req.DayOfWeek,
		OpenTime:  req.OpenTime,
		CloseTime: req.CloseTime,
		IsClosed:  req.IsClosed,
	}

	if err := h.operationHoursUsecase.Create(c.Request.Context(), hours); err != nil {
		log.Printf("Failed to create operation hours: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, hours)
}

// GetOperationHoursByFloor - GET /api/floors/:id/operation-hours
func (h *FloorOperationHoursHandler) GetOperationHoursByFloor(c *gin.Context) {
	floorID := c.Param("id")
	if floorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "フロアIDが必要です"})
		return
	}

	hours, err := h.operationHoursUsecase.GetByFloorID(c.Request.Context(), floorID)
	if err != nil {
		log.Printf("Failed to get operation hours for floor %s: %v", floorID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, hours)
}

// GetOperationHoursByFloorAndDay - GET /api/floors/:id/operation-hours/:day
func (h *FloorOperationHoursHandler) GetOperationHoursByFloorAndDay(c *gin.Context) {
	floorID := c.Param("id")
	dayStr := c.Param("day")

	if floorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "フロアIDが必要です"})
		return
	}

	if dayStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "曜日が必要です"})
		return
	}

	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 0 || day > 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "曜日は0-6の範囲で指定してください"})
		return
	}

	hours, err := h.operationHoursUsecase.GetByFloorAndDay(c.Request.Context(), floorID, day)
	if err != nil {
		if err == entity.ErrFloorOperationHoursNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "営業時間が見つかりません"})
			return
		}
		log.Printf("Failed to get operation hours for floor %s day %d: %v", floorID, day, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, hours)
}

// GetOperationHours - GET /api/admin/operation-hours/:id
func (h *FloorOperationHoursHandler) GetOperationHours(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "営業時間IDが必要です"})
		return
	}

	hours, err := h.operationHoursUsecase.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == entity.ErrFloorOperationHoursNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "営業時間が見つかりません"})
			return
		}
		log.Printf("Failed to get operation hours %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, hours)
}

// GetAllOperationHours - GET /api/admin/operation-hours
func (h *FloorOperationHoursHandler) GetAllOperationHours(c *gin.Context) {
	hours, err := h.operationHoursUsecase.GetAll(c.Request.Context())
	if err != nil {
		log.Printf("Failed to get all operation hours: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"operation_hours": hours,
		"count":           len(hours),
	})
}

// UpdateOperationHours - PUT /api/admin/operation-hours/:id
func (h *FloorOperationHoursHandler) UpdateOperationHours(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "営業時間IDが必要です"})
		return
	}

	var req UpdateFloorOperationHoursRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update entity
	updates := &entity.FloorOperationHours{}
	if req.DayOfWeek != nil {
		updates.DayOfWeek = *req.DayOfWeek
	}
	if req.OpenTime != nil {
		updates.OpenTime = *req.OpenTime
	}
	if req.CloseTime != nil {
		updates.CloseTime = *req.CloseTime
	}
	if req.IsClosed != nil {
		updates.IsClosed = *req.IsClosed
	}

	if err := h.operationHoursUsecase.Update(c.Request.Context(), id, updates); err != nil {
		log.Printf("Failed to update operation hours %s: %v", id, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch updated entity to return
	updatedHours, err := h.operationHoursUsecase.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新後の営業時間取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "営業時間を更新しました",
		"operation_hours": updatedHours,
	})
}

// DeleteOperationHours - DELETE /api/admin/operation-hours/:id
func (h *FloorOperationHoursHandler) DeleteOperationHours(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "営業時間IDが必要です"})
		return
	}

	if err := h.operationHoursUsecase.Delete(c.Request.Context(), id); err != nil {
		if err == entity.ErrFloorOperationHoursNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "営業時間が見つかりません"})
			return
		}
		log.Printf("Failed to delete operation hours %s: %v", id, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "営業時間を削除しました"})
}

// UpdateFloorOperationHoursBatch - PUT /api/admin/floors/:id/operation-hours
func (h *FloorOperationHoursHandler) UpdateFloorOperationHoursBatch(c *gin.Context) {
	floorID := c.Param("id")
	if floorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "フロアIDが必要です"})
		return
	}

	var req UpdateFloorOperationHoursBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert request to entities
	hoursEntities := make([]*entity.FloorOperationHours, 0, len(req.OperationHours))
	for _, oh := range req.OperationHours {
		hoursEntities = append(hoursEntities, &entity.FloorOperationHours{
			FloorID:   floorID,
			DayOfWeek: oh.DayOfWeek,
			OpenTime:  oh.OpenTime,
			CloseTime: oh.CloseTime,
			IsClosed:  oh.IsClosed,
		})
	}

	// Execute batch update
	updatedHours, err := h.operationHoursUsecase.UpdateFloorOperationHoursForWeek(
		c.Request.Context(),
		floorID,
		hoursEntities,
	)
	if err != nil {
		log.Printf("Failed to update floor operation hours for floor %s: %v", floorID, err)
		if err == entity.ErrFloorNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "フロアが見つかりません"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"operation_hours": updatedHours,
		"count":           len(updatedHours),
	})
}

// RegisterRoutes registers all routes for floor operation hours
func (h *FloorOperationHoursHandler) RegisterRoutes(r *gin.Engine) {
	// User endpoints (read-only)
	floors := r.Group("/api/floors/:id/operation-hours")
	floors.Use(middleware.ClerkAuthMiddleware())
	{
		floors.GET("", h.GetOperationHoursByFloor)
		floors.GET("/:day", h.GetOperationHoursByFloorAndDay)
	}

	// Admin endpoints (full CRUD)
	admin := r.Group("/api/admin/operation-hours")
	admin.Use(middleware.ClerkAuthMiddleware())
	admin.Use(middleware.RequireAdmin(h.userRepo))
	{
		admin.POST("", h.CreateOperationHours)
		admin.GET("", h.GetAllOperationHours)
		admin.GET("/:id", h.GetOperationHours)
		admin.PUT("/:id", h.UpdateOperationHours)
		admin.DELETE("/:id", h.DeleteOperationHours)
	}

	// Admin floor batch update endpoints
	floorAdmin := r.Group("/api/admin/floors/:id/operation-hours")
	floorAdmin.Use(middleware.ClerkAuthMiddleware())
	floorAdmin.Use(middleware.RequireAdmin(h.userRepo))
	{
		floorAdmin.PUT("", h.UpdateFloorOperationHoursBatch)
	}
}
