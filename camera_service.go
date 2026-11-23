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
	// the page still has to check in: a stalled, crashed or hidden frontend would otherwise
	// leave the timers counting with nobody there to report a face lost
	cameraPresenceTTL = 3 * time.Second
	// every this much counted screen time in a day, remind the user to take a break
	cameraReminderInterval = time.Hour
)

// CameraService owns all the timers. The frontend only reports edges:
// face found, face lost and blink.
type CameraService struct {
	repo    CameraRepository
	eventCh chan CameraEvent

	mu      sync.Mutex
	present bool
	// last heartbeat, so a frontend that went away stops counting
	lastSeen time.Time
	// unflushed data, hour -> value
	hourMillis map[int64]int64
	hourBlinks map[int64]int64
	blinkTimer *time.Timer
	// local start of the day todayMillis belongs to, so a new day resets the reminder
	dayStart      int64
	todayMillis   int64
	notifiedHours int64

	tick        time.Duration
	presenceTTL time.Duration
	reminder    time.Duration
	timeout     func() time.Duration
	flush       time.Duration

	done     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// timeout is read on every blink timer reset, so a changed setting applies without
// restarting the camera
func NewCameraService(repo CameraRepository, timeout func() time.Duration) *CameraService {
	return &CameraService{
		repo:        repo,
		eventCh:     make(chan CameraEvent, cameraEventBufferSize),
		hourMillis:  make(map[int64]int64),
		hourBlinks:  make(map[int64]int64),
		tick:        cameraTickInterval,
		presenceTTL: cameraPresenceTTL,
		reminder:    cameraReminderInterval,
		timeout:     timeout,
		flush:       cameraFlushInterval,
		done:        make(chan struct{}),
	}
}

func (svc *CameraService) Start() <-chan CameraEvent {
	svc.seedReminder(context.Background())

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

// A tick only counts as screen time while the user's face is on screen and the page is
// still checking in.
func (svc *CameraService) addMillis(hour int64, millis int64) {
	svc.mu.Lock()

	if !svc.fresh() {
		svc.mu.Unlock()
		return
	}

	svc.hourMillis[hour] += millis

	// the reminder is a day total, so a new day starts counting hours again
	if day := startOfDay(time.Unix(hour, 0)).Unix(); day != svc.dayStart {
		svc.dayStart = day
		svc.todayMillis = 0
		svc.notifiedHours = 0
	}

	svc.todayMillis += millis
	hours := svc.todayMillis / int64(svc.reminder/time.Millisecond)
	notify := int64(0)

	if hours > svc.notifiedHours {
		svc.notifiedHours = hours
		notify = hours
	}

	// emitted after the unlock: a send must not happen while the ticker holds the mutex
	svc.mu.Unlock()

	if notify > 0 {
		svc.emit(CameraEvent{typ: CameraEventScreenTimeReminder, hours: int(notify)})
	}
}

// seedReminder continues from the stored day total, so a relaunch doesn't announce the
// hours that were already reached. A failed read just starts the day from zero.
func (svc *CameraService) seedReminder(ctx context.Context) {
	today, err := svc.TodayStatistics(ctx)

	if err != nil {
		return
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.dayStart = startOfDay(time.Now()).Unix()
	svc.todayMillis = today.ScreenTime
	svc.notifiedHours = today.ScreenTime / int64(svc.reminder/time.Millisecond)
}

// caller must hold svc.mu
func (svc *CameraService) fresh() bool {
	return svc.present && time.Since(svc.lastSeen) < svc.presenceTTL
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

// statisticsRange is the span of hourly rows the page needs: the week can start in the
// previous month, so the month alone is not enough.
func statisticsRange(now time.Time) (from int64, to int64) {
	from = startOfWeek(now).Unix()

	if month := startOfMonth(now).Unix(); month < from {
		from = month
	}

	return from, startOfDay(now).AddDate(0, 0, 1).Unix()
}

// Statistics reads the hourly rows back and adds the buffer that hasn't been flushed yet, so
// the page also shows the blinks and the screen time of the current minute.
func (svc *CameraService) Statistics(ctx context.Context) (Statistics, error) {
	now := time.Now()
	from, to := statisticsRange(now)

	// the lock spans the reads: a flush in between would move data out of the buffer after
	// the read and the merge would miss it
	svc.mu.Lock()
	defer svc.mu.Unlock()

	blinks, err := svc.repo.ListBlinkCounts(ctx, from, to)

	if err != nil {
		return Statistics{}, err
	}

	screen, err := svc.repo.ListScreenTime(ctx, from, to)

	if err != nil {
		return Statistics{}, err
	}

	mergeHours(blinks, svc.hourBlinks, from, to)
	mergeHours(screen, svc.hourMillis, from, to)

	return Statistics{
		Blinks:     buildStatistics(now, blinks),
		ScreenTime: buildStatistics(now, screen),
	}, nil
}

// TodayStatistics is the cheap read for the camera widget: only today's rows are read and
// the unflushed buffer is added, so no buckets and no week/month range are built.
func (svc *CameraService) TodayStatistics(ctx context.Context) (TodayStatistics, error) {
	day := startOfDay(time.Now())
	from, to := day.Unix(), day.AddDate(0, 0, 1).Unix()

	// same lock span as Statistics: a flush in between would move data out of the buffer
	svc.mu.Lock()
	defer svc.mu.Unlock()

	blinks, err := svc.repo.ListBlinkCounts(ctx, from, to)

	if err != nil {
		return TodayStatistics{}, err
	}

	screen, err := svc.repo.ListScreenTime(ctx, from, to)

	if err != nil {
		return TodayStatistics{}, err
	}

	return TodayStatistics{
		Blinks:     sumHours(blinks, from, to) + sumHours(svc.hourBlinks, from, to),
		ScreenTime: sumHours(screen, from, to) + sumHours(svc.hourMillis, from, to),
	}, nil
}

// FaceDetected starts the screen time and blink timers. It doubles as the presence
// heartbeat: repeated calls only refresh the liveness stamp.
func (svc *CameraService) FaceDetected() {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	svc.lastSeen = time.Now()

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

	svc.emit(CameraEvent{typ: CameraEventBlinkDismiss})

	return svc.Flush(ctx)
}

func (svc *CameraService) Blink() {
	svc.addBlinks(hourStart(time.Now()))
	svc.emit(CameraEvent{typ: CameraEventBlinkDismiss})

	svc.mu.Lock()
	defer svc.mu.Unlock()

	// a blink comes from the same page, so it proves the page is still there
	svc.lastSeen = time.Now()

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
		fresh := svc.fresh()
		svc.mu.Unlock()

		if fresh {
			svc.emit(CameraEvent{typ: CameraEventBlinkReminder})
		}
	})
}

func (svc *CameraService) emit(ev CameraEvent) {
	select {
	case svc.eventCh <- ev:
	case <-svc.done:
	}
}
