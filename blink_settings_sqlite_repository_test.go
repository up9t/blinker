package main

import (
	"context"
	"errors"
	"testing"
)

func TestBlinkSettingsSqliteRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	db, err := NewSqliteDB(":memory:")

	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	repo := NewBlinkSettingsSqliteRepository(db)

	// migration must be safe to run on every start
	for range 2 {
		if err := repo.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := repo.Load(ctx); !errors.Is(err, errBlinkSettingsNotFound) {
		t.Fatalf("expected no stored settings yet, got %v", err)
	}

	updated := BlinkSettings{EyeCloseThreshold: 0.8, EyeOpenThreshold: 0.4, BlinkTimeoutMs: 900}

	// the second save rewrites the single row instead of adding one
	for _, settings := range []BlinkSettings{
		{EyeCloseThreshold: 0.7, EyeOpenThreshold: 0.3, BlinkTimeoutMs: 2500},
		updated,
	} {
		if err := repo.Save(ctx, settings); err != nil {
			t.Fatal(err)
		}
	}

	got, err := repo.Load(ctx)

	if err != nil {
		t.Fatal(err)
	}

	if got != updated {
		t.Fatalf("expected %+v, got %+v", updated, got)
	}

	var rows int

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+blinkSettingsTable).Scan(&rows); err != nil {
		t.Fatal(err)
	}

	if rows != 1 {
		t.Fatalf("expected a single settings row, got %d", rows)
	}
}
