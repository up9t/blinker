import type { IBreakpoint } from "./breakpoint.ts";

export interface ISettings {
  thresholdEyesClosed: number;
  thresholdEyesOpened: number;
  blinkTimeout: number;
  notifyBeforeSecond: number;
  autoStart: boolean;
}

export const DEFAULT_SETTINGS: ISettings = {
  thresholdEyesClosed: 0.525,
  thresholdEyesOpened: 0.5,
  blinkTimeout: 1.2,
  notifyBeforeSecond: 20,
  autoStart: false,
};

export interface ISettingsService {
  load(): Promise<ISettings>;
  save(state: ISettings): Promise<void>;
  reset(): Promise<ISettings>;
}
