package main

import (
	"context"
	"database/sql"
	"fmt"
)

type BreakpointSqliteRepository struct {
	table string
	db    *sql.DB
}

func NewBreakpointSqliteRepository(table string, db *sql.DB) *BreakpointSqliteRepository {
	return &BreakpointSqliteRepository{
		table: table,
		db:    db,
	}
}

func (repo *BreakpointSqliteRepository) Migrate(ctx context.Context) error {
	_, err := repo.db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		message TEXT NOT NULL DEFAULT '',
		interval INTEGER,
		interval_unit TEXT,
		duration INTEGER,
		duration_unit TEXT,
		audio TEXT NOT NULL DEFAULT '',
		layout TEXT NOT NULL DEFAULT '',
		enabled BOOLEAN
	)`, repo.table))

	if err != nil {
		return err
	}

	// a table created before these columns existed, they come with the empty defaults
	for _, column := range []string{"message", "audio", "layout"} {
		if err := repo.addColumnIfMissing(ctx, column); err != nil {
			return err
		}
	}

	return nil
}

func (repo *BreakpointSqliteRepository) addColumnIfMissing(ctx context.Context, column string) error {
	var count int
	err := repo.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = ?`, repo.table), column).Scan(&count)

	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	_, err = repo.db.ExecContext(ctx,
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s TEXT NOT NULL DEFAULT ''`, repo.table, column))

	return err
}

func (repo *BreakpointSqliteRepository) Create(ctx context.Context, bp Breakpoint) (int64, error) {
	result, err := repo.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s (
		name, 
		message,
		interval, 
		interval_unit, 
		duration, 
		duration_unit, 
		audio,
		layout,
		enabled
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, repo.table),
		bp.Name,
		bp.Message,
		bp.Interval,
		bp.IntervalUnit,
		bp.Duration,
		bp.DurationUnit,
		bp.Audio,
		bp.Layout,
		bp.Enabled,
	)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (repo *BreakpointSqliteRepository) GetByID(ctx context.Context, id string) (Breakpoint, error) {
	bp := Breakpoint{ID: id}
	err := repo.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT
		name,
		message,
		interval, 
		interval_unit, 
		duration, 
		duration_unit, 
		audio,
		layout,
		enabled
	FROM %s WHERE id = ?`, repo.table), id).Scan(
		&bp.Name,
		&bp.Message,
		&bp.Interval,
		&bp.IntervalUnit,
		&bp.Duration,
		&bp.DurationUnit,
		&bp.Audio,
		&bp.Layout,
		&bp.Enabled,
	)

	if err != nil {
		return Breakpoint{}, err
	}

	return bp, nil
}

func (repo *BreakpointSqliteRepository) List(ctx context.Context) ([]Breakpoint, error) {
	rows, err := repo.db.QueryContext(ctx, fmt.Sprintf(`SELECT
		id,
		name,
		message,
		interval, 
		interval_unit, 
		duration, 
		duration_unit, 
		audio,
		layout,
		enabled
	FROM %s`, repo.table))

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	bps := make([]Breakpoint, 0, 10)

	for rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}

		bp := Breakpoint{}
		err := rows.Scan(
			&bp.ID,
			&bp.Name,
			&bp.Message,
			&bp.Interval,
			&bp.IntervalUnit,
			&bp.Duration,
			&bp.DurationUnit,
			&bp.Audio,
			&bp.Layout,
			&bp.Enabled,
		)

		if err != nil {
			return nil, err
		}

		bps = append(bps, bp)
	}

	return bps, nil
}

func (repo *BreakpointSqliteRepository) Enable(ctx context.Context, id string) error {
	_, err := repo.db.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET enabled = ? WHERE id = ?`, repo.table), true, id)

	return err
}

func (repo *BreakpointSqliteRepository) Disable(ctx context.Context, id string) error {
	_, err := repo.db.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET enabled = ? WHERE id = ?`, repo.table), false, id)

	return err
}

func (repo *BreakpointSqliteRepository) Remove(ctx context.Context, id string) error {
	_, err := repo.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = ?", repo.table), id)

	return err
}

func (repo *BreakpointSqliteRepository) Update(ctx context.Context, bp Breakpoint) (Breakpoint, error) {
	_, err := repo.db.ExecContext(ctx, fmt.Sprintf(`UPDATE %s
		SET name = ?,
			message = ?,
			interval = ?,
			interval_unit = ?,
			duration = ?,
			duration_unit = ?,
			audio = ?,
			layout = ?,
			enabled = ? 
	WHERE id = ?`, repo.table), bp.Name, bp.Message, bp.Interval, bp.IntervalUnit, bp.Duration, bp.DurationUnit, bp.Audio, bp.Layout, bp.Enabled, bp.ID)

	if err != nil {
		return Breakpoint{}, err
	}

	return repo.GetByID(ctx, bp.ID)
}
