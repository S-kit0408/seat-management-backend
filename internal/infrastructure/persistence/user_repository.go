package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository はUserRepositoryの実装を返す
func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &userRepository{db: db}
}

// Create はユーザーを作成
func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// FindByID はIDでユーザーを検索
func (r *userRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// FindByClerkUserID はClerk User IDでユーザーを検索
func (r *userRepository) FindByClerkUserID(ctx context.Context, clerkUserID string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("clerk_user_id = ?", clerkUserID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail はEmailでユーザーを検索
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// Update はユーザー情報を更新
func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdateLastLogin は最終ログイン時刻を更新
func (r *userRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", userID).
		Update("last_login_at", time.Now()).
		Error
}

// Delete はユーザーを削除（ソフトデリート）
func (r *userRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.User{}, "id = ?", id).Error
}

// List はユーザー一覧を取得
func (r *userRepository) List(ctx context.Context, limit, offset int) ([]*entity.User, error) {
	var users []*entity.User
	err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Find(&users).Error
	return users, err
}
