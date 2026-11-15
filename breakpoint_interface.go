package main

import "context"

type BreakpointRepository interface {
	// returns (id, error)
	Create(ctx context.Context, bp Breakpoint) (int64, error)
	GetByID(ctx context.Context, id string) (Breakpoint, error)
	List(ctx context.Context) ([]Breakpoint, error)
	Enable(ctx context.Context, id string) error
	Disable(ctx context.Context, id string) error
	Remove(ctx context.Context, id string) error
	Update(ctx context.Context, bp Breakpoint) (Breakpoint, error)
}
