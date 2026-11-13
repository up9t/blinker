import { useEffect, useId, useState, type ChangeEvent } from "react";
import {
  getCamera,
  listCameras,
  requestCameraPermission,
  stopCameraStream,
} from "../lib/camera.ts";

interface CameraSelectorProps {
  onDeviceChange(deviceId: string): void;
}

export function CameraSelector(props: CameraSelectorProps) {
  const deviceSelectId = useId();
  const [cameras, setCameras] = useState<MediaDeviceInfo[]>([]);
  const [usedCamera, setUsedCamera] = useState<string>("");

  useEffect(() => {
    requestCameraPermission()
      .then(listCameras)
      .then((c) => {
        setUsedCamera(c[0]?.deviceId);
        return c;
      })
      .then(setCameras)
      .catch((err) => console.error("Error", err));
  }, []);

  const onDeviceChange = (e: ChangeEvent<HTMLSelectElement>) => {
    const deviceId = e.target.value;

    setUsedCamera(deviceId);
    props.onDeviceChange(deviceId);
  };

  return (
    <>
      <label htmlFor={deviceSelectId}>Camera: </label>
      <select name="" id={deviceSelectId} onChange={onDeviceChange} defaultValue={usedCamera}>
        <option value="">haha</option>
        {cameras.map((c) => (
          <option value={c.deviceId}>{c.label}</option>
        ))}
      </select>
    </>
  );
}
