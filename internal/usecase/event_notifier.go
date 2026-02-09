package usecase

import (
	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/infrastructure/websocket"
)

type EventNotifier struct {
	hub *websocket.Hub
}

func NewEventNotifier(hub *websocket.Hub) *EventNotifier {
	return &EventNotifier{
		hub: hub,
	}
}

func (n *EventNotifier) NotifySeatStatusChange(
	seatID string,
	status string,
	reservation *entity.Reservation,
) {
	if n.hub == nil {
		return
	}

	event := websocket.SeatEvent{
		SeatID: seatID,
		Status: status,
	}

	if reservation != nil {
		event.ReservationID = reservation.ID
		event.UserID = reservation.UserID
		event.StartTime = &reservation.StartTime
		event.EndTime = &reservation.EndTime

		if reservation.PrivacySetting != nil {
			event.PrivacySetting = *reservation.PrivacySetting
		} else if reservation.User.ID != "" {
			event.PrivacySetting = reservation.User.DefaultPrivacySetting
		} else {
			event.PrivacySetting = entity.PrivacyPublic
		}

		if event.PrivacySetting != entity.PrivacyPrivate && reservation.User.ID != "" {
			event.UserName = reservation.User.Name
		}
	} else {
		event.PrivacySetting = entity.PrivacyPublic
	}

	n.hub.BroadcastSeatEvent(event)
}
