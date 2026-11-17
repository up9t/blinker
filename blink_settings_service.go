package main

import (
	"context"
	"errors"
	"sync"
	"time"
)

// BlinkSettingsService keeps the user's blink parameters, in memory for the running app
// and in sqlite across restarts.
type BlinkSettingsService struct {
	repo    BlinkSettingsRepository
	mu      sync.Mutex
	current BlinkSettings
}

func NewBlinkSettingsService(repo BlinkSettingsRepository) *BlinkSettingsService {
	return &BlinkSettingsService{
		repo:    repo,
		current: DefaultBlinkSettings(),
	}
}

// Start loads the stored settings, so the camera timers already use them before the
// frontend mounts. Nothing saved yet keeps the defaults.
func (svc *BlinkSettingsService) Start(ctx context.Context) error {
	settings, err := svc.repo.Load(ctx)

	if errors.Is(err, errBlinkSettingsNotFound) {
		return nil
	}

	if err != nil {
		return err
	}

	svc.mu.Lock()
	svc.current = settings.sanitized()
	svc.mu.Unlock()

	return nil
}

func (svc *BlinkSettingsService) Load() BlinkSettings {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	return svc.current
}

func (svc *BlinkSettingsService) Save(ctx context.Context, settings BlinkSettings) (BlinkSettings, error) {
	if err := settings.validate(); err != nil {
		return BlinkSettings{}, err
	}

	if err := svc.repo.Save(ctx, settings); err != nil {
		return BlinkSettings{}, err
	}

	svc.mu.Lock()
	svc.current = settings
	svc.mu.Unlock()

	return settings, nil
}

func (svc *BlinkSettingsService) Reset(ctx context.Context) (BlinkSettings, error) {
	return svc.Save(ctx, DefaultBlinkSettings())
}

// read by the camera service on every blink timer reset
func (svc *BlinkSettingsService) blinkTimeout() time.Duration {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	return svc.current.timeout()
}
