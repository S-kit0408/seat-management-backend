package repository

import (
	"context"
	"seat-management-backend/internal/domain/entity"
)

type FloorOperationHoursRepository interface {
	// Create creates a new floor operation hours record
	Create(ctx context.Context, hours *entity.FloorOperationHours) error

	// FindByID retrieves floor operation hours by ID
	FindByID(ctx context.Context, id string) (*entity.FloorOperationHours, error)

	// FindByFloorID retrieves all operation hours for a specific floor (all days)
	FindByFloorID(ctx context.Context, floorID string) ([]*entity.FloorOperationHours, error)

	// FindByFloorIDAndDayOfWeek retrieves operation hours for specific floor and day
	FindByFloorIDAndDayOfWeek(ctx context.Context, floorID string, dayOfWeek int) (*entity.FloorOperationHours, error)

	// Update updates existing floor operation hours
	Update(ctx context.Context, hours *entity.FloorOperationHours) error

	// Delete soft deletes floor operation hours
	Delete(ctx context.Context, id string) error

	// FindAll retrieves all operation hours (admin use)
	FindAll(ctx context.Context) ([]*entity.FloorOperationHours, error)
}
