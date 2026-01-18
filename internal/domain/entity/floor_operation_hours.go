package entity

import (
	"fmt"
	"time"

	ulidpkg "seat-management-backend/pkg/ulid"

	"gorm.io/gorm"
)

type FloorOperationHours struct {
	ID        string `gorm:"type:char(26);primary_key" json:"id"`
	FloorID   string `gorm:"type:char(26);not null;index:idx_floor_operation_hours_unique,priority:1" json:"floor_id"`
	DayOfWeek int    `gorm:"type:int;not null;check:day_of_week >= 0 AND day_of_week <= 6;index:idx_floor_operation_hours_unique,priority:2" json:"day_of_week"` // 0=Sunday, 1=Monday, ..., 6=Saturday
	OpenTime  string `gorm:"type:time;not null" json:"open_time"`                                                                                                // "08:00:00"
	CloseTime string `gorm:"type:time;not null" json:"close_time"`                                                                                               // "22:00:00"
	IsClosed  bool   `gorm:"type:boolean;not null;default:false" json:"is_closed"`                                                                               // true = closed all day

	CreatedAt time.Time      `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relation
	Floor Floor `gorm:"foreignKey:FloorID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"floor,omitempty"`
}

func (FloorOperationHours) TableName() string {
	return "floor_operation_hours"
}

// BeforeCreate hook for ULID generation
func (foh *FloorOperationHours) BeforeCreate(tx *gorm.DB) error {
	if foh.ID == "" {
		foh.ID = ulidpkg.Generate()
	}
	return nil
}

// BeforeSave hook for validation
func (foh *FloorOperationHours) BeforeSave(tx *gorm.DB) error {
	return foh.Validate()
}

// Validate performs business logic validation
func (foh *FloorOperationHours) Validate() error {
	// Day of week validation
	if foh.DayOfWeek < 0 || foh.DayOfWeek > 6 {
		return ErrInvalidDayOfWeek
	}

	// If closed, skip time validation
	if foh.IsClosed {
		return nil
	}

	// Time format validation (HH:MM:SS)
	if err := validateTimeFormat(foh.OpenTime); err != nil {
		return fmt.Errorf("営業開始時刻の形式が無効です: %w", err)
	}
	if err := validateTimeFormat(foh.CloseTime); err != nil {
		return fmt.Errorf("営業終了時刻の形式が無効です: %w", err)
	}

	// Open time must be before close time
	if foh.OpenTime >= foh.CloseTime {
		return ErrInvalidOperationHours
	}

	return nil
}

// validateTimeFormat validates HH:MM:SS format
func validateTimeFormat(timeStr string) error {
	_, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		return fmt.Errorf("時刻は HH:MM:SS 形式である必要があります")
	}
	return nil
}

// IsOperatingOn checks if floor is operating (not closed)
func (foh *FloorOperationHours) IsOperatingOn() bool {
	return !foh.IsClosed
}

// GetOperatingDuration returns duration in hours (for display purposes)
func (foh *FloorOperationHours) GetOperatingDuration() (float64, error) {
	if foh.IsClosed {
		return 0, nil
	}

	openTime, err := time.Parse("15:04:05", foh.OpenTime)
	if err != nil {
		return 0, err
	}

	closeTime, err := time.Parse("15:04:05", foh.CloseTime)
	if err != nil {
		return 0, err
	}

	duration := closeTime.Sub(openTime)
	return duration.Hours(), nil
}
