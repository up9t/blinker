import type { ICameraService } from "../ports/camera.ts";
import { CameraService } from "../../../bindings/github.com/up9t/blinker/index.js";

export class WailsCameraService implements ICameraService {
  async faceDetected(): Promise<void> {
    await CameraService.FaceDetected();
  }

  async faceLost(): Promise<void> {
    await CameraService.FaceLost();
  }

  async blink(): Promise<void> {
    await CameraService.Blink();
  }
}
