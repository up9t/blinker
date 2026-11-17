import { beforeEach, describe, expect, it, vi } from "vitest";

const bindings = vi.hoisted(() => ({
  Add: vi.fn(),
  GetByID: vi.fn(),
  List: vi.fn(),
  Disable: vi.fn(),
  Enable: vi.fn(),
  Remove: vi.fn(),
  Update: vi.fn(),
  ActiveBreak: vi.fn(),
  Postpone: vi.fn(),
}));

vi.mock("../../../bindings/github.com/up9t/blinker/index.js", () => ({
  BreakpointService: bindings,
}));

import { WailsBreakpointService } from "./wails_breakpoint.ts";
import { DurationUnit, type IBreakpoint } from "../ports/breakpoint.ts";

const point: IBreakpoint = {
  id: "7",
  name: "Break",
  interval: 20,
  intervalUnit: DurationUnit.MINUTE,
  duration: 20,
  durationUnit: DurationUnit.SECOND,
  enabled: true,
};

const service = new WailsBreakpointService();

beforeEach(() => {
  vi.resetAllMocks();
});

describe("WailsBreakpointService", () => {
  it("treats a missing list as no breakpoints", async () => {
    bindings.List.mockResolvedValue(null);

    expect(await service.list()).toEqual([]);
  });

  it("adds without an id, so the backend generates one", async () => {
    bindings.Add.mockResolvedValue(point);

    await service.add({ ...point, id: "ignored" } as IBreakpoint);

    expect(bindings.Add).toHaveBeenCalledWith({ ...point, id: "" });
  });

  it("updates the whole record, id included", async () => {
    bindings.Update.mockResolvedValue(point);

    await service.update(point);

    expect(bindings.Update).toHaveBeenCalledWith(point);
  });

  it("toggles and removes by id", async () => {
    await service.enable(point.id);
    await service.disable(point.id);
    await service.remove(point.id);

    expect(bindings.Enable).toHaveBeenCalledWith(point.id);
    expect(bindings.Disable).toHaveBeenCalledWith(point.id);
    expect(bindings.Remove).toHaveBeenCalledWith(point.id);
  });

  it("returns no running break as null and postpones it", async () => {
    bindings.ActiveBreak.mockResolvedValue(null);

    expect(await service.activeBreak()).toBeNull();

    await service.postpone();

    expect(bindings.Postpone).toHaveBeenCalledOnce();
  });
});
