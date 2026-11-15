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
	fmt.Println("WWWWWWWWWWWWWWWWWWWWWWWWWWW")
	_, err := repo.db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		interval INTEGER,
		interval_unit TEXT,
		duration INTEGER,
		duration_unit TEXT,
		enabled BOOLEAN
	)`, repo.table))

	return err
}

func (repo *BreakpointSqliteRepository) Create(ctx context.Context, bp Breakpoint) (int64, error) {
	result, err := repo.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO %s (
		name, 
		interval, 
		interval_unit, 
		duration, 
		duration_unit, 
		enabled
	) VALUES (?, ?, ?, ?, ?)`, repo.table),
		bp.name,
		bp.interval,
		bp.intervalUnit,
		bp.duration,
		bp.durationUnit,
		bp.enabled,
	)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (repo *BreakpointSqliteRepository) GetByID(ctx context.Context, id string) (Breakpoint, error) {
	bp := Breakpoint{}
	err := repo.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT (
		name,
		interval, 
		interval_unit, 
		duration, 
		duration_unit, 
		enabled
	) FROM %s WHERE id = ?`, repo.table), id).Scan(&bp)

	if err != nil {
		return Breakpoint{}, err
	}

	return bp, nil
}

func (repo *BreakpointSqliteRepository) List(ctx context.Context) ([]Breakpoint, error) {
	rows, err := repo.db.QueryContext(ctx, fmt.Sprintf(`SELECT (
		name,
		interval, 
		interval_unit, 
		duration, 
		duration_unit, 
		enabled
	) FROM %s`, repo.table))

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
		err := rows.Scan(&bp)

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
			interval = ?,
			interval_unit = ?,
			duration = ?,
			duration_unit = ?,
			enabled = ? 
	WHERE id = ?`, repo.table), bp.name, bp.interval, bp.intervalUnit, bp.duration, bp.durationUnit, bp.enabled, bp.id)

	if err != nil {
		return Breakpoint{}, err
	}

	return repo.GetByID(ctx, bp.id)
}
