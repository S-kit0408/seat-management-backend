package usecase

import (
	"context"
	"errors"
	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"

	"gorm.io/gorm"
)

type ReservationSettingsUsecase interface {
	GetOrCreateActiveSettings(ctx context.Context) (*entity.ReservationSettings, error)
	Create(ctx context.Context, settings *entity.ReservationSettings) error
	GetByID(ctx context.Context, id string) (*entity.ReservationSettings, error)
	GetAll(ctx context.Context) ([]*entity.ReservationSettings, error)
	Update(ctx context.Context, id string, updates *entity.ReservationSettings) error
	Activate(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

type reservationSettingsUsecase struct {
	settingsRepo repository.ReservationSettingsRepository
	db           *gorm.DB
}

func NewReservationSettingsUsecase(
	settingsRepo repository.ReservationSettingsRepository,
	db *gorm.DB,
) ReservationSettingsUsecase {
	return &reservationSettingsUsecase{
		settingsRepo: settingsRepo,
		db:           db,
	}
}

// GetOrCreateActiveSettings returns active settings or creates default if none exist
func (u *reservationSettingsUsecase) GetOrCreateActiveSettings(
	ctx context.Context,
) (*entity.ReservationSettings, error) {
	settings, err := u.settingsRepo.GetActive(ctx)
	if err == nil {
		return settings, nil
	}

	if err != entity.ErrReservationSettingsNotFound {
		return nil, err
	}

	// Create default settings
	defaultSettings := entity.GetDefaultSettings()
	if err := u.settingsRepo.Create(ctx, defaultSettings); err != nil {
		return nil, err
	}

	return defaultSettings, nil
}

// Create creates new reservation settings
func (u *reservationSettingsUsecase) Create(ctx context.Context, settings *entity.ReservationSettings) error {
	// Validate
	if err := settings.Validate(); err != nil {
		return err
	}

	// If this is set to active, deactivate others first
	if settings.IsActive {
		if err := u.settingsRepo.DeactivateAll(ctx); err != nil {
			return err
		}
	}

	return u.settingsRepo.Create(ctx, settings)
}

// GetByID returns settings by ID
func (u *reservationSettingsUsecase) GetByID(ctx context.Context, id string) (*entity.ReservationSettings, error) {
	return u.settingsRepo.FindByID(ctx, id)
}

// GetAll returns all settings
func (u *reservationSettingsUsecase) GetAll(ctx context.Context) ([]*entity.ReservationSettings, error) {
	return u.settingsRepo.FindAll(ctx)
}

// Update updates existing reservation settings
func (u *reservationSettingsUsecase) Update(
	ctx context.Context,
	id string,
	updates *entity.ReservationSettings,
) error {
	// Get existing settings
	existing, err := u.settingsRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Apply updates (only non-zero values)
	if updates.CheckInMinutesBeforeStart != 0 {
		existing.CheckInMinutesBeforeStart = updates.CheckInMinutesBeforeStart
	}
	if updates.CheckInGracePeriodMinutes != 0 {
		existing.CheckInGracePeriodMinutes = updates.CheckInGracePeriodMinutes
	}
	if updates.MinReservationMinutes != 0 {
		existing.MinReservationMinutes = updates.MinReservationMinutes
	}
	if updates.MaxReservationMinutes != 0 {
		existing.MaxReservationMinutes = updates.MaxReservationMinutes
	}
	if updates.MaxAdvanceBookingDays != 0 {
		existing.MaxAdvanceBookingDays = updates.MaxAdvanceBookingDays
	}
	if updates.CancellationDeadlineMinutes != 0 {
		existing.CancellationDeadlineMinutes = updates.CancellationDeadlineMinutes
	}
	if updates.MaxExtensionMinutes != 0 {
		existing.MaxExtensionMinutes = updates.MaxExtensionMinutes
	}
	if updates.MaxExtensionCount != 0 {
		existing.MaxExtensionCount = updates.MaxExtensionCount
	}

	if updates.Description != "" {
		existing.Description = updates.Description
	}

	// Validate updated settings
	if err := existing.Validate(); err != nil {
		return err
	}

	return u.settingsRepo.Update(ctx, existing)
}

// Activate activates a settings profile (and deactivates others)
func (u *reservationSettingsUsecase) Activate(ctx context.Context, id string) error {
	return u.settingsRepo.SetActive(ctx, id)
}

// Delete deletes reservation settings
func (u *reservationSettingsUsecase) Delete(ctx context.Context, id string) error {
	// Check if settings exists
	settings, err := u.settingsRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Prevent deletion of active settings
	if settings.IsActive {
		return errors.New("アクティブな設定は削除できません。別の設定を有効化してから削除してください")
	}

	return u.settingsRepo.Delete(ctx, id)
}
