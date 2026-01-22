package entity

import (
	"time"

	ulidpkg "seat-management-backend/pkg/ulid"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Seat struct {
	ID          string `gorm:"type:varchar(26);primary_key" json:"id"`
	SeatNumber  string `gorm:"type:varchar(50);uniqueIndex;not null" json:"seat_number"`
	Description string `gorm:"type:text" json:"description,omitempty"`

	PositionX     float64 `gorm:"type:decimal(10,2);not null;default:0" json:"position_x"` // X座標（px or grid unit）
	PositionY     float64 `gorm:"type:decimal(10,2);not null;default:0" json:"position_y"` // Y座標
	RotationAngle int     `gorm:"type:int;not null;default:0;check:rotation_angle >= 0 AND rotation_angle < 360 AND rotation_angle % 15 = 0" json:"rotation_angle"`

	Width  float64   `gorm:"type:decimal(10,2);not null;default:100" json:"width"`       // 幅（px）
	Height float64   `gorm:"type:decimal(10,2);not null;default:100" json:"height"`      // 高さ（px）
	Shape  SeatShape `gorm:"type:varchar(20);not null;default:'rectangle'" json:"shape"` // 形状

	Attributes datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"attributes"`

	FloorID *string `gorm:"type:char(26);index" json:"floor_id,omitempty"`
	SpaceID *string `gorm:"type:char(26);index" json:"space_id,omitempty"`

	IsActive bool `gorm:"type:boolean;not null;default:true" json:"is_active"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (Seat) TableName() string {
	return "seats"
}

// レコード作成前に実行される
func (s *Seat) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = ulidpkg.Generate()
	}
	return nil
}

// 座席が予約可能かチェック
func (s *Seat) IsAvailable() bool {
	return s.IsActive
}

// 座席の形状が有効かバリデーション
func (s *Seat) ValidateShape() bool {
	return s.Shape.IsValid()
}

// 回転角度が有効かバリデーション
func (s *Seat) ValidateRotationAngle() bool {
	return s.RotationAngle >= 0 && s.RotationAngle < 360 && s.RotationAngle%15 == 0
}
