package persistence

import (
	"context"
	"errors"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"

	"gorm.io/gorm"
)

type seatRepository struct {
	db *gorm.DB
}

func NewSeatRepository(db *gorm.DB) repository.SeatRepository {
	return &seatRepository{db: db}
}

func (r *seatRepository) Create(ctx context.Context, seat *entity.Seat) error {
	return r.db.WithContext(ctx).Create(seat).Error
}

func (r *seatRepository) FindByID(ctx context.Context, id string) (*entity.Seat, error) {
	var seat entity.Seat
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&seat).Error
	if err != nil {
		// レコードが見つからない場合はカスタムエラーを返す
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrSeatNotFound
		}
		return nil, err
	}
	return &seat, nil
}

func (r *seatRepository) FindBySeatNumber(ctx context.Context, seatNumber string) (*entity.Seat, error) {
	var seat entity.Seat
	err := r.db.WithContext(ctx).Where("seat_number = ?", seatNumber).First(&seat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrSeatNotFound
		}
		return nil, err
	}
	return &seat, nil
}

func (r *seatRepository) Update(ctx context.Context, seat *entity.Seat) error {
	return r.db.WithContext(ctx).Save(seat).Error
}

func (r *seatRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Seat{}, "id = ?", id).Error
}

func (r *seatRepository) List(ctx context.Context, limit, offset int) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("seat_number ASC"). // 座席番号順にソート
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindAll(ctx context.Context) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindActive(ctx context.Context) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindActiveWithFloor(ctx context.Context) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Where("is_active = ? AND floor_id IS NOT NULL", true).
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindByFloorID(ctx context.Context, floorID string) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Where("floor_id = ?", floorID).
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindBySpaceID(ctx context.Context, spaceID string) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Where("space_id = ?", spaceID).
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindUnassignedSeats(ctx context.Context) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Where("floor_id IS NULL").
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}
