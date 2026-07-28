import type { Dispatch, SetStateAction } from "react";
import { useState } from "react";
import { useOnKeyChange } from "./useOnKeyChange";

/**
 * `useState` that re-seeds itself during render whenever `seedKey` changes.
 * Single-state convenience over `useOnKeyChange` — reach for that hook
 * directly when one key change must reset several states at once.
 */
export function useSeededState<T>(
  seedKey: string,
  seed: () => T
): [T, Dispatch<SetStateAction<T>>] {
  const [state, setState] = useState<T>(seed);
  useOnKeyChange(seedKey, () => setState(seed()));
  return [state, setState];
}
