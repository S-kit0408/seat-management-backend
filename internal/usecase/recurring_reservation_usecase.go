package usecase

import (
	"context"
	"fmt"
	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
	"time"

	"gorm.io/gorm"
)

// RecurringReservationUsecase は定期予約関連のビジネスロジックを定義します
type RecurringReservationUsecase interface {
	// 定期予約作成・管理
	CreateRecurringReservation(ctx context.Context, userID, seatID string, daysOfWeek int, startTime, endTime string, validFrom time.Time, validUntil *time.Time, privacySetting entity.PrivacySetting, autoExtend bool) (*entity.RecurringReservation, error)
	UpdateRecurringReservation(ctx context.Context, id string, updates map[string]interface{}) (*entity.RecurringReservation, error)
	DisableRecurringReservation(ctx context.Context, id string) (*entity.RecurringReservation, error)
	DeleteRecurringReservation(ctx context.Context, id string) error

	// 定期予約取得
	GetRecurringReservationByID(ctx context.Context, id string) (*entity.RecurringReservation, error)
	GetMyRecurringReservations(ctx context.Context, userID string) ([]*entity.RecurringReservation, error)
	GetActiveRecurringReservations(ctx context.Context, userID string) ([]*entity.RecurringReservation, error)
	GetRecurringReservationsBySeatID(ctx context.Context, seatID string) ([]*entity.RecurringReservation, error)
	CountActiveRecurringReservations(ctx context.Context, userID string) (int64, error)

	// 予約生成
	GenerateReservationsForDate(ctx context.Context, date time.Time) error
	GenerateReservationsForDateRange(ctx context.Context, startDate, endDate time.Time) error
}

type recurringReservationUsecase struct {
	recurringReservationRepo repository.RecurringReservationRepository
	reservationRepo          repository.ReservationRepository
	userRepo                 repository.UserRepository
	seatRepo                 repository.SeatRepository
	db                       *gorm.DB
}

func NewRecurringReservationUsecase(
	recurringReservationRepo repository.RecurringReservationRepository,
	reservationRepo repository.ReservationRepository,
	userRepo repository.UserRepository,
	seatRepo repository.SeatRepository,
	db *gorm.DB,
) RecurringReservationUsecase {
	return &recurringReservationUsecase{
		recurringReservationRepo: recurringReservationRepo,
		reservationRepo:          reservationRepo,
		userRepo:                 userRepo,
		seatRepo:                 seatRepo,
		db:                       db,
	}
}

// CreateRecurringReservation は定期予約を作成します
func (u *recurringReservationUsecase) CreateRecurringReservation(
	ctx context.Context,
	userID string,
	seatID string,
	daysOfWeek int,
	startTime string, // "09:00:00" 形式
	endTime string, // "18:00:00" 形式
	validFrom time.Time,
	validUntil *time.Time,
	privacySetting entity.PrivacySetting,
	autoExtend bool,
) (*entity.RecurringReservation, error) {
	// バリデーション：daysOfWeek（1-127）
	if daysOfWeek < 1 || daysOfWeek > 127 {
		return nil, entity.ErrInvalidDaysOfWeek
	}

	// ユーザー・座席存在確認
	_, err := u.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, entity.ErrUserNotFound
	}

	_, err = u.seatRepo.FindByID(ctx, seatID)
	if err != nil {
		return nil, entity.ErrSeatNotFound
	}

	// プライバシー設定の検証
	if !privacySetting.IsValid() {
		return nil, fmt.Errorf("invalid privacy setting: %s", privacySetting)
	}

	// startTime < endTime を検証（時刻文字列として）
	if startTime >= endTime {
		return nil, entity.ErrInvalidTimeRange
	}

	// validFrom <= validUntil を検証
	if validUntil != nil {
		validFromDate := time.Date(validFrom.Year(), validFrom.Month(), validFrom.Day(), 0, 0, 0, 0, validFrom.Location())
		validUntilDate := time.Date(validUntil.Year(), validUntil.Month(), validUntil.Day(), 0, 0, 0, 0, validUntil.Location())
		if validFromDate.After(validUntilDate) {
			return nil, entity.ErrInvalidTimeRange
		}
	}

	// 定期予約作成
	recurringReservation := &entity.RecurringReservation{
		UserID:         userID,
		SeatID:         seatID,
		DaysOfWeek:     daysOfWeek,
		StartTime:      startTime,
		EndTime:        endTime,
		ValidFrom:      validFrom,
		ValidUntil:     validUntil,
		PrivacySetting: privacySetting,
		AutoExtend:     autoExtend,
		IsActive:       true,
	}

	// 保存
	if err := u.recurringReservationRepo.Create(ctx, recurringReservation); err != nil {
		return nil, err
	}

	return recurringReservation, nil
}

// UpdateRecurringReservation は定期予約を更新します
func (u *recurringReservationUsecase) UpdateRecurringReservation(
	ctx context.Context,
	id string,
	updates map[string]interface{},
) (*entity.RecurringReservation, error) {
	// 既存の定期予約を取得
	recurringReservation, err := u.recurringReservationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, entity.ErrRecurringReservationNotFound
	}

	// 部分更新（渡されたフィールドのみ更新）
	if err := u.db.WithContext(ctx).Model(recurringReservation).Updates(updates).Error; err != nil {
		return nil, err
	}

	return recurringReservation, nil
}

