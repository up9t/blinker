package main

import "time"

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
	data Breakpoint
}

type DurationUnitType = int

const (
	DurationUnitSecond DurationUnitType = iota
	DurationUnitMinute
)

type Breakpoint struct {
	// id from database
	id           string
	name         string
	interval     int
	intervalUnit DurationUnitType
	duration     int
	durationUnit DurationUnitType
	enabled      bool
}

func (bp Breakpoint) getInterval() time.Duration {
	if bp.intervalUnit == DurationUnitSecond {
		return time.Duration(bp.interval) * time.Second
	}

	return time.Duration(bp.interval) * time.Minute
}

func (bp Breakpoint) getDuration() time.Duration {
	if bp.durationUnit == DurationUnitSecond {
		return time.Duration(bp.duration) * time.Second
	}

	return time.Duration(bp.duration) * time.Minute
}

func (bp Breakpoint) hasValidID() bool {
	return bp.id != ""
}
