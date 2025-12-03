package entity

import (
	"time"

	ulidpkg "seat-management-backend/pkg/ulid"

	"gorm.io/gorm"
)

type FriendRequest struct {
	ID          string        `gorm:"type:varchar(26);primary_key" json:"id"`
	RequesterID string        `gorm:"type:varchar(26);not null;index:idx_friend_requests_requester" json:"requester_id"`
	AddresseeID string        `gorm:"type:varchar(26);not null;index:idx_friend_requests_addressee" json:"addressee_id"`
	Status      RequestStatus `gorm:"type:request_status_enum;default:'pending';not null" json:"status"`
	Message     *string       `gorm:"type:text" json:"message,omitempty"`
	CreatedAt   time.Time     `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time     `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"updated_at"`

	Requester User `gorm:"foreignKey:RequesterID;references:ID;constraint:OnDelete:CASCADE" json:"requester,omitempty"`
	Addressee User `gorm:"foreignKey:AddresseeID;references:ID;constraint:OnDelete:CASCADE" json:"addressee,omitempty"`
}

func (FriendRequest) TableName() string {
	return "friend_requests"
}

func (fr *FriendRequest) BeforeCreate(tx *gorm.DB) error {
	if fr.ID == "" {
		fr.ID = ulidpkg.Generate()
	}
	return nil
}

func (fr *FriendRequest) IsPending() bool {
	return fr.Status == RequestStatusPending
}

func (fr *FriendRequest) Accept() {
	fr.Status = RequestStatusAccepted
}

func (fr *FriendRequest) Reject() {
	fr.Status = RequestStatusRejected
}

func (fr *FriendRequest) Cancel() {
	fr.Status = RequestStatusCancelled
}
