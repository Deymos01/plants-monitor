package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"plants-monitor/internal/models"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	store := &Store{db: db}

	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DeviceInternalID(ctx context.Context, publicDeviceID string) (int64, error) {
	query := `
		SELECT id
		FROM devices
		WHERE device_id = ?
		LIMIT 1;
`

	var id int64

	err := s.db.QueryRowContext(ctx, query, publicDeviceID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, sql.ErrNoRows
	}

	if err != nil {
		return 0, fmt.Errorf("device internal id: %w", err)
	}

	return id, nil
}

func (s *Store) TelegramUserID(ctx context.Context, chatID int64) (int64, error) {
	query := `
		SELECT id
		FROM telegram_users
		WHERE chat_id = ?
		LIMIT 1;
`

	var id int64

	err := s.db.QueryRowContext(ctx, query, chatID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, sql.ErrNoRows
	}

	if err != nil {
		return 0, fmt.Errorf("telegram user id: %w", err)
	}

	return id, nil
}

func (s *Store) InsertMeasurement(ctx context.Context, publicDeviceID string, input models.CreateMeasurementRequest) (models.Measurement, error) {
	query := `
		INSERT INTO measurements (
			device_id,
			soil_raw,
			soil_voltage,
			soil_percent,
			light_raw,
			light_voltage,
			battery_voltage
		)
		SELECT d.id, ?, ?,	?, ?, ?, ?
		FROM devices d
		WHERE d.device_id = ?
		RETURNING id;
`

	var measurementID int64

	err := s.db.QueryRowContext(
		ctx,
		query,
		input.SoilRaw,
		input.SoilVoltage,
		input.SoilPercent,
		input.LightRaw,
		input.LightVoltage,
		input.BatteryVoltage,
		publicDeviceID,
	).Scan(&measurementID)

	if errors.Is(err, sql.ErrNoRows) {
		return models.Measurement{}, sql.ErrNoRows
	}

	if err != nil {
		return models.Measurement{}, fmt.Errorf("insert measurement: %w", err)
	}

	measurement, err := s.MeasurementByID(ctx, measurementID)
	if err != nil {
		return models.Measurement{}, err
	}

	return measurement, nil

}

func (s *Store) MeasurementByID(ctx context.Context, measurementID int64) (models.Measurement, error) {
	query := `
		SELECT
			m.id,
			d.device_id,
			d.plant_name,
			m.soil_raw,
			m.soil_voltage,
			m.soil_percent,
			m.light_raw,
			m.light_voltage,
			m.battery_voltage,
			m.created_at
		FROM measurements m
		JOIN devices d ON d.id = m.device_id
		WHERE m.id = ?
		LIMIT 1;
`

	var m models.Measurement
	var createdAt string

	err := s.db.QueryRowContext(ctx, query, measurementID).Scan(
		&m.ID,
		&m.DeviceID,
		&m.PlantName,
		&m.SoilRaw,
		&m.SoilVoltage,
		&m.SoilPercent,
		&m.LightRaw,
		&m.LightVoltage,
		&m.BatteryVoltage,
		&createdAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.Measurement{}, sql.ErrNoRows
	}

	if err != nil {
		return models.Measurement{}, fmt.Errorf("measurement by id: %w", err)
	}

	t, err := parseSQLiteTime(createdAt)
	if err != nil {
		return models.Measurement{}, err
	}

	m.CreatedAt = t
	return m, nil
}

func (s *Store) LatestMeasurement(ctx context.Context, publicDeviceID string) (models.Measurement, error) {
	query := `
		SELECT
			m.id,
			d.device_id,
			d.plant_name,
			m.soil_raw,
			m.soil_voltage,
			m.soil_percent,
			m.light_raw,
			m.light_voltage,
			m.battery_voltage,
			m.created_at
		FROM measurements m
		JOIN devices d ON d.id = m.device_id
		WHERE d.device_id = ?
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT 1;
	`

	var m models.Measurement
	var createdAt string

	err := s.db.QueryRowContext(ctx, query, publicDeviceID).Scan(
		&m.ID,
		&m.DeviceID,
		&m.PlantName,
		&m.SoilRaw,
		&m.SoilVoltage,
		&m.SoilPercent,
		&m.LightRaw,
		&m.LightVoltage,
		&m.BatteryVoltage,
		&createdAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return models.Measurement{}, sql.ErrNoRows
	}

	if err != nil {
		return models.Measurement{}, fmt.Errorf("latest measurement: %w", err)
	}

	t, err := parseSQLiteTime(createdAt)
	if err != nil {
		return models.Measurement{}, err
	}

	m.CreatedAt = t
	return m, nil
}

