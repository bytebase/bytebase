import { useEffect, useRef, useState } from "react";

/**
 * Tracks the container element's available height so the inner table
 * can size its scroll area to fill the panel without spilling. React
 * counterpart of Vue's `useAutoHeightDataTable` — without naive-ui.
 *
 * Returns a `ref` to attach to the outer container plus the observed
 * `maxHeight` in pixels (0 until the first measurement).
 */
export function useTableContainerHeight() {
  const ref = useRef<HTMLDivElement>(null);
  const [maxHeight, setMaxHeight] = useState(0);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new ResizeObserver((entries) => {
      const next = Math.round(entries[0]?.contentRect.height ?? 0);
      setMaxHeight((current) => (current === next ? current : next));
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return { ref, maxHeight };
}
