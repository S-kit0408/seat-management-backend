package persistence

import (
	"context"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"

	"gorm.io/gorm"
)

type floorRepository struct {
	db *gorm.DB
}

func NewFloorRepository(db *gorm.DB) repository.FloorRepository {
	return &floorRepository{db: db}
}

func (r *floorRepository) Create(ctx context.Context, floor *entity.Floor) error {
	return r.db.WithContext(ctx).Create(floor).Error
}

func (r *floorRepository) FindByID(ctx context.Context, id string) (*entity.Floor, error) {
	var floor entity.Floor
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&floor).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entity.ErrFloorNotFound
		}
		return nil, err
	}
	return &floor, nil
}

func (r *floorRepository) FindByName(ctx context.Context, name string) (*entity.Floor, error) {
	var floor entity.Floor
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&floor).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entity.ErrFloorNotFound
		}
		return nil, err
	}
	return &floor, nil
}

func (r *floorRepository) FindAll(ctx context.Context) ([]*entity.Floor, error) {
	var floors []*entity.Floor
	err := r.db.WithContext(ctx).Order("sort_order ASC, name ASC").Find(&floors).Error
	if err != nil {
		return nil, err
	}
	return floors, nil
}

func (r *floorRepository) FindAllActive(ctx context.Context) ([]*entity.Floor, error) {
	var floors []*entity.Floor
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("sort_order ASC, name ASC").
		Find(&floors).Error
	if err != nil {
		return nil, err
	}
	return floors, nil
}

func (r *floorRepository) Update(ctx context.Context, floor *entity.Floor) error {
	return r.db.WithContext(ctx).Save(floor).Error
}

func (r *floorRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Floor{}, "id = ?", id).Error
}

func (r *floorRepository) List(ctx context.Context, limit, offset int) ([]*entity.Floor, error) {
	var floors []*entity.Floor
	err := r.db.WithContext(ctx).
		Order("sort_order ASC, name ASC").
		Limit(limit).
		Offset(offset).
		Find(&floors).Error
	if err != nil {
		return nil, err
	}
	return floors, nil
}
