// mirrors AssetType in asset.go
export const AssetType = {
  AUDIO: "audio",
  LAYOUT: "layout",
} as const;

export type AssetType = (typeof AssetType)[keyof typeof AssetType];

// mirrors Asset in asset.go
export interface IAsset {
  // the file name without its extension, unique inside its type
  name: string;
  type: AssetType;
}

export interface IAssetsService {
  listAudio(): Promise<IAsset[]>;
  listLayouts(): Promise<IAsset[]>;
}

// what each list feeds, used for the section titles on the assets page
export const ASSET_TYPE_LABEL: Record<AssetType, string> = {
  [AssetType.AUDIO]: "Audio",
  [AssetType.LAYOUT]: "Layout",
};
