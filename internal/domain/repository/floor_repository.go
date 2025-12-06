package repository

import (
	"context"

	"seat-management-backend/internal/domain/entity"
)

type FloorRepository interface {
	Create(ctx context.Context, floor *entity.Floor) error
	FindByID(ctx context.Context, id string) (*entity.Floor, error)
	FindByName(ctx context.Context, name string) (*entity.Floor, error)
	FindAll(ctx context.Context) ([]*entity.Floor, error)
	FindAllActive(ctx context.Context) ([]*entity.Floor, error)
	Update(ctx context.Context, floor *entity.Floor) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*entity.Floor, error)
}
