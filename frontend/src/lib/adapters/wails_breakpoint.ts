import {
  type IBreakInfo,
  type IBreakpoint,
  type IBreakpointService,
  type IBreakpointWithoutID,
} from "../ports/breakpoint.ts";
import { BreakpointService } from "../../../bindings/github.com/up9t/blinker/index.js";

export class WailsBreakpointService implements IBreakpointService {
  async add(point: IBreakpointWithoutID): Promise<IBreakpoint> {
    return await BreakpointService.Add({ ...point, id: "" });
  }

  async getById(id: IBreakpoint["id"]): Promise<IBreakpoint> {
    return await BreakpointService.GetByID(id);
  }

  async list(): Promise<IBreakpoint[]> {
    return (await BreakpointService.List()) ?? [];
  }

  async disable(id: IBreakpoint["id"]): Promise<void> {
    await BreakpointService.Disable(id);
  }

  async enable(id: IBreakpoint["id"]): Promise<void> {
    await BreakpointService.Enable(id);
  }

  async remove(id: IBreakpoint["id"]): Promise<void> {
    await BreakpointService.Remove(id);
  }

  async update(point: IBreakpoint): Promise<IBreakpoint> {
    return await BreakpointService.Update(point);
  }

  async activeBreak(): Promise<IBreakInfo | null> {
    return await BreakpointService.ActiveBreak();
  }

  async postpone(delayMs: number): Promise<void> {
    await BreakpointService.Postpone(delayMs);
  }
}
