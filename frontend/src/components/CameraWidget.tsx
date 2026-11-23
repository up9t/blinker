import { useEffect, useRef, useState } from "react";
import { CameraFrame } from "./CameraFrame.tsx";
import { CameraSelector } from "./CameraSelector.tsx";
import { BlinkSettingsPanel } from "./BlinkSettingsPanel.tsx";
import { WailsCameraService } from "../lib/adapters/wails_camera.ts";
import { WailsBlinkSettingsService } from "../lib/adapters/wails_blink_settings.ts";
import { WailsStatisticsService } from "../lib/adapters/wails_statistics.ts";
import { formatDuration } from "../lib/duration.ts";
import type { ITodayStatistics } from "../lib/ports/statistics.ts";
import { DEFAULT_BLINK_SETTINGS, type IBlinkSettings } from "../lib/ports/blink_settings.ts";

const cameraService = new WailsCameraService();
const blinkSettingsService = new WailsBlinkSettingsService();
const statisticsService = new WailsStatisticsService();

// sliders fire on every pixel, only the last value of a drag is worth a write
const SAVE_DEBOUNCE_MS = 300;
// the backend adds one tick of screen time per second while a face is on screen
const SCREEN_TIME_TICK_MS = 1000;

function call(request: Promise<unknown>) {
  request.catch((err) => console.error("Camera service call failed", err));
}

export function CameraWidget() {
  const [usedCamera, setUsedCamera] = useState("");
  const [today, setToday] = useState<ITodayStatistics | null>(null);
  const [blinkSettings, setBlinkSettings] = useState<IBlinkSettings>(DEFAULT_BLINK_SETTINGS);
  const pendingSave = useRef<{ timer: number; settings: IBlinkSettings } | null>(null);
  const present = useRef(false);

  // one snapshot on mount, the backend total already includes the unflushed buffer; the
  // widget then keeps the numbers live locally instead of polling the database
  useEffect(() => {
    statisticsService
      .today()
      .then(setToday)
      .catch((err) => console.error("Failed to load today's statistics", err));
  }, []);

  useEffect(() => {
    const timer = window.setInterval(() => {
      if (!present.current) {
        return;
      }

      setToday((prev) => prev && { ...prev, screenTime: prev.screenTime + SCREEN_TIME_TICK_MS });
      // doubles as the presence heartbeat: the backend stops counting a few seconds after
      // this stops arriving, so a stalled or hidden frontend can't keep screen time running
      call(cameraService.faceDetected());
    }, SCREEN_TIME_TICK_MS);

    return function () {
      window.clearInterval(timer);
    };
  }, []);

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
          setToday((prev) => prev && { ...prev, blinks: prev.blinks + 1 });
          call(cameraService.blink());
        }}
        onFaceDetected={() => {
          present.current = true;
          call(cameraService.faceDetected());
        }}
        onFaceLost={() => {
          present.current = false;
          call(cameraService.faceLost());
        }}
      />

      <p>Blinks today: {today?.blinks ?? "—"}</p>
      <p>Screen time today: {today ? formatDuration(today.screenTime) : "—"}</p>

      <CameraSelector onDeviceChange={(deviceId) => setUsedCamera(deviceId)} />

      <BlinkSettingsPanel
        settings={blinkSettings}
        onChange={changeSettings}
        onReset={resetSettings}
      />
    </>
  );
}
