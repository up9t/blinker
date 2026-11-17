package main

import (
	"fmt"
	"strings"
	"time"
)

type BreakpointEventType = int

const (
	// how long does the break last
	// set the overlay url to /break
	// emit event to the overlay window to show how long does the user have to wait
	BreakpointEventStartBreak BreakpointEventType = iota

	// fire a new interval cycle
	BreakpointEventFinishBreak
)

type BreakpointEvent struct {
	typ  BreakpointEventType
	data BreakInfo
}

type DurationUnitType = string

const (
	DurationUnitSecond DurationUnitType = "second"
	DurationUnitMinute DurationUnitType = "minute"
)

const (
	breakpointDefaultName     = "Break"
	breakpointDefaultInterval = 20
	breakpointDefaultDuration = 20

	breakpointMaxInterval = 24 * time.Hour
	breakpointMaxDuration = time.Hour

	// snooze: end the current break now, remind again in 30s
	breakpointPostponeDelay = 30 * time.Second
)

// json tags are what the binding generator uses for the frontend field names
type Breakpoint struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Interval     int              `json:"interval"`
	IntervalUnit DurationUnitType `json:"intervalUnit"`
	Duration     int              `json:"duration"`
	DurationUnit DurationUnitType `json:"durationUnit"`
	Enabled      bool             `json:"enabled"`
}

// only what the break overlay needs, sent with the start event
type BreakInfo struct {
	Name        string `json:"name"`
	RemainingMs int    `json:"remainingMs"`
}

func DefaultBreakpoint() Breakpoint {
	return Breakpoint{
		Name:         breakpointDefaultName,
		Interval:     breakpointDefaultInterval,
		IntervalUnit: DurationUnitMinute,
		Duration:     breakpointDefaultDuration,
		DurationUnit: DurationUnitSecond,
		Enabled:      true,
	}
}

func (bp Breakpoint) getInterval() time.Duration {
	if bp.IntervalUnit == DurationUnitSecond {
		return time.Duration(bp.Interval) * time.Second
	}

	return time.Duration(bp.Interval) * time.Minute
}

func (bp Breakpoint) getDuration() time.Duration {
	if bp.DurationUnit == DurationUnitSecond {
		return time.Duration(bp.Duration) * time.Second
	}

	return time.Duration(bp.Duration) * time.Minute
}

func (bp Breakpoint) hasValidID() bool {
	return bp.ID != ""
}

func (bp Breakpoint) validate() error {
	if strings.TrimSpace(bp.Name) == "" {
		return fmt.Errorf("breakpoint name is required")
	}

	for name, unit := range map[string]DurationUnitType{
		"interval unit": bp.IntervalUnit,
		"duration unit": bp.DurationUnit,
	} {
		if unit != DurationUnitSecond && unit != DurationUnitMinute {
			return fmt.Errorf("%s must be %q or %q, got %q", name, DurationUnitSecond, DurationUnitMinute, unit)
		}
	}

	if interval := bp.getInterval(); interval < time.Second || interval > breakpointMaxInterval {
		return fmt.Errorf("break interval must be between 1s and %s, got %s", breakpointMaxInterval, interval)
	}

	if duration := bp.getDuration(); duration < time.Second || duration > breakpointMaxDuration {
		return fmt.Errorf("break duration must be between 1s and %s, got %s", breakpointMaxDuration, duration)
	}

	return nil
}
