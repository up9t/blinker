import { useState } from "react";
import { CameraFrame } from "./CameraFrame.tsx";
import { CameraSelector } from "./CameraSelector.tsx";
import { WailsCameraService } from "../lib/adapters/wails_camera.ts";

const cameraService = new WailsCameraService();

function call(request: Promise<void>) {
  request.catch((err) => console.error("Camera service call failed", err));
}

export function CameraWidget() {
  const [usedCamera, setUsedCamera] = useState("");
  const [blinkCount, setBlinkCount] = useState(0);

  return (
    <>
      <CameraFrame
        deviceId={usedCamera}
        eyeCloseThreshold={4 / 10}
        eyeOpenThreshold={5 / 10}
        onBlink={() => {
          setBlinkCount((count) => count + 1);
          call(cameraService.blink());
        }}
        onFaceDetected={() => call(cameraService.faceDetected())}
        onFaceLost={() => call(cameraService.faceLost())}
      />

      <p>Blinks: {blinkCount}</p>

      <CameraSelector onDeviceChange={(deviceId) => setUsedCamera(deviceId)} />
    </>
  );
}