// DisableRecurringReservation は定期予約を無効化します（削除ではなく無効化）
func (u *recurringReservationUsecase) DisableRecurringReservation(
	ctx context.Context,
	id string,
) (*entity.RecurringReservation, error) {
	recurringReservation, err := u.recurringReservationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, entity.ErrRecurringReservationNotFound
	}

	recurringReservation.IsActive = false

	if err := u.recurringReservationRepo.Update(ctx, recurringReservation); err != nil {
		return nil, err
	}

	return recurringReservation, nil
}

// DeleteRecurringReservation は定期予約を削除します
func (u *recurringReservationUsecase) DeleteRecurringReservation(
	ctx context.Context,
	id string,
) error {
	// 存在確認
	_, err := u.recurringReservationRepo.FindByID(ctx, id)
	if err != nil {
		return entity.ErrRecurringReservationNotFound
	}

	// 削除
	if err := u.recurringReservationRepo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

// GetMyRecurringReservations はユーザーの定期予約を取得します
func (u *recurringReservationUsecase) GetMyRecurringReservations(
	ctx context.Context,
	userID string,
) ([]*entity.RecurringReservation, error) {
	recurringReservations, err := u.recurringReservationRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return recurringReservations, nil
}

// GetActiveRecurringReservations はユーザーのアクティブな定期予約を取得します
func (u *recurringReservationUsecase) GetActiveRecurringReservations(
	ctx context.Context,
	userID string,
) ([]*entity.RecurringReservation, error) {
	recurringReservations, err := u.recurringReservationRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return recurringReservations, nil
}

// GenerateReservationsForDate は指定日付に対して有効な定期予約から予約を生成します
// 最も重要なメソッド
func (u *recurringReservationUsecase) GenerateReservationsForDate(
	ctx context.Context,
	date time.Time,
) error {
	// 指定日付に対して有効な定期予約を取得
	recurringReservations, err := u.recurringReservationRepo.FindActiveForDate(ctx, date)
	if err != nil {
		return err
	}

	// トランザクション開始
	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 各定期予約から予約を生成
	for _, rr := range recurringReservations {
		// 指定日が有効か確認
		if !rr.ShouldGenerateReservation(date) {
			continue
		}

		// startTime と endTime を構築
		// time文字列 "09:00:00" から time.Time に変換
		startTimeStr := rr.StartTime
		endTimeStr := rr.EndTime

		// 日付とtime文字列を結合
		startTimeObj, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%04d-%02d-%02d %s", date.Year(), date.Month(), date.Day(), startTimeStr))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to parse start time: %w", err)
		}

		endTimeObj, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%04d-%02d-%02d %s", date.Year(), date.Month(), date.Day(), endTimeStr))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to parse end time: %w", err)
		}

		// 重複チェック（他の予約が既にあるかチェック）
		// 定期予約同士の重複は許可（異なるユーザーが同じ座席を予約できる）
		// しかし、同一ユーザーの同一座席での重複は避ける（オプション）

		// 予約作成
		reservation := &entity.Reservation{
			UserID:                 rr.UserID,
			SeatID:                 rr.SeatID,
			Type:                   entity.ReservationTypeRecurring,
			StartTime:              startTimeObj,
			EndTime:                endTimeObj,
			Status:                 entity.ReservationStatusReserved,
			PrivacySetting:         &rr.PrivacySetting,
			RecurringReservationID: &rr.ID,
			AutoExtend:             rr.AutoExtend,
		}

		// 保存
		if err := tx.Create(reservation).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create reservation for recurring %s: %w", rr.ID, err)
		}
	}

	// コミット
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

// GenerateReservationsForDateRange は指定期間の日付ごとに予約を生成します
func (u *recurringReservationUsecase) GenerateReservationsForDateRange(
	ctx context.Context,
	startDate time.Time,
	endDate time.Time,
) error {
	if startDate.After(endDate) {
		return entity.ErrInvalidTimeRange
	}

	// startDate から endDate まで日毎にループ
	currentDate := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endDateOnly := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())

	for currentDate.Before(endDateOnly) || currentDate.Equal(endDateOnly) {
		if err := u.GenerateReservationsForDate(ctx, currentDate); err != nil {
			return fmt.Errorf("failed to generate reservations for %s: %w", currentDate.Format("2006-01-02"), err)
		}
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return nil
}

// GetRecurringReservationByID は定期予約IDで定期予約を取得します
func (u *recurringReservationUsecase) GetRecurringReservationByID(
	ctx context.Context,
	id string,
) (*entity.RecurringReservation, error) {
	recurringReservation, err := u.recurringReservationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, entity.ErrRecurringReservationNotFound
	}
	return recurringReservation, nil
}

// CountActiveRecurringReservations はユーザーのアクティブな定期予約数をカウントします
func (u *recurringReservationUsecase) CountActiveRecurringReservations(
	ctx context.Context,
	userID string,
) (int64, error) {
	count, err := u.recurringReservationRepo.CountActiveByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetRecurringReservationsBySeatID は座席IDで定期予約を取得します
func (u *recurringReservationUsecase) GetRecurringReservationsBySeatID(
	ctx context.Context,
	seatID string,
) ([]*entity.RecurringReservation, error) {
	recurringReservations, err := u.recurringReservationRepo.FindBySeatID(ctx, seatID)
	if err != nil {
		return nil, err
	}
	return recurringReservations, nil
}
