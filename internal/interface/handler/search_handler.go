package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"seat-management-backend/internal/middleware"
	"seat-management-backend/internal/usecase"
)

// SearchHandler は自然言語検索ハンドラー
type SearchHandler struct {
	aiSearchUsecase usecase.AiSearchUsecase
}

// NewSearchHandler は SearchHandler を初期化
func NewSearchHandler(aiSearchUsecase usecase.AiSearchUsecase) *SearchHandler {
	return &SearchHandler{
		aiSearchUsecase: aiSearchUsecase,
	}
}

// RegisterRoutes は検索エンドポイントを登録
func (h *SearchHandler) RegisterRoutes(router *gin.Engine) {
	search := router.Group("/api/search")
	search.Use(middleware.ClerkAuthMiddleware())
	{
		search.POST("/seats", h.SearchSeats)
	}
}

// SearchSeats は POST /api/search/seats
// @Summary 自然言語で座席検索（全一致・部分一致）
// @Description 自然言語クエリで座席を検索します。結果は全一致と部分一致に分かれて返されます
// @Tags search
// @Accept json
// @Produce json
// @Param request body searchSeatsRequest true "検索クエリと検索モード"
// @Success 200 {object} searchSeatsResponse
// @Failure 400 {object} handler.ErrorResponse
// @Failure 401 {object} handler.ErrorResponse
// @Failure 500 {object} handler.ErrorResponse
// @Router /search/seats [post]
func (h *SearchHandler) SearchSeats(c *gin.Context) {
	var req struct {
		Query          string `json:"query" binding:"required"`
		ExactMatchOnly bool   `json:"exact_match_only"` // トグル：true=全一致のみ, false=全て表示
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithValidationError(c, err)
		return
	}

	if len(req.Query) > 500 {
		RespondWithError(c, http.StatusBadRequest, "query is too long (max 500 characters)", CodeValidation)
		return
	}

	searchResult, err := h.aiSearchUsecase.SearchSeats(c.Request.Context(), req.Query)
	if err != nil {
		HandleUsecaseError(c, err, "座席検索に失敗しました")
		return
	}

	// レスポンス作成
	var responseSeats interface{}
	var count int

	if req.ExactMatchOnly {
		// 全一致のみ表示
		responseSeats = searchResult.ExactMatches
		count = len(searchResult.ExactMatches)
	} else {
		// 全一致と部分一致を両方表示
		responseSeats = gin.H{
			"exact_matches":   searchResult.ExactMatches,
			"partial_matches": searchResult.PartialMatches,
		}
		count = len(searchResult.ExactMatches) + len(searchResult.PartialMatches)
	}

	c.JSON(http.StatusOK, gin.H{
		"seats":               responseSeats,
		"count":               count,
		"exact_match_count":   len(searchResult.ExactMatches),
		"partial_match_count": len(searchResult.PartialMatches),
	})
}

// リクエスト/レスポンス型定義（ドキュメント用）
type searchSeatsRequest struct {
	Query          string `json:"query" example:"静かな場所がいい"`
	ExactMatchOnly bool   `json:"exact_match_only" example:"false"`
}

type searchSeatsResponse struct {
	Seats             interface{} `json:"seats"`
	Count             int         `json:"count"`
	ExactMatchCount   int         `json:"exact_match_count"`
	PartialMatchCount int         `json:"partial_match_count"`
}

type seatResponse struct {
	ID            string      `json:"id"`
	SeatNumber    string      `json:"seat_number"`
	Description   string      `json:"description"`
	PositionX     float64     `json:"position_x"`
	PositionY     float64     `json:"position_y"`
	RotationAngle int         `json:"rotation_angle"`
	Width         float64     `json:"width"`
	Height        float64     `json:"height"`
	Shape         string      `json:"shape"`
	Attributes    interface{} `json:"attributes"`
	FloorID       *string     `json:"floor_id"`
	SpaceID       *string     `json:"space_id"`
	IsActive      bool        `json:"is_active"`
}

type errorResponse struct {
	Error string `json:"error"`
}
