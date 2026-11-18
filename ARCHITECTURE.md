# Blinker Architecture

## Stack

Backend (Desktop Framework): Wails v3 + Go
Frontend: Astro + React + Tailwind + TypeScript
Storage: Sqlite (Store user settings/preferences)

## Project Structure

- `main.go` -> backend entry point
- `frontend/` -> frontend directory

### Backend

- Write idiomatic, effective Go.
- Use clean architecture, separation between business logic, app logic, and adapter for external libraries. Domain layer shouldn't be using any external libraries.

### Frontend

- Use ports(interface)/adapters pattern when communicating with external libraries. Goal is to easily change the backend/library when needed.
- Separation of concerns.
- Keep the component script/logic thin.

## Blink Strategy

1. Setup a camera.
2. Detect user has blinked in a specific timeout period,
3. if the user blinked the timeout reset. 
4. if the user hasn't blinked and the time is out, emit backend event.
5. Backend shows another window called `overlay window`. Completely block the screen, forcing the user to blink.
6. Hide() method is called to hide the overlay window, when the user is finally blinked.

### Who owns what

The frontend only detects and reports edges to `CameraService` (`frontend/src/lib/camera_detector.ts`):

- `FaceDetected` -> backend starts the screen time and blink timers.
- `FaceLost` (face gone 3s, camera stopped, window hidden) -> backend stops the timers and flushes.
- `Blink` -> backend counts it and resets the blink timer.

All timers live in the backend (`camera_service.go`), together with the buffered screen time
and blink count written to sqlite every 30s. While nobody is on screen the frontend only runs
the detector once per second (efficiency mode).

The blink parameters (eye open/close thresholds, blink timeout) are user settings owned by
`BlinkSettingsService` (`blink_settings_service.go`): one row in sqlite, loaded on startup so the
camera timers use them before the frontend mounts, edited on the Camera page through
`BlinkSettingsPanel` and saved with a debounce. The camera service reads the timeout on every
timer reset, the detector reads the thresholds on every frame, so a change applies without a
restart. General app settings (editor path, ...) are a separate concern, not these.


## Break Strategy

Reminder to take a quick rest, like the 202020 rule, for eyes and to move the body. A user can
have several breakpoints at once.

A breakpoint is a row in its own `breakpoints` table (own domain, never mixed with
`blink_settings` or global settings): name, break interval, break duration, enabled.
`BreakpointService` (`breakpoint_service.go`) owns the whole schedule in the backend:

1. `Start` loads the stored breakpoints before the windows open and arms one interval timer
   per enabled breakpoint. An interval of 40s or more also gets a reminder timer
   (`breakpointNotifyLeadTime`) that emits `BreakpointEventBreakSoon` 40s earlier;
   `main.go` turns it into a desktop notification titled with the app name.
2. When an interval is due and no break is on screen, the service starts the break, emits
   `StartBreak`, and arms the duration timer.
3. `main.go` shows the shared overlay window with `/break/` and passes the `BreakInfo`
   payload through the `break:start` event. The page also reads `ActiveBreak()` on mount, so
   it hydrates whether or not the page was already loaded.
4. When the duration is over the service emits `FinishBreak`, `main.go` hides the overlay and
   the service rearms that breakpoint's interval timer: the cycle repeats.
5. A break that comes due while another break is on screen is queued, not replaced, and
   starts as soon as the running one ends.
6. `Postpone(delayMs)` snoozes: the running break ends now and that breakpoint reminds again
   after the delay the frontend sends. The delay is a frontend constant
   (`POSTPONE_DELAY_MS`, 30s today) and the backend only validates it. A break already in the
   queue starts right away.
7. The tray menu has "Reset next break", which restarts only the breakpoint whose break comes
   next (a break already queued behind a running one counts as next). The running break and
   every other timer keep their deadlines.

The timers share the camera timers' pattern (one mutex, a generation counter per breakpoint so
a stale timer can't fire, and no event channel close while callbacks may still run). Every
duration goes through one `speedup` division that stays 1 in production, so tests run a 20
minute cycle in milliseconds.

The Breakpoints page (`BreakpointPanel`) hydrates from `List()` on mount and writes every
add, update, enable/disable and delete immediately. The break overlay (`BreakOverlay`) only
reads `ActiveBreak()`, listens for `break:start` and calls `Postpone()`, so no break state
lives in the frontend.

## Statistics

The camera data is read back through `CameraService.Statistics(ctx)` (`statistics.go`): it
queries the current month (plus the days of this week that fall in the previous month) from
the two hourly tables via `CameraRepository` (`ListBlinkCounts`, `ListScreenTime`), merges the
buffer that has not been flushed yet (`hourBlinks`, `hourMillis`), and buckets both metrics
into today (24 hours), this week (7 days, Monday first) and this month (one bucket per day,
zero for days without data).
Every total is the sum of its own buckets.

`StatisticsPanel` (`statistics.astro`) reads it on mount and on Refresh and renders bare
tables, so styling the graphs is a frontend job; the backend only owns the numbers.