package repository

import (
	"context"
	"seat-management-backend/internal/domain/entity"
	"time"
)

type ReservationRepository interface {
	Create(ctx context.Context, reservation *entity.Reservation) error
	FindByID(ctx context.Context, id string) (*entity.Reservation, error)
	Update(ctx context.Context, reservation *entity.Reservation) error
	Delete(ctx context.Context, id string) error

	FindByUserID(ctx context.Context, userID string, statuses []entity.ReservationStatus) ([]*entity.Reservation, error)
	FindActiveReservationsByUserID(ctx context.Context, userID string) ([]*entity.Reservation, error)
	CountActiveReservationsByUserID(ctx context.Context, userID string) (int64, error)

	FindBySeatID(ctx context.Context, seatID string, startTime, endTime time.Time, statuses []entity.ReservationStatus) ([]*entity.Reservation, error)
	CheckOverlap(ctx context.Context, seatID string, startTime, endTime time.Time, excludeReservationID *string) (bool, error)

	// 時間範囲検索
	FindByTimeRange(ctx context.Context, startTime, endTime time.Time, statuses []entity.ReservationStatus) ([]*entity.Reservation, error)

	// 定期予約関連
	FindByRecurringReservationID(ctx context.Context, recurringReservationID string) ([]*entity.Reservation, error)

	// ステータス管理
	FindPendingCheckIns(ctx context.Context, threshold time.Time) ([]*entity.Reservation, error) // 自動キャンセル対象

	// プライバシーフィルタリング
	FindVisibleReservations(ctx context.Context, viewerUserID string, timeRange *TimeRange) ([]*entity.Reservation, error)
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}
