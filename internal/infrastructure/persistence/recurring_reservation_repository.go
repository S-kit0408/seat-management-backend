package persistence

import (
	"context"
	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
	"time"

	"gorm.io/gorm"
)

type RecurringReservationRepository struct {
	db *gorm.DB
}

func NewRecurringReservationRepository(db *gorm.DB) repository.RecurringReservationRepository {
	return &RecurringReservationRepository{db: db}
}

// 新しい定期予約を作成
func (r *RecurringReservationRepository) Create(ctx context.Context, recurringReservation *entity.RecurringReservation) error {
	return r.db.WithContext(ctx).Create(recurringReservation).Error
}

// 定期予約IDで取得
func (r *RecurringReservationRepository) FindByID(ctx context.Context, id string) (*entity.RecurringReservation, error) {
	var recurringReservation entity.RecurringReservation
	if err := r.db.WithContext(ctx).First(&recurringReservation, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entity.ErrRecurringReservationNotFound
		}
		return nil, err
	}
	return &recurringReservation, nil
}

// 定期予約を更新
func (r *RecurringReservationRepository) Update(ctx context.Context, recurringReservation *entity.RecurringReservation) error {
	return r.db.WithContext(ctx).Save(recurringReservation).Error
}

// 定期予約を削除
func (r *RecurringReservationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.RecurringReservation{}, "id = ?", id).Error
}

// ユーザーIDで定期予約取得
func (r *RecurringReservationRepository) FindByUserID(ctx context.Context, userID string) ([]*entity.RecurringReservation, error) {
	var recurringReservations []*entity.RecurringReservation
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&recurringReservations).Error; err != nil {
		return nil, err
	}
	return recurringReservations, nil
}

// ユーザーのアクティブな定期予約を取得
func (r *RecurringReservationRepository) FindActiveByUserID(ctx context.Context, userID string) ([]*entity.RecurringReservation, error) {
	var recurringReservations []*entity.RecurringReservation
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("is_active = ?", true).
		Where("valid_from <= ?", now).
		Where("valid_until IS NULL OR valid_until >= ?", now).
		Order("created_at DESC").
		Find(&recurringReservations).Error; err != nil {
		return nil, err
	}
	return recurringReservations, nil
}

// ユーザーのアクティブな定期予約数をカウント
func (r *RecurringReservationRepository) CountActiveByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&entity.RecurringReservation{}).
		Where("user_id = ?", userID).
		Where("is_active = ?", true).
		Where("valid_from <= ?", now).
		Where("valid_until IS NULL OR valid_until >= ?", now).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// 座席IDで定期予約を取得
func (r *RecurringReservationRepository) FindBySeatID(ctx context.Context, seatID string) ([]*entity.RecurringReservation, error) {
	var recurringReservations []*entity.RecurringReservation
	if err := r.db.WithContext(ctx).
		Where("seat_id = ?", seatID).
		Order("created_at DESC").
		Find(&recurringReservations).Error; err != nil {
		return nil, err
	}
	return recurringReservations, nil
}

// 指定にづけに対して有効な定期予約を取得
func (r *RecurringReservationRepository) FindActiveForDate(ctx context.Context, date time.Time) ([]*entity.RecurringReservation, error) {
	var recurringReservations []*entity.RecurringReservation
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Where("valid_from <= ?", dateOnly).
		Where("valid_until IS NULL OR valid_until >= ?", dateOnly).
		Order("created_at ASC").
		Find(&recurringReservations).Error; err != nil {
		return nil, err
	}

	return recurringReservations, nil
}

// すべての定期予約を取得
func (r *RecurringReservationRepository) FindAll(ctx context.Context) ([]*entity.RecurringReservation, error) {
	var recurringReservations []*entity.RecurringReservation
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&recurringReservations).Error; err != nil {
		return nil, err
	}
	return recurringReservations, nil
}
