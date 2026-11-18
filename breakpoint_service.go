package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const breakpointEventBufferSize = 16

// BreakpointService owns the break schedule so the overlay window only reacts to events:
// one interval timer per enabled breakpoint, one duration timer for the running break and
// a queue for the breaks that came due while another break was on screen.
type BreakpointService struct {
	repo    BreakpointRepository
	eventCh chan BreakpointEvent

	mu sync.Mutex
	// every known breakpoint, by id
	breakpoints map[string]Breakpoint
	// id -> unexpired interval timer
	intervalTimers map[string]*time.Timer
	// id -> timer that notifies shortly before that interval timer fires
	notifyTimers map[string]*time.Timer
	// id -> when that timer fires, so "reset next break" finds the soonest one
	intervalDeadlines map[string]time.Time
	// bumped on every reschedule, so a timer that already fired can't act twice
	generations map[string]int
	// breaks waiting for the running break to finish
	queue      []Breakpoint
	active     *Breakpoint
	breakTimer *time.Timer
	breakEnd   time.Time

	// test seam: divides every duration, 1 in production
	speedup time.Duration

	stopped  bool
	done     chan struct{}
	stopOnce sync.Once
}

func NewBreakpointService(repo BreakpointRepository) *BreakpointService {
	return &BreakpointService{
		repo:              repo,
		eventCh:           make(chan BreakpointEvent, breakpointEventBufferSize),
		breakpoints:       make(map[string]Breakpoint),
		intervalTimers:    make(map[string]*time.Timer),
		notifyTimers:      make(map[string]*time.Timer),
		intervalDeadlines: make(map[string]time.Time),
		generations:       make(map[string]int),
		speedup:           1,
		done:              make(chan struct{}),
	}
}

// Start loads the stored breakpoints and arms a timer for each enabled one, so breaks keep
// running before the frontend mounts. Rows the user can no longer use are skipped.
func (svc *BreakpointService) Start(ctx context.Context) (<-chan BreakpointEvent, error) {
	bps, err := svc.repo.List(ctx)

	if err != nil {
		return nil, err
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	for _, bp := range bps {
		svc.breakpoints[bp.ID] = bp

		if !bp.Enabled || bp.validate() != nil {
			continue
		}

		svc.scheduleIntervalLocked(bp, bp.getInterval())
	}

	return svc.eventCh, nil
}

// Stop is safe to call twice and from any state. The event channel is never closed: the
// timers may still be inside a callback when Stop returns.
func (svc *BreakpointService) Stop() {
	svc.stopOnce.Do(func() {
		svc.mu.Lock()
		defer svc.mu.Unlock()

		svc.stopped = true
		close(svc.done)

		for id := range svc.intervalTimers {
			svc.stopIntervalLocked(id)
		}

		if svc.breakTimer != nil {
			svc.breakTimer.Stop()
			svc.breakTimer = nil
		}

		svc.active = nil
		svc.queue = nil
	})
}

// caller must hold svc.mu
func (svc *BreakpointService) stopIntervalLocked(id string) {
	if timer, ok := svc.intervalTimers[id]; ok {
		timer.Stop()
		delete(svc.intervalTimers, id)
	}

	if timer, ok := svc.notifyTimers[id]; ok {
		timer.Stop()
		delete(svc.notifyTimers, id)
	}

	delete(svc.intervalDeadlines, id)
	svc.generations[id]++
}

// caller must hold svc.mu
func (svc *BreakpointService) scheduleIntervalLocked(bp Breakpoint, delay time.Duration) {
	if svc.stopped {
		return
	}

	svc.stopIntervalLocked(bp.ID)
	generation := svc.generations[bp.ID]
	scaled := svc.scale(delay)

	svc.intervalTimers[bp.ID] = time.AfterFunc(scaled, func() {
		svc.intervalElapsed(bp.ID, generation)
	})
	svc.intervalDeadlines[bp.ID] = time.Now().Add(scaled)

	// a break shorter than the lead time leaves no room for a heads up
	if delay >= breakpointNotifyLeadTime {
		svc.notifyTimers[bp.ID] = time.AfterFunc(svc.scale(delay-breakpointNotifyLeadTime), func() {
			svc.intervalReminder(bp.ID, generation)
		})
	}
}

// intervalReminder tells the user a break is coming. It shares the interval timer's
// generation, so a reschedule or disable silently drops the stale reminder.
func (svc *BreakpointService) intervalReminder(id string, generation int) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.stopped || svc.generations[id] != generation {
		return
	}

	// the timer just fired, it isn't reusable
	delete(svc.notifyTimers, id)

	bp, ok := svc.breakpoints[id]

	if !ok || !bp.Enabled {
		return
	}

	svc.emitLocked(BreakpointEvent{
		typ: BreakpointEventBreakSoon,
		data: BreakInfo{
			Name:        bp.Name,
			RemainingMs: int(breakpointNotifyLeadTime / time.Millisecond),
		},
	})
}

