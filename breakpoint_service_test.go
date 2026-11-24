package main

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"
)

type fakeBreakpointRepository struct {
	mu          sync.Mutex
	breakpoints map[string]Breakpoint
	nextID      int64
	fail        bool
}

func newFakeBreakpointRepository() *fakeBreakpointRepository {
	return &fakeBreakpointRepository{
		breakpoints: make(map[string]Breakpoint),
		nextID:      1,
	}
}

func (repo *fakeBreakpointRepository) Create(ctx context.Context, bp Breakpoint) (int64, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.fail {
		return 0, context.DeadlineExceeded
	}

	id := repo.nextID
	repo.nextID++
	bp.ID = strconv.FormatInt(id, 10)
	repo.breakpoints[bp.ID] = bp

	return id, nil
}

func (repo *fakeBreakpointRepository) GetByID(ctx context.Context, id string) (Breakpoint, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.fail {
		return Breakpoint{}, context.DeadlineExceeded
	}

	bp, ok := repo.breakpoints[id]

	if !ok {
		return Breakpoint{}, sql.ErrNoRows
	}

	return bp, nil
}

func (repo *fakeBreakpointRepository) List(ctx context.Context) ([]Breakpoint, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.fail {
		return nil, context.DeadlineExceeded
	}

	bps := make([]Breakpoint, 0, len(repo.breakpoints))

	for _, bp := range repo.breakpoints {
		bps = append(bps, bp)
	}

	return bps, nil
}

func (repo *fakeBreakpointRepository) setEnabled(id string, enabled bool) error {
	if repo.fail {
		return context.DeadlineExceeded
	}

	bp, ok := repo.breakpoints[id]

	if !ok {
		return sql.ErrNoRows
	}

	bp.Enabled = enabled
	repo.breakpoints[id] = bp

	return nil
}

func (repo *fakeBreakpointRepository) Enable(ctx context.Context, id string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	return repo.setEnabled(id, true)
}

func (repo *fakeBreakpointRepository) Disable(ctx context.Context, id string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	return repo.setEnabled(id, false)
}

func (repo *fakeBreakpointRepository) Remove(ctx context.Context, id string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.fail {
		return context.DeadlineExceeded
	}

	delete(repo.breakpoints, id)

	return nil
}

func (repo *fakeBreakpointRepository) Update(ctx context.Context, bp Breakpoint) (Breakpoint, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.fail {
		return Breakpoint{}, context.DeadlineExceeded
	}

	if _, ok := repo.breakpoints[bp.ID]; !ok {
		return Breakpoint{}, sql.ErrNoRows
	}

	repo.breakpoints[bp.ID] = bp

	return bp, nil
}

func (repo *fakeBreakpointRepository) count() int {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	return len(repo.breakpoints)
}

func (repo *fakeBreakpointRepository) seed(t *testing.T, bp Breakpoint) Breakpoint {
	t.Helper()

	id, err := repo.Create(context.Background(), bp)

	if err != nil {
		t.Fatal(err)
	}

	bp.ID = strconv.FormatInt(id, 10)

	return bp
}

// 1s becomes 1ms, so a whole 20 minute cycle runs in the test in 1.2s
func newTestBreakpointService(repo BreakpointRepository) *BreakpointService {
	svc := NewBreakpointService(repo)
	svc.speedup = 1000

	return svc
}

func newTestBreakpoint(name string, interval int, duration int) Breakpoint {
	return Breakpoint{
		Name:         name,
		Interval:     interval,
		IntervalUnit: DurationUnitSecond,
		Duration:     duration,
		DurationUnit: DurationUnitSecond,
		Enabled:      true,
	}
}

func waitBreakpointEvent(t *testing.T, events <-chan BreakpointEvent, typ BreakpointEventType) BreakpointEvent {
	t.Helper()

	timeout := time.After(2 * time.Second)

	for {
		select {
		case ev := <-events:
			if ev.typ == typ {
				return ev
			}
		case <-timeout:
			t.Fatalf("timed out waiting for breakpoint event %d", typ)
		}
	}
}

func expectNoBreakpointEvent(t *testing.T, events <-chan BreakpointEvent, wait time.Duration) {
	t.Helper()

	select {
	case ev := <-events:
		t.Fatalf("unexpected breakpoint event %d (%q)", ev.typ, ev.data.Name)
	case <-time.After(wait):
	}
}

