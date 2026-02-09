package websocket

import (
	"encoding/json"
	"time"

	"seat-management-backend/internal/domain/entity"
)

type MessageType string

const (
	MessageTypeSeatStatusUpdate MessageType = "seat_status_update"
	MessageTypePing             MessageType = "ping"
	MessageTypePong             MessageType = "pong"
)

type Message struct {
	Type      MessageType     `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

type SeatEvent struct {
	SeatID         string
	Status         string
	ReservationID  string
	UserID         string
	UserName       string
	StartTime      *time.Time
	EndTime        *time.Time
	PrivacySetting entity.PrivacySetting
}

type SeatStatusUpdateData struct {
	SeatID         string     `json:"seat_id"`
	Status         string     `json:"status"`
	ReservationID  string     `json:"reservation_id,omitempty"`
	UserID         string     `json:"user_id,omitempty"`
	UserName       string     `json:"user_name,omitempty"`
	StartTime      *time.Time `json:"start_time,omitempty"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	PrivacySetting string     `json:"privacy_setting,omitempty"`
}

func createMessageFromEvent(event SeatEvent) []byte {
	data := SeatStatusUpdateData{
		SeatID:         event.SeatID,
		Status:         event.Status,
		ReservationID:  event.ReservationID,
		UserID:         event.UserID,
		UserName:       event.UserName,
		StartTime:      event.StartTime,
		EndTime:        event.EndTime,
		PrivacySetting: string(event.PrivacySetting),
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		// マーシャルエラーの場合、エラーメッセージを含むメッセージを返す
		dataJSON = []byte(`{"error":"data marshal failed"}`)
	}

	msg := Message{
		Type:      MessageTypeSeatStatusUpdate,
		Data:      dataJSON,
		Timestamp: time.Now(),
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		// 全体的なマーシャルも失敗した場合、最小限のメッセージを返す
		return []byte(`{"type":"error","data":{"error":"message marshal failed"},"timestamp":""}`)
	}
	return msgJSON
}

func createPrivateMessage(event SeatEvent) []byte {
	data := SeatStatusUpdateData{
		SeatID: event.SeatID,
		Status: event.Status,
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		// マーシャルエラーの場合、エラーメッセージを含むメッセージを返す
		dataJSON = []byte(`{"error":"data marshal failed"}`)
	}

	msg := Message{
		Type:      MessageTypeSeatStatusUpdate,
		Data:      dataJSON,
		Timestamp: time.Now(),
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		// 全体的なマーシャルも失敗した場合、最小限のメッセージを返す
		return []byte(`{"type":"error","data":{"error":"message marshal failed"},"timestamp":""}`)
	}
	return msgJSON
}