// caller must hold svc.mu
func (svc *BreakpointService) startBreakLocked(bp Breakpoint) {
	svc.active = &bp
	duration := svc.scale(bp.getDuration())
	svc.breakEnd = time.Now().Add(duration)
	svc.breakTimer = time.AfterFunc(duration, svc.finishBreak)

	svc.emitLocked(BreakpointEvent{
		typ: BreakpointEventStartBreak,
		data: BreakInfo{
			Name:        bp.Name,
			RemainingMs: int(duration / time.Millisecond),
		},
	})
}

// caller must hold svc.mu
func (svc *BreakpointService) emitLocked(ev BreakpointEvent) {
	select {
	case svc.eventCh <- ev:
	case <-svc.done:
	}
}

func (svc *BreakpointService) intervalElapsed(id string, generation int) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.stopped || svc.generations[id] != generation {
		return
	}

	// the timer just fired, it isn't reusable
	delete(svc.intervalTimers, id)
	delete(svc.intervalDeadlines, id)

	bp, ok := svc.breakpoints[id]

	if !ok || !bp.Enabled {
		return
	}

	if svc.active != nil {
		svc.queue = append(svc.queue, bp)
		return
	}

	svc.startBreakLocked(bp)
}

// caller must hold svc.mu
func (svc *BreakpointService) finishBreak() {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.stopped {
		return
	}

	finished, ok := svc.endActiveBreakLocked()

	if !ok {
		return
	}

	// the finished breakpoint waits a full interval before its next break
	svc.armNextIntervalLocked(finished, finished.getInterval())
	svc.startQueuedBreakLocked()
}

// caller must hold svc.mu
func (svc *BreakpointService) endActiveBreakLocked() (Breakpoint, bool) {
	if svc.active == nil {
		return Breakpoint{}, false
	}

	if svc.breakTimer != nil {
		svc.breakTimer.Stop()
		svc.breakTimer = nil
	}

	finished := *svc.active
	svc.active = nil

	svc.emitLocked(BreakpointEvent{
		typ:  BreakpointEventFinishBreak,
		data: BreakInfo{Name: finished.Name},
	})

	return finished, true
}

// caller must hold svc.mu
func (svc *BreakpointService) armNextIntervalLocked(finished Breakpoint, delay time.Duration) {
	bp, ok := svc.breakpoints[finished.ID]

	if !ok || !bp.Enabled {
		return
	}

	svc.scheduleIntervalLocked(bp, delay)
}

// caller must hold svc.mu
func (svc *BreakpointService) startQueuedBreakLocked() {
	for len(svc.queue) > 0 {
		next := svc.queue[0]
		svc.queue = svc.queue[1:]

		// it may have been disabled or removed while waiting
		bp, ok := svc.breakpoints[next.ID]

		if !ok || !bp.Enabled {
			continue
		}

		svc.startBreakLocked(bp)

		return
	}
}

// caller must hold svc.mu
func (svc *BreakpointService) removeFromQueueLocked(id string) {
	kept := svc.queue[:0]

	for _, bp := range svc.queue {
		if bp.ID != id {
			kept = append(kept, bp)
		}
	}

	svc.queue = kept
}

// scale is the only place the test seam touches production code.
func (svc *BreakpointService) scale(d time.Duration) time.Duration {
	if svc.speedup <= 0 {
		return d
	}

	return d / svc.speedup
}

// ActiveBreak hydrates the break overlay on mount: the service already knows the break
// before it asks the overlay window to show the page.
func (svc *BreakpointService) ActiveBreak() *BreakInfo {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.active == nil {
		return nil
	}

	remaining := time.Until(svc.breakEnd)

	if remaining < 0 {
		remaining = 0
	}

	return &BreakInfo{
		Name:        svc.active.Name,
		RemainingMs: int(remaining / time.Millisecond),
	}
}

