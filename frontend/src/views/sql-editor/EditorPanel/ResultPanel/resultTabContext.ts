import Emittery from "emittery";

export type CloseTabAction =
  | "CLOSE"
  | "CLOSE_OTHERS"
  | "CLOSE_TO_THE_RIGHT"
  | "CLOSE_ALL";

/**
 * Module-level singleton replacing the per-instance Vue provide/inject in
 * `./context.ts`. The React `ContextMenu` (mounted per-tab inside
 * `BatchQuerySelect`) emits on this channel; `BatchQuerySelect` subscribes
 * to it to handle close-tab actions.
 *
 * Base UI's `ContextMenuTrigger` handles right-click positioning natively, so
 * the previous `{ x, y, index }` state from the Vue context is no longer
 * needed — only the close-tab event bus survives.
 */
export const resultTabEvents = new Emittery<{
  "close-tab": { index: number; action: CloseTabAction };
}>();
