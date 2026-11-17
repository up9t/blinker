package main

import (
	"errors"
	"fmt"
	"time"
)

// blink parameters tuned by the user, stored as a single row
const (
	blinkSettingsDefaultCloseThreshold = 0.525
	blinkSettingsDefaultOpenThreshold  = 0.5
	blinkSettingsDefaultTimeoutMs      = 1200

	blinkSettingsMinThreshold = 0.05
	blinkSettingsMaxThreshold = 0.95
	blinkSettingsMinTimeoutMs = 200
	blinkSettingsMaxTimeoutMs = 5000
)

var errBlinkSettingsNotFound = errors.New("blink settings not found")

// json tags are what the binding generator uses for the frontend field names
type BlinkSettings struct {
	EyeCloseThreshold float64 `json:"eyeCloseThreshold"`
	EyeOpenThreshold  float64 `json:"eyeOpenThreshold"`
	BlinkTimeoutMs    int     `json:"blinkTimeoutMs"`
}

func DefaultBlinkSettings() BlinkSettings {
	return BlinkSettings{
		EyeCloseThreshold: blinkSettingsDefaultCloseThreshold,
		EyeOpenThreshold:  blinkSettingsDefaultOpenThreshold,
		BlinkTimeoutMs:    blinkSettingsDefaultTimeoutMs,
	}
}

// validate reports why the settings can't be used. The close threshold must sit above
// the open one, otherwise the eye hysteresis inverts and every frame counts as closed.
func (s BlinkSettings) validate() error {
	for name, value := range map[string]float64{
		"eye close threshold": s.EyeCloseThreshold,
		"eye open threshold":  s.EyeOpenThreshold,
	} {
		if value < blinkSettingsMinThreshold || value > blinkSettingsMaxThreshold {
			return fmt.Errorf("%s must be between %v and %v, got %v",
				name, blinkSettingsMinThreshold, blinkSettingsMaxThreshold, value)
		}
	}

	if s.EyeOpenThreshold >= s.EyeCloseThreshold {
		return fmt.Errorf("eye close threshold (%v) must be above the eye open threshold (%v)",
			s.EyeCloseThreshold, s.EyeOpenThreshold)
	}

	if s.BlinkTimeoutMs < blinkSettingsMinTimeoutMs || s.BlinkTimeoutMs > blinkSettingsMaxTimeoutMs {
		return fmt.Errorf("blink timeout must be between %d and %d ms, got %d",
			blinkSettingsMinTimeoutMs, blinkSettingsMaxTimeoutMs, s.BlinkTimeoutMs)
	}

	return nil
}

// a stored row the user can no longer use (hand edited, older schema) falls back to defaults
func (s BlinkSettings) sanitized() BlinkSettings {
	if err := s.validate(); err != nil {
		return DefaultBlinkSettings()
	}

	return s
}

func (s BlinkSettings) timeout() time.Duration {
	return time.Duration(s.BlinkTimeoutMs) * time.Millisecond
}
