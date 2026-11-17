import type { FaceLandmarkerResult } from "@mediapipe/tasks-vision";
import { getCamera, stopCameraStream } from "./camera.ts";
import { getCameraResult, setupLandmarker, type EyeThreshold } from "./mediapipe.ts";
import { clearCanvas, resizeCanvas } from "./canvas.ts";

// face gone for this long counts as "user left the screen"
const FACE_LOST_DELAY_MS = 3000;
// while nobody is on screen, only check once per second (efficiency mode)
const EFFICIENCY_INTERVAL_MS = 1000;

export interface CameraDetectorOptions {
  // read on every frame, so a changed setting applies to the running camera
  getThresholds(): EyeThreshold;
  onBlink(): void;
  onFaceDetected(): void;
  onFaceLost(): void;
  onFace?(result: FaceLandmarkerResult): void;
}

export class CameraDetector {
  private stream: MediaStream | null = null;
  private frameId: number | null = null;
  private lastVideoTime = -1;
  private lastCheckAt = 0;
  // increments on every start/stop so an in flight start can be dropped
  private token = 0;
  private eyesClosed = false;
  private facePresent = false;
  private lastFaceSeenAt = 0;

  constructor(
    private video: HTMLVideoElement,
    private canvas: HTMLCanvasElement,
    private options: CameraDetectorOptions,
  ) {}

  async start(deviceId: string): Promise<void> {
    this.stop();

    // after stop(), so the token below is the only one in flight
    const token = ++this.token;

    const stream = await getCamera(deviceId);

    if (token !== this.token) {
      stopCameraStream(stream);
      return;
    }

    // a hidden window stops the frame loop, so nobody is on screen anymore
    document.addEventListener("visibilitychange", this.onVisibilityChange);

    this.stream = stream;
    this.video.srcObject = stream;
    this.lastVideoTime = -1;
    // negative, so the first frame is always checked while nobody is on screen
    this.lastCheckAt = -EFFICIENCY_INTERVAL_MS;
    this.frameId = requestAnimationFrame(this.frame);

    // elements can refuse to autoplay, ask explicitly and don't fail the whole start
    this.video.play().catch((err) => console.warn("Failed to play the camera stream", err));

    // the stream runs regardless, frames count as "no face" until the model is ready
    await setupLandmarker().catch((err) =>
      console.error("Face model failed to load, blink detection is disabled", err),
    );
  }

  stop(): void {
    this.token++;
    this.stopLoop();
    document.removeEventListener("visibilitychange", this.onVisibilityChange);

    if (this.stream) {
      stopCameraStream(this.stream);
      this.stream = null;
    }

    this.video.srcObject = null;

    if (this.facePresent) {
      this.facePresent = false;
      this.options.onFaceLost();
    }
  }

  private onVisibilityChange = () => {
    if (!document.hidden || !this.facePresent) return;

    this.facePresent = false;
    this.options.onFaceLost();
  };

  private stopLoop() {
    if (this.frameId !== null) {
      cancelAnimationFrame(this.frameId);
      this.frameId = null;
    }
  }

  private frame = (now: number) => {
    this.frameId = requestAnimationFrame(this.frame);

    const video = this.video;

    if (video.readyState < 2 || video.currentTime === this.lastVideoTime) return;

    if (!this.facePresent && now - this.lastCheckAt < EFFICIENCY_INTERVAL_MS) return;

    this.lastVideoTime = video.currentTime;
    this.lastCheckAt = now;

    const result = getCameraResult(video, now, this.options.getThresholds());

    if (result.faceFound) {
      if (!this.facePresent) {
        this.facePresent = true;
        this.options.onFaceDetected();
      }

      this.lastFaceSeenAt = now;

      if (result.eyesClosed && !this.eyesClosed) {
        this.options.onBlink();
      }

      if (result.eyesClosed !== null) {
        this.eyesClosed = result.eyesClosed;
      }
    } else if (this.facePresent && now - this.lastFaceSeenAt >= FACE_LOST_DELAY_MS) {
      this.facePresent = false;
      this.eyesClosed = false;
      this.options.onFaceLost();
    }

    resizeCanvas(this.canvas, video.videoWidth, video.videoHeight);
    clearCanvas(this.canvas.getContext("2d")!);

    if (result.faceFound && result.faceLandmarkResult !== null) {
      this.options.onFace?.(result.faceLandmarkResult);
    }
  };
}
