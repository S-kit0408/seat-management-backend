package usecase

import (
	"context"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
)

// UserUsecase はユーザー関連のビジネスロジックを定義
type UserUsecase interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id string) (*entity.User, error)
	GetByClerkUserID(ctx context.Context, clerkUserID string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	UpdateLastLogin(ctx context.Context, userID string) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*entity.User, error)
}

// userUsecase はUserUsecaseの実装
type userUsecase struct {
	userRepo repository.UserRepository
}

// NewUserUsecase はUserUsecaseの新しいインスタンスを作成
func NewUserUsecase(ur repository.UserRepository) UserUsecase {
	return &userUsecase{
		userRepo: ur,
	}
}

// Create は新しいユーザーを作成
func (u *userUsecase) Create(ctx context.Context, user *entity.User) error {
	// ビジネスロジック: デフォルト値の設定など
	if user.DefaultPrivacySetting == "" {
		user.DefaultPrivacySetting = entity.PrivacyPrivate
	}
	if user.PrimaryAuthProvider == "" {
		user.PrimaryAuthProvider = entity.AuthProviderUnknown
	}

	return u.userRepo.Create(ctx, user)
}

// GetByID はIDでユーザーを取得
func (u *userUsecase) GetByID(ctx context.Context, id string) (*entity.User, error) {
	return u.userRepo.FindByID(ctx, id)
}

// GetByClerkUserID はClerk User IDでユーザーを取得
func (u *userUsecase) GetByClerkUserID(ctx context.Context, clerkUserID string) (*entity.User, error) {
	return u.userRepo.FindByClerkUserID(ctx, clerkUserID)
}

// GetByEmail はEmailでユーザーを取得
func (u *userUsecase) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return u.userRepo.FindByEmail(ctx, email)
}

// Update はユーザー情報を更新
func (u *userUsecase) Update(ctx context.Context, user *entity.User) error {
	// ビジネスロジック: バリデーションなど
	if user.Email == "" {
		return entity.ErrInvalidEmail
	}
	if user.Name == "" {
		return entity.ErrInvalidName
	}

	return u.userRepo.Update(ctx, user)
}

// UpdateLastLogin は最終ログイン時刻を更新
func (u *userUsecase) UpdateLastLogin(ctx context.Context, userID string) error {
	return u.userRepo.UpdateLastLogin(ctx, userID)
}

// Delete はユーザーを削除（ソフトデリート）
func (u *userUsecase) Delete(ctx context.Context, id string) error {
	return u.userRepo.Delete(ctx, id)
}

// List はユーザー一覧を取得
func (u *userUsecase) List(ctx context.Context, limit, offset int) ([]*entity.User, error) {
	// ビジネスロジック: limitの最大値チェックなど
	if limit <= 0 || limit > 100 {
		limit = 20 // デフォルト値
	}
	if offset < 0 {
		offset = 0
	}

	return u.userRepo.List(ctx, limit, offset)
}
