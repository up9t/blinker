import { useState } from "react";
import { CameraFrame } from "./CameraFrame.tsx";
import { CameraSelector } from "./CameraSelector.tsx";

export function CameraWidget() {
  const [usedCamera, setUsedCamera] = useState("");

  return (
    <>
      <CameraFrame
        deviceId={usedCamera}
        eyeCloseThreshold={5.25 / 10}
        eyeOpenThreshold={5 / 10}
        onEyeClose={() => console.log("close")}
        onEyeOpen={() => {}}
      />

      <CameraSelector onDeviceChange={(deviceId) => setUsedCamera(deviceId)} />
    </>
  );
}
