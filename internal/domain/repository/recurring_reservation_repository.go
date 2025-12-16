package repository

import (
	"context"
	"seat-management-backend/internal/domain/entity"
	"time"
)

type RecurringReservationRepository interface {
	Create(ctx context.Context, recurringReservation *entity.RecurringReservation) error
	FindByID(ctx context.Context, id string) (*entity.RecurringReservation, error)
	Update(ctx context.Context, recurringReservation *entity.RecurringReservation) error
	Delete(ctx context.Context, id string) error

	FindByUserID(ctx context.Context, userID string) ([]*entity.RecurringReservation, error)
	FindActiveByUserID(ctx context.Context, userID string) ([]*entity.RecurringReservation, error)
	CountActiveByUserID(ctx context.Context, userID string) (int64, error)

	FindBySeatID(ctx context.Context, seatID string) ([]*entity.RecurringReservation, error)

	// 予約生成用
	FindActiveForDate(ctx context.Context, date time.Time) ([]*entity.RecurringReservation, error)
	FindAll(ctx context.Context) ([]*entity.RecurringReservation, error)
}
