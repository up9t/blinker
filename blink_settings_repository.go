package main

import "context"

type BlinkSettingsRepository interface {
	// returns errBlinkSettingsNotFound when nothing was saved yet
	Load(ctx context.Context) (BlinkSettings, error)
	Save(ctx context.Context, settings BlinkSettings) error
}
