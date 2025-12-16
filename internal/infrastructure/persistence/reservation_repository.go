package persistence

import (
	"context"
	"gorm.io/gorm"
	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
	"time"
)

type ReservationRepository struct {
	db *gorm.DB
}

func NewReservationRepository(db *gorm.DB) repository.ReservationRepository {
	return &ReservationRepository{db: db}
}

// 新しい予約の作成
func (r *ReservationRepository) Create(ctx context.Context, reservation *entity.Reservation) error {
	return r.db.WithContext(ctx).Create(reservation).Error
}

// 予約IDで取得
func (r *ReservationRepository) FindByID(ctx context.Context, id string) (*entity.Reservation, error) {
	var reservation entity.Reservation
	if err := r.db.WithContext(ctx).First(&reservation, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entity.ErrReservationNotFound
		}
	}
	return &reservation, nil
}

// 既存予約を編集
func (r *ReservationRepository) Update(ctx context.Context, reservation *entity.Reservation) error {
	return r.db.WithContext(ctx).Save(reservation).Error
}

// 予約削除
func (r *ReservationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Reservation{}, "id = ?", id).Error
}

// ユーザーIDで予約を取得
func (r *ReservationRepository) FindByUserID(ctx context.Context, userID string, statuses []entity.ReservationStatus) ([]*entity.Reservation, error) {
	var reservations []*entity.Reservation
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)

	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	if err := query.Order("start_time DESC").Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

// ユーザーのアクティブな予約を取得
func (r *ReservationRepository) FindActiveReservationsByUserID(ctx context.Context, userID string) ([]*entity.Reservation, error) {
	var reservations []*entity.Reservation
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("status IN ?", []entity.ReservationStatus{
			entity.ReservationStatusReserved,
			entity.ReservationStatusInUse,
		}).
		Order("start_time ASC").
		Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

// ユーザーのアクティブな予約数をカウント
func (r *ReservationRepository) CountActiveReservationsByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&entity.Reservation{}).
		Where("user_id = ?", userID).
		Where("status IN ?", []entity.ReservationStatus{
			entity.ReservationStatusReserved,
			entity.ReservationStatusInUse,
		}).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// 座席IDで予約を取得
func (r *ReservationRepository) FindBySeatID(ctx context.Context, seatID string, startTime, endTime time.Time, statuses []entity.ReservationStatus) ([]*entity.Reservation, error) {
	var reservations []*entity.Reservation
	query := r.db.WithContext(ctx).
		Where("seat_id = ?", seatID).
		Where("start_time < ?", endTime).
		Where("end_time > ?", startTime)

	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	if err := query.Order("start_time ASC").Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

// 指定時間に重複する予約が存在するか
func (r *ReservationRepository) CheckOverlap(ctx context.Context, seatID string, startTime, endTime time.Time, excludeReservationID *string) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Where("seat_id = ?", seatID).
		Where("start_time < ?", endTime).
		Where("end_time > ?", startTime).
		Where("status IN ?", []entity.ReservationStatus{
			entity.ReservationStatusReserved,
			entity.ReservationStatusInUse,
		})

	if excludeReservationID != nil {
		query = query.Where("id != ?", *excludeReservationID)
	}

	if err := query.Model(&entity.Reservation{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// 時間範囲で予約を検索
func (r *ReservationRepository) FindByTimeRange(ctx context.Context, startTime, endTime time.Time, statuses []entity.ReservationStatus) ([]*entity.Reservation, error) {
	var reservations []*entity.Reservation
	query := r.db.WithContext(ctx).
		Where("start_time < ?", endTime).
		Where("end_time > ?", startTime)

	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	if err := query.Order("start_time ASC").Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

// 定期予約IDで予約を取得
func (r *ReservationRepository) FindByRecurringReservationID(ctx context.Context, recurringReservationID string) ([]*entity.Reservation, error) {
	var reservations []*entity.Reservation
	if err := r.db.WithContext(ctx).
		Where("recurring_reservation_id = ?", recurringReservationID).
		Order("start_time ASC").
		Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

// チェックイン期限超過の予約を取得
func (r *ReservationRepository) FindPendingCheckIns(ctx context.Context, threshold time.Time) ([]*entity.Reservation, error) {
	var reservations []*entity.Reservation
	if err := r.db.WithContext(ctx).
		Where("status = ?", entity.ReservationStatusReserved).
		Where("start_time < ?", threshold).
		Where("checked_in_at IS NULL").
		Order("start_time ASC").
		Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

// プライバシー設定を考慮した予約を取得
func (r *ReservationRepository) FindVisibleReservations(ctx context.Context, viewerUserID string, timeRange *repository.TimeRange) ([]*entity.Reservation, error) {
	var reservations []*entity.Reservation

	query := r.db.WithContext(ctx).
		Preload("User").
		Where("status IN ?", []entity.ReservationStatus{
			entity.ReservationStatusReserved,
			entity.ReservationStatusInUse,
			entity.ReservationStatusCompleted,
		})

	// 時間範囲を指定している場合はフィルタリング
	if timeRange != nil {
		query = query.
			Where("start_time < ?", timeRange.End).
			Where("end_time > ?", timeRange.Start)
	}

	if err := query.Order("start_time ASC").Find(&reservations).Error; err != nil {
		return nil, err
	}

	// プライバシー設定に基づいてフィルタリング
	visibleReservations := make([]*entity.Reservation, 0)
	for _, res := range reservations {
		// 自分の予約は常に表示
		if res.UserID == viewerUserID {
			visibleReservations = append(visibleReservations, res)
			continue
		}

		// プライバシー設定を取得
		privacySetting := res.PrivacySetting
		if privacySetting == nil {
			privacySetting = &res.User.DefaultPrivacySetting
		}

		// プライバシーチェック
		switch *privacySetting {

		// 誰でも見える
		case entity.PrivacyPublic:
			visibleReservations = append(visibleReservations, res)
		// 誰にも見えない（所有者のみ）
		case entity.PrivacyPrivate:
			// スキップ
		// フレンドのみ見える（フレンドチェックはusecase層で実装推奨）
		case entity.PrivacyFriends:
			visibleReservations = append(visibleReservations, res)
		}
	}

	return visibleReservations, nil
}
