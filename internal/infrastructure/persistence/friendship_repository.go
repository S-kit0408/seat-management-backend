package persistence

import (
	"context"
	"errors"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"

	"gorm.io/gorm"
)

type friendshipRepository struct {
	db *gorm.DB
}

func NewFriendshipRepository(db *gorm.DB) repository.FriendshipRepository {
	return &friendshipRepository{db: db}
}

func (r *friendshipRepository) Create(ctx context.Context, friendship *entity.Friendship) error {
	return r.db.WithContext(ctx).Create(friendship).Error
}

func (r *friendshipRepository) FindByID(ctx context.Context, id string) (*entity.Friendship, error) {
	var friendship entity.Friendship
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&friendship).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrFriendshipNotFound
		}
		return nil, err
	}
	return &friendship, nil
}

func (r *friendshipRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Friendship{}, "id = ?", id).Error
}

func (r *friendshipRepository) DeleteByUserIDs(ctx context.Context, userID1, userID2 string) error {
	// 小さいIDをuser_id1に
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	return r.db.WithContext(ctx).
		Where("user_id1 = ? AND user_id2 = ?", userID1, userID2).
		Delete(&entity.Friendship{}).Error
}

func (r *friendshipRepository) FindByUserIDs(ctx context.Context, userID1, userID2 string) (*entity.Friendship, error) {
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	var friendship entity.Friendship
	err := r.db.WithContext(ctx).
		Where("user_id1 = ? AND user_id2 = ?", userID1, userID2).
		First(&friendship).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &friendship, nil
}

func (r *friendshipRepository) FindFriendsByUserID(ctx context.Context, userID string) ([]*entity.User, error) {
	var users []*entity.User

	// サブクエリでフレンドのIDを取得し、usersテーブルと結合
	err := r.db.WithContext(ctx).
		Raw(`
                        SELECT u.* FROM users u
                        INNER JOIN friendships f ON (
                                (f.user_id1 = ? AND f.user_id2 = u.id) OR
                                (f.user_id2 = ? AND f.user_id1 = u.id)
                        )
                        WHERE u.deleted_at IS NULL
                        ORDER BY f.created_at DESC
                `, userID, userID).
		Scan(&users).Error

	return users, err
}

func (r *friendshipRepository) CheckFriendship(ctx context.Context, userID1, userID2 string) (bool, error) {
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.Friendship{}).
		Where("user_id1 = ? AND user_id2 = ?", userID1, userID2).
		Count(&count).Error

	return count > 0, err
}

func (r *friendshipRepository) WithTx(tx *gorm.DB) repository.FriendshipRepository {
	return &friendshipRepository{db: tx}
}
