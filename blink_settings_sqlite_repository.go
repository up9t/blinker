package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const blinkSettingsTable = "blink_settings"

type BlinkSettingsSqliteRepository struct {
	db *sql.DB
}

func NewBlinkSettingsSqliteRepository(db *sql.DB) *BlinkSettingsSqliteRepository {
	return &BlinkSettingsSqliteRepository{db: db}
}

func (repo *BlinkSettingsSqliteRepository) Migrate(ctx context.Context) error {
	_, err := repo.db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		eye_close_threshold REAL NOT NULL,
		eye_open_threshold REAL NOT NULL,
		blink_timeout_ms INTEGER NOT NULL
	)`, blinkSettingsTable))

	return err
}

func (repo *BlinkSettingsSqliteRepository) Load(ctx context.Context) (BlinkSettings, error) {
	settings := BlinkSettings{}

	err := repo.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT
		eye_close_threshold,
		eye_open_threshold,
		blink_timeout_ms
	FROM %s WHERE id = 1`, blinkSettingsTable)).Scan(
		&settings.EyeCloseThreshold,
		&settings.EyeOpenThreshold,
		&settings.BlinkTimeoutMs,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return BlinkSettings{}, errBlinkSettingsNotFound
	}

	if err != nil {
		return BlinkSettings{}, err
	}

	return settings, nil
}

func (repo *BlinkSettingsSqliteRepository) Save(ctx context.Context, settings BlinkSettings) error {
	_, err := repo.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s (
		id,
		eye_close_threshold,
		eye_open_threshold,
		blink_timeout_ms
	) VALUES (1, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		eye_close_threshold = excluded.eye_close_threshold,
		eye_open_threshold = excluded.eye_open_threshold,
		blink_timeout_ms = excluded.blink_timeout_ms`, blinkSettingsTable),
		settings.EyeCloseThreshold,
		settings.EyeOpenThreshold,
		settings.BlinkTimeoutMs,
	)

	return err
}
