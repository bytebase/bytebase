import { useEffect, useState } from "react";

/**
 * Returns the whole seconds since beginMS, refreshed once per second, or
 * undefined when beginMS is unset.
 */
export function useElapsedSeconds(
  beginMS: number | undefined
): number | undefined {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!beginMS) return;
    setNow(Date.now());
    const handle = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(handle);
  }, [beginMS]);
  if (!beginMS) return undefined;
  return Math.max(0, Math.floor((now - beginMS) / 1000));
}
