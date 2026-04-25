package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"plants-monitor/internal/models"

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
		
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_measurements_device_created
		ON measurements(device_id, created_at DESC);
`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("migrate sqlite: %w", err)
	}

	return nil
}

func (s *Store) InsertMeasurement(ctx context.Context, input models.CreateMeasurementRequest) (models.Measurement, error) {
	return models.Measurement{}, nil
}
