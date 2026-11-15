import { useCallback, useEffect, useRef, useState } from "react";
import { getCamera, stopCameraStream } from "../lib/camera.ts";
import {
  getCameraResult,
  type IDrawLandmark,
  setupDrawLandmark,
  setupLandmarker,
} from "../lib/mediapipe.ts";
import type { FaceLandmarker } from "@mediapipe/tasks-vision";
import { clearCanvas, resizeCanvas } from "../lib/canvas.ts";

interface CameraFrameProps {
  deviceId: string;
  eyeCloseThreshold: number;
  eyeOpenThreshold: number;
  onEyeClose(): void;
  onEyeOpen(): void;
}

export function CameraFrame(props: CameraFrameProps) {
  const videoElementRef = useRef<HTMLVideoElement | null>(null);
  const canvasElementRef = useRef<HTMLCanvasElement | null>(null);
  const canvasContextRef = useRef<CanvasRenderingContext2D | null>(null);
  const frameIdRef = useRef<number | null>(null);
  const gpuCanvasRef = useRef(document.createElement("canvas"));
  const drawLandmarkFuncRef = useRef<IDrawLandmark>(() => { });
  const lastVideoTimeRef = useRef(-1);
  const [pause, setPause] = useState(true);

  useEffect(() => {
    if (canvasElementRef.current === null) {
      return;
    }

    const context2d = canvasElementRef.current.getContext("2d");

    if (context2d === null) {
      throw new Error("Failed to retrieve canvas context 2D rendering");
    }

    const contextgl = gpuCanvasRef.current.getContext("webgl2");

    if (contextgl === null) {
      throw new Error("Failed to retrieve webgl2 context");
    }

    let landmarker: FaceLandmarker | undefined = undefined;

    canvasContextRef.current = context2d;
    drawLandmarkFuncRef.current = setupDrawLandmark(context2d, contextgl);

    (async function () {
      landmarker = await setupLandmarker(gpuCanvasRef.current);
    })();

    return function () {
      landmarker?.close();
    };
  }, []);

  useEffect(() => {
    if (videoElementRef.current === null) {
      return;
    }

    if (canvasContextRef.current === null) {
      throw new Error("Failed to retrieve canvas context 2D rendering");
    }

    if (props.deviceId === "") {
      const stream = videoElementRef.current.srcObject;

      if (stream !== null) {
        stopCameraStream(stream as MediaStream);
      }

      videoElementRef.current.srcObject = null;

      return;
    }

    if (!pause) {
      getCamera(props.deviceId).then((stream) => {
        videoElementRef.current!.srcObject = stream;
      });
    }

    return function () {
      const stream = videoElementRef.current?.srcObject;

      if (stream) {
        stopCameraStream(stream as MediaStream);
      }
    };
  }, [props.deviceId]);

  const frame = useCallback(
    (now: number) => {
      if (pause) {
        cancelAnimationFrame(frameIdRef.current!);
        frameIdRef.current = null;
        return;
      }
      console.log(11);

      const video = videoElementRef.current;
      const videoReady = (video?.readyState ?? 0) >= 2;

      if (
        video === null ||
        canvasElementRef.current === null ||
        !videoReady ||
        video?.currentTime === lastVideoTimeRef.current
      ) {
        frameIdRef.current = requestAnimationFrame(frame);
        return;
      }

      const result = getCameraResult(video, now, {
        open: props.eyeOpenThreshold,
        close: props.eyeCloseThreshold,
      });

      resizeCanvas(canvasElementRef.current, video.videoWidth, video.videoHeight);
      clearCanvas(canvasElementRef.current.getContext("2d")!);

      console.log(result);

      if (!result.faceFound) {
        frameIdRef.current = requestAnimationFrame(frame);
        return; // don't continue
      }

      if (result.eyeClose) {
        props.onEyeClose();
      } else {
        props.onEyeOpen();
      }

      if (result.faceLandmarkResult !== null) {
        drawLandmarkFuncRef.current(result.faceLandmarkResult);
      }

      frameIdRef.current = requestAnimationFrame(frame);
    },
    [pause, props.eyeOpenThreshold, props.eyeCloseThreshold, props.onEyeClose, props.onEyeOpen],
  );

  useEffect(() => {
    if (pause) {
      if (frameIdRef.current) cancelAnimationFrame(frameIdRef.current);
      const stream = videoElementRef.current?.srcObject;
      if (stream) stopCameraStream(stream as MediaStream);

      return;
    }

    getCamera(props.deviceId).then((stream) => {
      videoElementRef.current!.srcObject = stream;
    });

    frameIdRef.current = requestAnimationFrame(frame);

    return function () {
      if (frameIdRef.current) cancelAnimationFrame(frameIdRef.current);
    };
  }, [pause, frame]);

  return (
    <>
      <div className="aspect-video relative max-h-2/3">
        <video
          ref={videoElementRef}
          autoPlay
          className="absolute size-full object-cover inset-0"
        ></video>
        <canvas ref={canvasElementRef} className="absolute size-full object-cover inset-0"></canvas>
      </div>
      <button onClick={() => setPause(!pause)}>{pause ? "Resume" : "Pause"}</button>
    </>
  );
}
