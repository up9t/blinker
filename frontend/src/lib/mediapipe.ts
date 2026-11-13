import {
  DrawingUtils,
  FaceLandmarker,
  type FaceLandmarkerResult,
  FilesetResolver,
} from "@mediapipe/tasks-vision";

let landmarker: FaceLandmarker | undefined;

export async function setupLandmarker(canvas: HTMLCanvasElement) {
  if (typeof landmarker !== "undefined") return landmarker;

  const modelAssetPath = import.meta.env.BASE_URL + "task-vision/face_landmarker.task";
  const modelPath = import.meta.env.BASE_URL + "task-vision/wasm";

  const resolver = await FilesetResolver.forVisionTasks(modelPath);

  landmarker = await FaceLandmarker.createFromOptions(resolver, {
    baseOptions: {
      delegate: "GPU",
      modelAssetPath: modelAssetPath,
    },
    runningMode: "VIDEO",
    outputFaceBlendshapes: true,
    numFaces: 1,
    canvas: canvas,
  });

  return landmarker;
}

export interface IDrawLandmark {
  (results: FaceLandmarkerResult): void;
}

export function setupDrawLandmark(
  context: CanvasRenderingContext2D,
  gpuContext: WebGL2RenderingContext,
): IDrawLandmark {
  const drawingUtils = new DrawingUtils(context, gpuContext);

  return function (results: FaceLandmarkerResult) {
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
  eyeClose: boolean;
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
  if (typeof landmarker === "undefined")
    return {
      faceFound: false,
      eyeClose: false,
      faceLandmarkResult: null,
    };

  const results = landmarker.detectForVideo(videoElement, now);

  if (results.faceBlendshapes && results.faceBlendshapes.length > 0) {
    const blinkEyes = results.faceBlendshapes
      .at(0)
      ?.categories.filter(
        (cat) => cat.categoryName === "eyeBlinkLeft" || cat.categoryName === "eyeBlinkRight",
      );

    // is eye's closing
    if (blinkEyes?.every((blinkEye) => blinkEye.score >= close)) {
      return {
        faceFound: true,
        eyeClose: true,
        faceLandmarkResult: results,
      };
    }

    // is eye's opening
    if (blinkEyes?.every((blinkEye) => blinkEye.score <= open)) {
      return {
        faceFound: true,
        eyeClose: false,
        faceLandmarkResult: results,
      };
    }
  }

  return {
    faceFound: false,
    eyeClose: false,
    faceLandmarkResult: results,
  };
}
