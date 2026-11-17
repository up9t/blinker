import { useEffect, useId, useState, type ChangeEvent } from "react";
import { listCameras, requestCameraPermission } from "../lib/camera.ts";

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
        setCameras(c);

        if (c[0]) {
          setUsedCamera(c[0].deviceId);
          props.onDeviceChange(c[0].deviceId);
        }
      })
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
      <select id={deviceSelectId} onChange={onDeviceChange} value={usedCamera}>
        <option value="">No camera</option>
        {cameras.map((c) => (
          <option key={c.deviceId} value={c.deviceId}>
            {c.label}
          </option>
        ))}
      </select>
    </>
  );
}
