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
 * For most node types both single- and double-click do the same thing
 * (`toggleNode`), so we skip the discriminator and fire the single-click
 * handler immediately. Only `table` and `view` rows have a distinct
 * double-click action (`selectAllFromTableOrView`) — those keep the
 * 250ms timer.
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

    const queueSingle = (node: TreeNode) => {
      pending = {
        timeout: setTimeout(() => {
          pending = undefined;
          handlersRef.current.onSingleClick(node);
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

      if (pending && pending.node.key === node.key) {
        clear();
        handlersRef.current.onDoubleClick(node);
        return;
      }
      clear();
      queueSingle(node);
    };

    return { handleClick };
  }, []);
}
