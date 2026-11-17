package main

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"testing"
)

func newTestBreakpointSqliteRepository(t *testing.T) (*BreakpointSqliteRepository, *sql.DB) {
	t.Helper()

	ctx := context.Background()
	db, err := NewSqliteDB(":memory:")

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { db.Close() })

	repo := NewBreakpointSqliteRepository("breakpoints", db)

	// migration must be safe to run on every start
	for range 2 {
		if err := repo.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}

	return repo, db
}

func TestBreakpointSqliteRepositoryCRUD(t *testing.T) {
	ctx := context.Background()
	repo, _ := newTestBreakpointSqliteRepository(t)

	created := DefaultBreakpoint()
	created.Name = "Twenty twenty"

	id, err := repo.Create(ctx, created)

	if err != nil {
		t.Fatal(err)
	}

	created.ID = strconv.FormatInt(id, 10)

	got, err := repo.GetByID(ctx, created.ID)

	if err != nil {
		t.Fatal(err)
	}

	if got != created {
		t.Fatalf("expected %+v, got %+v", created, got)
	}

	list, err := repo.List(ctx)

	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 1 || list[0] != created {
		t.Fatalf("expected the created breakpoint, got %+v", list)
	}

	updated := created
	updated.Name = "Renamed"
	updated.Interval = 45
	updated.IntervalUnit = DurationUnitSecond
	updated.Duration = 5
	updated.DurationUnit = DurationUnitMinute
	updated.Enabled = false

	returned, err := repo.Update(ctx, updated)

	if err != nil {
		t.Fatal(err)
	}

	if returned != updated {
		t.Fatalf("expected the updated row back, got %+v", returned)
	}

	if err := repo.Enable(ctx, created.ID); err != nil {
		t.Fatal(err)
	}

	got, err = repo.GetByID(ctx, created.ID)

	if err != nil {
		t.Fatal(err)
	}

	if !got.Enabled {
		t.Fatal("expected the breakpoint to be enabled")
	}

	if err := repo.Disable(ctx, created.ID); err != nil {
		t.Fatal(err)
	}

	got, err = repo.GetByID(ctx, created.ID)

	if err != nil {
		t.Fatal(err)
	}

	if got.Enabled {
		t.Fatal("expected the breakpoint to be disabled")
	}

	if err := repo.Remove(ctx, created.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.GetByID(ctx, created.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected the row to be gone, got %v", err)
	}

	list, err = repo.List(ctx)

	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 0 {
		t.Fatalf("expected an empty list, got %+v", list)
	}
}

func TestBreakpointSqliteRepositoryStoresUnitsAsText(t *testing.T) {
	ctx := context.Background()
	repo, db := newTestBreakpointSqliteRepository(t)

	bp := DefaultBreakpoint()
	bp.IntervalUnit = DurationUnitMinute
	bp.DurationUnit = DurationUnitMinute

	id, err := repo.Create(ctx, bp)

	if err != nil {
		t.Fatal(err)
	}

	var intervalUnit, durationUnit string

	if err := db.QueryRowContext(ctx, "SELECT interval_unit, duration_unit FROM breakpoints WHERE id = ?", id).Scan(&intervalUnit, &durationUnit); err != nil {
		t.Fatal(err)
	}

	if intervalUnit != DurationUnitMinute || durationUnit != DurationUnitMinute {
		t.Fatalf("expected readable units, got %q and %q", intervalUnit, durationUnit)
	}
}
