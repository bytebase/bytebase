import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";
import {
  type DelayedValueUpdate,
  useDelayedValue,
} from "@/react/hooks/useDelayedValue";
import type { Position } from "@/types";

/**
 * The hover-state shape consumed by SchemaPane's HoverPanel. Mirrors
 * `src/views/sql-editor/AsidePanel/SchemaPane/HoverPanel/hover-state.ts`
 * — a sparse path through the schema tree where the most-specific
 * populated field decides which `*Info` card the panel renders.
 */
export type HoverState = {
  database: string;
  schema?: string;
  table?: string;
  externalTable?: string;
  view?: string;
  column?: string;
  partition?: string;
};

export type HoverStateContextValue = {
  state: HoverState | undefined;
  position: Position;
  setPosition: (position: Position) => void;
  update: DelayedValueUpdate<HoverState | undefined>;
  cancel: () => void;
};

const DELAY_BEFORE = 1000;
const DELAY_AFTER = 100;

const HoverStateContext = createContext<HoverStateContextValue | null>(null);

/**
 * Mount under SchemaPane to provide hover-state to descendant `HoverPanel`
 * + tree nodes. Scoped to this surface — not a global context — so we
 * don't pay re-render costs for unrelated parts of the editor.
 */
export function HoverStateProvider({ children }: { children: ReactNode }) {
  const {
    value: state,
    update,
    cancel,
  } = useDelayedValue<HoverState | undefined>(undefined, {
    delayBefore: DELAY_BEFORE,
    delayAfter: DELAY_AFTER,
  });
  const [position, setPositionState] = useState<Position>({ x: 0, y: 0 });

  const setPosition = useCallback((next: Position) => {
    setPositionState(next);
  }, []);

  const value = useMemo<HoverStateContextValue>(
    () => ({ state, position, setPosition, update, cancel }),
    [state, position, setPosition, update, cancel]
  );

  return (
    <HoverStateContext.Provider value={value}>
      {children}
    </HoverStateContext.Provider>
  );
}

export function useHoverState(): HoverStateContextValue {
  const value = useContext(HoverStateContext);
  if (!value) {
    throw new Error("useHoverState must be used inside <HoverStateProvider>");
  }
  return value;
}
