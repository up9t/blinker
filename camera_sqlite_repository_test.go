package main

import (
	"context"
	"testing"
)

func TestCameraSqliteRepositoryAccumulates(t *testing.T) {
	ctx := context.Background()
	db, err := NewSqliteDB(":memory:")

	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	repo := NewCameraSqliteRepository(db)

	// migration must be safe to run on every start
	for range 2 {
		if err := repo.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}

	const hour = 1_700_000_000

	if err := repo.AddScreenTime(ctx, hour, 10); err != nil {
		t.Fatal(err)
	}

	if err := repo.AddScreenTime(ctx, hour, 5); err != nil {
		t.Fatal(err)
	}

	if err := repo.AddBlinkCount(ctx, hour, 2); err != nil {
		t.Fatal(err)
	}

	if err := repo.AddBlinkCount(ctx, hour, 3); err != nil {
		t.Fatal(err)
	}

	var millis, blinks int64

	if err := db.QueryRowContext(ctx, "SELECT millis FROM "+screenTimeTable+" WHERE hour = ?", hour).Scan(&millis); err != nil {
		t.Fatal(err)
	}

	if err := db.QueryRowContext(ctx, "SELECT blinks FROM "+blinkCountTable+" WHERE hour = ?", hour).Scan(&blinks); err != nil {
		t.Fatal(err)
	}

	if millis != 15 {
		t.Fatalf("expected 15 millis, got %d", millis)
	}

	if blinks != 5 {
		t.Fatalf("expected 5 blinks, got %d", blinks)
	}
}

func TestCameraSqliteRepositoryListHours(t *testing.T) {
	ctx := context.Background()
	db, err := NewSqliteDB(":memory:")

	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	repo := NewCameraSqliteRepository(db)

	if err := repo.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	for hour, value := range map[int64]int64{100: 10, 200: 20, 300: 30} {
		if err := repo.AddScreenTime(ctx, hour, value); err != nil {
			t.Fatal(err)
		}

		if err := repo.AddBlinkCount(ctx, hour, value); err != nil {
			t.Fatal(err)
		}
	}

	screen, err := repo.ListScreenTime(ctx, 200, 300)

	if err != nil {
		t.Fatal(err)
	}

	blinks, err := repo.ListBlinkCounts(ctx, 100, 200)

	if err != nil {
		t.Fatal(err)
	}

	// fromHour included, toHour excluded
	if len(screen) != 1 || screen[200] != 20 {
		t.Fatalf("expected only the 200 hour of screen time, got %v", screen)
	}

	if len(blinks) != 1 || blinks[100] != 10 {
		t.Fatalf("expected only the 100 hour of blinks, got %v", blinks)
	}
}
