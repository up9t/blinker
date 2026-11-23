import { beforeEach, describe, expect, it, vi } from "vitest";

const bindings = vi.hoisted(() => ({
  Statistics: vi.fn(),
  TodayStatistics: vi.fn(),
}));

vi.mock("../../../bindings/github.com/up9t/blinker/index.js", () => ({
  CameraService: bindings,
}));

import { WailsStatisticsService } from "./wails_statistics.ts";
import type { IStatPeriodSet, IStatistics } from "../ports/statistics.ts";

const emptyPeriod: IStatPeriodSet = {
  today: { total: 0, buckets: [] },
  week: { total: 0, buckets: [] },
  month: { total: 0, buckets: [] },
};

const stats: IStatistics = { blinks: emptyPeriod, screenTime: emptyPeriod };

const service = new WailsStatisticsService();

beforeEach(() => {
  vi.resetAllMocks();
});

describe("WailsStatisticsService", () => {
  it("reads the statistics from the camera service", async () => {
    bindings.Statistics.mockResolvedValue(stats);

    expect(await service.statistics()).toEqual(stats);
    expect(bindings.Statistics).toHaveBeenCalledOnce();
  });

  it("reads today's totals from the camera service", async () => {
    const today = { blinks: 12, screenTime: 90_000 };
    bindings.TodayStatistics.mockResolvedValue(today);

    expect(await service.today()).toEqual(today);
    expect(bindings.TodayStatistics).toHaveBeenCalledOnce();
  });
});
