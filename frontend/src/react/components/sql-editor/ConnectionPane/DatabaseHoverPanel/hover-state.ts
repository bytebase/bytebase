import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
} from "react";
import type { Position, SQLEditorTreeNode } from "@/types";

export type HoverState = {
  node?: SQLEditorTreeNode;
};

export type HoverStateUpdate = (
  value: HoverState | undefined,
  direction: "before" | "after",
  overrideDelay?: number
) => void;

export type HoverStateContextValue = {
  state: HoverState | undefined;
  position: Position;
  setPosition: (position: Position) => void;
  update: HoverStateUpdate;
};

const DELAY_BEFORE = 1000;
const DELAY_AFTER = 100;

const HoverStateContext = createContext<HoverStateContextValue | null>(null);

/**
 * React equivalent of the Vue `useHoverStateContext("connection-pane")`.
 * Wraps a delayed-update `HoverState` plus the last cursor `Position` so the
 * hover panel can render a database preview after the open-delay elapses.
 */
export function useProvideHoverState(): HoverStateContextValue {
  const [state, setState] = useState<HoverState | undefined>(undefined);
  const [position, setPosition] = useState<Position>({ x: 0, y: 0 });
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cancel = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const update = useCallback<HoverStateUpdate>(
    (value, direction, overrideDelay) => {
      const delay =
        overrideDelay ?? (direction === "before" ? DELAY_BEFORE : DELAY_AFTER);
      cancel();
      if (delay > 0) {
        timerRef.current = setTimeout(() => {
          setState(value);
        }, delay);
      } else {
        setState(value);
      }
    },
    [cancel]
  );

  // Memoize the context value so an unrelated parent re-render (e.g.
  // a Pinia tick triggering a `useVueState` update somewhere up the
  // tree) doesn't churn the `<HoverStateProvider value={…}>` reference.
  // Without this, every consumer of `useHoverState()` (every tree row
  // in `ConnectionPane`) re-renders, which can show up as the hover
  // panel rapidly toggling visibility / re-positioning while the user
  // holds the cursor over a row.
  return useMemo(
    () => ({ state, position, setPosition, update }),
    [state, position, setPosition, update]
  );
}

export const HoverStateProvider = HoverStateContext.Provider;

export function useHoverState(): HoverStateContextValue {
  const value = useContext(HoverStateContext);
  if (!value) {
    throw new Error(
      "useHoverState must be used within <HoverStateProvider value={useProvideHoverState()}>"
    );
  }
  return value;
}
