package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"
)

type FloorHandler struct {
	floorUsecase usecase.FloorUsecase
	userRepo     repository.UserRepository // 管理者認証用
}

type CreateFloorRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

type UpdateFloorRequest struct {
	Name        *string `json:"name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Description *string `json:"description,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

func NewFloorHandler(fu usecase.FloorUsecase, ur repository.UserRepository) *FloorHandler {
	return &FloorHandler{
		floorUsecase: fu,
		userRepo:     ur,
	}
}

// CreateFloor godoc
// @Summary Create floor
// @Description Create a new floor (admin only)
// @Tags floors
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateFloorRequest true "Floor creation request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Router /admin/floors [post]
func (h *FloorHandler) CreateFloor(c *gin.Context) {
	var req CreateFloorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	floor := &entity.Floor{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		IsActive:    req.IsActive,
	}

	if err := h.floorUsecase.Create(c.Request.Context(), floor); err != nil {
		HandleUsecaseError(c, err, "フロアの作成に失敗しました")
		return
	}

	c.JSON(http.StatusOK, floor)
}

// GetFloors godoc
// @Summary Get all floors
// @Description Get all floors including inactive ones (admin only)
// @Tags floors
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit (default: 100)"
// @Param offset query int false "Offset (default: 0)"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /admin/floors [get]
func (h *FloorHandler) GetFloors(c *gin.Context) {
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

	floors, err := h.floorUsecase.List(c.Request.Context(), limit, offset)
	if err != nil {
		HandleUsecaseError(c, err, "フロア一覧の取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"floors": floors,
		"count":  len(floors),
	})
}

// GetActiveFloors godoc
// @Summary Get active floors
// @Description Get only active floors (all authenticated users)
// @Tags floors
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /floors/active [get]
func (h *FloorHandler) GetActiveFloors(c *gin.Context) {
	floors, err := h.floorUsecase.GetActiveFloors(c.Request.Context())
	if err != nil {
		HandleUsecaseError(c, err, "アクティブなフロアの取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"floors": floors,
		"count":  len(floors),
	})
}

// GetFloor godoc
// @Summary Get floor by ID
// @Description Get floor details by ID (all authenticated users)
// @Tags floors
// @Security BearerAuth
// @Param id path string true "Floor ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 404 {object} handler.ErrorResponse "Floor not found"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /floors/{id} [get]
func (h *FloorHandler) GetFloor(c *gin.Context) {
	floorID := c.Param("id")
	if floorID == "" {
		RespondWithError(c, http.StatusBadRequest, "フロアIDが必要です", CodeValidation)
		return
	}

	floor, err := h.floorUsecase.GetByID(c.Request.Context(), floorID)
	if err != nil {
		HandleUsecaseError(c, err, "フロアの取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, floor)
}

// UpdateFloor godoc
// @Summary Update floor
// @Description Update floor details (admin only)
// @Tags floors
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Floor ID"
// @Param request body UpdateFloorRequest true "Floor update request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /admin/floors/{id} [put]
func (h *FloorHandler) UpdateFloor(c *gin.Context) {
	floorID := c.Param("id")
	if floorID == "" {
		RespondWithError(c, http.StatusBadRequest, "フロアIDが必要です", CodeValidation)
		return
	}

	var req UpdateFloorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	floor := &entity.Floor{}

	if req.Name != nil {
		floor.Name = *req.Name
	}
	if req.DisplayName != nil {
		floor.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		floor.Description = *req.Description
	}
	if req.SortOrder != nil {
		floor.SortOrder = *req.SortOrder
	}
	if req.IsActive != nil {
		floor.IsActive = *req.IsActive
	}

	if err := h.floorUsecase.Update(c.Request.Context(), floorID, floor); err != nil {
		HandleUsecaseError(c, err, "フロアの更新に失敗しました")
		return
	}

	updatedFloor, err := h.floorUsecase.GetByID(c.Request.Context(), floorID)
	if err != nil {
		HandleUsecaseError(c, err, "更新後のフロア取得に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "フロアを更新しました",
		"floor":   updatedFloor,
	})
}

// DeleteFloor godoc
// @Summary Delete floor
// @Description Delete a floor (admin only)
// @Tags floors
// @Security BearerAuth
// @Param id path string true "Floor ID"
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} handler.ErrorResponse "Bad request"
// @Failure 401 {object} handler.ErrorResponse "Unauthorized"
// @Failure 403 {object} handler.ErrorResponse "Forbidden"
// @Failure 404 {object} handler.ErrorResponse "Floor not found"
// @Failure 500 {object} handler.ErrorResponse "Internal server error"
// @Router /admin/floors/{id} [delete]
func (h *FloorHandler) DeleteFloor(c *gin.Context) {
	floorID := c.Param("id")
	if floorID == "" {
		RespondWithError(c, http.StatusBadRequest, "フロアIDが必要です", CodeValidation)
		return
	}

	if err := h.floorUsecase.Delete(c.Request.Context(), floorID); err != nil {
		HandleUsecaseError(c, err, "フロアの削除に失敗しました")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "フロアを削除しました"})
}

func (h *FloorHandler) RegisterRoutes(r *gin.Engine) {
	floors := r.Group("/api/floors")
	floors.Use(middleware.ClerkAuthMiddleware())
	{
		floors.GET("/active", h.GetActiveFloors)
		floors.GET("/:id", h.GetFloor)
	}

	admin := r.Group("/api/admin/floors")
	admin.Use(middleware.ClerkAuthMiddleware())
	admin.Use(middleware.RequireAdmin(h.userRepo))
	{
		admin.POST("", h.CreateFloor)
		admin.GET("", h.GetFloors)
		admin.PUT("/:id", h.UpdateFloor)
		admin.DELETE("/:id", h.DeleteFloor)
	}
}
