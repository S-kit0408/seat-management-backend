package usecase

import (
	"context"
	"fmt"
	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
	"time"

	"gorm.io/gorm"
)

type ReservationUsecase interface {
	// 予約作成
	CreateReservation(ctx context.Context, userID, seatID string, startTime, endTime time.Time) (*entity.Reservation, error)
	CreateInstantReservation(ctx context.Context, userID, seatID string, durationMinutes int) (*entity.Reservation, error)

	// チェックイン・チェックアウト
	CheckIn(ctx context.Context, reservationID string) (*entity.Reservation, error)
	CheckOut(ctx context.Context, reservationID string) (*entity.Reservation, error)

	// キャンセル・延長
	CancelReservation(ctx context.Context, reservationID, reason string) (*entity.Reservation, error)
	ExtendReservation(ctx context.Context, reservationID string, additionalMinutes int) (*entity.Reservation, error)

	// 予約取得
	GetReservationByID(ctx context.Context, reservationID string) (*entity.Reservation, error)
	GetMyReservations(ctx context.Context, userID string) ([]*entity.Reservation, error)
	GetActiveReservations(ctx context.Context, userID string) ([]*entity.Reservation, error)
	GetVisibleReservations(ctx context.Context, viewerUserID string, startTime, endTime *time.Time) ([]*entity.Reservation, error)
	GetReservationsByTimeRange(ctx context.Context, startTime, endTime time.Time, statuses []entity.ReservationStatus) ([]*entity.Reservation, error)
	GetReservationsBySeatID(ctx context.Context, seatID string, startTime, endTime time.Time, statuses []entity.ReservationStatus) ([]*entity.Reservation, error)

	// その他
	AutoCancelPendingCheckIns(ctx context.Context, thresholdMinutes int) error
	MarkAsNoShow(ctx context.Context, reservationID string) (*entity.Reservation, error)
}

type reservationUsecase struct {
	reservationRepo          repository.ReservationRepository
	recurringReservationRepo repository.RecurringReservationRepository
	userRepo                 repository.UserRepository
	seatRepo                 repository.SeatRepository
	friendshipRepo           repository.FriendshipRepository
	db                       *gorm.DB
}

func NewReservationUsecase(
	reservationRepo repository.ReservationRepository,
	recurringReservationRepo repository.RecurringReservationRepository,
	userRepo repository.UserRepository,
	seatRepo repository.SeatRepository,
	friendshipRepo repository.FriendshipRepository,
	db *gorm.DB,
) ReservationUsecase {
	return &reservationUsecase{
		reservationRepo:          reservationRepo,
		recurringReservationRepo: recurringReservationRepo,
		userRepo:                 userRepo,
		seatRepo:                 seatRepo,
		friendshipRepo:           friendshipRepo,
		db:                       db,
	}
}

// CreateReservation は予約を作成します
func (u *reservationUsecase) CreateReservation(
	ctx context.Context,
	userID string,
	seatID string,
	startTime time.Time,
	endTime time.Time,
) (*entity.Reservation, error) {
	// バリデーション：時間の妥当性
	if startTime.After(endTime) {
		return nil, entity.ErrInvalidReservationTime
	}

	// ユーザーが存在するか確認
	user, err := u.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, entity.ErrUserNotFound
	}

	// 座席が存在するか確認
	_, err = u.seatRepo.FindByID(ctx, seatID)
	if err != nil {
		return nil, entity.ErrSeatNotFound
	}

	// 重複チェック（reserved と in_use のみ重複判定）
	overlap, err := u.reservationRepo.CheckOverlap(ctx, seatID, startTime, endTime, nil)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, entity.ErrReservationOverlap
	}

	// 予約作成
	reservation := &entity.Reservation{
		UserID:         userID,
		SeatID:         seatID,
		Type:           entity.ReservationTypeScheduled,
		StartTime:      startTime,
		EndTime:        endTime,
		Status:         entity.ReservationStatusReserved,
		PrivacySetting: &user.DefaultPrivacySetting,
		AutoExtend:     false,
		ExtensionCount: 0,
	}

	// トランザクション内で保存
	if err := u.db.WithContext(ctx).Create(reservation).Error; err != nil {
		return nil, err
	}

	return reservation, nil
}

