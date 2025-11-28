package entity

import "errors"

var (
	// ユーザー関連のエラー
	ErrUserNotFound     = errors.New("ユーザーが見つかりません")
	ErrInvalidEmail     = errors.New("無効なメールアドレスです")
	ErrInvalidName      = errors.New("無効な名前です")
	ErrDuplicateEmail   = errors.New("このメールアドレスは既に使用されています")
	ErrDuplicateClerkID = errors.New("このClerk IDは既に使用されています")

	// 座席関連のエラー（今後追加）
	// ErrSeatNotFound = errors.New("座席が見つかりません")

	// 予約関連のエラー（今後追加）
	// ErrReservationNotFound = errors.New("予約が見つかりません")
)
