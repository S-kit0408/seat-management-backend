package entity

import "errors"

var (
	// ユーザー関連のエラー
	ErrUserNotFound     = errors.New("ユーザーが見つかりません")
	ErrInvalidEmail     = errors.New("無効なメールアドレスです")
	ErrInvalidName      = errors.New("無効な名前です")
	ErrInvalidRole      = errors.New("無効なロールです")
	ErrDuplicateEmail   = errors.New("このメールアドレスは既に使用されています")
	ErrDuplicateClerkID = errors.New("このClerk IDは既に使用されています")
	ErrUnauthorized     = errors.New("権限がありません")

	// フレンド機能関連のエラー
	ErrFriendRequestAlreadyExists = errors.New("すでにリクエストが存在しています")
	ErrFriendRequestNotFound      = errors.New("リクエストが見つかりません")
	ErrFriendshipNotFound         = errors.New("フレンド関係が見つかりません")
	ErrAlreadyFriends             = errors.New("すでにフレンドです")
	ErrCannotSendToSelf           = errors.New("自身へのリクエスト送信は行えません")
	ErrInvalidRequestStatus       = errors.New("無効なリクエストステータスです")
)
