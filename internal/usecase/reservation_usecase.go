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
	CreateReservation(ctx context.Context, userID, seatID string, startTime, endTime time.Time, privacySetting *entity.PrivacySetting) (*entity.Reservation, error)
	CreateInstantReservation(ctx context.Context, userID, seatID string, durationMinutes int, privacySetting *entity.PrivacySetting) (*entity.Reservation, error)

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
	settingsRepo             repository.ReservationSettingsRepository
	db                       *gorm.DB
}

func NewReservationUsecase(
	reservationRepo repository.ReservationRepository,
	recurringReservationRepo repository.RecurringReservationRepository,
	userRepo repository.UserRepository,
	seatRepo repository.SeatRepository,
	friendshipRepo repository.FriendshipRepository,
	settingsRepo repository.ReservationSettingsRepository,
	db *gorm.DB,
) ReservationUsecase {
	return &reservationUsecase{
		reservationRepo:          reservationRepo,
		recurringReservationRepo: recurringReservationRepo,
		userRepo:                 userRepo,
		seatRepo:                 seatRepo,
		friendshipRepo:           friendshipRepo,
		settingsRepo:             settingsRepo,
		db:                       db,
	}
}

// getSettingsWithFallback returns active settings or code defaults on error
func (u *reservationUsecase) getSettingsWithFallback(ctx context.Context) *entity.ReservationSettings {
	settings, err := u.settingsRepo.GetActive(ctx)
	if err != nil {
		// Fallback to code defaults
		return entity.GetDefaultSettings()
	}
	return settings
}

