package entity

import (
	"time"

	ulidpkg "seat-management-backend/pkg/ulid"

	"gorm.io/gorm"
)

type User struct {
	ID                    string         `gorm:"type:varchar(26);primary_key" json:"id"`
	ClerkUserID           string         `gorm:"type:varchar(255);uniqueIndex:idx_users_clerk_id;not null" json:"clerk_user_id"`
	Email                 string         `gorm:"type:varchar(255);uniqueIndex:idx_users_email;not null" json:"email"`
	Name                  string         `gorm:"type:varchar(100);not null" json:"name"`
	AvatarURL             *string        `gorm:"type:varchar(500)" json:"avatar_url,omitempty"`
	PrimaryAuthProvider   AuthProvider   `gorm:"type:auth_provider_enum;default:'unknown'" json:"primary_auth_provider"`
	DefaultPrivacySetting PrivacySetting `gorm:"type:privacy_setting_enum;default:'private'" json:"default_privacy_setting"`
	LastLoginAt           *time.Time     `gorm:"type:timestamp with time zone" json:"last_login_at,omitempty"`
	CreatedAt             time.Time      `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (User) TableName() string {
	return "users"
}

// BeforeCreate はレコード作成前に実行される
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		// ULIDを生成（後述のパッケージを使用）
		u.ID = ulidpkg.Generate()
	}
	return nil
}

// UpdateLastLogin は最終ログイン時刻を更新
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
}
