import { useMemo, useRef } from "react";
import type { TreeNode } from "./schemaTree";

const DOUBLE_CLICK_WINDOW_MS = 250;

type PendingState = {
  timeout: ReturnType<typeof setTimeout>;
  node: TreeNode;
};

export type ClickHandlers = {
  readonly onSingleClick: (node: TreeNode) => void;
  readonly onDoubleClick: (node: TreeNode) => void;
};

export type UseClickEventsResult = {
  handleClick: (node: TreeNode) => void;
};

/**
 * Single/double-click discriminator for the schema tree.
 *
 * Previously this routed clicks through an `Emittery` instance and the
 * consumer subscribed via `useEffect`. Emittery's `emit` is async — its
 * implementation `await resolvedPromise` before invoking listeners, so the
 * actual `toggleNode` work was deferred to a microtask after every click.
 * Combined with the downstream Vue-watch / React-render / react-arborist
 * recompute, that microtask hop pushed the paint out far enough that
 * users saw "no response until I move the mouse" (the next input event
 * was what gave the browser a chance to flush). Plain callbacks invoked
 * synchronously from `handleClick` keep all the work in the same tick.
 *
 * The single-click handler ALWAYS fires immediately (instant expand),
 * for every node type. Only `table` and `view` rows have a distinct
 * double-click action (`selectAllFromTableOrView`); for those, a second
 * click within `DOUBLE_CLICK_WINDOW_MS` runs that action. Because the
 * first click already expanded the node, we do not defer or re-toggle —
 * this avoids the 250ms per-click lag the previous timer-based approach
 * imposed on every table/view click.
 */
export function useClickEvents(handlers: ClickHandlers): UseClickEventsResult {
  // Refresh-on-render ref so the discriminator always invokes the latest
  // handlers (which capture the latest `expandedKeys`, etc.) without
  // having to re-create the discriminator state on every parent render.
  const handlersRef = useRef(handlers);
  handlersRef.current = handlers;

  return useMemo(() => {
    let pending: PendingState | undefined;

    const clear = () => {
      if (!pending) return;
      clearTimeout(pending.timeout);
      pending = undefined;
    };

    // Arms a short window during which a second click on the same node
    // counts as a double-click. The single-click action already ran on
    // the first click, so this only schedules window expiry — it does
    // not defer any work.
    const armDoubleClickWindow = (node: TreeNode) => {
      pending = {
        timeout: setTimeout(() => {
          pending = undefined;
        }, DOUBLE_CLICK_WINDOW_MS),
        node,
      };
    };

    const handleClick = (node: TreeNode) => {
      const type = node.meta.type;
      const hasDistinctDoubleClick = type === "table" || type === "view";

      if (!hasDistinctDoubleClick) {
        clear();
        handlersRef.current.onSingleClick(node);
        return;
      }

      // Second click within the window → run the distinct double-click
      // action (e.g. SELECT *). The first click already expanded the
      // node, so we don't re-toggle it here.
      if (pending && pending.node.key === node.key) {
        clear();
        handlersRef.current.onDoubleClick(node);
        return;
      }

      // First click: act IMMEDIATELY so expanding a table/view feels
      // instant. The previous implementation deferred this by 250ms to
      // disambiguate a double-click, which made every table click laggy.
      clear();
      handlersRef.current.onSingleClick(node);
      armDoubleClickWindow(node);
    };

    return { handleClick };
  }, []);
}
