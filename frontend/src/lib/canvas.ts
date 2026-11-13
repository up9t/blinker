export function resizeCanvas(canvas: HTMLCanvasElement, width: number, height: number) {
  if (canvas.width !== width || canvas.height !== height) {
    canvas.width = width;
    canvas.height = height;
  }
}
export function clearCanvas(ctx: CanvasRenderingContext2D) {
  ctx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height);
}