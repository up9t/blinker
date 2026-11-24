import { beforeEach, describe, expect, it, vi } from "vitest";

const bindings = vi.hoisted(() => ({
  ListAudio: vi.fn(),
  ListLayouts: vi.fn(),
}));

vi.mock("../../../bindings/github.com/up9t/blinker/index.js", () => ({
  AssetsService: bindings,
}));

import { WailsAssetsService } from "./wails_assets.ts";
import { AssetType, type IAsset } from "../ports/assets.ts";

const audio: IAsset[] = [{ name: "solemn", type: AssetType.AUDIO }];
const layouts: IAsset[] = [{ name: "default", type: AssetType.LAYOUT }];

const service = new WailsAssetsService();

beforeEach(() => {
  vi.resetAllMocks();
});

describe("WailsAssetsService", () => {
  it("lists the audio assets", async () => {
    bindings.ListAudio.mockResolvedValue(audio);

    expect(await service.listAudio()).toEqual(audio);
    expect(bindings.ListAudio).toHaveBeenCalledOnce();
  });

  it("lists the layout assets", async () => {
    bindings.ListLayouts.mockResolvedValue(layouts);

    expect(await service.listLayouts()).toEqual(layouts);
    expect(bindings.ListLayouts).toHaveBeenCalledOnce();
  });

  it("treats a missing list as no assets", async () => {
    bindings.ListAudio.mockResolvedValue(null);
    bindings.ListLayouts.mockResolvedValue(null);

    expect(await service.listAudio()).toEqual([]);
    expect(await service.listLayouts()).toEqual([]);
  });
});
