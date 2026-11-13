export async function listCameras() {
  const allDevices = await navigator.mediaDevices.enumerateDevices();

  return allDevices.filter((device) => device.kind === "videoinput");
}

export async function getCamera(deviceId: string) {
  const stream = await navigator.mediaDevices.getUserMedia({ video: { deviceId } });
  return stream;
}

export async function requestCameraPermission() {
  const stream = await navigator.mediaDevices.getUserMedia({ video: true });
  stream.getVideoTracks().forEach((t) => t.stop());
}

export async function stopCameraStream(stream: MediaStream) {
  stream.getTracks().forEach((track) => track.stop());
}
