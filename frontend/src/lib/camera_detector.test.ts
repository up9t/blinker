import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("./mediapipe.ts", () => ({
  setupLandmarker: vi.fn(async () => ({})),
  getCameraResult: vi.fn(),
}));

vi.mock("./camera.ts", () => ({
  getCamera: vi.fn(),
  stopCameraStream: vi.fn(),
}));

import { CameraDetector } from "./camera_detector.ts";
import { getCameraResult, setupLandmarker } from "./mediapipe.ts";
import { getCamera, stopCameraStream } from "./camera.ts";

const mockedGetCamera = vi.mocked(getCamera);
const mockedGetCameraResult = vi.mocked(getCameraResult);
const mockedSetupLandmarker = vi.mocked(setupLandmarker);
const mockedStopCameraStream = vi.mocked(stopCameraStream);

const stream = { id: "stream" } as unknown as MediaStream;

let detector: CameraDetector;
let video: {
  readyState: number;
  currentTime: number;
  srcObject: MediaStream | null;
  play: () => Promise<void>;
};
let canvas: {
  width: number;
  height: number;
  getContext: () => { canvas: unknown; clearRect(): void };
};
let queue: Map<number, FrameRequestCallback>;
let nextFrameId: number;
let now: number;
let events: { blinks: number; detected: number; lost: number };
let drawnFrames: number;

function tick(ms = 16) {
  now += ms;
  video.currentTime += 1;

  const callbacks = [...queue.values()];
  queue.clear();

  for (const callback of callbacks) callback(now);
}

function setResult(result: { faceFound: boolean; eyesClosed: boolean | null }) {
  mockedGetCameraResult.mockReturnValue({
    ...result,
    faceLandmarkResult: result.faceFound ? ({ faceLandmarks: [] } as never) : null,
  });
}

beforeEach(() => {
  queue = new Map();
  nextFrameId = 1;
  now = 0;
  events = { blinks: 0, detected: 0, lost: 0 };
  drawnFrames = 0;

  video = {
    readyState: 2,
    currentTime: 0,
    srcObject: null,
    play: vi.fn(async () => {}),
  };

  canvas = {
    width: 0,
    height: 0,
    getContext() {
      return { canvas: canvas, clearRect: () => {} };
    },
  };

  vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
    const id = nextFrameId++;
    queue.set(id, callback);
    return id;
  });

  vi.stubGlobal("cancelAnimationFrame", (id: number) => {
    queue.delete(id);
  });

  vi.stubGlobal("document", {
    hidden: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  });

  mockedGetCamera.mockReset();
  mockedGetCamera.mockResolvedValue(stream);
  mockedGetCameraResult.mockReset();
  mockedGetCameraResult.mockReturnValue({
    faceFound: false,
    eyesClosed: null,
    faceLandmarkResult: null,
  });
  mockedSetupLandmarker.mockReset();
  mockedSetupLandmarker.mockResolvedValue({} as never);
  mockedStopCameraStream.mockReset();

  vi.spyOn(console, "error").mockImplementation(() => {});
  vi.spyOn(console, "warn").mockImplementation(() => {});

  detector = new CameraDetector(
    video as unknown as HTMLVideoElement,
    canvas as unknown as HTMLCanvasElement,
    {
      eyeCloseThreshold: 0.525,
      eyeOpenThreshold: 0.5,
      onBlink: () => events.blinks++,
      onFaceDetected: () => events.detected++,
      onFaceLost: () => events.lost++,
      onFace: () => drawnFrames++,
    },
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("CameraDetector", () => {
  it("keeps the stream and starts the frame loop on start", async () => {
    await detector.start("camera-id");

    expect(video.srcObject).toBe(stream);
    expect(mockedStopCameraStream).not.toHaveBeenCalled();
    expect(queue.size).toBe(1);
  });

  it("keeps the camera running when the model fails to load", async () => {
    mockedSetupLandmarker.mockRejectedValueOnce(new Error("no webgl"));

    await detector.start("camera-id");

    expect(video.srcObject).toBe(stream);

    setResult({ faceFound: true, eyesClosed: false });
    tick();

    expect(events.detected).toBe(1);
    expect(drawnFrames).toBe(1);
  });

  it("drops a stream that arrives after the detector was stopped", async () => {
    let resolveStream: (value: MediaStream) => void = () => {};
    mockedGetCamera.mockReturnValueOnce(
      new Promise<MediaStream>((resolve) => {
        resolveStream = resolve;
      }),
    );

    const starting = detector.start("camera-id");
    detector.stop();
    resolveStream(stream);
    await starting;

    expect(video.srcObject).toBeNull();
    expect(mockedStopCameraStream).toHaveBeenCalledWith(stream);
  });

  it("reports a blink only on the open to closed edge", async () => {
    await detector.start("camera-id");

    setResult({ faceFound: true, eyesClosed: false });
    tick();
    setResult({ faceFound: true, eyesClosed: true });
    tick();
    tick(); // still closed

    expect(events.blinks).toBe(1);

    setResult({ faceFound: true, eyesClosed: false });
    tick();
    setResult({ faceFound: true, eyesClosed: true });
    tick();

    expect(events.blinks).toBe(2);
  });

  it("ignores in between blink scores", async () => {
    await detector.start("camera-id");

    setResult({ faceFound: true, eyesClosed: false });
    tick();
    setResult({ faceFound: true, eyesClosed: null });
    tick();
    setResult({ faceFound: true, eyesClosed: true });
    tick();

    expect(events.blinks).toBe(1);
  });

  it("reports face found once and face lost after 3s without a face", async () => {
    await detector.start("camera-id");

    setResult({ faceFound: true, eyesClosed: false });
    tick();
    tick();
    tick();

    expect(events.detected).toBe(1);

    setResult({ faceFound: false, eyesClosed: null });
    tick(1000);
    tick(1000);

    expect(events.lost).toBe(0);

    tick(1000);

    expect(events.lost).toBe(1);
  });

  it("only checks once per second while nobody is on screen", async () => {
    await detector.start("camera-id");

    tick(16);
    expect(mockedGetCameraResult).toHaveBeenCalledTimes(1);

    tick(16);
    tick(16);
    expect(mockedGetCameraResult).toHaveBeenCalledTimes(1);

    tick(1000);
    expect(mockedGetCameraResult).toHaveBeenCalledTimes(2);
  });

  it("stops the stream and reports face lost when stopped", async () => {
    await detector.start("camera-id");

    setResult({ faceFound: true, eyesClosed: false });
    tick();

    detector.stop();

    expect(mockedStopCameraStream).toHaveBeenCalledWith(stream);
    expect(video.srcObject).toBeNull();
    expect(events.lost).toBe(1);

    detector.stop();

    expect(events.lost).toBe(1);
    expect(queue.size).toBe(0);
  });
});
