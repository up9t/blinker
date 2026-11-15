package main

import (
	"context"
	"fmt"
	"strconv"
)

type BreakpointService struct {
	repo BreakpointRepository
}

type DurationUnitType = int

const (
	DurationUnitSecond DurationUnitType = iota
	DurationUnitMinute
)

type Breakpoint struct {
	id           string
	name         string
	interval     int
	intervalUnit DurationUnitType
	duration     int
	durationUnit DurationUnitType
	enabled      bool
}

func NewBreakpointService(repo BreakpointRepository) *BreakpointService {
	return &BreakpointService{repo}
}

func (svc *BreakpointService) Add(ctx context.Context, bp Breakpoint) (Breakpoint, error) {
	id, err := svc.repo.Create(ctx, bp)

	if err != nil {
		return Breakpoint{}, err
	}

	bp.id = strconv.FormatInt(id, 10)

	return bp, nil
}

func (svc *BreakpointService) GetByID(ctx context.Context, id string) (Breakpoint, error) {
	return svc.repo.GetByID(ctx, id)
}

func (svc *BreakpointService) List(ctx context.Context) []Breakpoint {
	bps, err := svc.repo.List(ctx)

	if err != nil {
		fmt.Println("Error", err)
		return nil
	}

	return bps
}

func (svc *BreakpointService) Disable(ctx context.Context, id string) error {
	return svc.repo.Disable(ctx, id)
}

func (svc *BreakpointService) Enable(ctx context.Context, id string) error {
	return svc.repo.Enable(ctx, id)
}

func (svc *BreakpointService) Remove(ctx context.Context, id string) error {
	return svc.repo.Remove(ctx, id)
}

func (svc *BreakpointService) Update(ctx context.Context, bp Breakpoint) (Breakpoint, error) {
	return svc.repo.Update(ctx, bp)
}
