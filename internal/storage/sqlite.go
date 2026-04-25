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

func (s *Store) migrate(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS measurements (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL,
		
			soil_raw REAL NOT NULL,
			soil_voltage REAL NOT NULL,
			soil_percent INTEGER NOT NULL,
		
			light_raw REAL NOT NULL,
			light_voltage REAL NOT NULL,
		
			battery_voltage REAL NULL,
		
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

			FOREIGN KEY (device_id) REFERENCES devices(device_id)
		);
		
		CREATE INDEX IF NOT EXISTS idx_measurements_device_created
		ON measurements(device_id, created_at DESC);

		CREATE TABLE IF NOT EXISTS devices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id TEXT NOT NULL UNIQUE,
			token_hash TEXT NOT NULL,
			plant_name TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen_at DATETIME NULL
		);
`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("migrate sqlite: %w", err)
	}

	return nil
}

func (s *Store) InsertMeasurement(ctx context.Context, deviceID string, input models.CreateMeasurementRequest) (models.Measurement, error) {
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
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING
			id,
			device_id,
			soil_raw,
			soil_voltage,
			soil_percent,
			light_raw,
			light_voltage,
			battery_voltage,
			created_at;
`

	var m models.Measurement
	var createdAt string

	err := s.db.QueryRowContext(
		ctx,
		query,
		deviceID,
		input.SoilRaw,
		input.SoilVoltage,
		input.SoilPercent,
		input.LightRaw,
		input.LightVoltage,
		input.BatteryVoltage,
	).Scan(
		&m.ID,
		&m.DeviceID,
		&m.SoilRaw,
		&m.SoilVoltage,
		&m.SoilPercent,
		&m.LightRaw,
		&m.LightVoltage,
		&m.BatteryVoltage,
		&createdAt,
	)

	if err != nil {
		return models.Measurement{}, fmt.Errorf("insert measurement: %w", err)
	}

	t, err := parseSQLiteTime(createdAt)
	if err != nil {
		return models.Measurement{}, err
	}

	m.CreatedAt = t
	return m, nil
}

func (s *Store) LatestMeasurement(ctx context.Context, deviceID string) (models.Measurement, error) {
	query := `
		SELECT
			id,
			device_id,
			soil_raw,
			soil_voltage,
			soil_percent,
			light_raw,
			light_voltage,
			battery_voltage,
			created_at
		FROM measurements
		WHERE device_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 1;
`

	var m models.Measurement
	var createdAt string

	err := s.db.QueryRowContext(ctx, query, deviceID).Scan(
		&m.ID,
		&m.DeviceID,
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
