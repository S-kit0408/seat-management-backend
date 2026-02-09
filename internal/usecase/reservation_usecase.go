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
	MarkPastReservationsAsNoShow(ctx context.Context) error // バッチ処理用
}

type reservationUsecase struct {
	reservationRepo          repository.ReservationRepository
	recurringReservationRepo repository.RecurringReservationRepository
	userRepo                 repository.UserRepository
	seatRepo                 repository.SeatRepository
	friendshipRepo           repository.FriendshipRepository
	settingsRepo             repository.ReservationSettingsRepository
	floorOperationHoursRepo  repository.FloorOperationHoursRepository
	db                       *gorm.DB
	eventNotifier            *EventNotifier
}

func NewReservationUsecase(
	reservationRepo repository.ReservationRepository,
	recurringReservationRepo repository.RecurringReservationRepository,
	userRepo repository.UserRepository,
	seatRepo repository.SeatRepository,
	friendshipRepo repository.FriendshipRepository,
	settingsRepo repository.ReservationSettingsRepository,
	floorOperationHoursRepo repository.FloorOperationHoursRepository,
	db *gorm.DB,
	eventNotifier *EventNotifier,
) ReservationUsecase {
	return &reservationUsecase{
		reservationRepo:          reservationRepo,
		recurringReservationRepo: recurringReservationRepo,
		userRepo:                 userRepo,
		seatRepo:                 seatRepo,
		friendshipRepo:           friendshipRepo,
		settingsRepo:             settingsRepo,
		floorOperationHoursRepo:  floorOperationHoursRepo,
		db:                       db,
		eventNotifier:            eventNotifier,
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

// validateOperatingHours checks if the reservation time is within floor operating hours
func (u *reservationUsecase) validateOperatingHours(
	ctx context.Context,
	seat *entity.Seat,
	startTime time.Time,
	endTime time.Time,
) error {
	// If seat has no floor, skip operating hours check
	if seat.FloorID == nil {
		return nil
	}

	// Convert to JST (Asia/Tokyo) for day of week calculation
	jst, _ := time.LoadLocation("Asia/Tokyo")
	startTimeJST := startTime.In(jst)
	endTimeJST := endTime.In(jst)

	// Get day of week for start and end times in JST
	startDay := int(startTimeJST.Weekday())
	endDay := int(endTimeJST.Weekday())

	// Check start time
	if err := u.checkTimeAgainstOperatingHours(ctx, *seat.FloorID, startTime, startDay); err != nil {
		return err
	}

	// Check end time (if different day)
	if startDay != endDay {
		if err := u.checkTimeAgainstOperatingHours(ctx, *seat.FloorID, endTime, endDay); err != nil {
			return err
		}
	}

	return nil
}

// checkTimeAgainstOperatingHours checks a specific time against floor operating hours
func (u *reservationUsecase) checkTimeAgainstOperatingHours(
	ctx context.Context,
	floorID string,
	checkTime time.Time,
	dayOfWeek int,
) error {
	// Get operating hours for this floor and day
	hours, err := u.floorOperationHoursRepo.FindByFloorIDAndDayOfWeek(ctx, floorID, dayOfWeek)
	if err != nil {
		// If no operating hours defined, allow reservation (no restrictions)
		if err == entity.ErrFloorOperationHoursNotFound {
			return nil
		}
		return err
	}

	// Check if floor is closed
	if hours.IsClosed {
		return entity.ErrFloorClosed
	}

	// Convert to JST (Asia/Tokyo) before extracting time component
	jst, _ := time.LoadLocation("Asia/Tokyo")
	timeStr := checkTime.In(jst).Format("15:04:05")

	// Compare with operating hours
	if timeStr < hours.OpenTime || timeStr >= hours.CloseTime {
		return entity.ErrOutsideOperatingHours
	}

	return nil
}

// validateReservationTimeWithinOperatingHours checks if reservation duration fits within operating hours
func (u *reservationUsecase) validateReservationTimeWithinOperatingHours(
	ctx context.Context,
	seat *entity.Seat,
	startTime time.Time,
	endTime time.Time,
	settings *entity.ReservationSettings,
) error {
	// If seat has no floor, skip check
	if seat.FloorID == nil {
		return nil
	}

	// Convert to JST (Asia/Tokyo) for day of week calculation
	jst, _ := time.LoadLocation("Asia/Tokyo")
	startTimeJST := startTime.In(jst)
	endTimeJST := endTime.In(jst)

	// Get start day operating hours
	startDay := int(startTimeJST.Weekday())
	hours, err := u.floorOperationHoursRepo.FindByFloorIDAndDayOfWeek(ctx, *seat.FloorID, startDay)
	if err != nil {
		// If no operating hours defined, allow reservation
		if err == entity.ErrFloorOperationHoursNotFound {
			return nil
		}
		return err
	}

	// If closed, already caught by validateOperatingHours
	if hours.IsClosed {
		return nil
	}

	// Check if reservation is on the same day
	endDay := int(endTimeJST.Weekday())
	if startDay == endDay {
		// Single day reservation: check if available time within operating hours is sufficient
		startTimeStr := startTimeJST.Format("15:04:05")
		endTimeStr := endTimeJST.Format("15:04:05")

		// Check if reservation fits within operating hours
		if startTimeStr < hours.OpenTime || endTimeStr > hours.CloseTime {
			return entity.ErrOutsideOperatingHours
		}

		// Check if available time window is sufficient for minimum duration
		// Calculate available minutes from start of operating hours to end of operating hours
		openTime, _ := time.Parse("15:04:05", hours.OpenTime)
		closeTime, _ := time.Parse("15:04:05", hours.CloseTime)
		availableMinutes := int(closeTime.Sub(openTime).Minutes())

		if availableMinutes < settings.MinReservationMinutes {
			return fmt.Errorf("営業時間内で最小予約時間(%d分)を満たす予約ができません", settings.MinReservationMinutes)
		}
	}

	return nil
}

// validateInstantReservationWithinOperatingHours checks if instant reservation fits within remaining operating hours
func (u *reservationUsecase) validateInstantReservationWithinOperatingHours(
	ctx context.Context,
	seat *entity.Seat,
	startTime time.Time,
	endTime time.Time,
) error {
	// If seat has no floor, skip check
	if seat.FloorID == nil {
		return nil
	}

	// Convert to JST (Asia/Tokyo) for day of week calculation
	jst, _ := time.LoadLocation("Asia/Tokyo")
	startTimeJST := startTime.In(jst)
	endTimeJST := endTime.In(jst)

	// For instant reservation, startTime is usually now
	// Check if now is within operating hours
	startDay := int(startTimeJST.Weekday())
	endDay := int(endTimeJST.Weekday())

	// Get start day's operating hours
	hours, err := u.floorOperationHoursRepo.FindByFloorIDAndDayOfWeek(ctx, *seat.FloorID, startDay)
	if err != nil {
		// If no operating hours defined, allow reservation
		if err == entity.ErrFloorOperationHoursNotFound {
			return nil
		}
		return err
	}

	// If closed, already caught by validateOperatingHours
	if hours.IsClosed {
		return nil
	}

	// Compare times as strings for simplicity and accuracy (in JST)
	startTimeStr := startTimeJST.Format("15:04:05")
	endTimeStr := endTimeJST.Format("15:04:05")

	// Check if start time is within operating hours
	if startTimeStr < hours.OpenTime || startTimeStr >= hours.CloseTime {
		return entity.ErrOutsideOperatingHours
	}

	// Check if end time fits within operating hours
	// For same day: end time must be <= close time (inclusive of the exact close time is excluded, so use <)
	if startDay == endDay {
		// Single day: end time must be strictly before close time
		if endTimeStr > hours.CloseTime {
			return fmt.Errorf("営業時間内に予約を完了できません。営業終了時刻: %s", hours.CloseTime)
		}
	} else {
		// Multi-day: check if end time is valid on end day
		endHours, err := u.floorOperationHoursRepo.FindByFloorIDAndDayOfWeek(ctx, *seat.FloorID, endDay)
		if err != nil {
			if err == entity.ErrFloorOperationHoursNotFound {
				return nil
			}
			return err
		}

		if endHours.IsClosed {
			return entity.ErrFloorClosed
		}

		// End time must be before close time on end day
		if endTimeStr > endHours.CloseTime {
			return fmt.Errorf("営業時間内に予約を完了できません。終了日の営業終了時刻: %s", endHours.CloseTime)
		}
	}

	return nil
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
	seat, err := u.seatRepo.FindByID(ctx, seatID)
	if err != nil {
		return nil, entity.ErrSeatNotFound
	}

	// 営業時間チェック
	if err := u.validateOperatingHours(ctx, seat, startTime, endTime); err != nil {
		return nil, err
	}

	// 営業時間内での予約時間制限チェック
	if err := u.validateReservationTimeWithinOperatingHours(ctx, seat, startTime, endTime, settings); err != nil {
		return nil, err
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

	// Notify WebSocket clients
	if u.eventNotifier != nil {
		u.eventNotifier.NotifySeatStatusChange(seatID, "reserved", reservation)
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

	seat, err := u.seatRepo.FindByID(ctx, seatID)
	if err != nil {
		return nil, entity.ErrSeatNotFound
	}

	// 時間設定
	now := time.Now()
	startTime := now
	endTime := now.Add(time.Duration(durationMinutes) * time.Minute)

	// 営業時間チェック
	if err := u.validateOperatingHours(ctx, seat, startTime, endTime); err != nil {
		return nil, err
	}

	// 営業時間内での即時予約実行可能性チェック
	if err := u.validateInstantReservationWithinOperatingHours(ctx, seat, startTime, endTime); err != nil {
		return nil, err
	}

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

	// Notify WebSocket clients
	if u.eventNotifier != nil {
		u.eventNotifier.NotifySeatStatusChange(seatID, "occupied", reservation)
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

	// Notify WebSocket clients
	if u.eventNotifier != nil {
		u.eventNotifier.NotifySeatStatusChange(reservation.SeatID, "occupied", reservation)
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

	// Notify WebSocket clients
	if u.eventNotifier != nil {
		u.eventNotifier.NotifySeatStatusChange(reservation.SeatID, "available", nil)
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

	// Notify WebSocket clients
	if u.eventNotifier != nil {
		u.eventNotifier.NotifySeatStatusChange(reservation.SeatID, "available", nil)
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

	// Notify WebSocket clients
	if u.eventNotifier != nil {
		u.eventNotifier.NotifySeatStatusChange(reservation.SeatID, "occupied", reservation)
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

// MarkPastReservationsAsNoShow はバッチ処理で利用終了時刻を超過してチェックインされていない予約をno-showに変更
// このメソッドは定期的に実行されることを想定（例：毎分実行）
func (u *reservationUsecase) MarkPastReservationsAsNoShow(ctx context.Context) error {
	now := time.Now()
	// 過去の全予約を対象に検索（EndTime が現在時刻より前）
	// StartTimeは遠い過去を指定
	pastTime := now.AddDate(-1, 0, 0) // 1年前から現在まで

	// reserved ステータスの予約で、現在時刻より前の終了時刻を持つものを取得
	reservations, err := u.reservationRepo.FindByTimeRange(
		ctx,
		pastTime,
		now,
		[]entity.ReservationStatus{entity.ReservationStatusReserved},
	)
	if err != nil {
		return fmt.Errorf("failed to fetch reservations: %w", err)
	}

	var processedCount int
	for _, reservation := range reservations {
		// チェックインされていない && 利用終了時刻を超過している
		if reservation.CheckedInAt == nil && reservation.EndTime.Before(now) {
			// no-showに変更
			reservation.MarkAsNoShow()

			// DB に保存
			if err := u.reservationRepo.Update(ctx, reservation); err != nil {
				fmt.Printf("[ERROR] Failed to mark reservation %s as no-show: %v\n", reservation.ID, err)
				continue
			}

			// WebSocket で座席が available になったことを通知
			if u.eventNotifier != nil {
				u.eventNotifier.NotifySeatStatusChange(reservation.SeatID, "available", nil)
			}

			processedCount++
			fmt.Printf("[INFO] Marked reservation %s as no-show (user: %s, seat: %s, end_time: %s)\n",
				reservation.ID, reservation.UserID, reservation.SeatID, reservation.EndTime)
		}
	}

	if processedCount > 0 {
		fmt.Printf("[INFO] No-show batch: %d reservations marked as no-show\n", processedCount)
	}

	return nil
}
