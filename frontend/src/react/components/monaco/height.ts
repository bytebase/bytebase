export interface ClampEditorHeightOptions {
  contentHeight: number;
  min: number;
  max: number;
}

export function clampEditorHeight({
  contentHeight,
  min,
  max,
}: ClampEditorHeightOptions): number {
  return Math.min(max, Math.max(min, contentHeight));
}