// CreateInstantReservation はその場での予約を作成します（Type: instant）
func (u *reservationUsecase) CreateInstantReservation(
	ctx context.Context,
	userID string,
	seatID string,
	durationMinutes int,
) (*entity.Reservation, error) {
	// ユーザー・座席存在確認
	user, err := u.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, entity.ErrUserNotFound
	}

	_, err = u.seatRepo.FindByID(ctx, seatID)
	if err != nil {
		return nil, entity.ErrSeatNotFound
	}

	// 時間設定
	now := time.Now()
	startTime := now
	endTime := now.Add(time.Duration(durationMinutes) * time.Minute)

	// 重複チェック
	overlap, err := u.reservationRepo.CheckOverlap(ctx, seatID, startTime, endTime, nil)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, entity.ErrReservationOverlap
	}

	// 予約作成
	reservation := &entity.Reservation{
		UserID:         userID,
		SeatID:         seatID,
		Type:           entity.ReservationTypeInstant,
		StartTime:      startTime,
		EndTime:        endTime,
		Status:         entity.ReservationStatusInUse,
		CheckedInAt:    &now,
		PrivacySetting: &user.DefaultPrivacySetting,
	}

	if err := u.db.WithContext(ctx).Create(reservation).Error; err != nil {
		return nil, err
	}

	return reservation, nil
}

// CheckIn はチェックインを実行します
func (u *reservationUsecase) CheckIn(ctx context.Context, reservationID string) (*entity.Reservation, error) {
	// 予約を取得
	reservation, err := u.reservationRepo.FindByID(ctx, reservationID)
	if err != nil {
		return nil, entity.ErrReservationNotFound
	}

	// チェックイン可能か確認
	if !reservation.CanCheckIn() {
		return nil, entity.ErrCannotCheckIn
	}

	// チェックイン実行
	reservation.CheckIn()

	// 保存
	if err := u.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, err
	}

	return reservation, nil
}

// CheckOut はチェックアウトを実行します
func (u *reservationUsecase) CheckOut(ctx context.Context, reservationID string) (*entity.Reservation, error) {
	// 予約を取得
	reservation, err := u.reservationRepo.FindByID(ctx, reservationID)
	if err != nil {
		return nil, entity.ErrReservationNotFound
	}

	// チェックアウト可能か確認
	if !reservation.CanCheckOut() {
		return nil, entity.ErrCannotCheckOut
	}

	// チェックアウト実行
	reservation.CheckOut()

	// 保存
	if err := u.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, err
	}

	return reservation, nil
}

// CancelReservation は予約をキャンセルします
func (u *reservationUsecase) CancelReservation(
	ctx context.Context,
	reservationID string,
	reason string,
) (*entity.Reservation, error) {
	// 予約を取得
	reservation, err := u.reservationRepo.FindByID(ctx, reservationID)
	if err != nil {
		return nil, entity.ErrReservationNotFound
	}

	// キャンセル可能か確認
	if !reservation.CanCancel() {
		return nil, entity.ErrCannotCancel
	}

	// キャンセル実行
	reservation.Cancel(reason)

	// 保存
	if err := u.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, err
	}

	return reservation, nil
}

// ExtendReservation は予約を延長します
func (u *reservationUsecase) ExtendReservation(
	ctx context.Context,
	reservationID string,
	additionalMinutes int,
) (*entity.Reservation, error) {
	// 予約を取得
	reservation, err := u.reservationRepo.FindByID(ctx, reservationID)
	if err != nil {
		return nil, entity.ErrReservationNotFound
	}

	// 延長可能か確認
	if !reservation.CanExtend() {
		return nil, entity.ErrCannotExtend
	}

	// 延長後の時間帯で重複がないか確認
	newEndTime := reservation.EndTime.Add(time.Duration(additionalMinutes) * time.Minute)
	overlap, err := u.reservationRepo.CheckOverlap(
		ctx,
		reservation.SeatID,
		reservation.StartTime,
		newEndTime,
		&reservation.ID,
	)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, entity.ErrReservationOverlap
	}

	// 延長実行
	reservation.Extend(additionalMinutes)

	// 保存
	if err := u.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, err
	}

	return reservation, nil
}

// GetMyReservations はユーザーの全予約を取得します
func (u *reservationUsecase) GetMyReservations(
	ctx context.Context,
	userID string,
) ([]*entity.Reservation, error) {
	reservations, err := u.reservationRepo.FindByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}
	return reservations, nil
}

// GetActiveReservations はユーザーのアクティブな予約を取得します（reserved / in_use）
func (u *reservationUsecase) GetActiveReservations(
	ctx context.Context,
	userID string,
) ([]*entity.Reservation, error) {
	reservations, err := u.reservationRepo.FindActiveReservationsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return reservations, nil
}

