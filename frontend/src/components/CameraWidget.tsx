import { useEffect, useRef, useState } from "react";
import { CameraFrame } from "./CameraFrame.tsx";
import { CameraSelector } from "./CameraSelector.tsx";
import { BlinkSettingsPanel } from "./BlinkSettingsPanel.tsx";
import { WailsCameraService } from "../lib/adapters/wails_camera.ts";
import { WailsBlinkSettingsService } from "../lib/adapters/wails_blink_settings.ts";
import { DEFAULT_BLINK_SETTINGS, type IBlinkSettings } from "../lib/ports/blink_settings.ts";

const cameraService = new WailsCameraService();
const blinkSettingsService = new WailsBlinkSettingsService();

// sliders fire on every pixel, only the last value of a drag is worth a write
const SAVE_DEBOUNCE_MS = 300;

function call(request: Promise<unknown>) {
  request.catch((err) => console.error("Camera service call failed", err));
}

export function CameraWidget() {
  const [usedCamera, setUsedCamera] = useState("");
  const [blinkCount, setBlinkCount] = useState(0);
  const [blinkSettings, setBlinkSettings] = useState<IBlinkSettings>(DEFAULT_BLINK_SETTINGS);
  const pendingSave = useRef<{ timer: number; settings: IBlinkSettings } | null>(null);

  useEffect(() => {
    blinkSettingsService
      .load()
      .then(setBlinkSettings)
      .catch((err) => console.error("Failed to load blink settings", err));
  }, []);

  // a drag left right before leaving the page still gets saved
  useEffect(() => {
    return function () {
      const pending = pendingSave.current;

      if (pending !== null) {
        window.clearTimeout(pending.timer);
        call(blinkSettingsService.save(pending.settings));
      }
    };
  }, []);

  const changeSettings = (settings: IBlinkSettings) => {
    setBlinkSettings(settings);

    if (pendingSave.current !== null) {
      window.clearTimeout(pendingSave.current.timer);
    }

    pendingSave.current = {
      settings,
      timer: window.setTimeout(() => {
        pendingSave.current = null;
        call(blinkSettingsService.save(settings));
      }, SAVE_DEBOUNCE_MS),
    };
  };

  const resetSettings = () => {
    if (pendingSave.current !== null) {
      window.clearTimeout(pendingSave.current.timer);
      pendingSave.current = null;
    }

    blinkSettingsService
      .reset()
      .then(setBlinkSettings)
      .catch((err) => console.error("Failed to reset blink settings", err));
  };

  return (
    <>
      <CameraFrame
        deviceId={usedCamera}
        eyeCloseThreshold={blinkSettings.eyeCloseThreshold}
        eyeOpenThreshold={blinkSettings.eyeOpenThreshold}
        onBlink={() => {
          setBlinkCount((count) => count + 1);
          call(cameraService.blink());
        }}
        onFaceDetected={() => call(cameraService.faceDetected())}
        onFaceLost={() => call(cameraService.faceLost())}
      />

      <p>Blinks: {blinkCount}</p>

      <CameraSelector onDeviceChange={(deviceId) => setUsedCamera(deviceId)} />

      <BlinkSettingsPanel
        settings={blinkSettings}
        onChange={changeSettings}
        onReset={resetSettings}
      />
    </>
  );
}
