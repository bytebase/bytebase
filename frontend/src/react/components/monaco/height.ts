export interface ReadonlyMonacoHeightOptions {
  minHeight?: number;
  maxHeight?: number;
  lineHeight?: number;
  padding?: number;
}

const DEFAULT_LINE_HEIGHT = 24;
const DEFAULT_PADDING = 16;

export function getReadonlyMonacoHeight(
  content: string,
  options: ReadonlyMonacoHeightOptions = {}
): number {
  const lines = Math.max(1, content.split(/\r?\n/).length);
  const lineHeight = options.lineHeight ?? DEFAULT_LINE_HEIGHT;
  const padding = options.padding ?? DEFAULT_PADDING;
  const rawHeight = lines * lineHeight + padding;
  const minHeight = options.minHeight ?? 0;
  const maxHeight = options.maxHeight ?? Number.MAX_SAFE_INTEGER;

  return Math.min(maxHeight, Math.max(minHeight, rawHeight));
}
