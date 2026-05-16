import { useEffect, useState } from "react";
import { MOBILE_BREAKPOINT_PX } from "../constants";

export type PlanDetailLayoutMode = "NONE" | "DESKTOP" | "MOBILE";

export function useLayoutMode(pageHost: HTMLElement | null) {
  const [layoutMode, setLayoutMode] = useState<PlanDetailLayoutMode>("NONE");
  const [containerWidth, setContainerWidth] = useState(0);

  useEffect(() => {
    if (!pageHost) {
      setLayoutMode("NONE");
      setContainerWidth(0);
      return;
    }

    const updateLayout = () => {
      const width = pageHost.getBoundingClientRect().width;
      const mode: PlanDetailLayoutMode =
        width === 0
          ? "NONE"
          : width < MOBILE_BREAKPOINT_PX
            ? "MOBILE"
            : "DESKTOP";
      setContainerWidth(width);
      setLayoutMode(mode);
    };

    updateLayout();
    const observer = new ResizeObserver(() => updateLayout());
    observer.observe(pageHost);

    return () => observer.disconnect();
  }, [pageHost]);

  return {
    layoutMode,
    containerWidth,
  };
}
