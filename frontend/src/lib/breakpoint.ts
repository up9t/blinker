// how far the "Postpone" button pushes the next break away. Tune it here; the backend
// only validates the value the overlay sends.
export const POSTPONE_DELAY_MS = 30_000;

// the break overlay counts down from a deadline, so a slow tick can't drift
export function remainingMs(endsAt: number, now: number): number {
  return Math.max(0, endsAt - now);
}

// rounds up, so a fresh 20s break reads 0:20 and the last second stays visible
export function formatCountdown(ms: number): string {
  const totalSeconds = Math.ceil(Math.max(0, ms) / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}
