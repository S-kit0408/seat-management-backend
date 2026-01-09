package repository

import (
	"context"
	"seat-management-backend/internal/domain/entity"
)

type ReservationSettingsRepository interface {
	// CRUD operations
	Create(ctx context.Context, settings *entity.ReservationSettings) error
	FindByID(ctx context.Context, id string) (*entity.ReservationSettings, error)
	FindAll(ctx context.Context) ([]*entity.ReservationSettings, error)
	Update(ctx context.Context, settings *entity.ReservationSettings) error
	Delete(ctx context.Context, id string) error

	// Active settings management
	GetActive(ctx context.Context) (*entity.ReservationSettings, error)
	SetActive(ctx context.Context, id string) error
	DeactivateAll(ctx context.Context) error
}
