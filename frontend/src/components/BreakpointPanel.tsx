import { useEffect, useId, useState, type ChangeEvent, type SubmitEvent } from "react";
import { WailsAssetsService } from "../lib/adapters/wails_assets.ts";
import { WailsBreakpointService } from "../lib/adapters/wails_breakpoint.ts";
import { AssetType, type IAsset } from "../lib/ports/assets.ts";
import {
  DEFAULT_BREAKPOINT,
  DurationUnit,
  type DurationUnitType,
  type IBreakpoint,
  type IBreakpointWithoutID,
} from "../lib/ports/breakpoint.ts";

const breakpointService = new WailsBreakpointService();
const assetsService = new WailsAssetsService();

const EMPTY_DRAFT: IBreakpoint = { ...DEFAULT_BREAKPOINT, id: "" };

export function BreakpointPanel() {
  const [breakpoints, setBreakpoints] = useState<IBreakpoint[]>([]);
  const [assets, setAssets] = useState<IAsset[]>([]);
  const [draft, setDraft] = useState<IBreakpoint>(EMPTY_DRAFT);
  const [error, setError] = useState("");
  const nameId = useId();
  const messageId = useId();
  const intervalId = useId();
  const durationId = useId();
  const audioId = useId();
  const layoutId = useId();
  const editing = draft.id !== "";

  useEffect(() => {
    refresh().catch((err) => setError(err instanceof Error ? err.message : String(err)));
    loadAssets().catch((err) => setError(err instanceof Error ? err.message : String(err)));
  }, []);

  async function refresh() {
    setBreakpoints(await breakpointService.list());
  }

  async function loadAssets() {
    setAssets([...(await assetsService.listAudio()), ...(await assetsService.listLayouts())]);
  }

  const assetNames = (type: IAsset["type"]) =>
    assets.filter((asset) => asset.type === type).map((asset) => asset.name);

  async function run(action: () => Promise<unknown>) {
    setError("");

    try {
      await action();
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  const change = (patch: Partial<IBreakpointWithoutID>) => setDraft({ ...draft, ...patch });

  const onUnitChange =
    (key: "intervalUnit" | "durationUnit") => (e: ChangeEvent<HTMLSelectElement>) => {
      change({ [key]: e.target.value as DurationUnitType });
    };

  const onAssetChange = (key: "audio" | "layout") => (e: ChangeEvent<HTMLSelectElement>) => {
    change({ [key]: e.target.value });
  };

  const onSubmit = (e: SubmitEvent) => {
    e.preventDefault();

    void run(async () => {
      if (editing) {
        await breakpointService.update(draft);
      } else {
        await breakpointService.add(draft);
      }

      setDraft(EMPTY_DRAFT);
    });
  };

  return (
    <>
      {breakpoints.length === 0 && <p>No breakpoints yet.</p>}

      <ul>
        {breakpoints.map((bp) => (
          <li key={bp.id}>
            <strong>{bp.name}</strong> every {bp.interval} {bp.intervalUnit} for {bp.duration}{" "}
            {bp.durationUnit}{" "}
            <label>
              <input
                type="checkbox"
                checked={bp.enabled}
                onChange={(e) =>
                  void run(() =>
                    e.target.checked
                      ? breakpointService.enable(bp.id)
                      : breakpointService.disable(bp.id),
                  )
                }
              />{" "}
              enabled
            </label>{" "}
            <button type="button" onClick={() => setDraft(bp)}>
              Edit
            </button>{" "}
            <button type="button" onClick={() => void run(() => breakpointService.remove(bp.id))}>
              Delete
            </button>
          </li>
        ))}
      </ul>

      <form onSubmit={onSubmit}>
        <p>
          <label htmlFor={nameId}>Name </label>
          <input
            id={nameId}
            required
            value={draft.name}
            onChange={(e) => change({ name: e.target.value })}
          />
        </p>
        <p>
          <label htmlFor={messageId}>Message </label>
          <input
            id={messageId}
            value={draft.message}
            onChange={(e) => change({ message: e.target.value })}
          />
        </p>
        <p>
          <label htmlFor={intervalId}>Break every </label>
          <input
            id={intervalId}
            type="number"
            min={1}
            required
            value={draft.interval}
            onChange={(e) => change({ interval: Number(e.target.value) })}
          />
          <select value={draft.intervalUnit} onChange={onUnitChange("intervalUnit")}>
            <option value={DurationUnit.MINUTE}>minutes</option>
            <option value={DurationUnit.SECOND}>seconds</option>
          </select>
        </p>
        <p>
          <label htmlFor={durationId}>Break for </label>
          <input
            id={durationId}
            type="number"
            min={1}
            required
            value={draft.duration}
            onChange={(e) => change({ duration: Number(e.target.value) })}
          />
          <select value={draft.durationUnit} onChange={onUnitChange("durationUnit")}>
            <option value={DurationUnit.SECOND}>seconds</option>
            <option value={DurationUnit.MINUTE}>minutes</option>
          </select>
        </p>
        <p>
          <label htmlFor={audioId}>Finish Sound </label>
          <select id={audioId} value={draft.audio} onChange={onAssetChange("audio")}>
            <option value="">None</option>
            {assetNames(AssetType.AUDIO).map((name) => (
              <option key={name} value={name}>
                {name}
              </option>
            ))}
          </select>
        </p>
        <p>
          <label htmlFor={layoutId}>Layout </label>
          <select id={layoutId} value={draft.layout} onChange={onAssetChange("layout")}>
            <option value="">None</option>
            {assetNames(AssetType.LAYOUT).map((name) => (
              <option key={name} value={name}>
                {name}
              </option>
            ))}
          </select>
        </p>
        <p>
          <label>
            <input
              type="checkbox"
              checked={draft.enabled}
              onChange={(e) => change({ enabled: e.target.checked })}
            />{" "}
            enabled
          </label>
        </p>
        <button type="submit">{editing ? "Save" : "Add breakpoint"}</button>{" "}
        {editing && (
          <button type="button" onClick={() => setDraft(EMPTY_DRAFT)}>
            Cancel
          </button>
        )}
        {error !== "" && <p role="alert">{error}</p>}
      </form>
    </>
  );
}
