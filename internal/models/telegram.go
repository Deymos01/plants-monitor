package models

import "time"

type TelegramUser struct {
	ChatID    int64     `json:"chat_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DeviceSubscription struct {
	ChatID    int64     `json:"chat_id"`
	DeviceID  string    `json:"device_id"`
	CreatedAt time.Time `json:"created_at"`
}
