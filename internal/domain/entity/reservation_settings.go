package entity

import (
	"errors"
	"time"

	ulidpkg "seat-management-backend/pkg/ulid"

	"gorm.io/gorm"
)

// Default constants (code-level fallback)
const (
	DefaultCheckInMinutesBeforeStart   = 15
	DefaultCheckInGracePeriodMinutes   = 15
	DefaultMinReservationMinutes       = 30
	DefaultMaxReservationMinutes       = 480
	DefaultMaxAdvanceBookingDays       = 14
	DefaultCancellationDeadlineMinutes = 60
	DefaultMaxExtensionMinutes         = 120
	DefaultMaxExtensionCount           = 2
	DefaultAllowInstantReservation     = true
)

type ReservationSettings struct {
	ID string `gorm:"type:char(26);primary_key" json:"id"`

	// チェックイン制限
	CheckInMinutesBeforeStart int `gorm:"type:int;not null;default:15" json:"check_in_minutes_before_start"` // 予約開始の何分前からチェックイン可能か
	CheckInGracePeriodMinutes int `gorm:"type:int;not null;default:15" json:"check_in_grace_period_minutes"` // 予約開始から何分以内にチェックインしないとno_showになるか

	// 予約時間制限
	MinReservationMinutes int `gorm:"type:int;not null;default:30" json:"min_reservation_minutes"`  // 最小予約時間（分）
	MaxReservationMinutes int `gorm:"type:int;not null;default:480" json:"max_reservation_minutes"` // 最大予約時間（分）
	MaxAdvanceBookingDays int `gorm:"type:int;not null;default:14" json:"max_advance_booking_days"` // 何日先まで予約可能か

	// キャンセル・延長ルール
	CancellationDeadlineMinutes int  `gorm:"type:int;not null;default:60" json:"cancellation_deadline_minutes"`   // 予約開始の何分前までキャンセル可能か
	MaxExtensionMinutes         int  `gorm:"type:int;not null;default:120" json:"max_extension_minutes"`          // 1回あたりの最大延長時間（分）
	MaxExtensionCount           int  `gorm:"type:int;not null;default:2" json:"max_extension_count"`              // 最大延長回数
	AllowInstantReservation     bool `gorm:"type:boolean;not null;default:true" json:"allow_instant_reservation"` // 即時予約を許可するか

	// メタ情報
	Description string    `gorm:"type:text" json:"description,omitempty"`       // 設定の説明
	IsActive    bool      `gorm:"type:boolean;not null;index" json:"is_active"` // アクティブな設定か
	CreatedAt   time.Time `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (ReservationSettings) TableName() string {
	return "reservation_settings"
}

func (rs *ReservationSettings) BeforeCreate(tx *gorm.DB) error {
	if rs.ID == "" {
		rs.ID = ulidpkg.Generate()
	}
	return nil
}

// Validate performs validation on reservation settings
func (rs *ReservationSettings) Validate() error {
	if rs.MinReservationMinutes <= 0 {
		return errors.New("最小予約時間は1分以上である必要があります")
	}
	if rs.MaxReservationMinutes <= rs.MinReservationMinutes {
		return errors.New("最大予約時間は最小予約時間より大きい必要があります")
	}
	if rs.MaxAdvanceBookingDays <= 0 {
		return errors.New("最大事前予約日数は1日以上である必要があります")
	}
	if rs.CheckInMinutesBeforeStart < 0 {
		return errors.New("チェックイン可能時間は0分以上である必要があります")
	}
	if rs.CheckInGracePeriodMinutes < 0 {
		return errors.New("チェックイン猶予時間は0分以上である必要があります")
	}
	if rs.CancellationDeadlineMinutes < 0 {
		return errors.New("キャンセル期限は0分以上である必要があります")
	}
	if rs.MaxExtensionMinutes <= 0 {
		return errors.New("最大延長時間は1分以上である必要があります")
	}
	if rs.MaxExtensionCount < 0 {
		return errors.New("最大延長回数は0回以上である必要があります")
	}
	return nil
}

// GetDefaultSettings returns code-level default settings
func GetDefaultSettings() *ReservationSettings {
	return &ReservationSettings{
		CheckInMinutesBeforeStart:   DefaultCheckInMinutesBeforeStart,
		CheckInGracePeriodMinutes:   DefaultCheckInGracePeriodMinutes,
		MinReservationMinutes:       DefaultMinReservationMinutes,
		MaxReservationMinutes:       DefaultMaxReservationMinutes,
		MaxAdvanceBookingDays:       DefaultMaxAdvanceBookingDays,
		CancellationDeadlineMinutes: DefaultCancellationDeadlineMinutes,
		MaxExtensionMinutes:         DefaultMaxExtensionMinutes,
		MaxExtensionCount:           DefaultMaxExtensionCount,
		AllowInstantReservation:     DefaultAllowInstantReservation,
		IsActive:                    true,
		Description:                 "Default system settings",
	}
}
