import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const detectForVideo = vi.fn();

vi.mock("@mediapipe/tasks-vision", () => ({
  FaceLandmarker: {
    createFromOptions: vi.fn(async () => ({ detectForVideo })),
    FACE_LANDMARKS_TESSELATION: {},
    FACE_LANDMARKS_RIGHT_EYE: {},
    FACE_LANDMARKS_RIGHT_EYEBROW: {},
    FACE_LANDMARKS_LEFT_EYE: {},
    FACE_LANDMARKS_LEFT_EYEBROW: {},
    FACE_LANDMARKS_FACE_OVAL: {},
    FACE_LANDMARKS_LIPS: {},
    FACE_LANDMARKS_RIGHT_IRIS: {},
    FACE_LANDMARKS_LEFT_IRIS: {},
  },
  FilesetResolver: {
    forVisionTasks: vi.fn(async () => ({})),
  },
  DrawingUtils: vi.fn(),
}));

import { getCameraResult, setupLandmarker } from "./mediapipe.ts";

const thresholds = { open: 0.5, close: 0.525 };
const video = {} as HTMLVideoElement;

beforeEach(() => {
  vi.stubGlobal("document", {
    createElement: () => ({}),
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
});

function detect(leftEyeScore: number, rightEyeScore: number, faceCount = 1) {
  detectForVideo.mockReturnValue({
    faceLandmarks: Array.from({ length: faceCount }, () => [{ x: 0, y: 0, z: 0 }]),
    faceBlendshapes:
      faceCount === 0
        ? []
        : [
            {
              categories: [
                { categoryName: "eyeBlinkLeft", score: leftEyeScore },
                { categoryName: "eyeBlinkRight", score: rightEyeScore },
              ],
            },
          ],
  });
}

describe("getCameraResult", () => {
  it("reports no face when no landmark was found", async () => {
    await setupLandmarker();
    detect(0, 0, 0);

    const result = getCameraResult(video, 1, thresholds);

    expect(result).toEqual({ faceFound: false, eyesClosed: null, faceLandmarkResult: null });
  });

  it("reports closed eyes above the close threshold", async () => {
    await setupLandmarker();
    detect(0.9, 0.8);

    expect(getCameraResult(video, 2, thresholds).eyesClosed).toBe(true);
  });

  it("reports open eyes below the open threshold", async () => {
    await setupLandmarker();
    detect(0.1, 0.2);

    expect(getCameraResult(video, 3, thresholds).eyesClosed).toBe(false);
  });

  it("reports an unknown blink state between the thresholds", async () => {
    await setupLandmarker();
    detect(0.51, 0.51);

    const result = getCameraResult(video, 4, thresholds);

    expect(result.faceFound).toBe(true);
    expect(result.eyesClosed).toBeNull();
  });
});
