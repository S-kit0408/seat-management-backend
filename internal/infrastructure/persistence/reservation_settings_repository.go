package persistence

import (
	"context"
	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"

	"gorm.io/gorm"
)

type reservationSettingsRepository struct {
	db *gorm.DB
}

func NewReservationSettingsRepository(db *gorm.DB) repository.ReservationSettingsRepository {
	return &reservationSettingsRepository{db: db}
}

func (r *reservationSettingsRepository) Create(ctx context.Context, settings *entity.ReservationSettings) error {
	return r.db.WithContext(ctx).Create(settings).Error
}

func (r *reservationSettingsRepository) FindByID(ctx context.Context, id string) (*entity.ReservationSettings, error) {
	var settings entity.ReservationSettings
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entity.ErrReservationSettingsNotFound
		}
		return nil, err
	}
	return &settings, nil
}

func (r *reservationSettingsRepository) FindAll(ctx context.Context) ([]*entity.ReservationSettings, error) {
	var settings []*entity.ReservationSettings
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&settings).Error
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *reservationSettingsRepository) Update(ctx context.Context, settings *entity.ReservationSettings) error {
	return r.db.WithContext(ctx).Save(settings).Error
}

func (r *reservationSettingsRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.ReservationSettings{}, "id = ?", id).Error
}

func (r *reservationSettingsRepository) GetActive(ctx context.Context) (*entity.ReservationSettings, error) {
	var settings entity.ReservationSettings
	err := r.db.WithContext(ctx).Where("is_active = ?", true).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entity.ErrReservationSettingsNotFound
		}
		return nil, err
	}
	return &settings, nil
}

// SetActive atomically activates one settings and deactivates all others
func (r *reservationSettingsRepository) SetActive(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Verify settings exists
		var settings entity.ReservationSettings
		if err := tx.Where("id = ?", id).First(&settings).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return entity.ErrReservationSettingsNotFound
			}
			return err
		}

		// Deactivate all
		if err := tx.Model(&entity.ReservationSettings{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
			return err
		}

		// Activate target
		settings.IsActive = true
		return tx.Save(&settings).Error
	})
}

func (r *reservationSettingsRepository) DeactivateAll(ctx context.Context) error {
	return r.db.WithContext(ctx).Model(&entity.ReservationSettings{}).
		Where("is_active = ?", true).
		Update("is_active", false).Error
}