func (s *Store) CreateDevice(ctx context.Context, input models.CreateDeviceRequest) (models.CreateDeviceResponse, error) {
	token, err := GenerateDeviceToken()
	if err != nil {
		return models.CreateDeviceResponse{}, err
	}

	tokenHash := HashDeviceToken(token)

	query := `
		INSERT INTO devices (
			device_id,
			token_hash,
			plant_name
		)
		VALUES (?, ?, ?);
`

	_, err = s.db.ExecContext(
		ctx,
		query,
		input.DeviceID,
		tokenHash,
		input.PlantName,
	)
	if err != nil {
		return models.CreateDeviceResponse{}, fmt.Errorf("create device: %w", err)
	}

	return models.CreateDeviceResponse{
		OK:          true,
		DeviceID:    input.DeviceID,
		PlantName:   input.PlantName,
		DeviceToken: token,
	}, nil
}

func (s *Store) AuthenticateDevice(ctx context.Context, deviceID string, token string) error {
	if deviceID == "" {
		return errors.New("missing device id")
	}

	if token == "" {
		return errors.New("missing device token")
	}

	query := `
		SELECT token_hash
		FROM devices
		WHERE device_id = ?
		LIMIT 1;
`

	var tokenHash string

	err := s.db.QueryRowContext(ctx, query, deviceID).Scan(&tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("device not found")
	}

	if err != nil {
		return fmt.Errorf("load device: %w", err)
	}

	if !CompareTokenHash(token, tokenHash) {
		return errors.New("invalid device token")
	}

	return nil
}

func (s *Store) TouchDevice(ctx context.Context, deviceID string) error {
	query := `
		UPDATE devices
		SET last_seen_at = CURRENT_TIMESTAMP
		WHERE device_id = ?;
`

	if _, err := s.db.ExecContext(ctx, query, deviceID); err != nil {
		return fmt.Errorf("touch device: %w", err)
	}

	return nil
}

func (s *Store) UpsertTelegramUser(ctx context.Context, chatID int64) error {
	query := `
		INSERT INTO telegram_users (chat_id)
		VALUES (?)
		ON CONFLICT(chat_id) DO UPDATE SET
			updated_at = CURRENT_TIMESTAMP;
`

	if _, err := s.db.ExecContext(ctx, query, chatID); err != nil {
		return fmt.Errorf("upsert telegram user: %w", err)
	}

	return nil
}

func (s *Store) SubscribeDevice(ctx context.Context, chatID int64, publicDeviceID string) error {
	telegramUserID, err := s.TelegramUserID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("resolve telegram user id: %w", err)
	}

	deviceID, err := s.DeviceInternalID(ctx, publicDeviceID)
	if err != nil {
		return fmt.Errorf("resolve device id: %w", err)
	}

	query := `
		INSERT INTO device_subscriptions (
			telegram_user_id,
			device_id
		)
		VALUES (?, ?)
		ON CONFLICT(telegram_user_id, device_id) DO NOTHING;
`

	if _, err := s.db.ExecContext(ctx, query, telegramUserID, deviceID); err != nil {
		return fmt.Errorf("subscribe device: %w", err)
	}

	return nil
}

func (s *Store) LatestSubscribedDeviceID(ctx context.Context, chatID int64) (string, error) {
	query := `
		SELECT d.device_id
		FROM device_subscriptions ds
			JOIN telegram_users tu ON tu.id = ds.telegram_user_id
			JOIN devices d ON d.id = ds.device_id
		WHERE tu.chat_id = ?
		ORDER BY ds.created_at DESC
		LIMIT 1;
`

	var deviceID string

	err := s.db.QueryRowContext(ctx, query, chatID).Scan(&deviceID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", sql.ErrNoRows
	}

	if err != nil {
		return "", fmt.Errorf("latest subscribed device id: %w", err)
	}

	return deviceID, nil
}

func (s *Store) IsSubscribed(ctx context.Context, chatID int64, publicDeviceID string) (bool, error) {
	query := `
		SELECT 1
		FROM device_subscriptions ds
			JOIN telegram_users tu ON tu.id = ds.telegram_user_id
			JOIN devices d ON d.id = ds.device_id
		WHERE tu.chat_id = ?
		  AND d.device_id = ?
		LIMIT 1;
`

	var exists int

	err := s.db.QueryRowContext(ctx, query, chatID, publicDeviceID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("check device subscription: %w", err)
	}

	return true, nil
}

