package main

import (
	"context"
	"sync"
	"time"
)

const (
	cameraTickInterval    = time.Second
	cameraFlushInterval   = 30 * time.Second
	cameraEventBufferSize = 16
)

// CameraService owns all the timers. The frontend only reports edges:
// face found, face lost and blink.
type CameraService struct {
	repo    CameraRepository
	eventCh chan CameraEvent

	mu      sync.Mutex
	present bool
	// unflushed data, hour -> value
	hourMillis map[int64]int64
	hourBlinks map[int64]int64
	blinkTimer *time.Timer

	tick    time.Duration
	timeout func() time.Duration
	flush   time.Duration

	done     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// timeout is read on every blink timer reset, so a changed setting applies without
// restarting the camera
func NewCameraService(repo CameraRepository, timeout func() time.Duration) *CameraService {
	return &CameraService{
		repo:       repo,
		eventCh:    make(chan CameraEvent, cameraEventBufferSize),
		hourMillis: make(map[int64]int64),
		hourBlinks: make(map[int64]int64),
		tick:       cameraTickInterval,
		timeout:    timeout,
		flush:      cameraFlushInterval,
		done:       make(chan struct{}),
	}
}

func (svc *CameraService) Start() <-chan CameraEvent {
	svc.wg.Add(2)
	go svc.runTicker()
	go svc.runFlusher()

	return svc.eventCh
}

func (svc *CameraService) Stop() {
	svc.stopOnce.Do(func() {
		close(svc.done)
	})

	svc.wg.Wait()

	svc.mu.Lock()
	if svc.blinkTimer != nil {
		svc.blinkTimer.Stop()
		svc.blinkTimer = nil
	}
	svc.mu.Unlock()

	// nothing left to do on shutdown if this fails
	svc.Flush(context.Background())
}

func (svc *CameraService) runTicker() {
	defer svc.wg.Done()

	ticker := time.NewTicker(svc.tick)
	defer ticker.Stop()

	for {
		select {
		case <-svc.done:
			return
		case <-ticker.C:
			svc.addMillis(hourStart(time.Now()), int64(svc.tick/time.Millisecond))
		}
	}
}

func (svc *CameraService) runFlusher() {
	defer svc.wg.Done()

	ticker := time.NewTicker(svc.flush)
	defer ticker.Stop()

	for {
		select {
		case <-svc.done:
			return
		case <-ticker.C:
			svc.Flush(context.Background())
		}
	}
}

// A tick only counts as screen time while the user's face is on screen.
func (svc *CameraService) addMillis(hour int64, millis int64) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if !svc.present {
		return
	}

	svc.hourMillis[hour] += millis
}

func (svc *CameraService) addBlinks(hour int64) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.hourBlinks[hour]++
}

// Flush writes the buffer to the database. Failed writes stay in the buffer.
func (svc *CameraService) Flush(ctx context.Context) error {
	svc.mu.Lock()
	millis := svc.hourMillis
	blinks := svc.hourBlinks
	svc.hourMillis = make(map[int64]int64)
	svc.hourBlinks = make(map[int64]int64)
	svc.mu.Unlock()

	var flushErr error

	for hour, value := range millis {
		if err := svc.repo.AddScreenTime(ctx, hour, value); err != nil {
			flushErr = err
			svc.restoreMillis(hour, value)
		}
	}

	for hour, value := range blinks {
		if err := svc.repo.AddBlinkCount(ctx, hour, value); err != nil {
			flushErr = err
			svc.restoreBlinks(hour, value)
		}
	}

	return flushErr
}

func (svc *CameraService) restoreMillis(hour int64, millis int64) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.hourMillis[hour] += millis
}

func (svc *CameraService) restoreBlinks(hour int64, blinks int64) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.hourBlinks[hour] += blinks
}

// FaceDetected starts the screen time and blink timers. Repeated calls are no-ops.
func (svc *CameraService) FaceDetected() {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.present {
		return
	}

	svc.present = true
	svc.resetBlinkTimer()
}

// FaceLost stops the timers and flushes what was collected so far.
func (svc *CameraService) FaceLost(ctx context.Context) error {
	svc.mu.Lock()

	if !svc.present {
		svc.mu.Unlock()
		return nil
	}

	svc.present = false
	if svc.blinkTimer != nil {
		svc.blinkTimer.Stop()
		svc.blinkTimer = nil
	}
	svc.mu.Unlock()

	svc.emit(CameraEventBlinkDismiss)

	return svc.Flush(ctx)
}

func (svc *CameraService) Blink() {
	svc.addBlinks(hourStart(time.Now()))
	svc.emit(CameraEventBlinkDismiss)

	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.present {
		svc.resetBlinkTimer()
	}
}

// caller must hold svc.mu
func (svc *CameraService) resetBlinkTimer() {
	if svc.blinkTimer != nil {
		svc.blinkTimer.Stop()
	}

	svc.blinkTimer = time.AfterFunc(svc.timeout(), func() {
		svc.mu.Lock()
		present := svc.present
		svc.mu.Unlock()

		if present {
			svc.emit(CameraEventBlinkReminder)
		}
	})
}

func (svc *CameraService) emit(typ CameraEventType) {
	select {
	case svc.eventCh <- CameraEvent{typ: typ}:
	case <-svc.done:
	}
}
