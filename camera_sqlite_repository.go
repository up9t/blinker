package main

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	screenTimeTable = "screen_time"
	blinkCountTable = "blink_count"
)

type CameraSqliteRepository struct {
	db *sql.DB
}

func NewCameraSqliteRepository(db *sql.DB) *CameraSqliteRepository {
	return &CameraSqliteRepository{db: db}
}

func (repo *CameraSqliteRepository) Migrate(ctx context.Context) error {
	for _, stmt := range []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			hour INTEGER PRIMARY KEY,
			millis INTEGER NOT NULL DEFAULT 0
		)`, screenTimeTable),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			hour INTEGER PRIMARY KEY,
			blinks INTEGER NOT NULL DEFAULT 0
		)`, blinkCountTable),
	} {
		if _, err := repo.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}

func (repo *CameraSqliteRepository) AddScreenTime(ctx context.Context, hour int64, millis int64) error {
	_, err := repo.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s (hour, millis) VALUES (?, ?)
		ON CONFLICT(hour) DO UPDATE SET millis = millis + excluded.millis`, screenTimeTable), hour, millis)

	return err
}

func (repo *CameraSqliteRepository) AddBlinkCount(ctx context.Context, hour int64, blinks int64) error {
	_, err := repo.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s (hour, blinks) VALUES (?, ?)
		ON CONFLICT(hour) DO UPDATE SET blinks = blinks + excluded.blinks`, blinkCountTable), hour, blinks)

	return err
}

func (repo *CameraSqliteRepository) ListScreenTime(ctx context.Context, fromHour int64, toHour int64) (map[int64]int64, error) {
	return repo.listHours(ctx, screenTimeTable, "millis", fromHour, toHour)
}

func (repo *CameraSqliteRepository) ListBlinkCounts(ctx context.Context, fromHour int64, toHour int64) (map[int64]int64, error) {
	return repo.listHours(ctx, blinkCountTable, "blinks", fromHour, toHour)
}

// table and column are package constants, never user input
func (repo *CameraSqliteRepository) listHours(ctx context.Context, table string, column string, fromHour int64, toHour int64) (map[int64]int64, error) {
	rows, err := repo.db.QueryContext(ctx, fmt.Sprintf(`SELECT hour, %s FROM %s
		WHERE hour >= ? AND hour < ?`, column, table), fromHour, toHour)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	hours := make(map[int64]int64)

	for rows.Next() {
		var hour, value int64

		if err := rows.Scan(&hour, &value); err != nil {
			return nil, err
		}

		hours[hour] = value
	}

	return hours, rows.Err()
}
