package usecase

import (
	"context"
	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"

	"gorm.io/gorm"
)

type FloorOperationHoursUsecase interface {
	Create(ctx context.Context, hours *entity.FloorOperationHours) error
	GetByID(ctx context.Context, id string) (*entity.FloorOperationHours, error)
	GetByFloorID(ctx context.Context, floorID string) ([]*entity.FloorOperationHours, error)
	GetByFloorAndDay(ctx context.Context, floorID string, dayOfWeek int) (*entity.FloorOperationHours, error)
	Update(ctx context.Context, id string, updates *entity.FloorOperationHours) error
	Delete(ctx context.Context, id string) error
	GetAll(ctx context.Context) ([]*entity.FloorOperationHours, error)
	UpdateFloorOperationHoursForWeek(ctx context.Context, floorID string, hours []*entity.FloorOperationHours) ([]*entity.FloorOperationHours, error)
}

type floorOperationHoursUsecase struct {
	hoursRepo repository.FloorOperationHoursRepository
	floorRepo repository.FloorRepository
	db        *gorm.DB
}

func NewFloorOperationHoursUsecase(
	hoursRepo repository.FloorOperationHoursRepository,
	floorRepo repository.FloorRepository,
	db *gorm.DB,
) FloorOperationHoursUsecase {
	return &floorOperationHoursUsecase{
		hoursRepo: hoursRepo,
		floorRepo: floorRepo,
		db:        db,
	}
}

// Create creates a new floor operation hours record
func (u *floorOperationHoursUsecase) Create(ctx context.Context, hours *entity.FloorOperationHours) error {
	// Step 1: Validate
	if err := hours.Validate(); err != nil {
		return err
	}

	// Step 2: Verify Floor exists
	_, err := u.floorRepo.FindByID(ctx, hours.FloorID)
	if err != nil {
		return entity.ErrFloorNotFound
	}

	// Step 3: Check for duplicate (FloorID, DayOfWeek)
	existing, err := u.hoursRepo.FindByFloorIDAndDayOfWeek(ctx, hours.FloorID, hours.DayOfWeek)
	if err != nil && err != entity.ErrFloorOperationHoursNotFound {
		return err
	}
	if existing != nil {
		return entity.ErrDuplicateFloorDayOfWeek
	}

	// Step 4: Create
	return u.hoursRepo.Create(ctx, hours)
}

// GetByID retrieves operation hours by ID
func (u *floorOperationHoursUsecase) GetByID(ctx context.Context, id string) (*entity.FloorOperationHours, error) {
	return u.hoursRepo.FindByID(ctx, id)
}

// GetByFloorID retrieves all operation hours for a specific floor
func (u *floorOperationHoursUsecase) GetByFloorID(ctx context.Context, floorID string) ([]*entity.FloorOperationHours, error) {
	return u.hoursRepo.FindByFloorID(ctx, floorID)
}

// GetByFloorAndDay retrieves operation hours for specific floor and day of week
func (u *floorOperationHoursUsecase) GetByFloorAndDay(ctx context.Context, floorID string, dayOfWeek int) (*entity.FloorOperationHours, error) {
	return u.hoursRepo.FindByFloorIDAndDayOfWeek(ctx, floorID, dayOfWeek)
}

// Update updates existing operation hours
func (u *floorOperationHoursUsecase) Update(ctx context.Context, id string, updates *entity.FloorOperationHours) error {
	// Step 1: Get existing record
	existing, err := u.hoursRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Step 2: Apply updates selectively (only non-zero/non-empty values)
	if updates.DayOfWeek != 0 && updates.DayOfWeek != existing.DayOfWeek {
		// Check for duplicate if DayOfWeek is being changed
		other, err := u.hoursRepo.FindByFloorIDAndDayOfWeek(ctx, existing.FloorID, updates.DayOfWeek)
		if err != nil && err != entity.ErrFloorOperationHoursNotFound {
			return err
		}
		if other != nil {
			return entity.ErrDuplicateFloorDayOfWeek
		}
		existing.DayOfWeek = updates.DayOfWeek
	}

	if updates.OpenTime != "" {
		existing.OpenTime = updates.OpenTime
	}

	if updates.CloseTime != "" {
		existing.CloseTime = updates.CloseTime
	}

	// IsClosed is a bool, so always apply from updates if it's a pointer update
	// For direct entity update, we only update if both open_time and close_time aren't being set alone
	if updates.OpenTime != "" || updates.CloseTime != "" || updates.IsClosed {
		// If IsClosed is true, skip time validation
		if !updates.IsClosed {
			existing.IsClosed = updates.IsClosed
		} else {
			existing.IsClosed = true
		}
	}

	// Step 3: Validate updated record
	if err := existing.Validate(); err != nil {
		return err
	}

	// Step 4: Update
	return u.hoursRepo.Update(ctx, existing)
}

// Delete soft deletes operation hours
func (u *floorOperationHoursUsecase) Delete(ctx context.Context, id string) error {
	// Verify existence
	_, err := u.hoursRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	return u.hoursRepo.Delete(ctx, id)
}

// GetAll retrieves all operation hours
func (u *floorOperationHoursUsecase) GetAll(ctx context.Context) ([]*entity.FloorOperationHours, error) {
	return u.hoursRepo.FindAll(ctx)
}

// UpdateFloorOperationHoursForWeek updates all operation hours for a floor in a transaction
func (u *floorOperationHoursUsecase) UpdateFloorOperationHoursForWeek(
	ctx context.Context,
	floorID string,
	hours []*entity.FloorOperationHours,
) ([]*entity.FloorOperationHours, error) {
	// Step 1: Verify Floor exists
	_, err := u.floorRepo.FindByID(ctx, floorID)
	if err != nil {
		return nil, entity.ErrFloorNotFound
	}

	// Step 2: Start transaction
	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	// Step 3: Delete existing operation hours for this floor (hard delete to avoid soft delete conflicts)
	if err := tx.Unscoped().Where("floor_id = ?", floorID).Delete(&entity.FloorOperationHours{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Step 4: Validate and create new operation hours
	createdHours := make([]*entity.FloorOperationHours, 0, len(hours))
	for _, h := range hours {
		// Validate
		if err := h.Validate(); err != nil {
			tx.Rollback()
			return nil, err
		}

		// Set FloorID
		h.FloorID = floorID

		// Create
		if err := tx.Create(h).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		createdHours = append(createdHours, h)
	}

	// Step 5: Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return createdHours, nil
}
