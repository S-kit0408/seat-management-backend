package entity

import (
	ulidpkg "seat-management-backend/pkg/ulid"
	"time"

	"gorm.io/gorm"
)

type Friendship struct {
	ID        string    `gorm:"type:varchar(26);primary_key" json:"id"`
	UserID1   string    `gorm:"type:varchar(26);not null;index:idx_friendships_user1" json:"user_id1"`
	UserID2   string    `gorm:"type:varchar(26);not null;index:idx_friendships_user2" json:"user_id2"`
	CreatedAt time.Time `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`

	User1 User `gorm:"foreignKey:UserID1;references:ID;constraint:OnDelete:CASCADE" json:"user1,omitempty"`
	User2 User `gorm:"foreignKey:UserID2;references:ID;constraint:OnDelete:CASCADE" json:"user2,omitempty"`
}

func (Friendship) TableName() string {
	return "friendships"
}

func (f *Friendship) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = ulidpkg.Generate()
	}

	// UserID1が常にUserID2より小さくなるように正規化
	if f.UserID1 > f.UserID2 {
		f.UserID1, f.UserID2 = f.UserID2, f.UserID1
	}

	return nil
}

// 指定されたユーザーのフレンドIDを取得
func (f *Friendship) GetFriendID(userID string) string {
	if f.UserID1 == userID {
		return f.UserID2
	}
	return f.UserID1
}

// 指定されたユーザーがこのフレンドシップに含まれるか
func (f *Friendship) ContainsUser(userID string) bool {
	return f.UserID1 == userID || f.UserID2 == userID
}