// CreateReservation は予約を作成します
func (u *reservationUsecase) CreateReservation(
	ctx context.Context,
	userID string,
	seatID string,
	startTime time.Time,
	endTime time.Time,
	privacySetting *entity.PrivacySetting,
) (*entity.Reservation, error) {
	// Get settings (with fallback)
	settings := u.getSettingsWithFallback(ctx)

	// バリデーション：時間の妥当性
	if startTime.After(endTime) {
		return nil, entity.ErrInvalidReservationTime
	}

	// バリデーション：予約時間制限
	durationMinutes := int(endTime.Sub(startTime).Minutes())
	if durationMinutes < settings.MinReservationMinutes {
		return nil, entity.ErrReservationTooShort
	}
	if durationMinutes > settings.MaxReservationMinutes {
		return nil, entity.ErrReservationTooLong
	}

	// バリデーション：事前予約期限
	now := time.Now()
	maxAdvanceTime := now.AddDate(0, 0, settings.MaxAdvanceBookingDays)
	if startTime.After(maxAdvanceTime) {
		return nil, entity.ErrAdvanceBookingExceeded
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

	// プライバシー設定を決定：リクエストで指定されていればそれを使用、未指定ならユーザーのデフォルト設定
	effectivePrivacySetting := privacySetting
	if effectivePrivacySetting == nil {
		effectivePrivacySetting = &user.DefaultPrivacySetting
	}

	// 予約作成
	reservation := &entity.Reservation{
		UserID:         userID,
		SeatID:         seatID,
		Type:           entity.ReservationTypeScheduled,
		StartTime:      startTime,
		EndTime:        endTime,
		Status:         entity.ReservationStatusReserved,
		PrivacySetting: effectivePrivacySetting,
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
	privacySetting *entity.PrivacySetting,
) (*entity.Reservation, error) {
	// Get settings (with fallback)
	settings := u.getSettingsWithFallback(ctx)

	// バリデーション：即時予約が許可されているか
	if !settings.AllowInstantReservation {
		return nil, entity.ErrInstantReservationNotAllowed
	}

	// バリデーション：予約時間制限
	if durationMinutes < settings.MinReservationMinutes {
		return nil, entity.ErrReservationTooShort
	}
	if durationMinutes > settings.MaxReservationMinutes {
		return nil, entity.ErrReservationTooLong
	}

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

	// プライバシー設定を決定：リクエストで指定されていればそれを使用、未指定ならユーザーのデフォルト設定
	effectivePrivacySetting := privacySetting
	if effectivePrivacySetting == nil {
		effectivePrivacySetting = &user.DefaultPrivacySetting
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
		PrivacySetting: effectivePrivacySetting,
	}

	if err := u.db.WithContext(ctx).Create(reservation).Error; err != nil {
		return nil, err
	}

	return reservation, nil
}

// CheckIn はチェックインを実行します
func (u *reservationUsecase) CheckIn(ctx context.Context, reservationID string) (*entity.Reservation, error) {
	// Get settings
	settings := u.getSettingsWithFallback(ctx)

	// 予約を取得
	reservation, err := u.reservationRepo.FindByID(ctx, reservationID)
	if err != nil {
		return nil, entity.ErrReservationNotFound
	}

	// チェックイン可能か確認
	if !reservation.CanCheckIn() {
		return nil, entity.ErrCannotCheckIn
	}

	// バリデーション：チェックイン時間窓
	now := time.Now()
	earliestCheckIn := reservation.StartTime.Add(-time.Duration(settings.CheckInMinutesBeforeStart) * time.Minute)
	latestCheckIn := reservation.StartTime.Add(time.Duration(settings.CheckInGracePeriodMinutes) * time.Minute)

	if now.Before(earliestCheckIn) {
		return nil, entity.ErrCheckInTooEarly
	}
	if now.After(latestCheckIn) {
		return nil, entity.ErrCheckInDeadlinePassed
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
	// Get settings
	settings := u.getSettingsWithFallback(ctx)

	// 予約を取得
	reservation, err := u.reservationRepo.FindByID(ctx, reservationID)
	if err != nil {
		return nil, entity.ErrReservationNotFound
	}

	// キャンセル可能か確認
	if !reservation.CanCancel() {
		return nil, entity.ErrCannotCancel
	}

	// バリデーション：キャンセル期限
	now := time.Now()
	deadline := reservation.StartTime.Add(-time.Duration(settings.CancellationDeadlineMinutes) * time.Minute)
	if now.After(deadline) {
		return nil, entity.ErrCancellationDeadlinePassed
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
	// Get settings
	settings := u.getSettingsWithFallback(ctx)

	// 予約を取得
	reservation, err := u.reservationRepo.FindByID(ctx, reservationID)
	if err != nil {
		return nil, entity.ErrReservationNotFound
	}

	// 延長可能か確認
	if !reservation.CanExtend() {
		return nil, entity.ErrCannotExtend
	}

	// バリデーション：延長回数上限
	if reservation.ExtensionCount >= settings.MaxExtensionCount {
		return nil, entity.ErrExtensionLimitExceeded
	}

	// バリデーション：延長時間上限
	if additionalMinutes > settings.MaxExtensionMinutes {
		return nil, entity.ErrExtensionTimeTooLong
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

	// 可視性フィルタリング
	filteredReservations := make([]*entity.Reservation, 0)

	for _, res := range reservations {
		// 自分の予約は常に表示
		if res.UserID == viewerUserID {
			filteredReservations = append(filteredReservations, res)
			continue
		}

		// in_use/reserved 状態はプライバシー設定を無視して全員に見せる
		// （座席の使用状況を表示するため）
		if res.Status == entity.ReservationStatusInUse || res.Status == entity.ReservationStatusReserved {
			filteredReservations = append(filteredReservations, res)
			continue
		}

		// completed/cancelled/no_show は表示しない
		if res.Status == entity.ReservationStatusCompleted ||
			res.Status == entity.ReservationStatusCancelled ||
			res.Status == entity.ReservationStatusNoShow {
			continue
		}

		// その他のステータスについては既存のプライバシー設定ロジックを適用
		// プライバシー設定を取得
		privacySetting := res.PrivacySetting
		if privacySetting == nil {
			privacySetting = &res.User.DefaultPrivacySetting
		}

		// private の場合は表示しない
		if *privacySetting == entity.PrivacyPrivate {
			continue
		}

		// friends 設定の場合はフレンドシップ確認
		if *privacySetting == entity.PrivacyFriends {
			isFriend, err := u.friendshipRepo.CheckFriendship(ctx, viewerUserID, res.UserID)
			if err != nil || !isFriend {
				continue // フレンドでなければスキップ
			}
		}

		// public またはフレンドが確認済みの場合は表示
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