// Postpone snoozes the running break: it ends now and that breakpoint reminds again after
// delayMs. The frontend owns the delay, so tuning the snooze is a frontend constant.
// A break that was queued while this one was up starts right away.
func (svc *BreakpointService) Postpone(delayMs int) error {
	delay := time.Duration(delayMs) * time.Millisecond

	if delay < time.Second || delay > breakpointMaxInterval {
		return fmt.Errorf("postpone delay must be between 1s and %s, got %s",
			breakpointMaxInterval, delay)
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	finished, ok := svc.endActiveBreakLocked()

	if !ok {
		return nil
	}

	svc.armNextIntervalLocked(finished, delay)
	svc.startQueuedBreakLocked()

	return nil
}

// ResetNextBreak restarts only the breakpoint whose break comes next, so that break is a
// full interval away. While a break runs, the next break is the first one in the queue; a
// queued break is taken out of the queue and gets its interval restarted instead. Every
// other timer keeps its deadline.
func (svc *BreakpointService) ResetNextBreak() {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.stopped {
		return
	}

	var nextID string

	if svc.active != nil && len(svc.queue) > 0 {
		nextID = svc.queue[0].ID
		svc.queue = svc.queue[1:]
	} else {
		var nextDeadline time.Time

		for id, deadline := range svc.intervalDeadlines {
			if nextID == "" || deadline.Before(nextDeadline) {
				nextID = id
				nextDeadline = deadline
			}
		}
	}

	bp, ok := svc.breakpoints[nextID]

	if !ok || !bp.Enabled || bp.validate() != nil {
		return
	}

	svc.scheduleIntervalLocked(bp, bp.getInterval())
}

func (svc *BreakpointService) Add(ctx context.Context, bp Breakpoint) (Breakpoint, error) {
	bp.ID = ""
	bp.Name = strings.TrimSpace(bp.Name)

	if err := bp.validate(); err != nil {
		return Breakpoint{}, err
	}

	id, err := svc.repo.Create(ctx, bp)

	if err != nil {
		return Breakpoint{}, err
	}

	bp.ID = strconv.FormatInt(id, 10)

	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.breakpoints[bp.ID] = bp

	if bp.Enabled {
		svc.scheduleIntervalLocked(bp, bp.getInterval())
	}

	return bp, nil
}

func (svc *BreakpointService) GetByID(ctx context.Context, id string) (Breakpoint, error) {
	return svc.repo.GetByID(ctx, id)
}

func (svc *BreakpointService) List(ctx context.Context) ([]Breakpoint, error) {
	return svc.repo.List(ctx)
}

func (svc *BreakpointService) Disable(ctx context.Context, id string) error {
	if err := svc.repo.Disable(ctx, id); err != nil {
		return err
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	if bp, ok := svc.breakpoints[id]; ok {
		bp.Enabled = false
		svc.breakpoints[id] = bp
	}

	svc.stopIntervalLocked(id)
	svc.removeFromQueueLocked(id)

	return nil
}

func (svc *BreakpointService) Enable(ctx context.Context, id string) error {
	if err := svc.repo.Enable(ctx, id); err != nil {
		return err
	}

	bp, err := svc.repo.GetByID(ctx, id)

	if err != nil {
		return err
	}

	if err := bp.validate(); err != nil {
		return err
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.breakpoints[bp.ID] = bp
	svc.scheduleIntervalLocked(bp, bp.getInterval())

	return nil
}

func (svc *BreakpointService) Remove(ctx context.Context, id string) error {
	if err := svc.repo.Remove(ctx, id); err != nil {
		return err
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	delete(svc.breakpoints, id)
	svc.stopIntervalLocked(id)
	svc.removeFromQueueLocked(id)

	return nil
}

func (svc *BreakpointService) Update(ctx context.Context, bp Breakpoint) (Breakpoint, error) {
	bp.Name = strings.TrimSpace(bp.Name)

	if !bp.hasValidID() {
		return Breakpoint{}, errors.New("breakpoint id is required")
	}

	if err := bp.validate(); err != nil {
		return Breakpoint{}, err
	}

	updated, err := svc.repo.Update(ctx, bp)

	if err != nil {
		return Breakpoint{}, err
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.breakpoints[updated.ID] = updated

	if updated.Enabled {
		svc.scheduleIntervalLocked(updated, updated.getInterval())
	} else {
		svc.stopIntervalLocked(updated.ID)
		svc.removeFromQueueLocked(updated.ID)
	}

	return updated, nil
}
