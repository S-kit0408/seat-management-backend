package entity

type PrivacySetting string

const (
	PrivacyPublic  PrivacySetting = "public"
	PrivacyFriends PrivacySetting = "friends"
	PrivacyPrivate PrivacySetting = "private"
)

// IsValid はPrivacySettingが有効かチェック
func (p PrivacySetting) IsValid() bool {
	switch p {
	case PrivacyPublic, PrivacyFriends, PrivacyPrivate:
		return true
	}
	return false
}

type AuthProvider string

const (
	AuthProviderEmail   AuthProvider = "email"
	AuthProviderGoogle  AuthProvider = "google"
	AuthProviderUnknown AuthProvider = "unknown"
)

// IsValid はAuthProviderが有効かチェック
func (a AuthProvider) IsValid() bool {
	switch a {
	case AuthProviderEmail, AuthProviderGoogle, AuthProviderUnknown:
		return true
	}
	return false
}

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
