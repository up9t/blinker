import { describe, expect, it } from "vitest";
import { formatDuration } from "./duration.ts";

describe("formatDuration", () => {
  it("shows seconds below a minute", () => {
    expect(formatDuration(45_000)).toBe("45s");
  });

  it("shows minutes below an hour", () => {
    expect(formatDuration(59 * 60_000)).toBe("59m");
  });

  it("shows hours and the remaining minutes from an hour on", () => {
    expect(formatDuration(60 * 60_000)).toBe("1h 0m");
    expect(formatDuration(65 * 60_000)).toBe("1h 5m");
    expect(formatDuration(150 * 60_000)).toBe("2h 30m");
  });
});
