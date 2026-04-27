package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"plants-monitor/internal/storage"
	"strconv"
	"strings"

	tgbot "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"

	appmodels "plants-monitor/internal/models"
)

type Service struct {
	store *storage.Store
	bot   *tgbot.Bot
	log   *slog.Logger
}

func NewService(store *storage.Store, log *slog.Logger) *Service {
	return &Service{
		store: store,
		log:   log,
	}
}

func (s *Service) Start(ctx context.Context, token string) error {
	b, err := tgbot.New(token)
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", err)
	}

	s.bot = b

	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypePrefix, s.handleStart)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/register", tgbot.MatchTypePrefix, s.handleRegister)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/subscribe", tgbot.MatchTypePrefix, s.handleSubscribe)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "/status", tgbot.MatchTypePrefix, s.handleStatus)

	s.log.Info("telegram bot started")

	b.Start(ctx)

	return nil
}

func (s *Service) handleStart(ctx context.Context, b *tgbot.Bot, update *tgmodels.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	chatID := update.Message.Chat.ID

	err := s.store.UpsertTelegramUser(
		ctx,
		chatID,
	)
	if err != nil {
		log.Printf("upsert telegram user: %v", err)
		s.sendText(ctx, b, chatID, "Не удалось сохранить пользователя. Попробуй позже.")
		return
	}

	s.log.InfoContext(
		ctx,
		"telegram user started bot",
		slog.Int64("chat_id", chatID),
	)

	text := `Привет! Я бот мониторинга растений 🌱

Доступные команды:

/register <device_id> <название растения>
Зарегистрировать новое устройство и получить device token.

/subscribe <device_id>
Привязать уже существующее устройство к этому чату.

/status
Показать последнее измерение привязанного устройства.

/status <device_id>
Показать последнее измерение конкретного устройства.`

	s.sendText(ctx, b, chatID, text)
}

func (s *Service) handleRegister(ctx context.Context, b *tgbot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	args := strings.Fields(update.Message.Text)

	if len(args) < 3 {
		s.sendText(ctx, b, chatID, "Использование: /register <device_id> <название растения>\n\nПример:\n/register plant-a4cf12345678 Фиалки")
		return
	}

	deviceID := args[1]
	plantName := strings.Join(args[2:], " ")

	if err := s.store.UpsertTelegramUser(ctx, chatID); err != nil {
		s.log.ErrorContext(
			ctx,
			"upsert telegram user failed",
			slog.Int64("chat_id", chatID),
			slog.String("error", err.Error()),
		)
		s.sendText(ctx, b, chatID, "Не удалось сохранить пользователя. Попробуй позже.")
		return
	}

	response, err := s.store.CreateDevice(ctx, appmodels.CreateDeviceRequest{
		DeviceID:  deviceID,
		PlantName: plantName,
	})
	if err != nil {
		s.log.ErrorContext(
			ctx,
			"create device from telegram failed",
			slog.Int64("chat_id", chatID),
			slog.String("device_id", deviceID),
			slog.String("plant_name", plantName),
			slog.String("error", err.Error()),
		)

		s.sendText(ctx, b, chatID, "Не удалось зарегистрировать устройство. Возможно, оно уже существует.")
		return
	}

	if err := s.store.SubscribeDevice(ctx, chatID, deviceID); err != nil {
		s.log.ErrorContext(
			ctx,
			"subscribe device after telegram register failed",
			slog.Int64("chat_id", chatID),
			slog.String("device_id", deviceID),
			slog.String("error", err.Error()),
		)

		s.sendText(ctx, b, chatID, "Устройство создано, но не удалось подписать чат на уведомления.")
		return
	}

	s.log.InfoContext(
		ctx,
		"device registered from telegram",
		slog.Int64("chat_id", chatID),
		slog.String("device_id", deviceID),
		slog.String("plant_name", plantName),
	)

	text := fmt.Sprintf(
		"✅ Устройство зарегистрировано\n\nРастение: %s\nDevice ID: %s\n\nDevice token:\n%s\n\nСкопируй этот токен в `plants_monitor/secrets.h`:\n\nconst char* DEVICE_TOKEN = \"%s\";\n\nЭтот токен показывается только один раз. Не пересылай его другим людям.",
		response.PlantName,
		response.DeviceID,
		response.DeviceToken,
		response.DeviceToken,
	)

	s.sendText(ctx, b, chatID, text)
}

