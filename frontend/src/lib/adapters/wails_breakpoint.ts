import {
  DurationUnit,
  type IBreakpoint,
  type IBreakpointService,
  type IBreakpointWithoutID,
} from "../ports/breakpoint.ts";

export class WailsBreakpointService implements IBreakpointService {
  private static readonly DEFAULT_VALUE = {
    interval: 20,
    intervalUnit: DurationUnit.MINUTE,
    duration: 20,
    durationUnit: DurationUnit.SECOND,
    enabled: true,
  };

  add(point: IBreakpointWithoutID): Promise<IBreakpoint | Error> {}

  disable(id: IBreakpoint["id"]): Promise<void | Error> {}

  enable(id: IBreakpoint["id"]): Promise<void | Error> {}

  getById(id: IBreakpoint["id"]): Promise<IBreakpoint | Error> {}

  list(): Promise<IBreakpoint[]> {}

  remove(id: IBreakpoint["id"]): Promise<void | Error> {}

  update(point: IBreakpoint): Promise<IBreakpoint | Error> {}
}
