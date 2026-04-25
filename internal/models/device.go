package models

import "time"

type Device struct {
	ID         int64      `json:"id"`
	DeviceID   string     `json:"device_id"`
	PlantName  string     `json:"plant_name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
}

type CreateDeviceRequest struct {
	DeviceID  string `json:"device_id"`
	PlantName string `json:"plant_name"`
}

type CreateDeviceResponse struct {
	OK          bool   `json:"ok"`
	DeviceID    string `json:"device_id"`
	PlantName   string `json:"plant_name"`
	DeviceToken string `json:"device_token"`
}
