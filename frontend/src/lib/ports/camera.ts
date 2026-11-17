export interface ICameraService {
  faceDetected(): Promise<void>;
  faceLost(): Promise<void>;
  blink(): Promise<void>;
}
