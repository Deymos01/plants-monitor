package alerts

import (
	"context"
	"fmt"
	"log"
	"plants-monitor/internal/models"
	"plants-monitor/internal/storage"
)

type Notifier interface {
	SendText(ctx context.Context, chatID int64, text string)
}

type Engine struct {
	store           *storage.Store
	notifier        Notifier
	soilLowPercent  int
	lightLowVoltage float64
}

func NewEngine(
	store *storage.Store,
	notifier Notifier,
	soilLowPercent int,
	lightLowVoltage float64,
) *Engine {
	return &Engine{
		store:           store,
		notifier:        notifier,
		soilLowPercent:  soilLowPercent,
		lightLowVoltage: lightLowVoltage,
	}
}

func (e *Engine) ProcessMeasurement(ctx context.Context, m models.Measurement) {
	e.processSoilLow(ctx, m)
	e.processLightLow(ctx, m)
}

func (e *Engine) processSoilLow(ctx context.Context, m models.Measurement) {
	isBad := m.SoilPercent < e.soilLowPercent

	if isBad {
		message := fmt.Sprintf(
			"⚠️ Недостаточно влаги\n\nУстройство: %s (%s)\nВлажность почвы: %d%%\nПорог: %d%%\n\nРастение нужно полить.",
			m.PlantName,
			m.DeviceID,
			m.SoilPercent,
			e.soilLowPercent,
		)

		e.createAndNotifyIfNew(ctx, m.DeviceID, models.AlertTypeSoilLow, message)
		return
	}

	resolved, err := e.store.ResolveAlert(ctx, m.DeviceID, models.AlertTypeSoilLow)
	if err != nil {
		log.Printf("resolve soil alert: %v", err)
		return
	}

	if resolved {
		message := fmt.Sprintf(
			"✅ Влажность восстановилась\n\nУстройство: %s (%s)\nТекущая влажность: %d%%",
			m.PlantName,
			m.DeviceID,
			m.SoilPercent,
		)

		e.notifySubscribers(ctx, m.DeviceID, message)
	}
}

func (e *Engine) processLightLow(ctx context.Context, m models.Measurement) {
	isBad := m.LightVoltage < e.lightLowVoltage

	if isBad {
		message := fmt.Sprintf(
			"⚠️ Недостаточно света\n\nУстройство: %s (%s)\nОсвещённость: %.3f V\nПорог: %.3f V\n\nПроверь, не стоит ли растение в слишком тёмном месте.",
			m.PlantName,
			m.DeviceID,
			m.LightVoltage,
			e.lightLowVoltage,
		)

		e.createAndNotifyIfNew(ctx, m.DeviceID, models.AlertTypeLightLow, message)
		return
	}

	resolved, err := e.store.ResolveAlert(ctx, m.DeviceID, models.AlertTypeLightLow)
	if err != nil {
		log.Printf("resolve light alert: %v", err)
		return
	}

	if resolved {
		message := fmt.Sprintf(
			"✅ Освещение восстановилось\n\nУстройство: %s (%s)\nТекущая освещённость: %.3f V",
			m.PlantName,
			m.DeviceID,
			m.LightVoltage,
		)

		e.notifySubscribers(ctx, m.DeviceID, message)
	}
}

func (e *Engine) createAndNotifyIfNew(ctx context.Context, deviceID string, alertType models.AlertType, message string) {
	exists, err := e.store.ActiveAlertExists(ctx, deviceID, alertType)
	if err != nil {
		log.Printf("check active alert: %v", err)
		return
	}

	if exists {
		return
	}

	if err := e.store.CreateAlert(ctx, deviceID, alertType, message); err != nil {
		log.Printf("create alert: %v", err)
		return
	}

	e.notifySubscribers(ctx, deviceID, message)
}

func (e *Engine) notifySubscribers(ctx context.Context, deviceID string, message string) {
	chatIDs, err := e.store.DeviceSubscriberChatIDs(ctx, deviceID)
	if err != nil {
		log.Printf("load subscribers: %v", err)
		return
	}

	for _, chatID := range chatIDs {
		e.notifier.SendText(ctx, chatID, message)
	}
}
