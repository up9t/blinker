export interface IBlinkSettings {
  eyeCloseThreshold: number;
  eyeOpenThreshold: number;
  blinkTimeoutMs: number;
}

// mirrors DefaultBlinkSettings() in blink_settings.go
export const DEFAULT_BLINK_SETTINGS: IBlinkSettings = {
  eyeCloseThreshold: 0.525,
  eyeOpenThreshold: 0.5,
  blinkTimeoutMs: 1200,
};

export interface IBlinkSettingsService {
  load(): Promise<IBlinkSettings>;
  save(settings: IBlinkSettings): Promise<IBlinkSettings>;
  reset(): Promise<IBlinkSettings>;
}
