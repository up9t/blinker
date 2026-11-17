import { describe, expect, it } from "vitest";
import { formatCountdown, remainingMs } from "./breakpoint.ts";

describe("formatCountdown", () => {
  it("shows minutes and padded seconds", () => {
    expect(formatCountdown(20_000)).toBe("0:20");
    expect(formatCountdown(65_000)).toBe("1:05");
    expect(formatCountdown(3_600_000)).toBe("60:00");
  });

  it("rounds up so the last second stays visible", () => {
    expect(formatCountdown(1)).toBe("0:01");
    expect(formatCountdown(0)).toBe("0:00");
    expect(formatCountdown(-5)).toBe("0:00");
  });
});

describe("remainingMs", () => {
  it("counts down to the deadline and stops at zero", () => {
    expect(remainingMs(1_000, 400)).toBe(600);
    expect(remainingMs(1_000, 1_500)).toBe(0);
  });
});
