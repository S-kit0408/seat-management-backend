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

// フロア作成（管理者のみ）
func (h *FloorHandler) CreateFloor(c *gin.Context) {
	var req CreateFloorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		log.Printf("Failed to create floor: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, floor)
}

// フロア一覧取得（全ユーザー）
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
		log.Printf("Failed to get floors: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "フロアの取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"floors": floors,
		"count":  len(floors),
	})
}

// アクティブなフロアのみを取得（全ユーザー）
func (h *FloorHandler) GetActiveFloors(c *gin.Context) {
	floors, err := h.floorUsecase.GetActiveFloors(c.Request.Context())
	if err != nil {
		log.Printf("Failed to get active floors: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "フロアの取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"floors": floors,
		"count":  len(floors),
	})
}

// フロア詳細を取得（全ユーザー）
func (h *FloorHandler) GetFloor(c *gin.Context) {
	floorID := c.Param("id")
	if floorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "フロアIDが必要です"})
		return
	}

	floor, err := h.floorUsecase.GetByID(c.Request.Context(), floorID)
	if err != nil {
		if err == entity.ErrFloorNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "フロアが見つかりません"})
			return
		}
		log.Printf("Failed to get floor %s: %v", floorID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "フロアの取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, floor)
}

// フロアを更新（管理者のみ）
func (h *FloorHandler) UpdateFloor(c *gin.Context) {
	floorID := c.Param("id")
	if floorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "フロアIDが必要です"})
		return
	}

	var req UpdateFloorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新用エンティティを作成（変更されたフィールドのみ設定）
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
		log.Printf("Failed to update floor %s: %v", floorID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新後のフロアを取得
	updatedFloor, err := h.floorUsecase.GetByID(c.Request.Context(), floorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新後のフロア取得に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "フロアを更新しました",
		"floor":   updatedFloor,
	})
}

// フロアを削除（管理者のみ）
func (h *FloorHandler) DeleteFloor(c *gin.Context) {
	floorID := c.Param("id")
	if floorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "フロアIDが必要です"})
		return
	}

	if err := h.floorUsecase.Delete(c.Request.Context(), floorID); err != nil {
		if err == entity.ErrFloorNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "フロアが見つかりません"})
			return
		}
		log.Printf("Failed to delete floor %s: %v", floorID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "フロアの削除に失敗しました"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "フロアを削除しました"})
}

// ルート登録
func (h *FloorHandler) RegisterRoutes(r *gin.Engine) {
	// ユーザーエンドポイント（認証必須、閲覧のみ）
	floors := r.Group("/api/floors")
	floors.Use(middleware.ClerkAuthMiddleware())
	{
		floors.GET("/active", h.GetActiveFloors)
		floors.GET("/:id", h.GetFloor)
	}

	// 管理者エンドポイント
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
