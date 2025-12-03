package persistence

import (
	"context"
	"errors"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"

	"gorm.io/gorm"
)

type friendRequestRepository struct {
	db *gorm.DB
}

func NewFriendRequestRepository(db *gorm.DB) repository.FriendRequestRepository {
	return &friendRequestRepository{db: db}
}

func (r *friendRequestRepository) Create(ctx context.Context, request *entity.FriendRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

func (r *friendRequestRepository) FindByID(ctx context.Context, id string) (*entity.FriendRequest, error) {
	var request entity.FriendRequest
	err := r.db.WithContext(ctx).
		Preload("Requester").
		Preload("Addressee").
		Where("id = ?", id).
		First(&request).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrFriendRequestNotFound
		}
		return nil, err
	}

	return &request, nil
}

func (r *friendRequestRepository) Update(ctx context.Context, request *entity.FriendRequest) error {
	return r.db.WithContext(ctx).Save(request).Error
}

func (r *friendRequestRepository) UpdateStatus(ctx context.Context, id string, status entity.RequestStatus) error {
	return r.db.WithContext(ctx).
		Model(&entity.FriendRequest{}).
		Where("id = ?", id).
		Update("status", status).
		Error
}

func (r *friendRequestRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.FriendRequest{}, "id = ?", id).Error
}

func (r *friendRequestRepository) FindByRequesterAndAddressee(ctx context.Context, requesterID, addresseeID string) (*entity.FriendRequest, error) {
	var request entity.FriendRequest
	err := r.db.WithContext(ctx).
		Where("requester_id = ? AND addressee_id = ?", requesterID, addresseeID).
		First(&request).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 存在しない場合はnilを返す
		}
		return nil, err
	}

	return &request, nil
}

func (r *friendRequestRepository) FindReceivedRequests(ctx context.Context, userID string, status entity.RequestStatus) ([]*entity.FriendRequest, error) {
	var requests []*entity.FriendRequest
	query := r.db.WithContext(ctx).
		Preload("Requester").
		Where("addressee_id = ?", userID)

	// statusが空文字列でない場合のみフィルタリング
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Order("created_at DESC").Find(&requests).Error
	return requests, err
}

func (r *friendRequestRepository) FindSentRequests(ctx context.Context, userID string, status entity.RequestStatus) ([]*entity.FriendRequest, error) {
	var requests []*entity.FriendRequest
	query := r.db.WithContext(ctx).
		Preload("Addressee").
		Where("requester_id = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Order("created_at DESC").Find(&requests).Error
	return requests, err
}
