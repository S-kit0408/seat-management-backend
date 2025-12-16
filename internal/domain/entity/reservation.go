package entity

import (
	ulidpkg "seat-management-backend/pkg/ulid"
	"time"

	"gorm.io/gorm"
)

type Reservation struct {
	ID     string `gorm:"type:char(26);primary_key" json:"id"`
	UserID string `gorm:"type:char(26);not null;index:idx_reservation_user" json:"user_id"`
	SeatID string `gorm:"type:char(26);not null;index:idx_reservation_seat" json:"seat_id"`
	// 予約タイプ
	Type ReservationType `gorm:"type:varchar(20);not null;index" json:"type"`

	StartTime time.Time `gorm:"type:timestamp with time zone;not null;index:idx_reservation_time" json:"start_time"`
	EndTime   time.Time `gorm:"type:timestamp with time zone;not null;index:idx_reservation_time" json:"end_time"`
	// チェックイン管理
	CheckedInAt  *time.Time `gorm:"type:timestamp with time zone;index" json:"checked_in_at,omitempty"`
	CheckedOutAt *time.Time `gorm:"type:timestamp with time zone" json:"checked_out_at,omitempty"`
	// ステータス
	Status ReservationStatus `gorm:"type:varchar(20);not null;index:idx_reservation_status" json:"status"`
	// プライバシー設定（ユーザーのデフォルト設定を上書き可能）
	PrivacySetting *PrivacySetting `gorm:"type:privacy_setting_enum" json:"privacy_setting,omitempty"`
	// 定期予約用
	RecurringReservationID *string `gorm:"type:char(26);index:idx_reservation_recurring" json:"recurring_reservation_id,omitempty"`
	// キャンセル情報
	CancelledAt        *time.Time `gorm:"type:timestamp with time zone" json:"cancelled_at,omitempty"`
	CancellationReason *string    `gorm:"type:text" json:"cancellation_reason,omitempty"`
	// 自動延長設定
	AutoExtend bool `gorm:"type:boolean;default:false" json:"auto_extend"`
	// 延長回数カウント（修正版）
	ExtensionCount int `gorm:"type:int;not null;default:0" json:"extension_count"`
	// メタ情報
	Notes string `gorm:"type:text" json:"notes,omitempty"`

	CreatedAt time.Time      `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// リレーション（GORM用、JSON出力では通常omitempty）
	User User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user,omitempty"`
	Seat Seat `gorm:"foreignKey:SeatID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"seat,omitempty"`
}

func (Reservation) TableName() string {
	return "reservations"
}

func (r *Reservation) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = ulidpkg.Generate()
	}
	return nil
}

// チェックイン可能か
func (r *Reservation) CanCheckIn() bool {
	return r.Status == ReservationStatusReserved && r.CheckedInAt == nil
}

// チェックアウト可能か
func (r *Reservation) CanCheckOut() bool {
	return r.Status == ReservationStatusInUse && r.CheckedInAt != nil && r.CheckedOutAt == nil
}

// キャンセル可能か
func (r *Reservation) CanCancel() bool {
	return r.Status == ReservationStatusReserved || r.Status == ReservationStatusInUse
}

// 延長可能か（基本チェックのみ、ルールチェックは後のフェーズで）
func (r *Reservation) CanExtend() bool {
	return r.Status == ReservationStatusInUse
}

// プライバシー設定を取得（予約固有 or ユーザーデフォルト）
func (r *Reservation) GetEffectivePrivacySetting(user *User) PrivacySetting {
	if r.PrivacySetting != nil {
		return *r.PrivacySetting
	}
	return user.DefaultPrivacySetting
}

// チェックイン実行
func (r *Reservation) CheckIn() {
	now := time.Now()
	r.CheckedInAt = &now
	r.Status = ReservationStatusInUse
}

// チェックアウト実行
func (r *Reservation) CheckOut() {
	now := time.Now()
	r.CheckedOutAt = &now
	r.Status = ReservationStatusCompleted
}

// キャンセル実行
func (r *Reservation) Cancel(reason string) {
	now := time.Now()
	r.CancelledAt = &now
	r.CancellationReason = &reason
	r.Status = ReservationStatusCancelled
}

// 延長実行
func (r *Reservation) Extend(additionalMinutes int) {
	r.EndTime = r.EndTime.Add(time.Duration(additionalMinutes) * time.Minute)
	r.ExtensionCount++
}

// 無断キャンセルに変更
func (r *Reservation) MarkAsNoShow() {
	r.Status = ReservationStatusNoShow
}
