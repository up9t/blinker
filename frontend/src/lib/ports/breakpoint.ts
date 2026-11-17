export const DurationUnit = {
  SECOND: "second",
  MINUTE: "minute",
} as const;

export type DurationUnitType = (typeof DurationUnit)[keyof typeof DurationUnit];

export interface IBreakpoint {
  id: string;
  name: string;
  interval: number;
  intervalUnit: DurationUnitType;
  duration: number;
  durationUnit: DurationUnitType;
  enabled: boolean;
}

export type IBreakpointWithoutID = Omit<IBreakpoint, "id">;

// mirrors BreakInfo in breakpoint.go
export interface IBreakInfo {
  name: string;
  remainingMs: number;
}

// mirrors DefaultBreakpoint() in breakpoint.go
export const DEFAULT_BREAKPOINT: IBreakpointWithoutID = {
  name: "Break",
  interval: 20,
  intervalUnit: DurationUnit.MINUTE,
  duration: 20,
  durationUnit: DurationUnit.SECOND,
  enabled: true,
};

export interface IBreakpointService {
  // id generated on the backend
  add(point: IBreakpointWithoutID): Promise<IBreakpoint>;
  getById(id: IBreakpoint["id"]): Promise<IBreakpoint>;
  list(): Promise<IBreakpoint[]>;
  disable(id: IBreakpoint["id"]): Promise<void>;
  enable(id: IBreakpoint["id"]): Promise<void>;
  remove(id: IBreakpoint["id"]): Promise<void>;
  update(point: IBreakpoint): Promise<IBreakpoint>;

  // the break overlay reads this on mount and after a queued break starts
  activeBreak(): Promise<IBreakInfo | null>;
  postpone(): Promise<void>;
}