// GetVisibleReservations はプライバシー設定を考慮した予約を取得します
func (u *reservationUsecase) GetVisibleReservations(
	ctx context.Context,
	viewerUserID string,
	startTime *time.Time,
	endTime *time.Time,
) ([]*entity.Reservation, error) {
	// TimeRange 構築
	var timeRange *repository.TimeRange
	if startTime != nil && endTime != nil {
		timeRange = &repository.TimeRange{
			Start: *startTime,
			End:   *endTime,
		}
	}

	// プライバシー対応の予約取得
	reservations, err := u.reservationRepo.FindVisibleReservations(ctx, viewerUserID, timeRange)
	if err != nil {
		return nil, err
	}

	// PrivacySetting=friends の場合、フレンドシップをチェック
	filteredReservations := make([]*entity.Reservation, 0)
	for _, res := range reservations {
		// 自分の予約は常に表示
		if res.UserID == viewerUserID {
			filteredReservations = append(filteredReservations, res)
			continue
		}

		// プライバシー設定を取得
		privacySetting := res.PrivacySetting
		if privacySetting == nil {
			privacySetting = &res.User.DefaultPrivacySetting
		}

		// friends 設定の場合はフレンドシップ確認
		if *privacySetting == entity.PrivacyFriends {
			isFriend, err := u.friendshipRepo.CheckFriendship(ctx, viewerUserID, res.UserID)
			if err != nil || !isFriend {
				continue // フレンドでなければスキップ
			}
		}

		filteredReservations = append(filteredReservations, res)
	}

	return filteredReservations, nil
}

// GetReservationsByTimeRange は時間範囲で予約を取得します
func (u *reservationUsecase) GetReservationsByTimeRange(
	ctx context.Context,
	startTime time.Time,
	endTime time.Time,
	statuses []entity.ReservationStatus,
) ([]*entity.Reservation, error) {
	if startTime.After(endTime) {
		return nil, entity.ErrInvalidReservationTime
	}

	reservations, err := u.reservationRepo.FindByTimeRange(ctx, startTime, endTime, statuses)
	if err != nil {
		return nil, err
	}
	return reservations, nil
}

// GetReservationsBySeatID は座席の予約を時間範囲で取得します
func (u *reservationUsecase) GetReservationsBySeatID(
	ctx context.Context,
	seatID string,
	startTime time.Time,
	endTime time.Time,
	statuses []entity.ReservationStatus,
) ([]*entity.Reservation, error) {
	if startTime.After(endTime) {
		return nil, entity.ErrInvalidReservationTime
	}

	reservations, err := u.reservationRepo.FindBySeatID(ctx, seatID, startTime, endTime, statuses)
	if err != nil {
		return nil, err
	}
	return reservations, nil
}

// AutoCancelPendingCheckIns はチェックイン期限超過の予約を自動キャンセルします
func (u *reservationUsecase) AutoCancelPendingCheckIns(ctx context.Context, thresholdMinutes int) error {
	// 期限を計算（now - thresholdMinutes）
	threshold := time.Now().Add(-time.Duration(thresholdMinutes) * time.Minute)

	// 期限超過の予約を取得
	reservations, err := u.reservationRepo.FindPendingCheckIns(ctx, threshold)
	if err != nil {
		return err
	}

	// トランザクション開始
	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 各予約をキャンセル
	for _, res := range reservations {
		res.Cancel("自動キャンセル：チェックイン期限超過")
		if err := tx.Save(res).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to cancel reservation %s: %w", res.ID, err)
		}
	}

	// コミット
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

// GetReservationByID は予約IDで予約を取得します
func (u *reservationUsecase) GetReservationByID(
	ctx context.Context,
	reservationID string,
) (*entity.Reservation, error) {
	reservation, err := u.reservationRepo.FindByID(ctx, reservationID)
	if err != nil {
		return nil, entity.ErrReservationNotFound
	}
	return reservation, nil
}

// MarkAsNoShow は予約を無断キャンセルにマークします
func (u *reservationUsecase) MarkAsNoShow(
	ctx context.Context,
	reservationID string,
) (*entity.Reservation, error) {
	reservation, err := u.reservationRepo.FindByID(ctx, reservationID)
	if err != nil {
		return nil, entity.ErrReservationNotFound
	}

	reservation.MarkAsNoShow()

	if err := u.reservationRepo.Update(ctx, reservation); err != nil {
		return nil, err
	}

	return reservation, nil
}
