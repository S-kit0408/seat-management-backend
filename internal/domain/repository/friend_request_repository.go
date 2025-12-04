package repository

import (
	"context"
	"seat-management-backend/internal/domain/entity"

	"gorm.io/gorm"
)

type FriendRequestRepository interface {
	// CRUD
	Create(ctx context.Context, request *entity.FriendRequest) error
	FindByID(ctx context.Context, id string) (*entity.FriendRequest, error)
	Update(ctx context.Context, request *entity.FriendRequest) error
	Delete(ctx context.Context, id string) error

	FindByRequesterAndAddressee(ctx context.Context, requesterID, addresseeID string) (*entity.FriendRequest, error)
	FindReceivedRequests(ctx context.Context, userID string, status entity.RequestStatus) ([]*entity.FriendRequest, error)
	FindSentRequests(ctx context.Context, userID string, status entity.RequestStatus) ([]*entity.FriendRequest, error)

	WithTx(tx *gorm.DB) FriendRequestRepository
}
