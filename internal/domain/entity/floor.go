package entity

import (
	"time"

	ulidpkg "seat-management-backend/pkg/ulid"

	"gorm.io/gorm"
)

type Floor struct {
	ID          string `gorm:"type:char(26);primary_key" json:"id"`
	Name        string `gorm:"type:varchar(50);not null;uniqueIndex" json:"name"`   // "1階", "2階", "B1"
	DisplayName string `gorm:"type:varchar(100)" json:"display_name,omitempty"`     // "1F ワークスペース"
	Description string `gorm:"type:text" json:"description,omitempty"`              // フロアの説明
	SortOrder   int    `gorm:"type:int;not null;default:0;index" json:"sort_order"` // 表示順序（昇順）
	IsActive    bool   `gorm:"type:boolean;not null;default:true" json:"is_active"` // アクティブ状態

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Floor) TableName() string {
	return "floors"
}

// レコード作成前に実行される
func (f *Floor) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = ulidpkg.Generate()
	}
	return nil
}

// フロアがアクティブかチェック
func (f *Floor) IsFloorActive() bool {
	return f.IsActive
}
