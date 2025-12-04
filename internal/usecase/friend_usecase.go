package usecase

import (
	"context"
	"errors"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"

	"gorm.io/gorm"
)

type FriendUsecase interface {
	// 申請
	SendFriendRequest(ctx context.Context, requesterID, addresseeID string, message *string) error
	GetReceivedRequests(ctx context.Context, userID string) ([]*entity.FriendRequest, error)
	GetSentRequests(ctx context.Context, userID string) ([]*entity.FriendRequest, error)

	// 申請処理
	AcceptFriendRequest(ctx context.Context, requestID, userID string) error
	RejectFriendRequest(ctx context.Context, requestID, userID string) error
	CancelFriendRequest(ctx context.Context, requestID, userID string) error

	// 管理
	GetFriendsList(ctx context.Context, userID string) ([]*entity.User, error)
	RemoveFriend(ctx context.Context, userID, friendID string) error
	CheckFriendship(ctx context.Context, userID, targetID string) (bool, error)
}

type friendUsecase struct {
	friendRequestRepo repository.FriendRequestRepository
	friendshipRepo    repository.FriendshipRepository
	userRepo          repository.UserRepository
	db                *gorm.DB
}

func NewFriendUsecase(
	frRepo repository.FriendRequestRepository,
	fRepo repository.FriendshipRepository,
	uRepo repository.UserRepository,
	db *gorm.DB,
) FriendUsecase {
	return &friendUsecase{
		friendRequestRepo: frRepo,
		friendshipRepo:    fRepo,
		userRepo:          uRepo,
		db:                db,
	}
}

// フレンド申請の送信
func (u *friendUsecase) SendFriendRequest(ctx context.Context, requesterID, addresseeID string, message *string) error {
	// 自分自身への申請チェック
	if requesterID == addresseeID {
		return entity.ErrCannotSendToSelf
	}

	// 相手が存在するかチェック
	_, err := u.userRepo.FindByID(ctx, addresseeID)
	if err != nil {
		return err
	}

	// 既存の申請チェック（双方向）
	existingRequest, err := u.friendRequestRepo.FindByRequesterAndAddressee(ctx, requesterID, addresseeID)
	if err != nil {
		return err
	}
	if existingRequest != nil {
		if existingRequest.Status == entity.RequestStatusPending {
			return entity.ErrFriendRequestAlreadyExists
		}
		// Accepted/Rejected/Cancelled の場合は既存レコードを再利用
		existingRequest.Status = entity.RequestStatusPending
		existingRequest.Message = message
		return u.friendRequestRepo.Update(ctx, existingRequest)
	}

	// 逆方向の申請チェック
	reverseRequest, err := u.friendRequestRepo.FindByRequesterAndAddressee(ctx, addresseeID, requesterID)
	if err != nil {
		return err
	}
	if reverseRequest != nil && reverseRequest.Status == entity.RequestStatusPending {
		return errors.New("相手から既に申請が届いています")
	}

	// 既にフレンドかチェック
	isFriend, err := u.friendshipRepo.CheckFriendship(ctx, requesterID, addresseeID)
	if err != nil {
		return err
	}
	if isFriend {
		return entity.ErrAlreadyFriends
	}

	// 申請を作成
	request := &entity.FriendRequest{
		RequesterID: requesterID,
		AddresseeID: addresseeID,
		Status:      entity.RequestStatusPending,
		Message:     message,
	}

	return u.friendRequestRepo.Create(ctx, request)
}

// 受信した申請を取得
func (u *friendUsecase) GetReceivedRequests(ctx context.Context, userID string) ([]*entity.FriendRequest, error) {
	return u.friendRequestRepo.FindReceivedRequests(ctx, userID, entity.RequestStatusPending)
}

// 送信した申請を取得
func (u *friendUsecase) GetSentRequests(ctx context.Context, userID string) ([]*entity.FriendRequest, error) {
	return u.friendRequestRepo.FindSentRequests(ctx, userID, entity.RequestStatusPending)
}

// 申請を承認
func (u *friendUsecase) AcceptFriendRequest(ctx context.Context, requestID, userID string) error {
	// トランザクション開始
	tx := u.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	txFriendRequestRepo := u.friendRequestRepo.WithTx(tx)
	txFriendshipRepo := u.friendshipRepo.WithTx(tx)

	// 申請を取得
	request, err := txFriendRequestRepo.FindByID(ctx, requestID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 権限チェック（受信者のみ承認可能）
	if request.AddresseeID != userID {
		tx.Rollback()
		return entity.ErrUnauthorized
	}

	// ステータスチェック
	if !request.IsPending() {
		tx.Rollback()
		return entity.ErrInvalidRequestStatus
	}

	// 申請を承認状態に更新
	request.Accept()
	if err := txFriendRequestRepo.Update(ctx, request); err != nil {
		tx.Rollback()
		return err
	}

	// フレンドシップを作成
	friendship := &entity.Friendship{
		UserID1: request.RequesterID,
		UserID2: request.AddresseeID,
	}
	if err := txFriendshipRepo.Create(ctx, friendship); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// 申請を拒否
func (u *friendUsecase) RejectFriendRequest(ctx context.Context, requestID, userID string) error {
	request, err := u.friendRequestRepo.FindByID(ctx, requestID)
	if err != nil {
		return err
	}

	if request.AddresseeID != userID {
		return entity.ErrUnauthorized
	}

	if !request.IsPending() {
		return entity.ErrInvalidRequestStatus
	}

	request.Reject()
	return u.friendRequestRepo.Update(ctx, request)
}

// 申請をキャンセル
func (u *friendUsecase) CancelFriendRequest(ctx context.Context, requestID, userID string) error {
	request, err := u.friendRequestRepo.FindByID(ctx, requestID)
	if err != nil {
		return err
	}

	// 送信者のみキャンセル可能
	if request.RequesterID != userID {
		return entity.ErrUnauthorized
	}

	if !request.IsPending() {
		return entity.ErrInvalidRequestStatus
	}

	request.Cancel()
	return u.friendRequestRepo.Update(ctx, request)
}

// フレンドリストを取得
func (u *friendUsecase) GetFriendsList(ctx context.Context, userID string) ([]*entity.User, error) {
	return u.friendshipRepo.FindFriendsByUserID(ctx, userID)
}

// フレンドを解除
func (u *friendUsecase) RemoveFriend(ctx context.Context, userID, friendID string) error {
	// フレンド関係を確認
	isFriend, err := u.friendshipRepo.CheckFriendship(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if !isFriend {
		return errors.New("フレンド関係が存在しません")
	}

	return u.friendshipRepo.DeleteByUserIDs(ctx, userID, friendID)
}

// フレンド関係を確認
func (u *friendUsecase) CheckFriendship(ctx context.Context, userID, targetID string) (bool, error) {
	return u.friendshipRepo.CheckFriendship(ctx, userID, targetID)
}
