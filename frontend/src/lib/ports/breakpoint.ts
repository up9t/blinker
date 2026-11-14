export const DurationUnit = {
  SECOND: 0,
  MINUTE: 1,
}

export type DurationUnitType = typeof DurationUnit[keyof typeof DurationUnit]

export interface IBreakpoint {
  id: string;
  interval: number;
  intervalUnit: DurationUnitType;
  duration: number;
  durationUnit: DurationUnitType;
  enabled: boolean;
}

export type IBreakpointWithoutID = Omit<IBreakpoint, "id">

export interface IBreakpointService {
  // id generated on the backend
  add(point: IBreakpointWithoutID): Promise<IBreakpoint | Error>
  getById(id: IBreakpoint["id"]): Promise<IBreakpoint | Error>
  list(): Promise<IBreakpoint[]>
  disable(id: IBreakpoint["id"]): Promise<void | Error>
  enable(id: IBreakpoint["id"]): Promise<void | Error>
  remove(id: IBreakpoint["id"]): Promise<void | Error>
  update(point: IBreakpoint): Promise<IBreakpoint | Error>
}