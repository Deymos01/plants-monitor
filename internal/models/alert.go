package models

import "time"

type AlertType string
type AlertStatus string

const (
	AlertTypeSoilLow  AlertType = "SOIL_LOW"
	AlertTypeLightLow AlertType = "LIGHT_LOW"

	AlertStatusActive   AlertStatus = "ACTIVE"
	AlertStatusResolved AlertStatus = "RESOLVED"
)

type Alert struct {
	ID         int64       `json:"id"`
	DeviceID   string      `json:"device_id"`
	Type       AlertType   `json:"type"`
	Status     AlertStatus `json:"status"`
	Message    string      `json:"message"`
	CreatedAt  time.Time   `json:"created_at"`
	ResolvedAt *time.Time  `json:"resolved_at,omitempty"`
}