func (s *Service) handleSubscribe(ctx context.Context, b *tgbot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	args := strings.Fields(update.Message.Text)

	if len(args) != 2 {
		s.sendText(ctx, b, chatID, "Использование: /subscribe <device_id>")
		return
	}

	deviceID := args[1]

	exists, err := s.store.DeviceExists(ctx, deviceID)
	if err != nil {
		log.Printf("check device exists: %v", err)
		s.sendText(ctx, b, chatID, "Не удалось проверить устройство. Попробуй позже.")
		return
	}

	if !exists {
		s.sendText(ctx, b, chatID, "Устройство не найдено: "+deviceID)
		return
	}

	if err := s.store.SubscribeDevice(ctx, chatID, deviceID); err != nil {
		log.Printf("subscribe device: %v", err)
		s.sendText(ctx, b, chatID, "Не удалось привязать устройство. Попробуй позже.")
		return
	}

	s.sendText(ctx, b, chatID, "Устройство привязано: "+deviceID)

	s.log.InfoContext(
		ctx,
		"telegram user subscribed to device",
		slog.Int64("chat_id", chatID),
		slog.String("device_id", deviceID),
	)
}

func (s *Service) handleStatus(ctx context.Context, b *tgbot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	args := strings.Fields(update.Message.Text)

	var deviceID string

	if len(args) >= 2 {
		deviceID = args[1]
	} else {
		var err error

		deviceID, err = s.store.LatestSubscribedDeviceID(ctx, chatID)
		if errors.Is(err, sql.ErrNoRows) {
			s.sendText(ctx, b, chatID, "Сначала привяжи устройство: /subscribe <device_id>")
			return
		}

		if err != nil {
			log.Printf("latest subscribed device id: %v", err)
			s.sendText(ctx, b, chatID, "Не удалось получить привязанное устройство.")
			return
		}
	}

	measurement, err := s.store.LatestMeasurement(ctx, deviceID)
	if errors.Is(err, sql.ErrNoRows) {
		s.sendText(ctx, b, chatID, "Для устройства пока нет измерений: "+deviceID)
		return
	}

	if err != nil {
		log.Printf("latest measurement: %v", err)
		s.sendText(ctx, b, chatID, "Не удалось получить последнее измерение.")
		return
	}

	text := formatStatusMessage(deviceID, measurement.PlantName, measurement.SoilPercent, measurement.SoilVoltage, measurement.LightVoltage)

	s.sendText(ctx, b, chatID, text)

	s.log.InfoContext(
		ctx,
		"telegram status sent",
		slog.Int64("chat_id", chatID),
		slog.String("device_id", deviceID),
	)
}

func formatStatusMessage(
	deviceID string,
	plantName string,
	soilPercent int,
	soilVoltage float64,
	lightVoltage float64) string {
	var b strings.Builder

	b.WriteString("🌱 Статус растения\n\n")
	b.WriteString("Устройство: ")
	b.WriteString(plantName)
	b.WriteString(" (" + deviceID + ")")
	b.WriteString("\n\n")

	b.WriteString("Влажность почвы: ")
	b.WriteString(strconv.Itoa(soilPercent))
	b.WriteString("%\n")

	b.WriteString("Напряжение датчика влажности: ")
	b.WriteString(fmt.Sprintf("%.3f V\n", soilVoltage))

	b.WriteString("Освещённость: ")
	b.WriteString(fmt.Sprintf("%.3f V\n", lightVoltage))

	return b.String()
}

func (s *Service) sendText(ctx context.Context, b *tgbot.Bot, chatID int64, text string) {
	_, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})

	if err != nil {
		s.log.ErrorContext(
			ctx,
			"send telegram message failed",
			slog.Int64("chat_id", chatID),
			slog.String("error", err.Error()),
		)
	}
}

func (s *Service) SendText(ctx context.Context, chatID int64, text string) {
	if s.bot == nil {
		log.Printf("telegram bot is not initialized")
		return
	}

	s.sendText(ctx, s.bot, chatID, text)
}
