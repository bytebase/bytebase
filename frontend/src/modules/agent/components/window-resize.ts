export type ResizeDirection = "n" | "s" | "e" | "w" | "ne" | "nw" | "se" | "sw";

export interface ResizeBounds {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface ResizeConstraints {
  minWidth: number;
  minHeight: number;
  viewportWidth: number;
  viewportHeight: number;
  margin: number;
}

interface ResizeWindowBoundsOptions {
  direction: ResizeDirection;
  startBounds: ResizeBounds;
  deltaX: number;
  deltaY: number;
  constraints: ResizeConstraints;
}

const clamp = (value: number, min: number, max: number) => {
  return Math.min(max, Math.max(min, Math.round(value)));
};

export function resizeWindowBounds({
  direction,
  startBounds,
  deltaX,
  deltaY,
  constraints,
}: ResizeWindowBoundsOptions): ResizeBounds {
  const right = startBounds.x + startBounds.width;
  const bottom = startBounds.y + startBounds.height;

  const maxWidth = Math.max(
    1,
    constraints.viewportWidth - constraints.margin * 2
  );
  const maxHeight = Math.max(
    1,
    constraints.viewportHeight - constraints.margin * 2
  );
  const minWidth = Math.min(constraints.minWidth, maxWidth);
  const minHeight = Math.min(constraints.minHeight, maxHeight);

  let left = startBounds.x;
  let top = startBounds.y;
  let nextRight = right;
  let nextBottom = bottom;

  if (direction.includes("w")) {
    left = clamp(
      startBounds.x + deltaX,
      constraints.margin,
      nextRight - minWidth
    );
  }

  if (direction.includes("e")) {
    nextRight = clamp(
      right + deltaX,
      left + minWidth,
      constraints.viewportWidth - constraints.margin
    );
  }

  if (direction.includes("n")) {
    top = clamp(
      startBounds.y + deltaY,
      constraints.margin,
      nextBottom - minHeight
    );
  }

  if (direction.includes("s")) {
    nextBottom = clamp(
      bottom + deltaY,
      top + minHeight,
      constraints.viewportHeight - constraints.margin
    );
  }

  return {
    x: left,
    y: top,
    width: nextRight - left,
    height: nextBottom - top,
  };
}
