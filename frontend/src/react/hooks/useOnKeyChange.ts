import { useState } from "react";

/**
 * Runs `onChange` during render whenever `key` changes — the render-time reset
 * idiom (see React docs "storing information from previous renders" and
 * BYT-9763). Resetting state here, instead of in a post-paint effect, means
 * the first paint after a key change already shows the reseeded state rather
 * than flashing the previous key's state and patching it up a frame later.
 *
 * `onChange` should only call state setters (of the same component) and update
 * refs; React re-renders immediately with the new state.
 */
export function useOnKeyChange(key: string, onChange: () => void): void {
  const [trackedKey, setTrackedKey] = useState(key);
  if (key !== trackedKey) {
    setTrackedKey(key);
    onChange();
  }
}
