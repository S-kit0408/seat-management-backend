package repository

import (
	"context"
	"seat-management-backend/internal/domain/entity"

	"gorm.io/gorm"
)

type FriendshipRepository interface {
	Create(ctx context.Context, friendship *entity.Friendship) error
	FindByID(ctx context.Context, id string) (*entity.Friendship, error)
	Delete(ctx context.Context, id string) error
	DeleteByUserIDs(ctx context.Context, userID1, userID2 string) error

	FindByUserIDs(ctx context.Context, userID1, userID2 string) (*entity.Friendship, error)
	FindFriendsByUserID(ctx context.Context, userID string) ([]*entity.User, error)
	CheckFriendship(ctx context.Context, userID1, userID2 string) (bool, error)

	WithTx(tx *gorm.DB) FriendshipRepository
}
