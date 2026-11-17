import {
  DrawingUtils,
  FaceLandmarker,
  type FaceLandmarkerResult,
  FilesetResolver,
} from "@mediapipe/tasks-vision";

let landmarker: FaceLandmarker | undefined;
// the canvas the task renders into, its WebGL context is needed for drawing
let landmarkerCanvas: HTMLCanvasElement | undefined;

export async function setupLandmarker() {
  if (typeof landmarker !== "undefined") return landmarker;

  const modelAssetPath = import.meta.env.BASE_URL + "task-vision/face_landmarker.task";
  const modelPath = import.meta.env.BASE_URL + "task-vision/wasm";

  const resolver = await FilesetResolver.forVisionTasks(modelPath);
  // MediaPipe creates a WebGL context on this canvas, so nobody else may touch it.
  const canvas = document.createElement("canvas");

  const create = (delegate: "GPU" | "CPU") =>
    FaceLandmarker.createFromOptions(resolver, {
      baseOptions: {
        delegate: delegate,
        modelAssetPath: modelAssetPath,
      },
      runningMode: "VIDEO",
      outputFaceBlendshapes: true,
      numFaces: 1,
      canvas: canvas,
    });

  try {
    landmarker = await create("GPU");
  } catch (err) {
    // no usable WebGL on this machine, CPU is slower but keeps the feature working
    console.warn("GPU delegate failed, falling back to CPU", err);
    landmarker = await create("CPU");
  }

  landmarkerCanvas = canvas;

  return landmarker;
}

export interface IDrawLandmark {
  (results: FaceLandmarkerResult): void;
}

export function setupDrawLandmark(context: CanvasRenderingContext2D): IDrawLandmark {
  let drawingUtils: DrawingUtils | undefined;

  return function (results: FaceLandmarkerResult) {
    // built once, on the first frame, because the task canvas only exists after setupLandmarker()
    if (!drawingUtils) {
      drawingUtils = new DrawingUtils(context, landmarkerCanvas?.getContext("webgl2") ?? undefined);
    }

    for (const landmarks of results.faceLandmarks) {
      drawingUtils.drawConnectors(landmarks, FaceLandmarker.FACE_LANDMARKS_TESSELATION, {
        color: "#C0C0C070",
        lineWidth: 1,
      });
      drawingUtils.drawConnectors(landmarks, FaceLandmarker.FACE_LANDMARKS_RIGHT_EYE, {
        color: "#FF3030",
      });
      drawingUtils.drawConnectors(landmarks, FaceLandmarker.FACE_LANDMARKS_RIGHT_EYEBROW, {
        color: "#FF3030",
      });
      drawingUtils.drawConnectors(landmarks, FaceLandmarker.FACE_LANDMARKS_LEFT_EYE, {
        color: "#30FF30",
      });
      drawingUtils.drawConnectors(landmarks, FaceLandmarker.FACE_LANDMARKS_LEFT_EYEBROW, {
        color: "#30FF30",
      });
      drawingUtils.drawConnectors(landmarks, FaceLandmarker.FACE_LANDMARKS_FACE_OVAL, {
        color: "#E0E0E0",
      });
      drawingUtils.drawConnectors(landmarks, FaceLandmarker.FACE_LANDMARKS_LIPS, {
        color: "#E0E0E0",
      });
      drawingUtils.drawConnectors(landmarks, FaceLandmarker.FACE_LANDMARKS_RIGHT_IRIS, {
        color: "#FF3030",
      });
      drawingUtils.drawConnectors(landmarks, FaceLandmarker.FACE_LANDMARKS_LEFT_IRIS, {
        color: "#30FF30",
      });
    }
  };
}

interface CameraResult {
  faceFound: boolean;
  // null when the blink score sits between the open and close thresholds
  eyesClosed: boolean | null;
  faceLandmarkResult: FaceLandmarkerResult | null;
}

interface EyeThreshold {
  open: number;
  close: number;
}

export function getCameraResult(
  videoElement: HTMLVideoElement,
  now: number,
  { open, close }: EyeThreshold,
): CameraResult {
  if (typeof landmarker === "undefined") {
    return {
      faceFound: false,
      eyesClosed: null,
      faceLandmarkResult: null,
    };
  }

  const results = landmarker.detectForVideo(videoElement, now);

  if (!results.faceLandmarks || results.faceLandmarks.length === 0) {
    return {
      faceFound: false,
      eyesClosed: null,
      faceLandmarkResult: null,
    };
  }

  const blinkEyes = results.faceBlendshapes
    .at(0)
    ?.categories.filter(
      (cat) => cat.categoryName === "eyeBlinkLeft" || cat.categoryName === "eyeBlinkRight",
    );

  let eyesClosed: boolean | null = null;

  if (blinkEyes?.length === 2) {
    if (blinkEyes.every((blinkEye) => blinkEye.score >= close)) {
      eyesClosed = true;
    } else if (blinkEyes.every((blinkEye) => blinkEye.score <= open)) {
      eyesClosed = false;
    }
  }

  return {
    faceFound: true,
    eyesClosed,
    faceLandmarkResult: results,
  };
}
