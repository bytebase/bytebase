import Emittery from "emittery";

export type CloseTabAction =
  | "CLOSE"
  | "CLOSE_OTHERS"
  | "CLOSE_TO_THE_RIGHT"
  | "CLOSE_ALL";

/**
 * Module-level singleton emittery used by `BatchQuerySelect`'s
 * right-click context menu (mounted once per database tab) to broadcast
 * close-tab actions back to the parent. Lives outside any component so
 * the menu's child trigger and the parent's listener can communicate
 * without a context provider.
 *
 * The emitter is *only* used by `BatchQuerySelect` for its database
 * tabs. The inner per-query-context tabs in `ResultPanel.tsx` use a
 * local `onSelect` prop instead — keeping the channel single-consumer
 * avoids cross-listener fan-out when both strips are visible.
 */
export type ResultTabEvents = {
  "close-tab": { index: number; action: CloseTabAction };
};

// Annotated rather than inferred: Emittery's default type argument names the
// listenerAdded and listenerRemoved symbols, which it does not export, so tsc
// cannot write the inferred type into a declaration file (TS4023).
export const resultTabEvents: Emittery<ResultTabEvents> =
  new Emittery<ResultTabEvents>();
