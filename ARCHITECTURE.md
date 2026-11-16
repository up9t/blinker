# Blinker Architecture

## Stack

Backend (Desktop Framework): Wails v3 + Go
Frontend: Astro + React + Tailwind + TypeScript
Storage: Sqlite (Store user settings/preferences)

## Project Structure

- `main.go` -> backend entry point
- `frontend/` -> frontend directory

## Blink Strategy

1. Setup a camera.
2. Detect user has blinked in a specific timeout period,
3. if the user blinked the timeout reset. 
4. if the user hasn't blinked and the time is out, emit backend event.
5. Backend shows another window called `overlay window`. Completely block the screen, forcing the user to blink.
6. Hide() method is called to hide the overlay window, when the user is finally blinked.

## Break Strategy

Reminder to take a quick rest, like 202020 rule, for eyes and move body to keep them healthy.

1. User set a breakpoint (break interval, break duration)
2. When interval time is done, show the same overlay window but with different url or page than the blink overlay, with better ui to show info such as time left, button to postpone 30 seconds, and show current time.
3. After break overlay showed for a X duration, kicks the cycle again.