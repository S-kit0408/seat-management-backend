package entity

import (
	"github.com/lib/pq"
	"gorm.io/gorm"
	ulidpkg "seat-management-backend/pkg/ulid"
	"time"
)

type RecurringReservation struct {
	ID     string `gorm:"type:char(26);primary_key" json:"id"`
	UserID string `gorm:"type:char(26);not null;index:idx_recurring_user" json:"user_id"`
	SeatID string `gorm:"type:char(26);not null;index:idx_recurring_seat" json:"seat_id"`

	// 繰り返しパターン（ビットフラグ: 日=1, 月=2, 火=4, 水=8, 木=16, 金=32, 土=64）
	DaysOfWeek int `gorm:"type:int;not null;check:days_of_week > 0 AND days_of_week <= 127" json:"days_of_week"`

	StartTime string `gorm:"type:time;not null" json:"start_time"` // "09:00:00"形式
	EndTime   string `gorm:"type:time;not null" json:"end_time"`   // "18:00:00"形式

	// 有効期間
	ValidFrom  time.Time  `gorm:"type:date;not null;index" json:"valid_from"`
	ValidUntil *time.Time `gorm:"type:date;index" json:"valid_until,omitempty"` // NULLの場合は無期限
	// プライバシー設定
	PrivacySetting PrivacySetting `gorm:"type:privacy_setting_enum;not null" json:"privacy_setting"`
	// 自動延長設定
	AutoExtend bool `gorm:"type:boolean;default:false" json:"auto_extend"`

	IsActive bool `gorm:"type:boolean;default:true;index" json:"is_active"`

	Notes string `gorm:"type:text" json:"notes,omitempty"`
	// 例外日管理（祝日などを除外）
	ExcludedDates pq.StringArray `gorm:"type:date[]" json:"excluded_dates,omitempty"`

	CreatedAt time.Time      `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	User User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user,omitempty"`
	Seat Seat `gorm:"foreignKey:SeatID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"seat,omitempty"`
}

func (RecurringReservation) TableName() string {
	return "recurring_reservations"
}

func (rr *RecurringReservation) BeforeCreate(tx *gorm.DB) error {
	if rr.ID == "" {
		rr.ID = ulidpkg.Generate()
	}
	return nil
}

// 指定日が定期予約の対象曜日か
func (rr *RecurringReservation) IsValidDayOfWeek(date time.Time) bool {
	dayBit := 1 << int(date.Weekday()) // 日曜=1, 月曜=2, ..., 土曜=64
	return (rr.DaysOfWeek & dayBit) != 0
}

// 指定日が例外日（除外日）か
func (rr *RecurringReservation) IsExcludedDate(date time.Time) bool {
	dateStr := date.Format("2006-01-02")
	for _, excluded := range rr.ExcludedDates {
		if excluded == dateStr {
			return true
		}
	}
	return false
}

// 指定日が有効期間内か
func (rr *RecurringReservation) IsValidDate(date time.Time) bool {
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	if dateOnly.Before(rr.ValidFrom) {
		return false
	}

	if rr.ValidUntil != nil && dateOnly.After(*rr.ValidUntil) {
		return false
	}

	return true
}

// 指定日に予約を生成すべきか
func (rr *RecurringReservation) ShouldGenerateReservation(date time.Time) bool {
	if !rr.IsActive {
		return false
	}

	if !rr.IsValidDate(date) {
		return false
	}

	if !rr.IsValidDayOfWeek(date) {
		return false
	}

	if rr.IsExcludedDate(date) {
		return false
	}

	return true
}
