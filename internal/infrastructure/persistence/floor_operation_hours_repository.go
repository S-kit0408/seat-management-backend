package persistence

import (
	"context"
	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"

	"gorm.io/gorm"
)

type floorOperationHoursRepository struct {
	db *gorm.DB
}

func NewFloorOperationHoursRepository(db *gorm.DB) repository.FloorOperationHoursRepository {
	return &floorOperationHoursRepository{db: db}
}

func (r *floorOperationHoursRepository) Create(ctx context.Context, hours *entity.FloorOperationHours) error {
	return r.db.WithContext(ctx).Create(hours).Error
}

func (r *floorOperationHoursRepository) FindByID(ctx context.Context, id string) (*entity.FloorOperationHours, error) {
	var hours entity.FloorOperationHours
	err := r.db.WithContext(ctx).
		Preload("Floor").
		Where("id = ?", id).
		First(&hours).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entity.ErrFloorOperationHoursNotFound
		}
		return nil, err
	}
	return &hours, nil
}

func (r *floorOperationHoursRepository) FindByFloorID(ctx context.Context, floorID string) ([]*entity.FloorOperationHours, error) {
	var hours []*entity.FloorOperationHours
	err := r.db.WithContext(ctx).
		Where("floor_id = ?", floorID).
		Order("day_of_week ASC").
		Find(&hours).Error
	if err != nil {
		return nil, err
	}
	return hours, nil
}

func (r *floorOperationHoursRepository) FindByFloorIDAndDayOfWeek(ctx context.Context, floorID string, dayOfWeek int) (*entity.FloorOperationHours, error) {
	var hours entity.FloorOperationHours
	err := r.db.WithContext(ctx).
		Preload("Floor").
		Where("floor_id = ? AND day_of_week = ?", floorID, dayOfWeek).
		First(&hours).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entity.ErrFloorOperationHoursNotFound
		}
		return nil, err
	}
	return &hours, nil
}

func (r *floorOperationHoursRepository) Update(ctx context.Context, hours *entity.FloorOperationHours) error {
	return r.db.WithContext(ctx).Save(hours).Error
}

func (r *floorOperationHoursRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.FloorOperationHours{}, "id = ?", id).Error
}

func (r *floorOperationHoursRepository) FindAll(ctx context.Context) ([]*entity.FloorOperationHours, error) {
	var hours []*entity.FloorOperationHours
	err := r.db.WithContext(ctx).
		Preload("Floor").
		Order("floor_id ASC, day_of_week ASC").
		Find(&hours).Error
	if err != nil {
		return nil, err
	}
	return hours, nil
}
