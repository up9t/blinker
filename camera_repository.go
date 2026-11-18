package main

import "context"

type CameraRepository interface {
	// hour is the unix timestamp of the start of the hour (local time)
	AddScreenTime(ctx context.Context, hour int64, millis int64) error
	AddBlinkCount(ctx context.Context, hour int64, blinks int64) error

	// rows keyed by that same hour, fromHour included and toHour excluded
	ListScreenTime(ctx context.Context, fromHour int64, toHour int64) (map[int64]int64, error)
	ListBlinkCounts(ctx context.Context, fromHour int64, toHour int64) (map[int64]int64, error)
}
