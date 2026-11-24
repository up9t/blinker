import { useEffect, useState } from "react";
import { WailsAssetsService } from "../lib/adapters/wails_assets.ts";
import { ASSET_TYPE_LABEL, type IAsset } from "../lib/ports/assets.ts";

const assetsService = new WailsAssetsService();

export function AssetsPanel() {
  const [assets, setAssets] = useState<IAsset[] | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    void refresh();
  }, []);

  async function refresh() {
    setError("");

    try {
      setAssets([...(await assetsService.listAudio()), ...(await assetsService.listLayouts())]);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <>
      <button type="button" onClick={() => void refresh()}>
        Refresh
      </button>
      {error !== "" && <p role="alert">{error}</p>}
      {assets === null && error === "" && <p>Loading…</p>}

      {assets !== null &&
        Object.entries(ASSET_TYPE_LABEL).map(([type, label]) => (
          <section key={type}>
            <h2>{label}</h2>
            <ul>
              {assets
                .filter((asset) => asset.type === type)
                .map((asset) => (
                  <li key={asset.name}>{asset.name}</li>
                ))}
            </ul>
          </section>
        ))}
    </>
  );
}
