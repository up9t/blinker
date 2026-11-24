import type { IAsset, IAssetsService } from "../ports/assets.ts";
import { AssetsService } from "../../../bindings/github.com/up9t/blinker/index.js";

export class WailsAssetsService implements IAssetsService {
  async listAudio(): Promise<IAsset[]> {
    return (await AssetsService.ListAudio()) ?? [];
  }

  async listLayouts(): Promise<IAsset[]> {
    return (await AssetsService.ListLayouts()) ?? [];
  }
}