func (s *Store) DeviceExists(ctx context.Context, deviceID string) (bool, error) {
	query := `
		SELECT 1
		FROM devices
		WHERE device_id = ?
		LIMIT 1;
`

	var exists int

	err := s.db.QueryRowContext(ctx, query, deviceID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("device exists: %w", err)
	}

	return true, nil
}

func (s *Store) AlertTypeID(ctx context.Context, alertType models.AlertType) (int64, error) {
	query := `
		SELECT id
		FROM alert_types
		WHERE type = ?
		LIMIT 1;
	`

	var id int64

	err := s.db.QueryRowContext(ctx, query, string(alertType)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, sql.ErrNoRows
	}

	if err != nil {
		return 0, fmt.Errorf("alert type id: %w", err)
	}

	return id, nil
}

func (s *Store) ActiveAlertExists(ctx context.Context, publicDeviceID string, alertType models.AlertType) (bool, error) {
	query := `
		SELECT 1
		FROM alerts a
			JOIN devices d ON d.id = a.device_id
			JOIN alert_types at ON at.id = a.type_id
		WHERE d.device_id = ?
		  AND at.type = ?
		  AND a.status = ?
		LIMIT 1;
`

	var exists int

	err := s.db.QueryRowContext(
		ctx,
		query,
		publicDeviceID,
		string(alertType),
		string(models.AlertStatusActive),
	).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("active alert exists: %w", err)
	}

	return true, nil
}

func (s *Store) CreateAlert(ctx context.Context, publicDeviceID string, alertType models.AlertType, message string) error {
	deviceID, err := s.DeviceInternalID(ctx, publicDeviceID)
	if err != nil {
		return fmt.Errorf("resolve device id: %w", err)
	}

	alertTypeID, err := s.AlertTypeID(ctx, alertType)
	if err != nil {
		return fmt.Errorf("resolve alert type id: %w", err)
	}

	query := `
		INSERT INTO alerts (
			device_id,
			type_id,
			status,
			message
		)
		VALUES (?, ?, ?, ?);
	`

	if _, err := s.db.ExecContext(
		ctx,
		query,
		deviceID,
		alertTypeID,
		string(models.AlertStatusActive),
		message,
	); err != nil {
		return fmt.Errorf("create alert: %w", err)
	}

	return nil
}

func (s *Store) ResolveAlert(ctx context.Context, publicDeviceID string, alertType models.AlertType) (bool, error) {
	query := `
		UPDATE alerts
		SET status = ?,
			resolved_at = CURRENT_TIMESTAMP
		WHERE id IN (
			SELECT a.id
			FROM alerts a
				JOIN devices d ON d.id = a.device_id
				JOIN alert_types at ON at.id = a.type_id
			WHERE d.device_id = ?
			  AND at.type = ?
			  AND a.status = ?
		);
	`

	result, err := s.db.ExecContext(
		ctx,
		query,
		string(models.AlertStatusResolved),
		publicDeviceID,
		string(alertType),
		string(models.AlertStatusActive),
	)
	if err != nil {
		return false, fmt.Errorf("resolve alert: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("resolve alert rows affected: %w", err)
	}

	return affected > 0, nil
}

func (s *Store) DeviceSubscriberChatIDs(ctx context.Context, publicDeviceID string) ([]int64, error) {
	query := `
		SELECT tu.chat_id
		FROM device_subscriptions ds
			JOIN telegram_users tu ON tu.id = ds.telegram_user_id
			JOIN devices d ON d.id = ds.device_id
		WHERE d.device_id = ?;
`

	rows, err := s.db.QueryContext(ctx, query, publicDeviceID)
	if err != nil {
		return nil, fmt.Errorf("query device subscribers: %w", err)
	}
	defer rows.Close()

	var chatIDs []int64

	for rows.Next() {
		var chatID int64

		if err := rows.Scan(&chatID); err != nil {
			return nil, fmt.Errorf("scan device subscriber: %w", err)
		}

		chatIDs = append(chatIDs, chatID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device subscribers: %w", err)
	}

	return chatIDs, nil
}

func parseSQLiteTime(value string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("parse sqlite time %q", value)
}

func GenerateDeviceToken() (string, error) {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return "pmon_" + hex.EncodeToString(bytes), nil
}

func HashDeviceToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func CompareTokenHash(token string, expectedHash string) bool {
	actualHash := HashDeviceToken(token)

	actualBytes, err := hex.DecodeString(actualHash)
	if err != nil {
		return false
	}

	expectedBytes, err := hex.DecodeString(expectedHash)
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(actualBytes, expectedBytes) == 1
}
