package main

import "context"

type CameraRepository interface {
	// hour is the unix timestamp of the start of the hour (local time)
	AddScreenTime(ctx context.Context, hour int64, millis int64) error
	AddBlinkCount(ctx context.Context, hour int64, blinks int64) error
}
