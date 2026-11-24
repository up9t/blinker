export const DurationUnit = {
  SECOND: "second",
  MINUTE: "minute",
} as const;

export type DurationUnitType = (typeof DurationUnit)[keyof typeof DurationUnit];

export interface IBreakpoint {
  id: string;
  name: string;
  message: string;
  interval: number;
  intervalUnit: DurationUnitType;
  duration: number;
  durationUnit: DurationUnitType;
  // asset names from the assets manager, empty means the user picked none
  audio: string;
  layout: string;
  enabled: boolean;
}

export type IBreakpointWithoutID = Omit<IBreakpoint, "id">;

// mirrors BreakInfo in breakpoint.go
export interface IBreakInfo {
  name: string;
  remainingMs: number;
  // a break that came back from a snooze can't be snoozed again
  postponed: boolean;
}

// mirrors DefaultBreakpoint() in breakpoint.go
export const DEFAULT_BREAKPOINT: IBreakpointWithoutID = {
  name: "Break",
  message: "",
  interval: 20,
  intervalUnit: DurationUnit.MINUTE,
  duration: 20,
  durationUnit: DurationUnit.SECOND,
  audio: "",
  layout: "",
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
  postpone(delayMs: number): Promise<void>;
}
