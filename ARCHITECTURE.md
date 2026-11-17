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

Reminder to take a quick rest, like 202020 rule, for eyes and move body to keep them healthy.

1. User set a breakpoint (break interval, break duration)
2. When interval time is done, show the same overlay window but with different url or page than the blink overlay, with better ui to show info such as time left, button to postpone 30 seconds, and show current time.
3. After break overlay showed for a X duration, kicks the cycle again.