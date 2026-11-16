package main

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

type BreakpointService struct {
	repo    BreakpointRepository
	eventCh chan BreakpointEvent
	// id -> timer
	activeTimer map[string]*time.Timer
}

func NewBreakpointService(repo BreakpointRepository) *BreakpointService {
	return &BreakpointService{
		repo:        repo,
		eventCh:     make(chan BreakpointEvent),
		activeTimer: make(map[string]*time.Timer),
	}
}

func (svc *BreakpointService) Start(ctx context.Context) (<-chan BreakpointEvent, error) {
	// 1. get data from database
	bps, err := svc.repo.List(ctx)

	if err != nil {
		return nil, err
	}

	// 2. set timer
	for _, bp := range bps {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if !bp.enabled {
			continue
		}

		go svc.StartBreakpoint(ctx, bp)
	}

	return svc.eventCh, nil
}

func (svc *BreakpointService) Stop() {
	close(svc.eventCh)

	for id, t := range svc.activeTimer {
		t.Stop()
		delete(svc.activeTimer, id)
	}
}
func (svc *BreakpointService) StopBreakpoint(ctx context.Context, id string) {
	timer, ok := svc.activeTimer[id]

	if !ok {
		return
	}

	timer.Stop()
	delete(svc.activeTimer, id)
}

// this function is blocking
func (svc *BreakpointService) StartBreakpoint(ctx context.Context, bp Breakpoint) error {
	if !bp.enabled {
		return fmt.Errorf("breakpoint isn't enabled: %s", bp.id)
	}

	if !bp.hasValidID() {
		return fmt.Errorf("invalid id: %s", bp.id)
	}

	fmt.Println("starting timer for (seconds): ", bp.getInterval().Seconds())
	timer := time.NewTimer(bp.getInterval())
	// this might be a race condition
	svc.activeTimer[bp.id] = timer

	select {
	case _, ok := <-timer.C:
		if ok {
			svc.eventCh <- BreakpointEvent{
				typ:  BreakpointEventStartBreak,
				data: bp,
			}
		}
	case <-ctx.Done():
		timer.Stop()
	}

	return nil
}

func (svc *BreakpointService) Add(ctx context.Context, bp Breakpoint) (Breakpoint, error) {
	bp.id = ""
	id, err := svc.repo.Create(ctx, bp)

	if err != nil {
		return Breakpoint{}, err
	}

	bp.id = strconv.FormatInt(id, 10)

	if bp.enabled {
		go svc.StartBreakpoint(ctx, bp)
	}

	return bp, nil
}

func (svc *BreakpointService) GetByID(ctx context.Context, id string) (Breakpoint, error) {
	return svc.repo.GetByID(ctx, id)
}

func (svc *BreakpointService) List(ctx context.Context) ([]Breakpoint, error) {
	bps, err := svc.repo.List(ctx)

	if err != nil {
		return nil, err
	}

	return bps, nil
}

func (svc *BreakpointService) Disable(ctx context.Context, id string) error {
	if err := svc.repo.Disable(ctx, id); err != nil {
		return err
	}

	svc.StopBreakpoint(ctx, id)

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

	go svc.StartBreakpoint(ctx, bp)

	return nil
}

func (svc *BreakpointService) Remove(ctx context.Context, id string) error {
	if err := svc.repo.Remove(ctx, id); err != nil {
		return err
	}

	svc.StopBreakpoint(ctx, id)

	return nil
}

func (svc *BreakpointService) Update(ctx context.Context, bp Breakpoint) (Breakpoint, error) {
	bp, err := svc.repo.Update(ctx, bp)

	if err != nil {
		return Breakpoint{}, nil
	}

	if bp.enabled {
		svc.StartBreakpoint(ctx, bp)
	} else {
		svc.StopBreakpoint(ctx, bp.id)
	}

	return bp, nil
}
