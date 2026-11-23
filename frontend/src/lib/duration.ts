// screen time arrives in millis, which are unreadable as a total
export function formatDuration(millis: number): string {
  const minutes = Math.floor(millis / 60_000);

  if (minutes < 1) {
    return `${Math.floor(millis / 1000)}s`;
  }

  if (minutes < 60) {
    return `${minutes}m`;
  }

  return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
}
