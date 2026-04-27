import Emittery from "emittery";
import { useMemo } from "react";
import type { TreeNode } from "./schemaTree";

const DELAY = 250;

type ClickEvents = Emittery<{
  "single-click": { node: TreeNode };
  "double-click": { node: TreeNode };
}>;

type PendingState = {
  timeout: ReturnType<typeof setTimeout>;
  node: TreeNode;
};

export type UseClickEventsResult = {
  events: ClickEvents;
  handleClick: (node: TreeNode) => void;
};

/**
 * React port of `src/views/sql-editor/AsidePanel/SchemaPane/click.ts`.
 *
 * 250ms single/double-click discriminator. Same Emittery API surface,
 * but the timing state lives in a closure (returned via `useMemo`) so
 * it persists across renders without resubscribing.
 */
export function useClickEvents(): UseClickEventsResult {
  return useMemo(() => {
    let pending: PendingState | undefined;
    const events: ClickEvents = new Emittery();

    const clear = () => {
      if (!pending) return;
      clearTimeout(pending.timeout);
      pending = undefined;
    };

    const queue = (node: TreeNode) => {
      pending = {
        timeout: setTimeout(() => {
          events.emit("single-click", { node });
          clear();
        }, DELAY),
        node,
      };
    };

    const handleClick = (node: TreeNode) => {
      if (pending && pending.node.key === node.key) {
        events.emit("double-click", { node });
        clear();
        return;
      }
      clear();
      queue(node);
    };

    return { events, handleClick };
  }, []);
}
