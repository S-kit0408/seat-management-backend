package usecase

import (
	"context"
	"fmt"
	"log"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
	"seat-management-backend/internal/infrastructure/ai"
)

// SearchResult は検索結果（全一致・部分一致）
type SearchResult struct {
	ExactMatches   []*entity.Seat `json:"exact_matches"`
	PartialMatches []*entity.Seat `json:"partial_matches"`
}

// AiSearchUsecase は自然言語検索のビジネスロジック
type AiSearchUsecase interface {
	SearchSeats(ctx context.Context, query string) (*SearchResult, error)
}

type aiSearchUsecase struct {
	geminiService *ai.GeminiService
	seatRepo      repository.SeatRepository
}

// NewAiSearchUsecase は AiSearchUsecase を初期化
func NewAiSearchUsecase(
	geminiService *ai.GeminiService,
	seatRepo repository.SeatRepository,
) AiSearchUsecase {
	return &aiSearchUsecase{
		geminiService: geminiService,
		seatRepo:      seatRepo,
	}
}

// SearchSeats は自然言語クエリで座席を検索
// 全一致結果と部分一致結果を返す
func (u *aiSearchUsecase) SearchSeats(
	ctx context.Context,
	query string,
) (*SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// 1. Gemini APIでクエリをパース（キーワード抽出）
	params, err := u.geminiService.ParseSearchQuery(ctx, query)
	if err != nil {
		log.Printf("[AI Search] Gemini parse error: %v", err)
		return nil, fmt.Errorf("failed to parse query with AI: %w", err)
	}

	log.Printf("[AI Search] Extracted keywords: %v, floor: %v", params.Keywords, params.FloorID)

	// 2. リポジトリで全一致・部分一致検索を実行
	exactMatches, partialMatches, err := u.seatRepo.FindByKeywords(ctx, params.Keywords, params.FloorID)
	if err != nil {
		log.Printf("[AI Search] Repository search error: %v", err)
		return nil, fmt.Errorf("failed to search seats: %w", err)
	}

	log.Printf("[AI Search] Found exact: %d, partial: %d", len(exactMatches), len(partialMatches))

	return &SearchResult{
		ExactMatches:   exactMatches,
		PartialMatches: partialMatches,
	}, nil
}
