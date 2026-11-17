package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeBlinkSettingsRepository struct {
	mu       sync.Mutex
	settings BlinkSettings
	found    bool
	fail     bool
}

func (repo *fakeBlinkSettingsRepository) Load(ctx context.Context) (BlinkSettings, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.fail {
		return BlinkSettings{}, context.DeadlineExceeded
	}

	if !repo.found {
		return BlinkSettings{}, errBlinkSettingsNotFound
	}

	return repo.settings, nil
}

func (repo *fakeBlinkSettingsRepository) Save(ctx context.Context, settings BlinkSettings) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.fail {
		return context.DeadlineExceeded
	}

	repo.settings = settings
	repo.found = true

	return nil
}

func TestBlinkSettingsServiceDefaultsWithoutStoredRow(t *testing.T) {
	svc := NewBlinkSettingsService(&fakeBlinkSettingsRepository{})

	if err := svc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	if svc.Load() != DefaultBlinkSettings() {
		t.Fatalf("expected the defaults, got %+v", svc.Load())
	}

	if svc.blinkTimeout() != 1200*time.Millisecond {
		t.Fatalf("expected a 1.2s default timeout, got %s", svc.blinkTimeout())
	}
}

func TestBlinkSettingsServiceStartLoadsStoredRow(t *testing.T) {
	repo := &fakeBlinkSettingsRepository{
		found: true,
		settings: BlinkSettings{
			EyeCloseThreshold: 0.8,
			EyeOpenThreshold:  0.2,
			BlinkTimeoutMs:    3000,
		},
	}
	svc := NewBlinkSettingsService(repo)

	if err := svc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	if svc.blinkTimeout() != 3*time.Second {
		t.Fatalf("expected the stored timeout, got %s", svc.blinkTimeout())
	}
}

func TestBlinkSettingsServiceStartFallsBackOnCorruptRow(t *testing.T) {
	repo := &fakeBlinkSettingsRepository{
		found: true,
		// inverted thresholds, unusable
		settings: BlinkSettings{
			EyeCloseThreshold: 0.1,
			EyeOpenThreshold:  0.9,
			BlinkTimeoutMs:    3000,
		},
	}
	svc := NewBlinkSettingsService(repo)

	if err := svc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	if svc.Load() != DefaultBlinkSettings() {
		t.Fatalf("expected the defaults, got %+v", svc.Load())
	}
}

func TestBlinkSettingsServiceStartFails(t *testing.T) {
	svc := NewBlinkSettingsService(&fakeBlinkSettingsRepository{fail: true})

	if err := svc.Start(context.Background()); err == nil {
		t.Fatal("expected the repository error")
	}
}

func TestBlinkSettingsServiceSavePersistsAndApplies(t *testing.T) {
	ctx := context.Background()
	repo := &fakeBlinkSettingsRepository{}
	svc := NewBlinkSettingsService(repo)

	next := BlinkSettings{EyeCloseThreshold: 0.7, EyeOpenThreshold: 0.3, BlinkTimeoutMs: 2500}

	saved, err := svc.Save(ctx, next)

	if err != nil {
		t.Fatal(err)
	}

	if saved != next || svc.Load() != next {
		t.Fatalf("expected %+v, got %+v", next, svc.Load())
	}

	if svc.blinkTimeout() != 2500*time.Millisecond {
		t.Fatalf("expected the new timeout, got %s", svc.blinkTimeout())
	}

	if repo.settings != next {
		t.Fatalf("expected the settings to be persisted, got %+v", repo.settings)
	}
}

func TestBlinkSettingsServiceSaveRejectsInvalid(t *testing.T) {
	repo := &fakeBlinkSettingsRepository{}
	svc := NewBlinkSettingsService(repo)

	for name, invalid := range map[string]BlinkSettings{
		"close below open": {EyeCloseThreshold: 0.3, EyeOpenThreshold: 0.6, BlinkTimeoutMs: 1200},
		"threshold above 1": {
			EyeCloseThreshold: 1.2, EyeOpenThreshold: 0.5, BlinkTimeoutMs: 1200,
		},
		"timeout too short": {EyeCloseThreshold: 0.6, EyeOpenThreshold: 0.5, BlinkTimeoutMs: 10},
		"timeout too long":  {EyeCloseThreshold: 0.6, EyeOpenThreshold: 0.5, BlinkTimeoutMs: 60_000},
	} {
		if _, err := svc.Save(context.Background(), invalid); err == nil {
			t.Fatalf("%s: expected a validation error", name)
		}
	}

	if svc.Load() != DefaultBlinkSettings() {
		t.Fatalf("an invalid save changed the settings: %+v", svc.Load())
	}

	if repo.found {
		t.Fatal("an invalid save reached the repository")
	}
}

func TestBlinkSettingsServiceSaveKeepsStateOnFailure(t *testing.T) {
	repo := &fakeBlinkSettingsRepository{fail: true}
	svc := NewBlinkSettingsService(repo)

	next := BlinkSettings{EyeCloseThreshold: 0.7, EyeOpenThreshold: 0.3, BlinkTimeoutMs: 2500}

	if _, err := svc.Save(context.Background(), next); err == nil {
		t.Fatal("expected the repository error")
	}

	if svc.Load() != DefaultBlinkSettings() {
		t.Fatalf("a failed save changed the settings: %+v", svc.Load())
	}
}

func TestBlinkSettingsServiceReset(t *testing.T) {
	repo := &fakeBlinkSettingsRepository{}
	svc := NewBlinkSettingsService(repo)

	_, err := svc.Save(context.Background(), BlinkSettings{
		EyeCloseThreshold: 0.7,
		EyeOpenThreshold:  0.3,
		BlinkTimeoutMs:    2500,
	})

	if err != nil {
		t.Fatal(err)
	}

	reset, err := svc.Reset(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if reset != DefaultBlinkSettings() || svc.Load() != DefaultBlinkSettings() {
		t.Fatalf("expected the defaults after reset, got %+v", svc.Load())
	}
}
