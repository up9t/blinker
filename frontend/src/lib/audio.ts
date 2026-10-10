const audioCtx = new window.AudioContext();

// cache
const srcToBuffer = new Map<string, AudioBuffer>();

// this won't play on linux.
// function playAudio() {
//   new Audio(audioSrc).play();
// }

async function loadAudio(audioSrc: string) {
  if (srcToBuffer.has(audioSrc)) {
    return srcToBuffer.get(audioSrc)!;
  }

  const response = await fetch(audioSrc);
  const arrayBuffer = await response.arrayBuffer();
  const buffer = await audioCtx.decodeAudioData(arrayBuffer);

  srcToBuffer.set(audioSrc, buffer);

  return buffer;
}

export async function playAudio(audioSrc: string) {
  if (audioCtx.state === "suspended") {
    await audioCtx.resume();
  }

  const audioBuffer = await loadAudio(audioSrc);

  const source = audioCtx.createBufferSource();
  source.buffer = audioBuffer;
  source.connect(audioCtx.destination);
  source.start();
}
