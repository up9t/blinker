package main

import "time"

type CameraEventType = int

const (
	// show the blink overlay, user hasn't blinked for a while
	CameraEventBlinkReminder CameraEventType = iota

	// hide the blink overlay, user blinked or left the screen
	CameraEventBlinkDismiss
)

type CameraEvent struct {
	typ CameraEventType
}

// local hour boundaries, so statistics can group by day/hour in local time
func hourStart(t time.Time) int64 {
	t = t.Local()
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.Local).Unix()
}
