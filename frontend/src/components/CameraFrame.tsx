import { useEffect, useRef, useState } from "react";
import { Pause, Play } from "lucide-react";
import { CameraDetector } from "../lib/camera_detector.ts";
import { setupDrawLandmark, type IDrawLandmark } from "../lib/mediapipe.ts";

interface CameraFrameProps {
  deviceId: string;
  eyeCloseThreshold: number;
  eyeOpenThreshold: number;
  onBlink(): void;
  onFaceDetected(): void;
  onFaceLost(): void;
}

export function CameraFrame(props: CameraFrameProps) {
  const videoElementRef = useRef<HTMLVideoElement | null>(null);
  const canvasElementRef = useRef<HTMLCanvasElement | null>(null);
  const detectorRef = useRef<CameraDetector | null>(null);
  // keeps the detector callbacks stable, so the camera isn't restarted on every render
  const propsRef = useRef(props);
  const [pause, setPause] = useState(true);

  useEffect(() => {
    propsRef.current = props;
  });

  useEffect(() => {
    const video = videoElementRef.current;
    const canvas = canvasElementRef.current;

    if (video === null || canvas === null) {
      return;
    }

    const context2d = canvas.getContext("2d");

    if (context2d === null) {
      throw new Error("Failed to retrieve canvas context for landmark drawing");
    }

    const drawLandmark: IDrawLandmark = setupDrawLandmark(context2d);

    detectorRef.current = new CameraDetector(video, canvas, {
      eyeCloseThreshold: propsRef.current.eyeCloseThreshold,
      eyeOpenThreshold: propsRef.current.eyeOpenThreshold,
      onBlink: () => propsRef.current.onBlink(),
      onFaceDetected: () => propsRef.current.onFaceDetected(),
      onFaceLost: () => propsRef.current.onFaceLost(),
      onFace: (result) => drawLandmark(result),
    });

    return function () {
      detectorRef.current?.stop();
      detectorRef.current = null;
    };
  }, []);

  useEffect(() => {
    const detector = detectorRef.current;

    if (detector === null || pause || props.deviceId === "") {
      return;
    }

    detector.start(props.deviceId).catch((err) => console.error("Failed to start camera", err));

    return function () {
      detector.stop();
    };
  }, [pause, props.deviceId]);

  return (
    <div className="aspect-video relative max-h-2/3 border-4 border-red-400">
      <video
        ref={videoElementRef}
        autoPlay
        muted
        className="absolute size-full object-cover inset-0"
      ></video>
      <canvas ref={canvasElementRef} className="absolute size-full object-cover inset-0"></canvas>
      <button
        aria-label={pause ? "Start camera" : "Pause camera"}
        className="absolute inset-0 m-auto size-14 rounded-full bg-black/50 text-white grid place-items-center"
        onClick={() => setPause(!pause)}
      >
        {pause ? <Play /> : <Pause />}
      </button>
    </div>
  );
}
