package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeCameraRepository struct {
	mu     sync.Mutex
	fail   bool
	screen map[int64]int64
	blinks map[int64]int64
}

func newFakeCameraRepository() *fakeCameraRepository {
	return &fakeCameraRepository{
		screen: make(map[int64]int64),
		blinks: make(map[int64]int64),
	}
}

func (repo *fakeCameraRepository) AddScreenTime(ctx context.Context, hour int64, millis int64) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.fail {
		return context.DeadlineExceeded
	}

	repo.screen[hour] += millis

	return nil
}

func (repo *fakeCameraRepository) AddBlinkCount(ctx context.Context, hour int64, blinks int64) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.fail {
		return context.DeadlineExceeded
	}

	repo.blinks[hour] += blinks

	return nil
}

func (repo *fakeCameraRepository) totals() (millis int64, blinks int64) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	for _, v := range repo.screen {
		millis += v
	}

	for _, v := range repo.blinks {
		blinks += v
	}

	return millis, blinks
}

// short tick, explicit flushing only, blink reminder effectively disabled
func newTestCameraService(repo CameraRepository) *CameraService {
	svc := NewCameraService(repo, func() time.Duration { return time.Hour })
	svc.tick = 10 * time.Millisecond
	svc.flush = time.Hour

	return svc
}

func TestCameraServiceScreenTimeOnlyWhilePresent(t *testing.T) {
	ctx := context.Background()
	repo := newFakeCameraRepository()
	svc := newTestCameraService(repo)
	svc.Start()
	defer svc.Stop()

	svc.FaceDetected()
	svc.FaceDetected() // repeated edge must not double count
	time.Sleep(100 * time.Millisecond)
	svc.FaceLost(ctx)

	afterLost, _ := repo.totals()

	if afterLost == 0 {
		t.Fatal("expected screen time to be flushed on face lost")
	}

	time.Sleep(100 * time.Millisecond)
	svc.Flush(ctx)

	afterIdle, _ := repo.totals()

	if afterIdle != afterLost {
		t.Fatalf("screen time kept counting with no face: %d -> %d", afterLost, afterIdle)
	}
}

func TestCameraServiceFaceLostFlushes(t *testing.T) {
	ctx := context.Background()
	repo := newFakeCameraRepository()
	svc := newTestCameraService(repo)
	svc.Start()
	defer svc.Stop()

	svc.FaceDetected()
	svc.Blink()
	svc.Blink()
	svc.Blink()

	if err := svc.FaceLost(ctx); err != nil {
		t.Fatal(err)
	}

	if _, blinks := repo.totals(); blinks != 3 {
		t.Fatalf("expected 3 blinks flushed, got %d", blinks)
	}

	// a second face lost is a no-op
	if err := svc.FaceLost(ctx); err != nil {
		t.Fatal(err)
	}

	if _, blinks := repo.totals(); blinks != 3 {
		t.Fatalf("second face lost changed the blink count: %d", blinks)
	}
}

func TestCameraServiceFailedFlushKeepsBuffer(t *testing.T) {
	ctx := context.Background()
	repo := newFakeCameraRepository()
	repo.fail = true
	svc := newTestCameraService(repo)
	svc.Start()
	defer svc.Stop()

	svc.FaceDetected()
	svc.Blink()

	if err := svc.Flush(ctx); err == nil {
		t.Fatal("expected an error from a failing repository")
	}

	repo.mu.Lock()
	repo.fail = false
	repo.mu.Unlock()

	if err := svc.Flush(ctx); err != nil {
		t.Fatal(err)
	}

	if _, blinks := repo.totals(); blinks != 1 {
		t.Fatalf("expected the buffered blink to be retried, got %d", blinks)
	}
}

func TestCameraServiceBlinkReminder(t *testing.T) {
	ctx := context.Background()
	repo := newFakeCameraRepository()
	svc := NewCameraService(repo, func() time.Duration { return 20 * time.Millisecond })
	svc.tick = time.Hour
	svc.flush = time.Hour

	events := svc.Start()
	defer svc.Stop()

	svc.FaceDetected()

	select {
	case ev := <-events:
		if ev.typ != CameraEventBlinkReminder {
			t.Fatalf("expected a blink reminder, got %d", ev.typ)
		}
	case <-time.After(time.Second):
		t.Fatal("no blink reminder after the timeout")
	}

	// blinking dismisses the overlay
	svc.FaceLost(ctx)
	svc.Blink()

	select {
	case ev := <-events:
		if ev.typ != CameraEventBlinkDismiss {
			t.Fatalf("expected a blink dismiss, got %d", ev.typ)
		}
	case <-time.After(time.Second):
		t.Fatal("blink did not dismiss the overlay")
	}
}

func TestCameraServiceNoReminderWhenAbsent(t *testing.T) {
	repo := newFakeCameraRepository()
	svc := NewCameraService(repo, func() time.Duration { return 20 * time.Millisecond })
	svc.tick = time.Hour
	svc.flush = time.Hour

	events := svc.Start()
	defer svc.Stop()

	select {
	case ev := <-events:
		t.Fatalf("unexpected event while nobody is on screen: %d", ev.typ)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestCameraServiceReadsTimeoutOnEveryReset(t *testing.T) {
	repo := newFakeCameraRepository()
	timeout := time.Hour
	svc := NewCameraService(repo, func() time.Duration { return timeout })
	svc.tick = time.Hour
	svc.flush = time.Hour

	events := svc.Start()
	defer svc.Stop()

	svc.FaceDetected()

	select {
	case ev := <-events:
		t.Fatalf("expected no reminder with an hour long timeout, got %d", ev.typ)
	case <-time.After(50 * time.Millisecond):
	}

	// a saved setting must reach the running camera, no restart
	timeout = 20 * time.Millisecond
	svc.Blink()

	select {
	case ev := <-events:
		if ev.typ != CameraEventBlinkDismiss {
			t.Fatalf("expected the blink dismiss, got %d", ev.typ)
		}
	case <-time.After(time.Second):
		t.Fatal("blink did not dismiss the overlay")
	}

	select {
	case ev := <-events:
		if ev.typ != CameraEventBlinkReminder {
			t.Fatalf("expected a reminder with the new timeout, got %d", ev.typ)
		}
	case <-time.After(time.Second):
		t.Fatal("the timer kept the old timeout")
	}
}

func TestHourStart(t *testing.T) {
	now := time.Date(2026, time.November, 17, 13, 45, 30, 0, time.Local)
	start := time.Unix(hourStart(now), 0)

	if start.Hour() != 13 || start.Minute() != 0 || start.Second() != 0 {
		t.Fatalf("expected the start of the local hour, got %s", start)
	}

	if start.Day() != 17 {
		t.Fatalf("expected the same day, got %d", start.Day())
	}
}
