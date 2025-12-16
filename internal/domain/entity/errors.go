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

	// 座席関連のエラー
	ErrSeatNotFound        = errors.New("座席が見つかりません")
	ErrDuplicateSeatNumber = errors.New("この座席番号は既に使用されています")
	ErrInvalidSeatShape    = errors.New("無効な座席形状です")
	ErrInvalidRotation     = errors.New("無効な回転角度です（15度刻みで0-359の範囲）")

	// フロア関連のエラー
	ErrFloorNotFound      = errors.New("フロアが見つかりません")
	ErrDuplicateFloorName = errors.New("このフロア名は既に使用されています")

	// 予約関連エラー
	ErrReservationNotFound      = errors.New("予約が見つかりません")
	ErrReservationOverlap       = errors.New("指定時間帯に既に予約が存在します")
	ErrInvalidReservationTime   = errors.New("予約時間が無効です（終了時刻が開始時刻より前です）")
	ErrCannotCheckIn            = errors.New("チェックインできません")
	ErrCannotCheckOut           = errors.New("チェックアウトできません")
	ErrCannotCancel             = errors.New("キャンセルできません")
	ErrCannotExtend             = errors.New("延長できません")
	ErrInvalidReservationType   = errors.New("無効な予約タイプです")
	ErrInvalidReservationStatus = errors.New("無効な予約ステータスです")

	ErrRecurringReservationNotFound = errors.New("定期予約が見つかりません")
	ErrInvalidDaysOfWeek            = errors.New("無効な曜日指定です")
	ErrInvalidTimeRange             = errors.New("無効な時間範囲です")
)
