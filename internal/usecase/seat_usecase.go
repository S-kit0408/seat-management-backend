package usecase

import (
	"context"
	"errors"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
)

type SeatUsecase interface {
	Create(ctx context.Context, seat *entity.Seat) error
	GetByID(ctx context.Context, id string) (*entity.Seat, error)
	GetBySeatNumber(ctx context.Context, seatNumber string) (*entity.Seat, error)
	Update(ctx context.Context, id string, seat *entity.Seat) error
	Delete(ctx context.Context, id string) error

	List(ctx context.Context, limit, offset int) ([]*entity.Seat, error)
	GetAllSeats(ctx context.Context) ([]*entity.Seat, error)
	GetActiveSeats(ctx context.Context) ([]*entity.Seat, error)
	GetActiveSeatsWithFloor(ctx context.Context) ([]*entity.Seat, error)

	GetSeatsByFloorID(ctx context.Context, floorID string) ([]*entity.Seat, error)
	GetSeatsBySpaceID(ctx context.Context, spaceID string) ([]*entity.Seat, error)
	GetUnassignedSeats(ctx context.Context) ([]*entity.Seat, error)
}

type seatUsecase struct {
	seatRepo repository.SeatRepository
}

func NewSeatUsecase(sr repository.SeatRepository) SeatUsecase {
	return &seatUsecase{
		seatRepo: sr,
	}
}

func (u *seatUsecase) Create(ctx context.Context, seat *entity.Seat) error {
	if seat.SeatNumber == "" {
		return errors.New("座席番号は必須です")
	}

	if !seat.ValidateShape() {
		return entity.ErrInvalidSeatShape
	}

	if !seat.ValidateRotationAngle() {
		return entity.ErrInvalidRotation
	}

	existingSeat, err := u.seatRepo.FindBySeatNumber(ctx, seat.SeatNumber)
	if err != nil && err != entity.ErrSeatNotFound {
		return err
	}
	if existingSeat != nil {
		return entity.ErrDuplicateSeatNumber
	}

	if seat.Shape == "" {
		seat.Shape = entity.SeatShapeRectangle
	}
	if seat.Width == 0 {
		seat.Width = 100
	}
	if seat.Height == 0 {
		seat.Height = 100
	}

	return u.seatRepo.Create(ctx, seat)
}

// IDで座席取得
func (u *seatUsecase) GetByID(ctx context.Context, id string) (*entity.Seat, error) {
	if id == "" {
		return nil, errors.New("座席IDは必須です")
	}
	return u.seatRepo.FindByID(ctx, id)
}

// 座席番号で取得
func (u *seatUsecase) GetBySeatNumber(ctx context.Context, seatNumber string) (*entity.Seat, error) {
	if seatNumber == "" {
		return nil, errors.New("座席番号は必須です")
	}
	return u.seatRepo.FindBySeatNumber(ctx, seatNumber)
}

// 座席情報更新
func (u *seatUsecase) Update(ctx context.Context, id string, seat *entity.Seat) error {
	// 既存の座席を取得
	existingSeat, err := u.seatRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// 座席番号が変更される場合
	if seat.SeatNumber != "" && seat.SeatNumber != existingSeat.SeatNumber {
		duplicate, err := u.seatRepo.FindBySeatNumber(ctx, seat.SeatNumber)
		if err != nil && err != entity.ErrSeatNotFound {
			return err
		}
		if duplicate != nil {
			return entity.ErrDuplicateSeatNumber
		}
		existingSeat.SeatNumber = seat.SeatNumber
	}

	// その他のフィールドを更新
	if seat.Description != "" {
		existingSeat.Description = seat.Description
	}

	// 位置情報の更新
	if seat.PositionX != 0 || seat.PositionY != 0 {
		existingSeat.PositionX = seat.PositionX
		existingSeat.PositionY = seat.PositionY
	}

	// 回転角度の更新
	if seat.RotationAngle != existingSeat.RotationAngle {
		if !seat.ValidateRotationAngle() {
			return entity.ErrInvalidRotation
		}
		existingSeat.RotationAngle = seat.RotationAngle
	}

	if seat.Width != 0 {
		existingSeat.Width = seat.Width
	}
	if seat.Height != 0 {
		existingSeat.Height = seat.Height
	}

	// 形状の更新
	if seat.Shape != "" && seat.Shape != existingSeat.Shape {
		if !seat.ValidateShape() {
			return entity.ErrInvalidSeatShape
		}
		existingSeat.Shape = seat.Shape
	}

	// 属性の更新
	if seat.Attributes != nil {
		existingSeat.Attributes = seat.Attributes
	}

	// フロア/スペースIDの更新
	if seat.FloorID != nil {
		existingSeat.FloorID = seat.FloorID
	}
	if seat.SpaceID != nil {
		existingSeat.SpaceID = seat.SpaceID
	}

	existingSeat.IsActive = seat.IsActive

	return u.seatRepo.Update(ctx, existingSeat)
}

// 座席削除
func (u *seatUsecase) Delete(ctx context.Context, id string) error {
	_, err := u.seatRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	return u.seatRepo.Delete(ctx, id)
}

// 座席一覧取得
func (u *seatUsecase) List(ctx context.Context, limit, offset int) ([]*entity.Seat, error) {
	if limit <= 0 || limit > 100 {
		limit = 100 // デフォルト値
	}
	if offset < 0 {
		offset = 0
	}

	return u.seatRepo.List(ctx, limit, offset)
}

// 全座席取得
func (u *seatUsecase) GetAllSeats(ctx context.Context) ([]*entity.Seat, error) {
	return u.seatRepo.FindAll(ctx)
}

// アクティブな座席のみを取得
func (u *seatUsecase) GetActiveSeats(ctx context.Context) ([]*entity.Seat, error) {
	return u.seatRepo.FindActive(ctx)
}

// アクティブかつフロアが設定されている座席を取得（一般ユーザー向け）
func (u *seatUsecase) GetActiveSeatsWithFloor(ctx context.Context) ([]*entity.Seat, error) {
	return u.seatRepo.FindActiveWithFloor(ctx)
}

// フロアIDで取得
func (u *seatUsecase) GetSeatsByFloorID(ctx context.Context, floorID string) ([]*entity.Seat, error) {
	if floorID == "" {
		return nil, errors.New("フロアIDは必須です")
	}
	return u.seatRepo.FindByFloorID(ctx, floorID)
}

// スペースIDで取得
func (u *seatUsecase) GetSeatsBySpaceID(ctx context.Context, spaceID string) ([]*entity.Seat, error) {
	if spaceID == "" {
		return nil, errors.New("スペースIDは必須です")
	}
	return u.seatRepo.FindBySpaceID(ctx, spaceID)
}

// フロア未設定の座席を取得（管理者向け）
func (u *seatUsecase) GetUnassignedSeats(ctx context.Context) ([]*entity.Seat, error) {
	return u.seatRepo.FindUnassignedSeats(ctx)
}
