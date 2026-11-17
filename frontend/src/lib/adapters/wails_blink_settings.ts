import type { IBlinkSettings, IBlinkSettingsService } from "../ports/blink_settings.ts";
import { BlinkSettingsService } from "../../../bindings/github.com/up9t/blinker/index.js";

export class WailsBlinkSettingsService implements IBlinkSettingsService {
  async load(): Promise<IBlinkSettings> {
    return await BlinkSettingsService.Load();
  }

  async save(settings: IBlinkSettings): Promise<IBlinkSettings> {
    return await BlinkSettingsService.Save(settings);
  }

  async reset(): Promise<IBlinkSettings> {
    return await BlinkSettingsService.Reset();
  }
}
