package repository

import (
	"context"
	"seat-management-backend/internal/domain/entity"
)

type SeatRepository interface {
	Create(ctx context.Context, seat *entity.Seat) error
	FindByID(ctx context.Context, id string) (*entity.Seat, error)
	FindBySeatNumber(ctx context.Context, seatNumber string) (*entity.Seat, error)
	Update(ctx context.Context, seat *entity.Seat) error
	Delete(ctx context.Context, id string) error

	List(ctx context.Context, limit, offset int) ([]*entity.Seat, error)
	FindAll(ctx context.Context) ([]*entity.Seat, error)
	FindActive(ctx context.Context) ([]*entity.Seat, error)
	FindActiveWithFloor(ctx context.Context) ([]*entity.Seat, error) // is_active=true AND floor_id IS NOT NULL

	FindByFloorID(ctx context.Context, floorID string) ([]*entity.Seat, error)
	FindBySpaceID(ctx context.Context, spaceID string) ([]*entity.Seat, error)
	FindUnassignedSeats(ctx context.Context) ([]*entity.Seat, error) // floor_id IS NULL の座席

	// AI検索用メソッド
	FindByAttributes(ctx context.Context, params map[string]interface{}) ([]*entity.Seat, error)
	FindByKeywords(ctx context.Context, keywords []string, floorID *string) (exactMatches []*entity.Seat, partialMatches []*entity.Seat, err error)
}
