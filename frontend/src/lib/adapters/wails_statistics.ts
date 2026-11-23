import type { IStatistics, IStatisticsService, ITodayStatistics } from "../ports/statistics.ts";
import { CameraService } from "../../../bindings/github.com/up9t/blinker/index.js";

// the blink and screen time rows live with the camera service, buffer included
export class WailsStatisticsService implements IStatisticsService {
  async statistics(): Promise<IStatistics> {
    return await CameraService.Statistics();
  }

  async today(): Promise<ITodayStatistics> {
    return await CameraService.TodayStatistics();
  }
}
