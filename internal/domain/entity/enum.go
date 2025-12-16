package entity

// プライバシー設定
type PrivacySetting string

const (
	PrivacyPublic  PrivacySetting = "public"
	PrivacyFriends PrivacySetting = "friends"
	PrivacyPrivate PrivacySetting = "private"
)

func (p PrivacySetting) IsValid() bool {
	switch p {
	case PrivacyPublic, PrivacyFriends, PrivacyPrivate:
		return true
	}
	return false
}

// ログインプロバイダー
type AuthProvider string

const (
	AuthProviderEmail   AuthProvider = "email"
	AuthProviderGoogle  AuthProvider = "google"
	AuthProviderUnknown AuthProvider = "unknown"
)

func (a AuthProvider) IsValid() bool {
	switch a {
	case AuthProviderEmail, AuthProviderGoogle, AuthProviderUnknown:
		return true
	}
	return false
}

// ユーザーロール
type UserRole string

const (
	RoleUser      UserRole = "user"
	RoleAdmin     UserRole = "admin"
	RoleModerator UserRole = "moderator"
)

func (r UserRole) IsValid() bool {
	switch r {
	case RoleUser, RoleAdmin, RoleModerator:
		return true
	}
	return false
}

// フレンド申請ステータス
type RequestStatus string

const (
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusAccepted  RequestStatus = "accepted"
	RequestStatusRejected  RequestStatus = "rejected"
	RequestStatusCancelled RequestStatus = "cancelled"
)

func (rs RequestStatus) String() string {
	return string(rs)
}

func (rs RequestStatus) IsValid() bool {
	switch rs {
	case RequestStatusPending, RequestStatusAccepted, RequestStatusRejected, RequestStatusCancelled:
		return true
	}
	return false
}

// 座席の形状
type SeatShape string

const (
	SeatShapeRectangle SeatShape = "rectangle" // 長方形（default）
	SeatShapeCircle    SeatShape = "circle"    // 円
	SeatShapeSquare    SeatShape = "square"    // 正方形
	SeatShapeOval      SeatShape = "oval"      // 楕円
)

func (s SeatShape) IsValid() bool {
	switch s {
	case SeatShapeRectangle, SeatShapeCircle, SeatShapeSquare, SeatShapeOval:
		return true
	}
	return false
}

// 予約タイプ
type ReservationType string

const (
	ReservationTypeInstant   ReservationType = "instant"
	ReservationTypeScheduled ReservationType = "scheduled"
	ReservationTypeRecurring ReservationType = "recurring"
)

func (rt ReservationType) IsValid() bool {
	switch rt {
	case ReservationTypeInstant, ReservationTypeScheduled, ReservationTypeRecurring:
		return true
	}
	return false
}

// 予約ステータス
type ReservationStatus string

const (
	ReservationStatusReserved  ReservationStatus = "reserved"  // 予約済み（チェックイン前）
	ReservationStatusInUse     ReservationStatus = "in_use"    // 利用中（チェックイン済み）
	ReservationStatusCompleted ReservationStatus = "completed" // 完了（チェックアウト済み）
	ReservationStatusCancelled ReservationStatus = "cancelled" // キャンセル
	ReservationStatusNoShow    ReservationStatus = "no_show"   // 無断キャンセル
)

func (rs ReservationStatus) IsValid() bool {
	switch rs {
	case ReservationStatusReserved, ReservationStatusInUse, ReservationStatusCompleted,
		ReservationStatusCancelled, ReservationStatusNoShow:
		return true
	}
	return false
}
