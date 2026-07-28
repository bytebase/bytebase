export interface ClampEditorHeightOptions {
  lineCount: number;
  lineHeight: number;
  min: number;
  max: number;
}

export function clampEditorHeight({
  lineCount,
  lineHeight,
  min,
  max,
}: ClampEditorHeightOptions): number {
  return Math.min(max, Math.max(min, lineCount * lineHeight));
}
