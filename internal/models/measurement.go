package models

import "time"

type Measurement struct {
	ID             int64     `json:"id"`
	DeviceID       string    `json:"device_id"`
	PlantName      string    `json:"plant_name"`
	SoilRaw        float64   `json:"soil_raw"`
	SoilVoltage    float64   `json:"soil_voltage"`
	SoilPercent    int       `json:"soil_percent"`
	LightRaw       float64   `json:"light_raw"`
	LightVoltage   float64   `json:"light_voltage"`
	BatteryVoltage *float64  `json:"battery_voltage,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateMeasurementRequest struct {
	SoilRaw        float64  `json:"soil_raw"`
	SoilVoltage    float64  `json:"soil_voltage"`
	SoilPercent    int      `json:"soil_percent"`
	LightRaw       float64  `json:"light_raw"`
	LightVoltage   float64  `json:"light_voltage"`
	BatteryVoltage *float64 `json:"battery_voltage,omitempty"`
}