func startTestBreakpointService(t *testing.T, svc *BreakpointService) <-chan BreakpointEvent {
	t.Helper()

	events, err := svc.Start(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(svc.Stop)

	return events
}

func TestBreakpointServiceCyclesBreaks(t *testing.T) {
	repo := newFakeBreakpointRepository()
	repo.seed(t, newTestBreakpoint("Break", 1, 1))

	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	ev := waitBreakpointEvent(t, events, BreakpointEventStartBreak)

	if ev.data.Name != "Break" || ev.data.RemainingMs != 1 {
		t.Fatalf("unexpected break info: %+v", ev.data)
	}

	waitBreakpointEvent(t, events, BreakpointEventFinishBreak)

	// the interval timer is armed again after the break
	waitBreakpointEvent(t, events, BreakpointEventStartBreak)
}

func TestBreakpointServiceNotifiesBeforeTheBreak(t *testing.T) {
	repo := newFakeBreakpointRepository()
	// a 60s interval scaled to 60ms puts the reminder 40s (40ms) before the break
	repo.seed(t, newTestBreakpoint("Focus", 60, 1))

	svc := newTestBreakpointService(repo)

	started := time.Now()
	events := startTestBreakpointService(t, svc)

	ev := waitBreakpointEvent(t, events, BreakpointEventBreakSoon)

	if ev.data.Name != "Focus" || ev.data.RemainingMs != 40_000 {
		t.Fatalf("unexpected reminder info: %+v", ev.data)
	}

	// the channel order already proves it came before the break; this rejects an
	// immediate reminder at schedule time
	if lead := time.Since(started); lead < 15*time.Millisecond {
		t.Fatalf("the reminder came after %s, expected it to wait out the lead time", lead)
	}

	waitBreakpointEvent(t, events, BreakpointEventStartBreak)
}

func TestBreakpointServiceSkipsReminderForShortIntervals(t *testing.T) {
	repo := newFakeBreakpointRepository()
	// 39s is below the 40s lead time, so the break comes with no heads up
	repo.seed(t, newTestBreakpoint("Short", 39, 1))

	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	select {
	case ev := <-events:
		if ev.typ != BreakpointEventStartBreak {
			t.Fatalf("unexpected event %d before the break", ev.typ)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the break")
	}
}

func TestBreakpointServiceSkipsDisabledStoredBreakpoints(t *testing.T) {
	repo := newFakeBreakpointRepository()
	disabled := newTestBreakpoint("Disabled", 1, 1)
	disabled.Enabled = false
	repo.seed(t, disabled)

	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	expectNoBreakpointEvent(t, events, 100*time.Millisecond)
}

func TestBreakpointServiceStartSkipsCorruptRows(t *testing.T) {
	repo := newFakeBreakpointRepository()
	corrupt := newTestBreakpoint("", 0, 0)
	repo.seed(t, corrupt)

	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	expectNoBreakpointEvent(t, events, 100*time.Millisecond)
}

func TestBreakpointServiceAddPersistsAndSchedules(t *testing.T) {
	ctx := context.Background()
	repo := newFakeBreakpointRepository()
	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	added, err := svc.Add(ctx, newTestBreakpoint("  Deep work  ", 1, 1))

	if err != nil {
		t.Fatal(err)
	}

	if added.ID == "" || added.Name != "Deep work" {
		t.Fatalf("expected a stored, trimmed breakpoint, got %+v", added)
	}

	if repo.count() != 1 {
		t.Fatalf("expected one stored breakpoint, got %d", repo.count())
	}

	waitBreakpointEvent(t, events, BreakpointEventStartBreak)
}

func TestBreakpointServiceAddDoesNotScheduleDisabled(t *testing.T) {
	ctx := context.Background()
	repo := newFakeBreakpointRepository()
	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	disabled := newTestBreakpoint("Disabled", 1, 1)
	disabled.Enabled = false

	if _, err := svc.Add(ctx, disabled); err != nil {
		t.Fatal(err)
	}

	expectNoBreakpointEvent(t, events, 100*time.Millisecond)
}

func TestBreakpointServiceAddValidates(t *testing.T) {
	ctx := context.Background()
	repo := newFakeBreakpointRepository()
	svc := newTestBreakpointService(repo)

	for name, invalid := range map[string]Breakpoint{
		"no name":       newTestBreakpoint("   ", 1, 1),
		"zero interval": newTestBreakpoint("Break", 0, 1),
		"zero duration": newTestBreakpoint("Break", 1, 0),
		"long interval": func() Breakpoint {
			bp := newTestBreakpoint("Break", 24*60+1, 1)
			bp.IntervalUnit = DurationUnitMinute
			return bp
		}(),
		"long duration": func() Breakpoint {
			bp := newTestBreakpoint("Break", 1, 61)
			bp.DurationUnit = DurationUnitMinute
			return bp
		}(),
		"unknown interval unit": func() Breakpoint {
			bp := newTestBreakpoint("Break", 1, 1)
			bp.IntervalUnit = "hour"
			return bp
		}(),
	} {
		if _, err := svc.Add(ctx, invalid); err == nil {
			t.Fatalf("%s: expected a validation error", name)
		}
	}

	if repo.count() != 0 {
		t.Fatalf("an invalid breakpoint reached the repository: %d", repo.count())
	}
}

func TestBreakpointServiceUpdateReturnsWithoutWaitingForTheInterval(t *testing.T) {
	ctx := context.Background()
	repo := newFakeBreakpointRepository()
	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	disabled := newTestBreakpoint("Break", 1, 1)
	disabled.Enabled = false

	added, err := svc.Add(ctx, disabled)

	if err != nil {
		t.Fatal(err)
	}

	updated := added
	updated.Interval = 600
	updated.Enabled = true

	started := time.Now()
	returned, err := svc.Update(ctx, updated)
	elapsed := time.Since(started)

	if err != nil {
		t.Fatal(err)
	}

	if elapsed > 200*time.Millisecond {
		t.Fatalf("update blocked for %s", elapsed)
	}

	if returned != updated {
		t.Fatalf("expected the updated breakpoint, got %+v", returned)
	}

	waitBreakpointEvent(t, events, BreakpointEventStartBreak)
}

func TestBreakpointServiceUpdateRejectsMissingID(t *testing.T) {
	svc := newTestBreakpointService(newFakeBreakpointRepository())

	if _, err := svc.Update(context.Background(), newTestBreakpoint("Break", 1, 1)); err == nil {
		t.Fatal("expected an error for a breakpoint without an id")
	}
}

func TestBreakpointServiceDisableAndEnable(t *testing.T) {
	ctx := context.Background()
	repo := newFakeBreakpointRepository()
	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	// one minute, so the first break doesn't come before the test can disable it
	bp := newTestBreakpoint("Break", 1, 1)
	bp.IntervalUnit = DurationUnitMinute

	added, err := svc.Add(ctx, bp)

	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Disable(ctx, added.ID); err != nil {
		t.Fatal(err)
	}

	expectNoBreakpointEvent(t, events, 150*time.Millisecond)

	if err := svc.Enable(ctx, added.ID); err != nil {
		t.Fatal(err)
	}

	waitBreakpointEvent(t, events, BreakpointEventStartBreak)
}

func TestBreakpointServiceRemoveStopsTheTimer(t *testing.T) {
	ctx := context.Background()
	repo := newFakeBreakpointRepository()
	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	bp := newTestBreakpoint("Break", 1, 1)
	bp.IntervalUnit = DurationUnitMinute

	added, err := svc.Add(ctx, bp)

	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Remove(ctx, added.ID); err != nil {
		t.Fatal(err)
	}

	expectNoBreakpointEvent(t, events, 150*time.Millisecond)

	if repo.count() != 0 {
		t.Fatalf("expected the breakpoint to be removed, got %d", repo.count())
	}
}

func TestBreakpointServiceQueuesOverlappingBreaks(t *testing.T) {
	repo := newFakeBreakpointRepository()
	repo.seed(t, newTestBreakpoint("Long", 1, 50))
	repo.seed(t, newTestBreakpoint("Short", 1, 1))

	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	first := waitBreakpointEvent(t, events, BreakpointEventStartBreak)
	waitBreakpointEvent(t, events, BreakpointEventFinishBreak)
	second := waitBreakpointEvent(t, events, BreakpointEventStartBreak)

	if first.data.Name == second.data.Name {
		t.Fatalf("the queued break replaced the running one: %q", first.data.Name)
	}
}

func TestBreakpointServicePostponeSnoozes(t *testing.T) {
	ctx := context.Background()
	repo := newFakeBreakpointRepository()
	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	// 1s interval, 60s break: scaling gives 1ms and 60ms
	if _, err := svc.Add(ctx, newTestBreakpoint("Break", 1, 60)); err != nil {
		t.Fatal(err)
	}

	waitBreakpointEvent(t, events, BreakpointEventStartBreak)

	postponedAt := time.Now()

	// what the break overlay sends today: a 30s snooze
	if err := svc.Postpone(30_000); err != nil {
		t.Fatal(err)
	}

	waitBreakpointEvent(t, events, BreakpointEventFinishBreak)

	if ended := time.Since(postponedAt); ended > 30*time.Millisecond {
		t.Fatalf("the break kept running for %s after postpone", ended)
	}

	// not the normal 1ms interval: the 30s snooze scaled to 30ms
	expectNoBreakpointEvent(t, events, 10*time.Millisecond)
	waitBreakpointEvent(t, events, BreakpointEventStartBreak)

	if snoozed := time.Since(postponedAt); snoozed < 20*time.Millisecond {
		t.Fatalf("the next break came after %s, expected the snooze delay", snoozed)
	}

	if svc.ActiveBreak() == nil {
		t.Fatal("expected the snoozed breakpoint to come back")
	}
}

func TestBreakpointServicePostponedBreakCantBePostponedAgain(t *testing.T) {
	ctx := context.Background()
	repo := newFakeBreakpointRepository()
	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	// 1s interval (1ms), 60s break (60ms), 30s snooze (30ms)
	if _, err := svc.Add(ctx, newTestBreakpoint("Break", 1, 60)); err != nil {
		t.Fatal(err)
	}

	first := waitBreakpointEvent(t, events, BreakpointEventStartBreak)

	if first.data.Postponed {
		t.Fatal("expected a scheduled break not to be a postponed one")
	}

	if err := svc.Postpone(30_000); err != nil {
		t.Fatal(err)
	}

	waitBreakpointEvent(t, events, BreakpointEventFinishBreak)
	snoozed := waitBreakpointEvent(t, events, BreakpointEventStartBreak)

	if !snoozed.data.Postponed {
		t.Fatal("expected the break that came back to be marked postponed")
	}

	if info := svc.ActiveBreak(); info == nil || !info.Postponed {
		t.Fatalf("expected the running break to be a postponed one, got %+v", info)
	}

	// the overlay hides the button, the backend is what actually enforces the rule
	if err := svc.Postpone(30_000); !errors.Is(err, errBreakpointPostponed) {
		t.Fatalf("expected the second postpone to be rejected, got %v", err)
	}

	if svc.ActiveBreak() == nil {
		t.Fatal("the rejected postpone ended the break")
	}

	waitBreakpointEvent(t, events, BreakpointEventFinishBreak)

	// once the postponed break is over, the cycle is normal again
	next := waitBreakpointEvent(t, events, BreakpointEventStartBreak)

	if next.data.Postponed {
		t.Fatal("expected a normal break after the postponed one finished")
	}

	if err := svc.Postpone(30_000); err != nil {
		t.Fatalf("expected postpone to be available again, got %v", err)
	}
}

func TestBreakpointServiceQueuedPostponedBreakKeepsTheFlag(t *testing.T) {
	ctx := context.Background()
	repo := newFakeBreakpointRepository()
	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	// Long starts right away, Short starts 100ms later and runs 500ms, so the 300ms snooze
	// comes due while Short is on screen and has to wait in the queue
	if _, err := svc.Add(ctx, newTestBreakpoint("Long", 1, 300)); err != nil {
		t.Fatal(err)
	}

	short := newTestBreakpoint("Short", 100, 500)
	short.IntervalUnit = DurationUnitSecond

	if _, err := svc.Add(ctx, short); err != nil {
		t.Fatal(err)
	}

	if first := waitBreakpointEvent(t, events, BreakpointEventStartBreak); first.data.Name != "Long" {
		t.Fatalf("expected Long to start first, got %q", first.data.Name)
	}

	if err := svc.Postpone(300_000); err != nil {
		t.Fatal(err)
	}

	waitBreakpointEvent(t, events, BreakpointEventFinishBreak)

	if second := waitBreakpointEvent(t, events, BreakpointEventStartBreak); second.data.Name != "Short" {
		t.Fatalf("expected the queued Short break, got %q", second.data.Name)
	}

	waitBreakpointEvent(t, events, BreakpointEventFinishBreak)
	third := waitBreakpointEvent(t, events, BreakpointEventStartBreak)

	if third.data.Name != "Long" || !third.data.Postponed {
		t.Fatalf("expected the postponed break to keep its flag through the queue, got %+v", third.data)
	}

	if err := svc.Postpone(30_000); !errors.Is(err, errBreakpointPostponed) {
		t.Fatalf("expected the second postpone to be rejected, got %v", err)
	}
}

func TestBreakpointServicePostponeStartsQueuedBreakRightAway(t *testing.T) {
	repo := newFakeBreakpointRepository()
	repo.seed(t, newTestBreakpoint("Long", 1, 50))
	repo.seed(t, newTestBreakpoint("Short", 1, 1))

	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	first := waitBreakpointEvent(t, events, BreakpointEventStartBreak)

	if err := svc.Postpone(30_000); err != nil {
		t.Fatal(err)
	}

	waitBreakpointEvent(t, events, BreakpointEventFinishBreak)
	second := waitBreakpointEvent(t, events, BreakpointEventStartBreak)

	if first.data.Name == second.data.Name {
		t.Fatalf("expected the queued break to start, got %q twice", first.data.Name)
	}
}

func TestBreakpointServicePostponeWithoutBreakIsANoOp(t *testing.T) {
	svc := newTestBreakpointService(newFakeBreakpointRepository())
	events := startTestBreakpointService(t, svc)

	if err := svc.Postpone(30_000); err != nil {
		t.Fatal(err)
	}

	expectNoBreakpointEvent(t, events, 50*time.Millisecond)
}

func TestBreakpointServicePostponeValidatesTheDelay(t *testing.T) {
	svc := newTestBreakpointService(newFakeBreakpointRepository())

	for _, delayMs := range []int{0, -1, 999, 24*60*60*1000 + 1} {
		if err := svc.Postpone(delayMs); err == nil {
			t.Fatalf("%d ms: expected a validation error", delayMs)
		}
	}
}

func TestBreakpointServiceResetNextBreak(t *testing.T) {
	ctx := context.Background()
	repo := newFakeBreakpointRepository()
	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	// two minutes scaled to 120ms, so its reminder comes at 80ms
	bp := newTestBreakpoint("Break", 2, 1)
	bp.IntervalUnit = DurationUnitMinute

	if _, err := svc.Add(ctx, bp); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	// the original schedule sent its heads up before the reset
	waitBreakpointEvent(t, events, BreakpointEventBreakSoon)
	svc.ResetNextBreak()

	// the stale deadline would fire 20ms from now; the reset one reminds after 80ms
	select {
	case ev := <-events:
		if ev.typ != BreakpointEventBreakSoon {
			t.Fatalf("the old deadline fired after the reset: event %d", ev.typ)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the reset schedule")
	}

	waitBreakpointEvent(t, events, BreakpointEventStartBreak)
}

func TestBreakpointServiceResetNextBreakOnlyResetsTheSoonest(t *testing.T) {
	repo := newFakeBreakpointRepository()

	soon := newTestBreakpoint("Soon", 1, 1)
	soon.IntervalUnit = DurationUnitMinute // 60ms
	later := newTestBreakpoint("Later", 2, 1)
	later.IntervalUnit = DurationUnitMinute // 120ms

	seededSoon := repo.seed(t, soon)
	seededLater := repo.seed(t, later)

	svc := newTestBreakpointService(repo)
	startTestBreakpointService(t, svc)

	svc.mu.Lock()
	beforeSoon := svc.intervalDeadlines[seededSoon.ID]
	beforeLater := svc.intervalDeadlines[seededLater.ID]
	svc.mu.Unlock()

	time.Sleep(20 * time.Millisecond)
	svc.ResetNextBreak()

	svc.mu.Lock()
	afterSoon := svc.intervalDeadlines[seededSoon.ID]
	afterLater := svc.intervalDeadlines[seededLater.ID]
	svc.mu.Unlock()

	if !afterSoon.After(beforeSoon) {
		t.Fatalf("the soonest timer was not reset: %s -> %s", beforeSoon, afterSoon)
	}

	if !afterLater.Equal(beforeLater) {
		t.Fatalf("another breakpoint's timer was reset: %s -> %s", beforeLater, afterLater)
	}
}

func TestBreakpointServiceResetNextBreakResetsTheQueuedBreak(t *testing.T) {
	repo := newFakeBreakpointRepository()

	long := newTestBreakpoint("Long", 1, 200)
	queued := newTestBreakpoint("Queued", 1, 1)
	queued.IntervalUnit = DurationUnitMinute // 60ms

	seededQueued := repo.seed(t, queued)
	repo.seed(t, long)

	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	first := waitBreakpointEvent(t, events, BreakpointEventStartBreak)

	if first.data.Name != "Long" {
		t.Fatalf("expected the 1ms interval to start first, got %q", first.data.Name)
	}

	// the minute long (60ms) interval comes due while the long break is still running
	time.Sleep(80 * time.Millisecond)

	svc.mu.Lock()
	queuedBefore := len(svc.queue)
	breakEndBefore := svc.breakEnd
	svc.mu.Unlock()

	if queuedBefore != 1 {
		t.Fatalf("expected one queued break, got %d", queuedBefore)
	}

	svc.ResetNextBreak()

	svc.mu.Lock()
	queuedAfter := len(svc.queue)
	deadline, armed := svc.intervalDeadlines[seededQueued.ID]
	breakEndAfter := svc.breakEnd
	svc.mu.Unlock()

	// the running break keeps its own duration
	if !breakEndAfter.Equal(breakEndBefore) {
		t.Fatalf("the running break's duration changed: %s -> %s", breakEndBefore, breakEndAfter)
	}

	if queuedAfter != 0 {
		t.Fatalf("expected the reset break to leave the queue, got %d", queuedAfter)
	}

	if !armed {
		t.Fatal("expected the queued breakpoint to get a fresh interval timer")
	}

	if until := time.Until(deadline); until < 40*time.Millisecond {
		t.Fatalf("expected a full interval, got %s", until)
	}
}

func TestBreakpointServiceActiveBreak(t *testing.T) {
	repo := newFakeBreakpointRepository()
	repo.seed(t, newTestBreakpoint("Break", 1, 50))

	svc := newTestBreakpointService(repo)
	events := startTestBreakpointService(t, svc)

	if svc.ActiveBreak() != nil {
		t.Fatal("expected no break before the interval")
	}

	waitBreakpointEvent(t, events, BreakpointEventStartBreak)
	info := svc.ActiveBreak()

	if info == nil || info.Name != "Break" || info.RemainingMs <= 0 {
		t.Fatalf("expected the running break, got %+v", info)
	}

	waitBreakpointEvent(t, events, BreakpointEventFinishBreak)

	if svc.ActiveBreak() != nil {
		t.Fatal("expected no break after the break finished")
	}
}

func TestBreakpointServiceStopIsSafe(t *testing.T) {
	repo := newFakeBreakpointRepository()
	repo.seed(t, newTestBreakpoint("Break", 1, 1))

	svc := newTestBreakpointService(repo)
	events, err := svc.Start(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	waitBreakpointEvent(t, events, BreakpointEventStartBreak)
	svc.Stop()
	svc.Stop()

	// events emitted just before Stop may still sit in the buffer
	for draining := true; draining; {
		select {
		case <-events:
		case <-time.After(20 * time.Millisecond):
			draining = false
		}
	}

	expectNoBreakpointEvent(t, events, 50*time.Millisecond)
}

func TestBreakpointServiceRepositoryErrors(t *testing.T) {
	ctx := context.Background()
	repo := newFakeBreakpointRepository()
	svc := newTestBreakpointService(repo)

	if _, err := svc.Start(ctx); err != nil {
		t.Fatal(err)
	}

	repo.mu.Lock()
	repo.fail = true
	repo.mu.Unlock()

	if _, err := svc.Add(ctx, newTestBreakpoint("Break", 1, 1)); err == nil {
		t.Fatal("expected the repository error")
	}

	if err := svc.Disable(ctx, "1"); err == nil {
		t.Fatal("expected the repository error")
	}

	if err := svc.Enable(ctx, "1"); err == nil {
		t.Fatal("expected the repository error")
	}

	if err := svc.Remove(ctx, "1"); err == nil {
		t.Fatal("expected the repository error")
	}

	invalidUpdate := newTestBreakpoint("Break", 1, 1)
	invalidUpdate.ID = "1"

	if _, err := svc.Update(ctx, invalidUpdate); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected the repository error, got %v", err)
	}
}
