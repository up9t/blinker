import { useId, type ChangeEvent } from "react";
import type { IBlinkSettings } from "../lib/ports/blink_settings.ts";

// kept one step apart so the close threshold always sits above the open one
const THRESHOLD_STEP = 0.01;
const MIN_THRESHOLD = 0.05;
const MAX_THRESHOLD = 0.95;
const MIN_TIMEOUT_SECONDS = 0.2;
const MAX_TIMEOUT_SECONDS = 5;
const TIMEOUT_STEP_SECONDS = 0.1;

interface BlinkSettingsPanelProps {
  settings: IBlinkSettings;
  onChange(settings: IBlinkSettings): void;
  onReset(): void;
}

export function BlinkSettingsPanel(props: BlinkSettingsPanelProps) {
  const closeId = useId();
  const openId = useId();
  const timeoutId = useId();

  const onNumberChange =
    (key: "eyeCloseThreshold" | "eyeOpenThreshold") => (e: ChangeEvent<HTMLInputElement>) => {
      props.onChange({ ...props.settings, [key]: Number(e.target.value) });
    };

  const onTimeoutChange = (e: ChangeEvent<HTMLInputElement>) => {
    props.onChange({
      ...props.settings,
      blinkTimeoutMs: Math.round(Number(e.target.value) * 1000),
    });
  };

  return (
    <details>
      <summary>Blink settings</summary>

      <p>
        <label htmlFor={closeId}>Eyes closed when the score is at least: </label>
        <input
          id={closeId}
          type="range"
          min={props.settings.eyeOpenThreshold + THRESHOLD_STEP}
          max={MAX_THRESHOLD}
          step={THRESHOLD_STEP}
          value={props.settings.eyeCloseThreshold}
          onChange={onNumberChange("eyeCloseThreshold")}
        />
        <output>{props.settings.eyeCloseThreshold.toFixed(2)}</output>
      </p>

      <p>
        <label htmlFor={openId}>Eyes open when the score is at most: </label>
        <input
          id={openId}
          type="range"
          min={MIN_THRESHOLD}
          max={props.settings.eyeCloseThreshold - THRESHOLD_STEP}
          step={THRESHOLD_STEP}
          value={props.settings.eyeOpenThreshold}
          onChange={onNumberChange("eyeOpenThreshold")}
        />
        <output>{props.settings.eyeOpenThreshold.toFixed(2)}</output>
      </p>

      <p>
        <label htmlFor={timeoutId}>Remind me when I haven't blinked for: </label>
        <input
          id={timeoutId}
          type="range"
          min={MIN_TIMEOUT_SECONDS}
          max={MAX_TIMEOUT_SECONDS}
          step={TIMEOUT_STEP_SECONDS}
          value={props.settings.blinkTimeoutMs / 1000}
          onChange={onTimeoutChange}
        />
        <output>{(props.settings.blinkTimeoutMs / 1000).toFixed(1)}s</output>
      </p>

      <button onClick={props.onReset}>Reset to defaults</button>
    </details>
  );
}
