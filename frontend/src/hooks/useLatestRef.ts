import { type RefObject, useRef } from "react";

/**
 * A ref that always holds the latest `value`, updated during render.
 *
 * For effects/callbacks that should be keyed on stable derived strings while
 * still reading the freshest object: polling pages replace proto objects with
 * fresh identities every tick, so depending on the object directly would
 * re-run the effect (and refetch) per tick even when nothing changed.
 */
export function useLatestRef<T>(value: T): RefObject<T> {
  const ref = useRef(value);
  ref.current = value;
  return ref;
}
