package repository

import (
	"context"
	"seat-management-backend/internal/domain/entity"
)

type FriendRequestRepository interface {
	// CRUD
	Create(ctx context.Context, request *entity.FriendRequest) error
	FindByID(ctx context.Context, id string) (*entity.FriendRequest, error)
	Update(ctx context.Context, request *entity.FriendRequest) error
	UpdateStatus(ctx context.Context, id string, status entity.RequestStatus) error
	Delete(ctx context.Context, id string) error

	FindByRequesterAndAddressee(ctx context.Context, requesterID, addresseeID string) (*entity.FriendRequest, error)
	FindReceivedRequests(ctx context.Context, userID string, status entity.RequestStatus) ([]*entity.FriendRequest, error)
	FindSentRequests(ctx context.Context, userID string, status entity.RequestStatus) ([]*entity.FriendRequest, error)
}
