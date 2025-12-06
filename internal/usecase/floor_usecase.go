package usecase

import (
	"context"
	"errors"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
)

type FloorUsecase interface {
	Create(ctx context.Context, floor *entity.Floor) error
	GetByID(ctx context.Context, id string) (*entity.Floor, error)
	GetByName(ctx context.Context, name string) (*entity.Floor, error)
	Update(ctx context.Context, id string, floor *entity.Floor) error
	Delete(ctx context.Context, id string) error

	List(ctx context.Context, limit, offset int) ([]*entity.Floor, error)
	GetAllFloors(ctx context.Context) ([]*entity.Floor, error)
	GetActiveFloors(ctx context.Context) ([]*entity.Floor, error)
}

type floorUsecase struct {
	floorRepo repository.FloorRepository
}

func NewFloorUsecase(fr repository.FloorRepository) FloorUsecase {
	return &floorUsecase{
		floorRepo: fr,
	}
}

func (u *floorUsecase) Create(ctx context.Context, floor *entity.Floor) error {
	if floor.Name == "" {
		return errors.New("フロア名は必須です")
	}

	// 重複チェック
	existingFloor, err := u.floorRepo.FindByName(ctx, floor.Name)
	if err != nil && err != entity.ErrFloorNotFound {
		return err
	}
	if existingFloor != nil {
		return entity.ErrDuplicateFloorName
	}

	return u.floorRepo.Create(ctx, floor)
}

func (u *floorUsecase) GetByID(ctx context.Context, id string) (*entity.Floor, error) {
	if id == "" {
		return nil, errors.New("フロアIDは必須です")
	}
	return u.floorRepo.FindByID(ctx, id)
}

func (u *floorUsecase) GetByName(ctx context.Context, name string) (*entity.Floor, error) {
	if name == "" {
		return nil, errors.New("フロア名は必須です")
	}
	return u.floorRepo.FindByName(ctx, name)
}

func (u *floorUsecase) Update(ctx context.Context, id string, floor *entity.Floor) error {
	// 既存のフロアを取得
	existingFloor, err := u.floorRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// フロア名が変更される場合、重複チェック
	if floor.Name != "" && floor.Name != existingFloor.Name {
		duplicate, err := u.floorRepo.FindByName(ctx, floor.Name)
		if err != nil && err != entity.ErrFloorNotFound {
			return err
		}
		if duplicate != nil {
			return entity.ErrDuplicateFloorName
		}
		existingFloor.Name = floor.Name
	}

	// その他のフィールドを更新
	if floor.DisplayName != "" {
		existingFloor.DisplayName = floor.DisplayName
	}
	if floor.Description != "" {
		existingFloor.Description = floor.Description
	}

	// SortOrderは0も有効な値なので、必ず更新
	existingFloor.SortOrder = floor.SortOrder

	// IsActiveも必ず更新
	existingFloor.IsActive = floor.IsActive

	return u.floorRepo.Update(ctx, existingFloor)
}

func (u *floorUsecase) Delete(ctx context.Context, id string) error {
	_, err := u.floorRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	return u.floorRepo.Delete(ctx, id)
}

func (u *floorUsecase) List(ctx context.Context, limit, offset int) ([]*entity.Floor, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return u.floorRepo.List(ctx, limit, offset)
}

func (u *floorUsecase) GetAllFloors(ctx context.Context) ([]*entity.Floor, error) {
	return u.floorRepo.FindAll(ctx)
}

func (u *floorUsecase) GetActiveFloors(ctx context.Context) ([]*entity.Floor, error) {
	return u.floorRepo.FindAllActive(ctx)
}
