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
